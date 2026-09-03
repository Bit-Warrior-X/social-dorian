#!/usr/bin/env python3
"""
Facebook bulk account creator — Selenium automation for the signup form.

Reads account rows from accounts.csv, opens Chrome/Chromium (optionally via
SOCKS proxy), fills https://www.facebook.com/r.php, and submits signup.

Prerequisites:
  - Google Chrome or Chromium + matching chromedriver on PATH
  - accounts.csv with id,name,password,gmail,gmail_password,birthday,gender,proxy
    Optional fb_id,fb_url — if both are set for a row, that account is skipped (already created)
  - Gmail App Password in gmail_password if email verification is required later
    (use fb_login.py + verification_code.py after signup)

Usage:
  python fb_creator.py

  # From Python:
  from fb_creator import FacebookBulkCreator
  creator = FacebookBulkCreator()
  creator.run_creation_process(max_accounts=1, headless=False)

Return values from create_facebook_account():
  success              — landed on welcome/home
  verification_needed  — confirmemail / verify URL (complete with fb_login.py)
  submitted            — form sent, outcome unclear
  error                — form missing, submit failed, or exception

Related scripts in this project:
  fb_login.py   — log in + email confirmation + save fb_id/fb_url
  fb_post.py    — publish post.txt to your timeline
  fb_reply.py   — comment on a post URL using reply.txt
  fb_browse.py  — human-like feed browsing (scroll, occasional Like)

Screenshots on failure: form_not_visible.png, error_<email>.png
"""

import time
import random
import os
import logging
import csv
import tempfile
import json
from selenium import webdriver
from selenium.webdriver.chrome.service import Service as ChromeService
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.common.exceptions import TimeoutException, NoSuchElementException
from selenium.webdriver.support.ui import Select
import subprocess
import re
import urllib.parse

# Set up logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

# --- CONFIGURATION ---
DATA_FILE = 'accounts.csv'  # Single CSV file with all account rows


class FacebookBulkCreator:
    """
    Automates Facebook registration for every row in accounts.csv.

    One browser session per account. Each account can use its own proxy from
    the CSV. The driver is quit after each attempt so the next row starts fresh.
    """

    def __init__(self):
        self.accounts = []  # Parsed rows from accounts.csv
        self.driver = None  # Active Selenium WebDriver (one account at a time)

    # ------------------------------------------------------------------ DATA
    def load_data(self):
        """
        Load account rows from CSV that still need signup.

        CSV columns:
          id, name, password, gmail, gmail_password, birthday, gender, proxy
          [, fb_id, fb_url]  — optional; if both are set, the row is skipped
                              (account already created / profile known)

        birthday format: DD/MM/YYYY (e.g. 15/01/1990)
        proxy format: socks5://user:pass@host:port

        Also splits name into first_name / last_name and birthday into day/month/year.
        """
        try:
            with open(DATA_FILE, 'r', encoding='utf-8') as f:
                # Detect delimiter (comma or semicolon)
                first_line = f.readline()
                f.seek(0)
                
                if ';' in first_line and ',' not in first_line:
                    delimiter = ';'
                else:
                    delimiter = ','
                
                reader = csv.DictReader(f, delimiter=delimiter)
                skipped_existing = 0
                
                for row in reader:
                    # Parse birthday (format: DD/MM/YYYY)
                    birthday_parts = row.get('birthday', '').strip().split('/')
                    fb_id = (row.get('fb_id') or '').strip()
                    fb_url = (row.get('fb_url') or '').strip()
                    
                    account = {
                        'id': row.get('id', '').strip(),
                        'name': row.get('name', '').strip(),
                        'password': row.get('password', '').strip(),
                        'gmail': row.get('gmail', '').strip(),
                        'gmail_password': row.get('gmail_password', '').strip(),
                        'birthday': row.get('birthday', '').strip(),
                        'gender': row.get('gender', '').strip(),
                        'proxy': row.get('proxy', '').strip(),
                        'fb_id': fb_id,
                        'fb_url': fb_url,
                        # Parse birthday components
                        'day': birthday_parts[0] if len(birthday_parts) >= 1 and birthday_parts[0] else '15',
                        'month': self.get_month_name(birthday_parts[1]) if len(birthday_parts) >= 2 else 'January',
                        'year': birthday_parts[2] if len(birthday_parts) >= 3 else '1990',
                        'first_name': row.get('name', '').strip().split()[0] if row.get('name', '').strip() else 'User',
                        'last_name': ' '.join(row.get('name', '').strip().split()[1:]) if len(row.get('name', '').strip().split()) > 1 else 'Doe'
                    }
                    
                    # Validate required fields
                    if not account['name'] or not account['gmail'] or not account['password']:
                        logger.warning(f"Skipping incomplete account: {account.get('gmail') or account}")
                        continue

                    # Already created — fb_login saved profile id/url
                    if fb_id and fb_url:
                        skipped_existing += 1
                        logger.info(
                            "Skipping id=%s (%s) — already has fb_id=%s",
                            account['id'],
                            account['gmail'],
                            fb_id,
                        )
                        continue
                    
                    self.accounts.append(account)
            
            if skipped_existing:
                logger.info(
                    "Skipped %d already-created account(s) (fb_id + fb_url present)",
                    skipped_existing,
                )
            logger.info(f"Loaded {len(self.accounts)} account(s) to create from {DATA_FILE}")
            
            # Log first account for verification (without showing full password)
            if self.accounts:
                first = self.accounts[0]
                proxy_preview = (first['proxy'][:30] + '...') if first['proxy'] else '(none)'
                logger.info(
                    f"First pending: ID={first['id']}, Name={first['name']}, "
                    f"Gmail={first['gmail']}, Gender={first['gender']}, Proxy={proxy_preview}"
                )
            else:
                logger.info("No pending accounts to create.")
            
            return len(self.accounts) > 0
            
        except FileNotFoundError:
            logger.error(f"Data file {DATA_FILE} not found!")
            logger.info("Please create a CSV file with the following format:")
            logger.info("id,name,password,gmail,gmail_password,birthday,gender,proxy[,fb_id,fb_url]")
            logger.info("Example: 1,John Doe,MyPass123,john@gmail.com,gmailpass,15/01/1990,Male,socks5://user:pass@host:port")
            return False
        except Exception as e:
            logger.error(f"Error loading data: {e}")
            return False

    def get_month_name(self, month_number):
        """Convert birthday month number (1–12) to English name for the signup dropdown."""
        months = {
            '1': 'January', '01': 'January',
            '2': 'February', '02': 'February',
            '3': 'March', '03': 'March',
            '4': 'April', '04': 'April',
            '5': 'May', '05': 'May',
            '6': 'June', '06': 'June',
            '7': 'July', '07': 'July',
            '8': 'August', '08': 'August',
            '9': 'September', '09': 'September',
            '10': 'October',
            '11': 'November',
            '12': 'December'
        }
        return months.get(str(month_number).strip(), 'January')

    # --------------------------------------------------------------- PROXY
    def parse_proxy(self, proxy_string):
        """
        Parse proxy string and extract components.

        Supported: socks5://user:pass@host:port or host:port without auth.
        Returns dict with keys: type, host, port, username, password.
        """
        if not proxy_string:
            return None
            
        try:
            # Default values
            proxy_type = "socks5"
            username = None
            password = None
            host = None
            port = None
            
            # Remove protocol if present
            if '://' in proxy_string:
                proxy_type = proxy_string.split('://')[0]
                proxy_string = proxy_string.split('://')[1]
            
            # Check for authentication
            if '@' in proxy_string:
                auth_part, host_part = proxy_string.split('@', 1)
                if ':' in auth_part:
                    username, password = auth_part.split(':', 1)
                else:
                    username = auth_part
                proxy_string = host_part
            
            # Split host and port
            if ':' in proxy_string:
                host, port = proxy_string.split(':', 1)
            
            return {
                'type': proxy_type,
                'host': host,
                'port': port,
                'username': username,
                'password': password
            }
            
        except Exception as e:
            logger.error(f"Error parsing proxy {proxy_string}: {e}")
            return None

    def create_proxy_extension(self, proxy_info):
        """
        Build a temporary Chrome extension for SOCKS/HTTP proxy auth.

        Chrome cannot pass proxy username/password via CLI alone; the extension
        registers onAuthRequired and supplies credentials automatically.
        """
        try:
            # Create the manifest.json
            manifest = {
                "version": "1.0.0",
                "manifest_version": 2,
                "name": "Proxy Extension",
                "permissions": [
                    "proxy",
                    "tabs",
                    "unlimitedStorage",
                    "storage",
                    "<all_urls>",
                    "webRequest",
                    "webRequestBlocking"
                ],
                "background": {
                    "scripts": ["background.js"]
                },
                "minimum_chrome_version": "22.0.0"
            }
            
            # Create the background script
            background_js = f"""
                var config = {{
                    mode: "fixed_servers",
                    rules: {{
                        singleProxy: {{
                            scheme: "{proxy_info['type']}",
                            host: "{proxy_info['host']}",
                            port: parseInt("{proxy_info['port']}")
                        }},
                        bypassList: ["localhost", "127.0.0.1"]
                    }}
                }};
                
                chrome.proxy.settings.set({{value: config, scope: "regular"}}, function() {{}});
                
                function callbackFn(details) {{
                    return {{
                        authCredentials: {{
                            username: "{proxy_info['username']}",
                            password: "{proxy_info['password']}"
                        }}
                    }};
                }}
                
                chrome.webRequest.onAuthRequired.addListener(
                    callbackFn,
                    {{urls: ["<all_urls>"]}},
                    ['blocking']
                );
            """
            
            # Create the extension directory
            extension_dir = tempfile.mkdtemp()
            
            # Write manifest.json
            with open(os.path.join(extension_dir, "manifest.json"), "w") as f:
                json.dump(manifest, f)
            
            # Write background.js
            with open(os.path.join(extension_dir, "background.js"), "w") as f:
                f.write(background_js)
            
            logger.info(f"Created proxy extension for: {proxy_info['host']}:{proxy_info['port']}")
            return extension_dir
            
        except Exception as e:
            logger.error(f"Error creating proxy extension: {e}")
            return None

    # -------------------------------------------------------------- BROWSER
    def find_chrome_path(self):
        """Locate Chrome or Chromium binary (common Linux install paths)."""
        possible_paths = [
            '/usr/bin/google-chrome-stable',
            '/usr/bin/google-chrome',
            '/opt/google/chrome/chrome',
            '/snap/bin/google-chrome',
            '/snap/bin/chromium',
            '/usr/bin/chromium-browser',
            '/usr/bin/chromium',
        ]
        
        for path in possible_paths:
            if os.path.exists(path) and os.access(path, os.X_OK):
                logger.info(f"Found Chrome at: {path}")
                return path
        
        try:
            for cmd in ['google-chrome-stable', 'google-chrome', 'chromium-browser', 'chromium']:
                result = subprocess.run(['which', cmd], capture_output=True, text=True)
                if result.returncode == 0 and result.stdout.strip():
                    path = result.stdout.strip()
                    logger.info(f"Found Chrome via which: {path}")
                    return path
        except:
            pass
        
        logger.error("Chrome not found.")
        return None

    def get_chrome_version(self, chrome_path):
        """Get Chrome version."""
        try:
            result = subprocess.run([chrome_path, '--version'], capture_output=True, text=True)
            version = result.stdout.strip()
            logger.info(f"Chrome version: {version}")
            match = re.search(r'(\d+\.\d+\.\d+\.\d+)', version)
            if match:
                return match.group(1)
            return version
        except:
            return None

    def find_chromedriver(self, chrome_version=None):
        """Locate chromedriver executable compatible with installed Chrome."""
        possible_paths = [
            '/usr/bin/chromedriver-151',
            '/usr/bin/chromedriver',
            '/usr/local/bin/chromedriver',
            '/snap/bin/chromedriver',
            '/usr/lib/chromium-browser/chromedriver',
        ]
        
        try:
            result = subprocess.run(['which', 'chromedriver'], capture_output=True, text=True)
            if result.returncode == 0 and result.stdout.strip():
                path = result.stdout.strip()
                if path not in possible_paths:
                    possible_paths.insert(0, path)
        except:
            pass
        
        for path in possible_paths:
            if os.path.exists(path) and os.access(path, os.X_OK):
                try:
                    result = subprocess.run([path, '--version'], capture_output=True, text=True)
                    version_output = result.stdout.strip()
                    logger.info(f"Found ChromeDriver at: {path} - {version_output}")
                    return path
                except:
                    continue
        
        logger.error("ChromeDriver not found.")
        return None

    # ----------------------------------------------------------- UI HELPERS
    def handle_cookie_consent(self):
        """Dismiss Meta cookie banner ('Allow all cookies') so the form is reachable."""
        try:
            logger.info("Looking for cookie consent popup...")
            time.sleep(2)
            
            # Check if cookie consent is present
            try:
                consent_popup = self.driver.find_element(By.XPATH, "//span[contains(text(), 'Allow the use of cookies from Facebook on this browser?')]")
                if consent_popup:
                    logger.info("Cookie consent popup detected!")
            except:
                logger.info("No cookie consent popup found, continuing...")
                return True
            
            # Try multiple strategies
            try:
                buttons = self.driver.find_elements(By.XPATH, "//button[contains(text(), 'Allow all cookies')]")
                for button in buttons:
                    if button.is_displayed() and button.is_enabled():
                        logger.info("Found 'Allow all cookies' button, clicking...")
                        self.driver.execute_script("arguments[0].scrollIntoView(true);", button)
                        time.sleep(0.5)
                        self.driver.execute_script("arguments[0].click();", button)
                        time.sleep(2)
                        try:
                            self.driver.find_element(By.XPATH, "//span[contains(text(), 'Allow the use of cookies from Facebook on this browser?')]")
                            return False
                        except:
                            logger.info("Cookie popup dismissed successfully!")
                            return True
            except Exception as e:
                logger.warning(f"Strategy 1 failed: {e}")
            
            try:
                buttons = self.driver.find_elements(By.XPATH, "//div[@role='button' and contains(@aria-label, 'Allow all cookies')]")
                for button in buttons:
                    if button.is_displayed():
                        logger.info("Found 'Allow all cookies' by aria-label, clicking...")
                        self.driver.execute_script("arguments[0].scrollIntoView(true);", button)
                        time.sleep(0.5)
                        self.driver.execute_script("arguments[0].click();", button)
                        time.sleep(2)
                        try:
                            self.driver.find_element(By.XPATH, "//span[contains(text(), 'Allow the use of cookies from Facebook on this browser?')]")
                            return False
                        except:
                            return True
            except Exception as e:
                logger.warning(f"Strategy 2 failed: {e}")
            
            logger.warning("Could not click 'Allow all cookies' button")
            return False
            
        except Exception as e:
            logger.warning(f"Cookie consent handling error: {e}")
            return False

    def wait_for_form_to_be_visible(self, timeout=30):
        """Wait until cookie overlay is gone and signup text inputs appear."""
        logger.info("Waiting for form to be visible...")
        
        # Wait for the cookie popup to disappear
        try:
            WebDriverWait(self.driver, 20).until(
                EC.invisibility_of_element_located((By.XPATH, "//span[contains(text(), 'Allow the use of cookies from Facebook on this browser?')]"))
            )
            logger.info("Cookie popup is gone!")
        except:
            logger.warning("Timeout waiting for cookie popup to disappear")
            self.handle_cookie_consent()
            time.sleep(2)
        
        # Wait for the form fields
        try:
            WebDriverWait(self.driver, 15).until(
                EC.presence_of_element_located((By.XPATH, "//input[@type='text']"))
            )
            logger.info("Form is visible!")
            return True
        except:
            logger.warning("Form still not visible!")
            return False

    def simulate_typing(self, element, text, delay_range=(0.05, 0.15)):
        """Type character-by-character with random delay (less bot-like than send_keys bulk)."""
        try:
            element.clear()
            for char in str(text):
                element.send_keys(char)
                time.sleep(random.uniform(delay_range[0], delay_range[1]))
            return True
        except:
            return False

    def find_input_by_label_text(self, label_text):
        """Find signup input via associated <label for='...'> on Facebook's form."""
        try:
            # Try to find label by text
            try:
                label = self.driver.find_element(By.XPATH, f"//span[text()='{label_text}']")
                label_element = label.find_element(By.XPATH, "./ancestor::label")
                input_id = label_element.get_attribute('for')
                if input_id:
                    input_element = self.driver.find_element(By.ID, input_id)
                    if input_element and input_element.is_enabled():
                        logger.info(f"Found input for '{label_text}' via label parent")
                        return input_element
            except:
                pass
            
            # Try to find label by containing text
            try:
                label = self.driver.find_element(By.XPATH, f"//span[contains(text(), '{label_text}')]")
                label_element = label.find_element(By.XPATH, "./ancestor::label")
                input_id = label_element.get_attribute('for')
                if input_id:
                    input_element = self.driver.find_element(By.ID, input_id)
                    if input_element and input_element.is_enabled():
                        logger.info(f"Found input for '{label_text}' via contains")
                        return input_element
            except:
                pass
            
            # Try to find label directly
            try:
                label = self.driver.find_element(By.XPATH, f"//label[contains(text(), '{label_text}')]")
                input_id = label.get_attribute('for')
                if input_id:
                    input_element = self.driver.find_element(By.ID, input_id)
                    if input_element and input_element.is_enabled():
                        logger.info(f"Found input for '{label_text}' via label direct")
                        return input_element
            except:
                pass
            
            logger.warning(f"Could not find input for label: {label_text}")
            return None
            
        except Exception as e:
            logger.error(f"Error finding input for label '{label_text}': {e}")
            return None

    def fill_field_by_label(self, label_text, value):
        """Fill input field by finding the label and using its 'for' attribute."""
        try:
            input_element = self.find_input_by_label_text(label_text)
            if input_element:
                self.driver.execute_script("arguments[0].scrollIntoView(true);", input_element)
                time.sleep(0.3)
                input_element.click()
                time.sleep(0.3)
                self.simulate_typing(input_element, value)
                logger.info(f"Filled '{label_text}' with: {value[:3]}...")
                return True
            else:
                # Fallback: try to find by position
                try:
                    inputs = self.driver.find_elements(By.XPATH, "//input[@type='text']")
                    if "First name" in label_text and len(inputs) > 0:
                        inputs[0].click()
                        self.simulate_typing(inputs[0], value)
                        logger.info(f"Filled '{label_text}' using fallback (position 1)")
                        return True
                    elif "Surname" in label_text and len(inputs) > 1:
                        inputs[1].click()
                        self.simulate_typing(inputs[1], value)
                        logger.info(f"Filled '{label_text}' using fallback (position 2)")
                        return True
                except:
                    pass
                logger.warning(f"Could not find input for label: {label_text}")
                return False
        except Exception as e:
            logger.error(f"Error filling field '{label_text}': {e}")
            return False

    def close_all_dropdowns(self):
        """Close any open dropdowns by clicking on the page body."""
        try:
            self.driver.execute_script("document.body.click();")
            time.sleep(0.3)
        except Exception as e:
            logger.warning(f"Error closing dropdowns: {e}")

    def select_gender(self, gender_value):
        """Open 'Select your gender' combobox and pick Male / Female / Custom."""
        logger.info(f"Selecting gender: {gender_value}")
        
        # First, close any open dropdowns
        self.close_all_dropdowns()
        time.sleep(0.5)
        
        try:
            # Scroll to the Gender section
            try:
                gender_label = self.driver.find_element(By.XPATH, "//span[contains(text(), 'Gender')]")
                self.driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", gender_label)
                time.sleep(0.5)
                logger.info("Scrolled to Gender section")
            except:
                pass
            
            # Find the dropdown combobox
            gender_dropdown = None
            
            # Strategy 1: Find by the display text "Select your gender"
            try:
                display_span = self.driver.find_element(By.XPATH, "//span[text()='Select your gender']")
                gender_dropdown = display_span.find_element(By.XPATH, "./ancestor::div[@role='combobox']")
                logger.info("Found gender dropdown by display text 'Select your gender'")
            except:
                pass
            
            # Strategy 2: Find by aria-label containing "gender"
            if not gender_dropdown:
                try:
                    gender_dropdown = self.driver.find_element(By.XPATH, "//div[@role='combobox' and contains(@aria-label, 'Gender')]")
                    logger.info("Found gender dropdown by aria-label containing 'Gender'")
                except:
                    pass
            
            # Strategy 3: Find by aria-label "Select your gender"
            if not gender_dropdown:
                try:
                    gender_dropdown = self.driver.find_element(By.XPATH, "//div[@role='combobox' and @aria-label='Select your gender']")
                    logger.info("Found gender dropdown by aria-label 'Select your gender'")
                except:
                    pass
            
            # Strategy 4: Find any combobox after the "Gender" label
            if not gender_dropdown:
                try:
                    gender_label = self.driver.find_element(By.XPATH, "//span[contains(text(), 'Gender')]")
                    parent = gender_label.find_element(By.XPATH, "./ancestor::div[contains(@class, 'x1n2onr6')]")
                    gender_dropdown = parent.find_element(By.XPATH, ".//div[@role='combobox']")
                    logger.info("Found gender dropdown by locating combobox after Gender label")
                except:
                    pass
            
            if not gender_dropdown:
                logger.error("Could not find gender dropdown")
                return False
            
            # Click the dropdown to open it
            self.driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", gender_dropdown)
            time.sleep(0.5)
            self.driver.execute_script("arguments[0].click();", gender_dropdown)
            time.sleep(1)
            logger.info("Clicked gender dropdown, waiting for options...")
            
            # Find and click the gender option
            try:
                option = WebDriverWait(self.driver, 5).until(
                    EC.presence_of_element_located((By.XPATH, f"//div[@role='option' and text()='{gender_value}']"))
                )
                self.driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", option)
                time.sleep(0.3)
                self.driver.execute_script("arguments[0].click();", option)
                time.sleep(0.5)
                logger.info(f"Selected gender option: {gender_value}")
            except:
                # Try to find option by containing text
                options = self.driver.find_elements(By.XPATH, "//div[@role='option']")
                for option in options:
                    if gender_value in option.text:
                        self.driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", option)
                        time.sleep(0.3)
                        self.driver.execute_script("arguments[0].click();", option)
                        time.sleep(0.5)
                        logger.info(f"Selected gender option: {gender_value} (by contains)")
                        break
            
            # Close the dropdown
            self.close_all_dropdowns()
            time.sleep(0.5)
            
            return True
            
        except Exception as e:
            logger.error(f"Error selecting gender: {e}")
            return False

    def select_dropdown_by_label(self, label_text, value):
        """Select dropdown option by label text and value."""
        try:
            # Close any open dropdowns first
            self.close_all_dropdowns()
            
            # Find the combobox
            dropdown = None
            
            # Try to find by exact aria-label
            try:
                dropdown = self.driver.find_element(By.XPATH, f"//div[@role='combobox' and @aria-label='{label_text}']")
            except:
                pass
            
            # Try to find by containing aria-label
            if not dropdown:
                try:
                    dropdown = self.driver.find_element(By.XPATH, f"//div[@role='combobox' and contains(@aria-label, '{label_text}')]")
                except:
                    pass
            
            if not dropdown:
                logger.warning(f"Could not find dropdown for label: {label_text}")
                return False
            
            # Scroll to dropdown and click
            self.driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", dropdown)
            time.sleep(0.5)
            self.driver.execute_script("arguments[0].click();", dropdown)
            time.sleep(0.8)
            
            # Find and click the option
            try:
                option = self.driver.find_element(By.XPATH, f"//div[@role='option' and text()='{value}']")
                self.driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", option)
                time.sleep(0.3)
                self.driver.execute_script("arguments[0].click();", option)
                time.sleep(0.5)
                
                # Close the dropdown
                self.close_all_dropdowns()
                
                logger.info(f"Selected '{value}' for '{label_text}'")
                return True
            except:
                # Try to find by containing text
                options = self.driver.find_elements(By.XPATH, "//div[@role='option']")
                for option in options:
                    if value in option.text:
                        self.driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", option)
                        time.sleep(0.3)
                        self.driver.execute_script("arguments[0].click();", option)
                        time.sleep(0.5)
                        
                        # Close the dropdown
                        self.close_all_dropdowns()
                        
                        logger.info(f"Selected '{value}' for '{label_text}'")
                        return True
            
            logger.warning(f"Could not find option '{value}' in dropdown")
            return False
            
        except Exception as e:
            logger.error(f"Error selecting dropdown '{label_text}': {e}")
            return False

    def create_driver_with_proxy(self, proxy_string=None, use_headless=False):
        """
        Start Chrome with anti-automation flags, optional proxy, and CDP patches.

        If proxy_string includes user:pass, loads a temp auth extension.
        Performs a quick google.com load to verify the session works.
        """
        
        chrome_path = self.find_chrome_path()
        if not chrome_path:
            return None
        
        chrome_version = self.get_chrome_version(chrome_path)
        
        options = webdriver.ChromeOptions()
        options.binary_location = chrome_path
        
        # Set window size
        options.add_argument("--no-sandbox")
        options.add_argument("--disable-dev-shm-usage")
        options.add_argument("--disable-gpu")
        options.add_argument("--window-size=1280,1080")
        options.add_argument("--disable-extensions")
        options.add_argument("--disable-setuid-sandbox")
        options.add_argument("--disable-web-security")
        options.add_argument("--disable-features=VizDisplayCompositor")
        options.add_argument("--disable-software-rasterizer")
        options.add_argument("--disable-popup-blocking")
        options.add_argument("--disable-background-timer-throttling")
        options.add_argument("--disable-backgrounding-occluded-windows")
        options.add_argument("--disable-renderer-backgrounding")
        options.add_argument("--disable-notifications")
        
        options.add_argument("--disable-blink-features=AutomationControlled")
        options.add_experimental_option("excludeSwitches", ["enable-automation"])
        options.add_experimental_option('useAutomationExtension', False)
        
        if use_headless:
            options.add_argument("--headless=new")
            options.add_argument("--disable-logging")
            options.add_argument("--log-level=3")
            options.add_argument("--silent")
        
        user_agent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.7922.173 Safari/537.36"
        options.add_argument(f"--user-agent={user_agent}")
        options.add_argument("--lang=en-US,en;q=0.9")
        options.add_argument("--accept-lang=en-US,en")
        
        # Proxy handling using extension for authentication
        if proxy_string:
            logger.info(f"Using proxy: {proxy_string[:50]}...")
            
            # Parse proxy info
            proxy_info = self.parse_proxy(proxy_string)
            if proxy_info and proxy_info['username'] and proxy_info['password']:
                # Create proxy extension for authentication
                extension_dir = self.create_proxy_extension(proxy_info)
                if extension_dir:
                    # Load the extension
                    options.add_argument(f'--disable-extensions-except={extension_dir}')
                    options.add_argument(f'--load-extension={extension_dir}')
                    logger.info("Using proxy extension for authentication")
                else:
                    # Fallback: try the simple approach
                    formatted_proxy = f"{proxy_info['type']}://{proxy_info['host']}:{proxy_info['port']}"
                    options.add_argument(f'--proxy-server={formatted_proxy}')
                    logger.warning("Proxy extension creation failed, using simple proxy without auth")
            elif proxy_info:
                # No authentication needed
                options.add_argument(f'--proxy-server={proxy_string}')

        chrome_prefs = {
            "profile.default_content_settings.images": 1,
            "download.default_directory": os.path.join(os.getcwd(), "downloads"),
            "plugins.always_open_pdf_externally": True,
            "credentials_enable_service": False,
            "profile.password_manager_enabled": False,
            "profile.default_content_setting_values.notifications": 2,
            "profile.managed_default_content_settings.images": 2,
            "profile.default_content_setting_values.cookies": 1,
            "profile.block_third_party_cookies": False,
        }
        options.add_experimental_option("prefs", chrome_prefs)

        chromedriver_path = self.find_chromedriver(chrome_version)
        if not chromedriver_path:
            return None
        
        try:
            logger.info(f"Using ChromeDriver: {chromedriver_path}")
            service = ChromeService(executable_path=chromedriver_path)
            driver = webdriver.Chrome(service=service, options=options)
            
            driver.execute_cdp_cmd('Page.addScriptToEvaluateOnNewDocument', {
                'source': '''
                    Object.defineProperty(navigator, 'webdriver', {
                        get: () => undefined
                    });
                    Object.defineProperty(navigator, 'plugins', {
                        get: () => [1, 2, 3, 4, 5]
                    });
                    Object.defineProperty(navigator, 'languages', {
                        get: () => ['en-US', 'en']
                    });
                    window.chrome = {
                        runtime: {}
                    };
                    delete window.cdc_adoQpoasnfa76pfcZLmcfl_Array;
                    delete window.cdc_adoQpoasnfa76pfcZLmcfl_Promise;
                    delete window.cdc_adoQpoasnfa76pfcZLmcfl_Symbol;
                '''
            })
            
            # Test if driver works
            driver.get("https://www.google.com")
            time.sleep(2)
            logger.info("Driver created successfully!")
            return driver
            
        except Exception as e:
            logger.error(f"Failed to create driver: {e}")
            return None

    # ----------------------------------------------------------- SIGNUP FLOW
    def create_facebook_account(self, account):
        """
        Fill and submit the Facebook registration form for one account dict.

        Steps: open r.php → cookies → name/email/password/birthday/gender → Submit.
        Does NOT complete email verification; use fb_login.py afterward if needed.
        """
        
        try:
            email = account['gmail']
            password = account['password']
            first_name = account['first_name']
            last_name = account['last_name']
            day = account['day']
            month = account['month']
            year = account['year']
            gender = account['gender']
            
            logger.info(f"Navigating to Facebook signup for {email}...")
            self.driver.get("https://www.facebook.com/r.php")
            
            logger.info("Waiting for page to load...")
            time.sleep(5)
            
            # --- HANDLE COOKIE CONSENT ---
            logger.info("Checking for cookie consent...")
            cookie_consent_handled = False
            
            for attempt in range(5):
                logger.info(f"Cookie consent attempt {attempt + 1}/5...")
                if self.handle_cookie_consent():
                    cookie_consent_handled = True
                    logger.info("Cookie consent handled successfully!")
                    time.sleep(3)
                    break
                time.sleep(2)
            
            if not cookie_consent_handled:
                logger.warning("Could not handle cookie consent automatically. Waiting 10 seconds...")
                time.sleep(10)
            
            # --- WAIT FOR FORM ---
            if not self.wait_for_form_to_be_visible():
                logger.error("Form is not visible after waiting!")
                self.driver.save_screenshot("form_not_visible.png")
                return "error"
            
            # --- FILL FORM FIELDS ---
            
            # 1. First Name
            logger.info(f"Filling First name: {first_name}")
            self.fill_field_by_label("First name", first_name)
            time.sleep(0.5)
            
            # 2. Last Name / Surname
            logger.info(f"Filling Surname: {last_name}")
            self.fill_field_by_label("Surname", last_name)
            time.sleep(0.5)
            
            # 3. Mobile number or email address
            logger.info(f"Filling Email: {email}")
            self.fill_field_by_label("Mobile number or email address", email)
            time.sleep(0.5)
            
            # 4. Password
            logger.info("Filling Password...")
            self.fill_field_by_label("Password", password)
            time.sleep(0.5)
            
            # 5. Birthday
            logger.info(f"Selecting birthday: {day}/{month}/{year}")
            
            # Day
            self.select_dropdown_by_label("Select day", day)
            time.sleep(0.5)
            
            # Month
            self.select_dropdown_by_label("Select month", month)
            time.sleep(0.5)
            
            # Year
            self.select_dropdown_by_label("Select year", year)
            time.sleep(0.5)
            
            # 6. Gender
            logger.info(f"Selecting gender: {gender}")
            gender_selected = self.select_gender(gender)
            
            if not gender_selected:
                logger.warning("Gender selection failed, trying alternative...")
                try:
                    elements = self.driver.find_elements(By.XPATH, "//*[contains(text(), 'Select your gender')]")
                    for elem in elements:
                        try:
                            dropdown = elem.find_element(By.XPATH, "./ancestor::div[@role='combobox']")
                            if dropdown:
                                self.driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", dropdown)
                                time.sleep(0.5)
                                self.driver.execute_script("arguments[0].click();", dropdown)
                                time.sleep(1)
                                
                                option = self.driver.find_element(By.XPATH, f"//div[@role='option' and text()='{gender}']")
                                self.driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", option)
                                time.sleep(0.3)
                                self.driver.execute_script("arguments[0].click();", option)
                                time.sleep(0.5)
                                
                                self.close_all_dropdowns()
                                logger.info(f"Gender selected: {gender} (alternative)")
                                gender_selected = True
                                break
                        except:
                            continue
                except Exception as e:
                    logger.error(f"Alternative gender selection also failed: {e}")
            
            # Close any open dropdowns
            self.close_all_dropdowns()
            time.sleep(0.5)
            
            # 7. Submit the form
            logger.info("Submitting form...")
            try:
                submit_button = self.driver.find_element(By.XPATH, "//span[contains(text(), 'Submit')]/ancestor::div[@role='button']")
                self.driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", submit_button)
                time.sleep(0.5)
                self.driver.execute_script("arguments[0].click();", submit_button)
                logger.info(f"Account creation submitted for {email}")
            except Exception as e:
                logger.error(f"Failed to click submit: {e}")
                try:
                    submit_button = self.driver.find_element(By.XPATH, "//div[@role='button']//span[contains(text(), 'Submit')]")
                    self.driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", submit_button)
                    time.sleep(0.5)
                    self.driver.execute_script("arguments[0].click();", submit_button)
                    logger.info(f"Account creation submitted for {email}")
                except Exception as e2:
                    logger.error(f"Alternative submit also failed: {e2}")
                    return "error"
            
            # Wait for the page to load after submission
            time.sleep(3)
            
            current_url = self.driver.current_url
            logger.info(f"Current URL after signup: {current_url}")
            
            if "confirm" in current_url or "verify" in current_url:
                logger.info(f"Verification needed for {email}")
                return "verification_needed"
            elif "welcome" in current_url or "home" in current_url:
                logger.info(f"Account created successfully for {email}")
                return "success"
            else:
                logger.info(f"Account submitted, current URL: {current_url}")
                return "submitted"
                
        except Exception as e:
            logger.error(f"Error creating account: {str(e)}")
            try:
                self.driver.save_screenshot(f"error_{email.replace('@', '_')}.png")
            except:
                pass
            return "error"

    # -------------------------------------------------------------- MAIN LOOP
    def run_creation_process(self, max_accounts=None, headless=False):
        """
        Process accounts from CSV sequentially.

        Args:
            max_accounts: Limit rows to process (None = all rows in CSV).
            headless: True = no visible browser window.

        For each row: test proxy → open browser → signup → quit → pause 15–30s.
        """
        
        if not self.load_data():
            return
        
        # Determine how many accounts to process
        if max_accounts is None or max_accounts > len(self.accounts):
            max_accounts = len(self.accounts)
        
        logger.info(f"Will process {max_accounts} account(s) from {len(self.accounts)} total")
        
        for i in range(max_accounts):
            try:
                account = self.accounts[i]
                email = account['gmail']
                
                logger.info(f"\n{'='*60}")
                logger.info(f"Processing Account {i+1}/{max_accounts}: {email}")
                logger.info(f"Name: {account['name']}")
                logger.info(f"Gender: {account['gender']}")
                logger.info(f"Birthday: {account['birthday']}")
                if account['proxy']:
                    logger.info(f"Proxy: {account['proxy'][:50]}...")
                logger.info(f"{'='*60}")
                
                # Test proxy if available
                if account['proxy']:
                    logger.info("Testing proxy connection...")
                    test_driver = self.create_driver_with_proxy(proxy_string=account['proxy'], use_headless=True)
                    if test_driver:
                        logger.info("Proxy test successful!")
                        test_driver.quit()
                    else:
                        logger.warning("Proxy test failed. Trying without proxy...")
                        account['proxy'] = None
                
                # Create driver with proxy
                self.driver = self.create_driver_with_proxy(
                    proxy_string=account.get('proxy'),
                    use_headless=headless
                )
                
                if not self.driver:
                    logger.error("Failed to create driver. Skipping...")
                    continue
                
                # Create Facebook account
                result = self.create_facebook_account(account)
                
                logger.info(f"Result for {email}: {result}")
                
                # Close driver
                if self.driver:
                    self.driver.quit()
                    self.driver = None
                
                # Wait between accounts
                if i < max_accounts - 1:
                    delay = random.uniform(15, 30)
                    logger.info(f"Waiting {delay:.0f} seconds before next account...")
                    time.sleep(delay)
                
            except Exception as e:
                logger.error(f"Failed to process account {i+1}: {str(e)}")
                if self.driver:
                    self.driver.quit()
                    self.driver = None
                continue

        logger.info("Bulk creation process finished.")
        logger.info(f"Processed {max_accounts} account(s)")


def main():
    """
    Entry point when run as: python fb_creator.py

    Edit the arguments below, or import FacebookBulkCreator from another script.
    """
    creator = FacebookBulkCreator()
    # headless=False — show the browser (recommended while debugging selectors)
    # max_accounts=None — process every row in accounts.csv; set to 1 for a test run
    creator.run_creation_process(max_accounts=None, headless=False)


if __name__ == "__main__":
    main()