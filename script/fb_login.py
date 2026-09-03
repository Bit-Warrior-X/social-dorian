#!/usr/bin/env python3
"""
Facebook login helper — uses your local Google Chrome / Chromium.

Loads accounts from accounts.csv (same format as fb_creator.py) and logs in
with a visible browser window. Optionally fetches an email verification code
from Gmail via verification_code.py.

Examples:
  python fb_login.py
  python fb_login.py --id 1
  python fb_login.py --email volkodavanton89@gmail.com
  python fb_login.py --no-proxy --keep-open
"""

from __future__ import annotations

import argparse
import csv
import json
import logging
import os
import random
import re
import subprocess
import tempfile
import time
import urllib.parse
from typing import Optional, Tuple

from selenium import webdriver
from selenium.common.exceptions import TimeoutException
from selenium.webdriver.chrome.service import Service as ChromeService
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

DATA_FILE = "accounts.csv"
LOGIN_URL = "https://www.facebook.com/login"
PROFILE_ME_URL = "https://www.facebook.com/me"


class FacebookLogin:
    def __init__(self):
        self.accounts = []
        self.driver = None

    # ------------------------------------------------------------------ data
    def load_data(self) -> bool:
        try:
            with open(DATA_FILE, "r", encoding="utf-8") as f:
                first_line = f.readline()
                f.seek(0)
                delimiter = ";" if ";" in first_line and "," not in first_line else ","
                reader = csv.DictReader(f, delimiter=delimiter)

                for row in reader:
                    account = {
                        "id": row.get("id", "").strip(),
                        "name": row.get("name", "").strip(),
                        "password": row.get("password", "").strip(),
                        "gmail": row.get("gmail", "").strip(),
                        "gmail_password": row.get("gmail_password", "").strip(),
                        "proxy": row.get("proxy", "").strip(),
                    }
                    if not account["gmail"] or not account["password"]:
                        logger.warning("Skipping incomplete row: %s", account.get("gmail"))
                        continue
                    self.accounts.append(account)

            logger.info("Loaded %d account(s) from %s", len(self.accounts), DATA_FILE)
            return bool(self.accounts)
        except FileNotFoundError:
            logger.error("%s not found", DATA_FILE)
            return False
        except Exception as exc:
            logger.error("Error loading data: %s", exc)
            return False

    def find_account(
        self,
        account_id: Optional[str] = None,
        email: Optional[str] = None,
        index: int = 0,
    ) -> Optional[dict]:
        if account_id:
            for acc in self.accounts:
                if acc["id"] == str(account_id):
                    return acc
            logger.error("No account with id=%s", account_id)
            return None
        if email:
            email_l = email.lower()
            for acc in self.accounts:
                if acc["gmail"].lower() == email_l:
                    return acc
            logger.error("No account with email=%s", email)
            return None
        if 0 <= index < len(self.accounts):
            return self.accounts[index]
        logger.error("Index %d out of range (have %d accounts)", index, len(self.accounts))
        return None

    # --------------------------------------------------------------- browser
    def parse_proxy(self, proxy_string: str) -> Optional[dict]:
        if not proxy_string:
            return None
        try:
            proxy_type = "socks5"
            username = password = host = port = None
            if "://" in proxy_string:
                proxy_type, proxy_string = proxy_string.split("://", 1)
            if "@" in proxy_string:
                auth_part, host_part = proxy_string.split("@", 1)
                if ":" in auth_part:
                    username, password = auth_part.split(":", 1)
                else:
                    username = auth_part
                proxy_string = host_part
            if ":" in proxy_string:
                host, port = proxy_string.split(":", 1)
            return {
                "type": proxy_type,
                "host": host,
                "port": port,
                "username": username,
                "password": password,
            }
        except Exception as exc:
            logger.error("Error parsing proxy: %s", exc)
            return None

    def create_proxy_extension(self, proxy_info: dict) -> Optional[str]:
        try:
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
                    "webRequestBlocking",
                ],
                "background": {"scripts": ["background.js"]},
                "minimum_chrome_version": "22.0.0",
            }
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
                    callbackFn, {{urls: ["<all_urls>"]}}, ['blocking']
                );
            """
            extension_dir = tempfile.mkdtemp()
            with open(os.path.join(extension_dir, "manifest.json"), "w") as f:
                json.dump(manifest, f)
            with open(os.path.join(extension_dir, "background.js"), "w") as f:
                f.write(background_js)
            return extension_dir
        except Exception as exc:
            logger.error("Error creating proxy extension: %s", exc)
            return None

    def find_chrome_path(self) -> Optional[str]:
        possible_paths = [
            "/usr/bin/google-chrome-stable",
            "/usr/bin/google-chrome",
            "/opt/google/chrome/chrome",
            "/snap/bin/google-chrome",
            "/snap/bin/chromium",
            "/usr/bin/chromium-browser",
            "/usr/bin/chromium",
        ]
        for path in possible_paths:
            if os.path.exists(path) and os.access(path, os.X_OK):
                logger.info("Found Chrome/Chromium at: %s", path)
                return path
        for cmd in ["google-chrome-stable", "google-chrome", "chromium-browser", "chromium"]:
            try:
                result = subprocess.run(["which", cmd], capture_output=True, text=True)
                if result.returncode == 0 and result.stdout.strip():
                    path = result.stdout.strip()
                    logger.info("Found Chrome via which: %s", path)
                    return path
            except Exception:
                pass
        logger.error("Chrome/Chromium not found")
        return None

    def find_chromedriver(self) -> Optional[str]:
        possible_paths = [
            "/usr/bin/chromedriver-151",
            "/usr/bin/chromedriver",
            "/usr/local/bin/chromedriver",
            "/snap/bin/chromedriver",
            "/usr/lib/chromium-browser/chromedriver",
        ]
        try:
            result = subprocess.run(["which", "chromedriver"], capture_output=True, text=True)
            if result.returncode == 0 and result.stdout.strip():
                path = result.stdout.strip()
                if path not in possible_paths:
                    possible_paths.insert(0, path)
        except Exception:
            pass
        for path in possible_paths:
            if os.path.exists(path) and os.access(path, os.X_OK):
                logger.info("Found ChromeDriver at: %s", path)
                return path
        logger.error("ChromeDriver not found")
        return None

    def create_driver(
        self,
        proxy_string: Optional[str] = None,
        headless: bool = False,
        user_data_dir: Optional[str] = None,
    ):
        chrome_path = self.find_chrome_path()
        if not chrome_path:
            return None

        options = webdriver.ChromeOptions()
        options.binary_location = chrome_path
        options.add_argument("--no-sandbox")
        options.add_argument("--disable-dev-shm-usage")
        options.add_argument("--disable-gpu")
        options.add_argument("--window-size=1280,1080")
        options.add_argument("--disable-setuid-sandbox")
        options.add_argument("--disable-popup-blocking")
        options.add_argument("--disable-notifications")
        options.add_argument("--disable-blink-features=AutomationControlled")
        options.add_experimental_option("excludeSwitches", ["enable-automation"])
        options.add_experimental_option("useAutomationExtension", False)

        if user_data_dir:
            os.makedirs(user_data_dir, exist_ok=True)
            options.add_argument(f"--user-data-dir={user_data_dir}")
            logger.info("Using Chrome profile: %s", user_data_dir)

        if headless:
            options.add_argument("--headless=new")
            options.add_argument("--log-level=3")

        options.add_argument(
            "--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
            "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.7922.173 Safari/537.36"
        )
        options.add_argument("--lang=en-US,en;q=0.9")

        if proxy_string:
            logger.info("Using proxy: %s...", proxy_string[:50])
            proxy_info = self.parse_proxy(proxy_string)
            if proxy_info and proxy_info.get("username") and proxy_info.get("password"):
                extension_dir = self.create_proxy_extension(proxy_info)
                if extension_dir:
                    options.add_argument(f"--disable-extensions-except={extension_dir}")
                    options.add_argument(f"--load-extension={extension_dir}")
                else:
                    options.add_argument(
                        f"--proxy-server={proxy_info['type']}://{proxy_info['host']}:{proxy_info['port']}"
                    )
            elif proxy_info:
                options.add_argument(f"--proxy-server={proxy_string}")

        options.add_experimental_option(
            "prefs",
            {
                "credentials_enable_service": False,
                "profile.password_manager_enabled": False,
                "profile.default_content_setting_values.notifications": 2,
            },
        )

        chromedriver_path = self.find_chromedriver()
        if not chromedriver_path:
            return None

        try:
            service = ChromeService(executable_path=chromedriver_path)
            driver = webdriver.Chrome(service=service, options=options)
            driver.execute_cdp_cmd(
                "Page.addScriptToEvaluateOnNewDocument",
                {
                    "source": """
                        Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
                        window.chrome = {runtime: {}};
                    """
                },
            )
            logger.info("Driver created successfully")
            return driver
        except Exception as exc:
            logger.error("Failed to create driver: %s", exc)
            return None

    # --------------------------------------------------------------- helpers
    def simulate_typing(self, element, text: str, delay_range=(0.04, 0.12)) -> bool:
        try:
            element.clear()
            for char in str(text):
                element.send_keys(char)
                time.sleep(random.uniform(*delay_range))
            return True
        except Exception:
            return False

    def handle_cookie_consent(self, retries: int = 3) -> bool:
        """
        Dismiss Facebook/Meta cookie dialogs.

        Covers both the simple banner and the detailed sheet with:
          - Allow all cookies
          - Decline optional cookies
        """
        labels = (
            "Allow all cookies",
            "Allow essential and optional cookies",
            "Accept all",
            "Accept All",
            "Decline optional cookies",
        )
        dialog_markers = (
            "Allow the use of cookies from Facebook on this browser?",
            "Allow all cookies",
            "Decline optional cookies",
            "Cookies from other companies",
            "Choose cookies by category",
        )

        dismissed_any = False
        for attempt in range(retries):
            time.sleep(1.0 if attempt == 0 else 1.5)

            # Is a cookie UI present?
            present = False
            page_lower = ""
            try:
                page_lower = self.driver.page_source.lower()
            except Exception:
                pass
            for marker in dialog_markers:
                if marker.lower() in page_lower:
                    # Only treat as cookie UI if an action button is also findable
                    present = True
                    break

            clicked = False
            for text in labels:
                xpaths = (
                    f"//button[normalize-space()='{text}']",
                    f"//*[@role='button' and normalize-space()='{text}']",
                    f"//button[contains(., '{text}')]",
                    f"//*[@role='button' and contains(., '{text}')]",
                    f"//div[@aria-label='{text}']",
                    f"//span[normalize-space()='{text}']/ancestor::button[1]",
                    f"//span[normalize-space()='{text}']/ancestor::*[@role='button'][1]",
                )
                for xpath in xpaths:
                    try:
                        for btn in self.driver.find_elements(By.XPATH, xpath):
                            if not btn.is_displayed():
                                continue
                            self.driver.execute_script(
                                "arguments[0].scrollIntoView({block:'center'});", btn
                            )
                            time.sleep(0.2)
                            try:
                                btn.click()
                            except Exception:
                                self.driver.execute_script("arguments[0].click();", btn)
                            logger.info("Dismissed cookie consent via: %s", text)
                            clicked = True
                            dismissed_any = True
                            time.sleep(1.5)
                            break
                    except Exception:
                        continue
                    if clicked:
                        break
                if clicked:
                    break

            if not clicked:
                if not present or not any(t.lower() in page_lower for t in ("allow all cookies", "decline optional")):
                    if not dismissed_any:
                        logger.info("No cookie consent popup")
                    return True
                logger.warning("Cookie dialog still visible (attempt %d/%d)", attempt + 1, retries)

        return dismissed_any

    def _find_login_fields(self):
        """Return (email_input, password_input) using several Facebook layouts."""
        email_el = None
        pass_el = None

        for by, value in (
            (By.ID, "email"),
            (By.NAME, "email"),
            (By.CSS_SELECTOR, "input[name='email']"),
            (By.CSS_SELECTOR, "input[type='text'][name='email']"),
            (By.CSS_SELECTOR, "input[autocomplete='username']"),
        ):
            try:
                email_el = self.driver.find_element(by, value)
                if email_el.is_displayed():
                    break
                email_el = None
            except Exception:
                continue

        for by, value in (
            (By.ID, "pass"),
            (By.NAME, "pass"),
            (By.CSS_SELECTOR, "input[name='pass']"),
            (By.CSS_SELECTOR, "input[type='password']"),
            (By.CSS_SELECTOR, "input[autocomplete='current-password']"),
        ):
            try:
                pass_el = self.driver.find_element(by, value)
                if pass_el.is_displayed():
                    break
                pass_el = None
            except Exception:
                continue

        return email_el, pass_el

    def _click_login_button(self) -> bool:
        for xpath in (
            "//button[@name='login']",
            "//button[@id='loginbutton']",
            "//button[@type='submit']",
            "//div[@role='button' and contains(., 'Log in')]",
            "//button[contains(., 'Log in')]",
        ):
            try:
                btn = self.driver.find_element(By.XPATH, xpath)
                if btn.is_displayed():
                    self.driver.execute_script("arguments[0].click();", btn)
                    return True
            except Exception:
                continue
        return False

    # ------------------------------------------------------- verification
    def needs_email_confirmation(self) -> bool:
        """True on confirmemail.php / 'Enter the confirmation code' screens."""
        url = self.driver.current_url.lower()
        if "confirmemail" in url or "confirm_email" in url:
            return True
        try:
            for text in (
                "Enter the confirmation code",
                "Confirmation code",
                "enter the code we sent",
            ):
                els = self.driver.find_elements(By.XPATH, f"//*[contains(text(), '{text}')]")
                if any(el.is_displayed() for el in els):
                    return True
        except Exception:
            pass
        page = self.driver.page_source.lower()
        return "confirmation code" in page or "enter the code" in page

    def _find_confirmation_code_input(self):
        """Locate the confirmation-code field on confirmemail.php."""
        xpaths = (
            "//input[contains(translate(@placeholder,'ABCDEFGHIJKLMNOPQRSTUVWXYZ','abcdefghijklmnopqrstuvwxyz'),'confirmation code')]",
            "//input[contains(translate(@aria-label,'ABCDEFGHIJKLMNOPQRSTUVWXYZ','abcdefghijklmnopqrstuvwxyz'),'confirmation code')]",
            "//input[@name='code' or @id='code' or @name='approvals_code']",
            "//input[@autocomplete='one-time-code']",
            "//input[@type='text' or @type='tel' or @type='number']",
        )
        for xpath in xpaths:
            try:
                for el in self.driver.find_elements(By.XPATH, xpath):
                    if el.is_displayed() and el.is_enabled():
                        return el
            except Exception:
                continue
        return None

    def _click_by_texts(self, texts: tuple[str, ...], timeout: float = 8) -> bool:
        """Click the first visible button/link/menu row matching any of the given labels."""
        deadline = time.time() + timeout
        while time.time() < deadline:
            for text in texts:
                xpaths = (
                    f"//*[@role='button' and normalize-space()='{text}']",
                    f"//button[normalize-space()='{text}']",
                    f"//a[normalize-space()='{text}']",
                    f"//*[@role='menuitem' and contains(., '{text}')]",
                    f"//*[@role='option' and contains(., '{text}')]",
                    f"//*[@role='dialog']//*[normalize-space()='{text}']",
                    f"//*[@role='dialog']//*[contains(., '{text}')]",
                    f"//div[@aria-label='{text}']",
                    f"//*[@role='button' and contains(., '{text}')]",
                    f"//button[contains(., '{text}')]",
                    f"//a[contains(., '{text}')]",
                    f"//span[normalize-space()='{text}']/ancestor::*[@role='button'][1]",
                    f"//span[normalize-space()='{text}']/ancestor::button[1]",
                    # Facebook modal list rows (text + chevron)
                    f"//span[normalize-space()='{text}']/ancestor::div[@role='button'][1]",
                    f"//div[normalize-space()='{text}']",
                    f"//*[normalize-space()='{text}']",
                )
                for xpath in xpaths:
                    try:
                        for el in self.driver.find_elements(By.XPATH, xpath):
                            if not el.is_displayed():
                                continue
                            # Prefer a clickable ancestor for plain text nodes
                            target = el
                            try:
                                if el.tag_name.lower() in {"span", "div"} and el.get_attribute("role") not in {
                                    "button",
                                    "menuitem",
                                    "option",
                                    "link",
                                }:
                                    clickable = el.find_elements(
                                        By.XPATH,
                                        "./ancestor::*[@role='button' or @role='menuitem' or self::button or self::a][1]",
                                    )
                                    if clickable:
                                        target = clickable[0]
                            except Exception:
                                pass
                            self.driver.execute_script(
                                "arguments[0].scrollIntoView({block:'center'});", target
                            )
                            time.sleep(0.2)
                            try:
                                target.click()
                            except Exception:
                                self.driver.execute_script("arguments[0].click();", target)
                            logger.info("Clicked: %s", text)
                            return True
                    except Exception:
                        continue
            time.sleep(0.5)
        return False

    def _click_continue(self) -> bool:
        return self._click_by_texts(("Continue", "Confirm", "Submit"), timeout=5)

    def _wait_for_resend_modal(self, timeout: float = 10) -> bool:
        """Wait for the 'Haven't received the code?' dialog."""
        deadline = time.time() + timeout
        while time.time() < deadline:
            try:
                for text in ("Haven't received the code?", "Haven’t received the code?", "Resend confirmation code"):
                    els = self.driver.find_elements(By.XPATH, f"//*[contains(text(), \"{text}\")]")
                    if any(el.is_displayed() for el in els):
                        logger.info("Resend modal is visible")
                        return True
            except Exception:
                pass
            time.sleep(0.4)
        return False

    def _resend_confirmation_code(self) -> bool:
        """
        Match Facebook GUI:
          1) Click 'I didn't get the code'
          2) Wait for modal 'Haven't received the code?'
          3) Click 'Resend confirmation code'
        """
        logger.info("Clicking 'I didn't get the code'...")
        if not self._click_by_texts(
            ("I didn't get the code", "I didn’t get the code", "Didn't get the code"),
            timeout=8,
        ):
            logger.error("Could not find 'I didn't get the code' button")
            return False

        if not self._wait_for_resend_modal(timeout=10):
            logger.error("Resend modal ('Haven't received the code?') did not appear")
            return False

        time.sleep(0.5)
        logger.info("Clicking 'Resend confirmation code' in modal...")
        if not self._click_by_texts(
            ("Resend confirmation code", "Resend code"),
            timeout=10,
        ):
            logger.error("Could not find 'Resend confirmation code' in modal")
            return False

        time.sleep(2)
        logger.info("Resend requested — waiting for a new email")
        return True

    def _fetch_fb_code(self, email: str, gmail_pass: str, timeout: int, since_ts: Optional[float] = None):
        """Poll Gmail for Facebook's 5-digit confirmation code."""
        from verification_code import wait_for_verification_code, get_verification_code

        # Prefer codes that arrived after since_ts (post-resend)
        max_age = 15
        if since_ts is not None:
            max_age = max(2, int((time.time() - since_ts) / 60) + 2)

        common = dict(
            timeout=timeout,
            poll_interval=5,
            max_age_minutes=max_age,
            debug=True,
            preferred_length=5,  # confirmemail.php asks for 5-digit code
        )

        code = wait_for_verification_code(
            email,
            gmail_pass,
            sender="facebookmail.com",
            **common,
        )
        if code and len(code) == 5:
            return code
        if code:
            logger.warning("Ignoring non-5-digit code from Facebook sender: %s", code)

        logger.info("Retrying Gmail search with Facebook subject filter...")
        code = wait_for_verification_code(
            email,
            gmail_pass,
            subject_contains="Facebook",
            timeout=min(30, timeout),
            poll_interval=5,
            max_age_minutes=max_age,
            debug=True,
            preferred_length=5,
        )
        if code and len(code) == 5:
            return code

        code = get_verification_code(
            email,
            gmail_pass,
            sender="facebookmail.com",
            max_age_minutes=max_age,
            debug=True,
            preferred_length=5,
        )
        if code and len(code) == 5:
            return code

        logger.warning("No 5-digit Facebook confirmation code found")
        return None

    def _submit_confirmation_code(self, code: str) -> bool:
        if len(code) != 5 or not code.isdigit():
            logger.error("Facebook expects a 5-digit code, got %r", code)
            return False

        code_input = self._find_confirmation_code_input()
        if not code_input:
            logger.error("Could not find code input field — enter %s manually", code)
            return False

        code_input.click()
        time.sleep(0.2)
        # Clear any previous wrong attempt (red error state)
        try:
            code_input.send_keys(Keys.CONTROL, "a")
            code_input.send_keys(Keys.BACKSPACE)
        except Exception:
            try:
                code_input.clear()
            except Exception:
                pass
        time.sleep(0.2)
        self.simulate_typing(code_input, code)
        time.sleep(0.5)

        if not self._click_continue():
            try:
                code_input.send_keys(Keys.RETURN)
            except Exception:
                pass

        time.sleep(4)

        # Detect Facebook's red error: "Please check the email... 5-digit code"
        try:
            err = self.driver.find_elements(
                By.XPATH,
                "//*[contains(text(), 'Please check the email') or "
                "contains(text(), '5-digit code') or contains(text(), 'incorrect')]",
            )
            if any(el.is_displayed() for el in err) and self.needs_email_confirmation():
                logger.warning("Facebook rejected the code (error visible on page)")
                return False
        except Exception:
            pass

        return True

    def handle_email_verification(self, account: dict, timeout: int = 60) -> bool:
        """
        Pull Facebook's email code from Gmail and submit it.

        Flow:
          1) Wait up to `timeout` seconds for a code
          2) If none → click "I didn't get the code" → "Resend confirmation code"
          3) Wait again for a new code
          4) If still none → give up (return False)
        """
        if not self.needs_email_confirmation():
            return False

        logger.info("Email confirmation page detected: %s", self.driver.current_url)

        gmail_pass = (account.get("gmail_password") or "").replace(" ", "").strip()
        placeholder = not gmail_pass or gmail_pass.lower() in {"gmailpass", "password", "apppassword"}
        if placeholder:
            logger.warning(
                "Email verification required, but no real Gmail App Password in accounts.csv. "
                "Enter the 5-digit code manually in the browser."
            )
            return False

        try:
            from verification_code import wait_for_verification_code  # noqa: F401
        except ImportError:
            logger.error("verification_code.py not importable")
            return False

        email = account["gmail"]

        # --- Attempt 1: wait for existing / first email ---
        logger.info("Waiting for Facebook verification email (up to %ss)...", timeout)
        attempt_started = time.time()
        code = self._fetch_fb_code(email, gmail_pass, timeout=timeout, since_ts=None)

        # --- Attempt 2: resend then wait again ---
        if not code:
            logger.warning("No code after %ss — requesting a resend...", timeout)
            if not self._resend_confirmation_code():
                logger.error("Resend UI failed — giving up login")
                return False

            resend_ts = time.time()
            logger.info("Waiting for resent verification email (up to %ss)...", timeout)
            code = self._fetch_fb_code(email, gmail_pass, timeout=timeout, since_ts=resend_ts)

        if not code:
            logger.error("Still no verification code after resend — giving up login")
            return False

        logger.info("Got code: %s — submitting...", code)
        if not self._submit_confirmation_code(code):
            return False

        # If Facebook rejected the code / still on page, don't pretend success
        if self.needs_email_confirmation():
            logger.warning("Still on confirmation page after submitting code")
            return False

        logger.info("Email confirmation submitted (waited %.0fs total)", time.time() - attempt_started)
        return True

    def _classify_after_login(self) -> str:
        url = self.driver.current_url.lower()
        if self.needs_email_confirmation() or "confirmemail" in url:
            return "verification_needed"
        if any(x in url for x in ("checkpoint", "auth_platform", "two_step", "approvals")):
            return "checkpoint"
        if "login" in url and ("login.php" in url or url.rstrip("/").endswith("/login")):
            return "failed"
        # Home / feed / profile
        if any(x in url for x in ("facebook.com/?", "facebook.com/home", "/feed", "/profile")):
            return "success"
        if "facebook.com" in url and "login" not in url and "confirm" not in url:
            return "success"
        return "checkpoint"

    # -------------------------------------------------------- profile id/url
    def _extract_fb_id_from_text(self, text: str) -> Optional[str]:
        """Pull numeric Facebook user id from page HTML / URL text."""
        patterns = (
            r'"userID"\s*:\s*"(\d{5,})"',
            r'"USER_ID"\s*:\s*"(\d{5,})"',
            r'"actorID"\s*:\s*"(\d{5,})"',
            r'"profile_owner"\s*:\s*\{\s*"id"\s*:\s*"(\d{5,})"',
            r'"profile_id"\s*:\s*"(\d{5,})"',
            r'profile\.php\?id=(\d{5,})',
            r'"userID"\s*:\s*(\d{5,})',
            r'"actorID"\s*:\s*(\d{5,})',
        )
        for pattern in patterns:
            match = re.search(pattern, text)
            if match:
                return match.group(1)
        return None

    def _public_url_from_current(self, fb_id: Optional[str]) -> str:
        url = self.driver.current_url.split("#")[0].split("?")[0].rstrip("/")
        # If still on /me, prefer stable profile.php?id= form
        if url.rstrip("/").endswith("/me") and fb_id:
            return f"https://www.facebook.com/profile.php?id={fb_id}"
        if "profile.php" in self.driver.current_url and fb_id:
            return f"https://www.facebook.com/profile.php?id={fb_id}"
        if fb_id and re.search(r"facebook\.com/\d+$", url):
            return f"https://www.facebook.com/profile.php?id={fb_id}"
        # Vanity username URL
        if re.search(r"facebook\.com/[^/]+$", url) and "/me" not in url:
            return url
        if fb_id:
            return f"https://www.facebook.com/profile.php?id={fb_id}"
        return self.driver.current_url

    def fetch_profile_info(self) -> Tuple[Optional[str], Optional[str]]:
        """
        After login, open /me and return (fb_id, public_url).
        """
        try:
            logger.info("Fetching profile id / public URL via %s ...", PROFILE_ME_URL)
            self.driver.get(PROFILE_ME_URL)
            time.sleep(4)
            self.handle_cookie_consent()
            time.sleep(1)

            current = self.driver.current_url
            logger.info("Profile URL after /me redirect: %s", current)

            fb_id = None
            parsed = urllib.parse.urlparse(current)
            qs = urllib.parse.parse_qs(parsed.query)
            if "id" in qs and qs["id"]:
                fb_id = qs["id"][0]

            if not fb_id:
                # Numeric path: facebook.com/1000...
                m = re.search(r"facebook\.com/(\d{5,})/?$", current.split("?")[0])
                if m:
                    fb_id = m.group(1)

            if not fb_id:
                try:
                    source = self.driver.page_source
                except Exception:
                    source = ""
                fb_id = self._extract_fb_id_from_text(source)

            public_url = self._public_url_from_current(fb_id)

            if fb_id:
                logger.info("Facebook ID: %s", fb_id)
            else:
                logger.warning("Could not determine numeric Facebook ID")
            logger.info("Public URL: %s", public_url)

            return fb_id, public_url
        except Exception as exc:
            logger.error("Failed to fetch profile info: %s", exc)
            return None, None

    def save_profile_to_csv(self, account: dict, fb_id: Optional[str], public_url: Optional[str]) -> bool:
        """Update accounts.csv row with fb_id and fb_url columns."""
        if not fb_id and not public_url:
            return False
        try:
            with open(DATA_FILE, "r", encoding="utf-8", newline="") as f:
                first = f.readline()
                f.seek(0)
                delimiter = ";" if ";" in first and "," not in first else ","
                reader = csv.DictReader(f, delimiter=delimiter)
                fieldnames = list(reader.fieldnames or [])
                rows = list(reader)

            for col in ("fb_id", "fb_url"):
                if col not in fieldnames:
                    fieldnames.append(col)

            target_email = account["gmail"].lower()
            updated = False
            for row in rows:
                if (row.get("gmail") or "").strip().lower() == target_email:
                    if fb_id:
                        row["fb_id"] = fb_id
                    if public_url:
                        row["fb_url"] = public_url
                    updated = True
                    break

            if not updated:
                logger.warning("No matching row in %s for %s", DATA_FILE, account["gmail"])
                return False

            with open(DATA_FILE, "w", encoding="utf-8", newline="") as f:
                writer = csv.DictWriter(f, fieldnames=fieldnames, delimiter=delimiter, extrasaction="ignore")
                writer.writeheader()
                writer.writerows(rows)

            logger.info("Saved fb_id / fb_url to %s for %s", DATA_FILE, account["gmail"])
            account["fb_id"] = fb_id
            account["fb_url"] = public_url
            return True
        except Exception as exc:
            logger.error("Failed to update %s: %s", DATA_FILE, exc)
            return False

    # ----------------------------------------------------------------- login
    def _finish_verification_if_needed(self, account: dict, auto_verify: bool) -> Optional[str]:
        """
        If on confirmemail page, try to submit the email code.
        Returns a status string if handled, or None to continue normal flow.
        """
        if not self.needs_email_confirmation():
            return None

        logger.info("Already on email confirmation page (saved session): %s", self.driver.current_url)
        if not auto_verify:
            return "verification_needed"

        verified = self.handle_email_verification(account, timeout=60)
        if verified:
            time.sleep(2)
            # Cookie sheet often reappears right after confirming the account
            self.handle_cookie_consent()
            time.sleep(1)
            if self.needs_email_confirmation():
                logger.warning("Still on confirmation page after submitting code — giving up")
                return "failed"
            state = self._classify_after_login()
            if state == "success":
                logger.info("Login appears successful after email confirmation")
            return state

        logger.error("Email verification failed after wait + resend — giving up login")
        return "failed"

    def login(self, account: dict, auto_verify: bool = True) -> str:
        """
        Log into Facebook.

        Returns one of: success, checkpoint, verification_needed, failed, error
        """
        email = account["gmail"]
        password = account["password"]

        try:
            logger.info("Opening Facebook login for %s ...", email)
            self.driver.get(LOGIN_URL)
            time.sleep(3)
            self.handle_cookie_consent()
            time.sleep(1)

            logger.info("Current URL: %s", self.driver.current_url)

            # Saved Chrome profile may already be mid-login on confirmemail.php
            early = self._finish_verification_if_needed(account, auto_verify)
            if early is not None:
                self.handle_cookie_consent()
                return early

            # Or already fully logged in from a previous session
            already = self._classify_after_login()
            if already == "success":
                self.handle_cookie_consent()
                logger.info("Already logged in via saved Chrome profile")
                return "success"

            email_el, pass_el = self._find_login_fields()
            if not email_el or not pass_el:
                # One more check — redirect may have landed on confirm after cookies
                early = self._finish_verification_if_needed(account, auto_verify)
                if early is not None:
                    return early
                logger.error("Login form not found (URL: %s)", self.driver.current_url)
                self.driver.save_screenshot(f"login_form_missing_{email.replace('@', '_')}.png")
                return "error"

            logger.info("Filling email...")
            email_el.click()
            time.sleep(0.2)
            self.simulate_typing(email_el, email)

            logger.info("Filling password...")
            pass_el.click()
            time.sleep(0.2)
            self.simulate_typing(pass_el, password)
            time.sleep(0.4)

            if not self._click_login_button():
                pass_el.send_keys(Keys.RETURN)

            logger.info("Submitted login form, waiting...")
            time.sleep(5)

            # Cookie dialog can reappear after login
            self.handle_cookie_consent()

            logger.info("URL after login: %s", self.driver.current_url)

            early = self._finish_verification_if_needed(account, auto_verify)
            if early is not None:
                self.handle_cookie_consent()
                return early

            state = self._classify_after_login()
            self.handle_cookie_consent()
            if state == "failed":
                logger.error("Still on login page — credentials may be wrong")
                self.driver.save_screenshot(f"login_failed_{email.replace('@', '_')}.png")
            elif state == "success":
                logger.info("Login appears successful")
            elif state == "checkpoint":
                logger.warning("Checkpoint / extra verification required — check the browser")
            elif state == "verification_needed":
                logger.warning("Email confirmation still required")
            return state

        except Exception as exc:
            logger.error("Login error: %s", exc)
            try:
                self.driver.save_screenshot(f"login_error_{email.replace('@', '_')}.png")
            except Exception:
                pass
            return "error"

    def run(
        self,
        account_id: Optional[str] = None,
        email: Optional[str] = None,
        index: int = 0,
        use_proxy: bool = True,
        headless: bool = False,
        keep_open: bool = True,
        auto_verify: bool = True,
        profile_dir: Optional[str] = None,
        fetch_profile: bool = True,
    ) -> Optional[str]:
        if not self.load_data():
            return None

        account = self.find_account(account_id=account_id, email=email, index=index)
        if not account:
            return None

        proxy = account.get("proxy") if use_proxy else None
        if profile_dir is None:
            safe = re.sub(r"[^a-zA-Z0-9_.-]", "_", account["gmail"])
            profile_dir = os.path.join(os.getcwd(), "chrome_profiles", safe)

        self.driver = self.create_driver(
            proxy_string=proxy,
            headless=headless,
            user_data_dir=profile_dir,
        )
        if not self.driver:
            return None

        result = self.login(account, auto_verify=auto_verify)
        logger.info("Result for %s: %s", account["gmail"], result)

        if result == "success" and fetch_profile:
            fb_id, public_url = self.fetch_profile_info()
            if fb_id or public_url:
                self.save_profile_to_csv(account, fb_id, public_url)

        if keep_open and not headless:
            logger.info("Browser left open — press Ctrl+C in this terminal when done.")
            try:
                while True:
                    time.sleep(1)
            except KeyboardInterrupt:
                logger.info("Closing browser...")
        else:
            time.sleep(2)

        if self.driver:
            try:
                self.driver.quit()
            except Exception:
                pass
            self.driver = None

        return result


def main() -> int:
    parser = argparse.ArgumentParser(description="Log into Facebook with your local Chrome/Chromium.")
    parser.add_argument("--id", dest="account_id", help="Account id from accounts.csv")
    parser.add_argument("--email", help="Gmail / Facebook email from accounts.csv")
    parser.add_argument("--index", type=int, default=0, help="Row index if id/email not given (default: 0)")
    parser.add_argument("--no-proxy", action="store_true", help="Ignore proxy column")
    parser.add_argument("--headless", action="store_true", help="Run without a visible window")
    parser.add_argument(
        "--no-keep-open",
        action="store_true",
        help="Close the browser when login finishes (default: keep open)",
    )
    parser.add_argument(
        "--no-verify",
        action="store_true",
        help="Do not auto-fetch email verification codes",
    )
    parser.add_argument(
        "--no-profile-info",
        action="store_true",
        help="Skip fetching Facebook id / public URL after login",
    )
    parser.add_argument(
        "--profile-dir",
        default=None,
        help="Chrome user-data-dir (default: ./chrome_profiles/<email>)",
    )
    args = parser.parse_args()

    bot = FacebookLogin()
    result = bot.run(
        account_id=args.account_id,
        email=args.email,
        index=args.index,
        use_proxy=not args.no_proxy,
        headless=args.headless,
        keep_open=not args.no_keep_open,
        auto_verify=not args.no_verify,
        profile_dir=args.profile_dir,
        fetch_profile=not args.no_profile_info,
    )
    if result in ("success", "checkpoint", "verification_needed"):
        return 0
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
