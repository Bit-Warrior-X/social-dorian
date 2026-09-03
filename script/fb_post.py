#!/usr/bin/env python3
"""
Create a Facebook post from a text file using your own account.

Flow:
  1) Login via accounts.csv (same as fb_login.py)
  2) Open Facebook home
  3) Open the create-post composer
  4) Paste content from post.txt
  5) Click Post and save a confirmation screenshot

post.txt format (optional Title line):
  Title: My headline

  Body paragraph one...
  Body paragraph two...

Examples:
  python fb_post.py
  python fb_post.py --message-file post.txt --id 2
  python fb_post.py --no-proxy --no-keep-open
"""

from __future__ import annotations

import argparse
import logging
import os
import random
import re
import time
from typing import Optional, Tuple

from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.common.action_chains import ActionChains

from fb_login import FacebookLogin

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

DEFAULT_MESSAGE_FILE = "post.txt"
HOME_URL = "https://www.facebook.com/"


class FacebookPoster(FacebookLogin):
    def load_post_file(self, path: str) -> Optional[Tuple[Optional[str], str]]:
        """
        Return (title, body_text_to_publish).

        Title line is optional metadata. It is only prepended when the body
        does NOT already start with the same title (avoids HealthcareHealthcare).
        """
        try:
            with open(path, "r", encoding="utf-8") as f:
                raw = f.read().strip()
            if not raw:
                logger.error("Post file is empty: %s", path)
                return None

            lines = raw.splitlines()
            title = None
            body_lines = lines
            if lines and re.match(r"(?i)^title\s*:", lines[0]):
                title = re.sub(r"(?i)^title\s*:", "", lines[0]).strip()
                body_lines = lines[1:]
                while body_lines and not body_lines[0].strip():
                    body_lines = body_lines[1:]

            body = "\n".join(body_lines).strip()
            if not body and not title:
                logger.error("No post content in %s", path)
                return None

            if not body:
                text = title or ""
            elif title:
                # Body already opens with the title → publish body only
                body_start = body.lstrip()
                if body_start.lower().startswith(title.lower()):
                    text = body
                    logger.info("Title already present in body — not prepending")
                else:
                    # Blank line between title and body (Facebook keeps \\n from insertText)
                    text = f"{title}\n\n{body}"
            else:
                text = body

            logger.info(
                "Loaded post from %s (title=%r, %d chars to publish)",
                path,
                title,
                len(text),
            )
            return title, text
        except FileNotFoundError:
            logger.error("Post file not found: %s", path)
            return None
        except Exception as exc:
            logger.error("Could not read %s: %s", path, exc)
            return None

    def _is_login_page(self) -> bool:
        url = self.driver.current_url.lower()
        if "login" in url or "checkpoint" in url or "confirmemail" in url:
            return True
        try:
            email_el, pass_el = self._find_login_fields()
            if email_el and pass_el and email_el.is_displayed() and pass_el.is_displayed():
                return True
        except Exception:
            pass
        return False

    def ensure_logged_in(self, account: dict, auto_verify: bool = True) -> str:
        logger.info("Ensuring login for %s ...", account["gmail"])
        result = self.login(account, auto_verify=auto_verify)
        logger.info("Login result: %s", result)
        return result

    def _click_create_post_entry(self) -> bool:
        """Open the create-post dialog from home."""
        # Close leftover modals first
        try:
            ActionChains(self.driver).send_keys(Keys.ESCAPE).perform()
            time.sleep(0.5)
        except Exception:
            pass

        labels = (
            "Create a post",
            "What's on your mind",
            "What's on your mind?",
            "Make Post",
            "Create post",
        )
        xpaths = []
        for label in labels:
            xpaths.extend(
                [
                    f"//*[@role='button' and contains(@aria-label, \"{label}\")]",
                    f"//span[contains(text(), \"{label}\")]/ancestor::*[@role='button'][1]",
                    f"//div[@role='button' and contains(., \"What's on your mind\")]",
                    f"//*[@role='textbox' and contains(@aria-label, \"What's on your mind\")]",
                ]
            )
        # Floating compose (pencil) button bottom-right on some layouts
        xpaths.extend(
            [
                "//*[@role='button' and @aria-label='Create post']",
                "//*[@aria-label='Create a post' and @role='button']",
                "//div[@aria-label='Create a post']",
            ]
        )

        for xpath in xpaths:
            try:
                for el in self.driver.find_elements(By.XPATH, xpath):
                    if el.is_displayed():
                        self.driver.execute_script(
                            "arguments[0].scrollIntoView({block:'center'});", el
                        )
                        time.sleep(0.2)
                        try:
                            el.click()
                        except Exception:
                            self.driver.execute_script("arguments[0].click();", el)
                        logger.info("Opened create-post via: %s", xpath[:80])
                        time.sleep(2)
                        return True
            except Exception:
                continue

        # Name-based: "What's on your mind, Anton?"
        try:
            for el in self.driver.find_elements(
                By.XPATH,
                "//*[contains(@aria-label, \"What's on your mind\") or "
                "contains(text(), \"What's on your mind\")]",
            ):
                if el.is_displayed():
                    clickable = el
                    try:
                        clickable = el.find_element(
                            By.XPATH, "./ancestor::*[@role='button'][1]"
                        )
                    except Exception:
                        pass
                    self.driver.execute_script("arguments[0].click();", clickable)
                    logger.info("Opened create-post via What's on your mind")
                    time.sleep(2)
                    return True
        except Exception:
            pass

        return False

    def _dismiss_review_audience_dialog(self, timeout: float = 8) -> bool:
        """
        Dismiss Facebook's 'Review audience' / posts+reels merge announcement
        that blocks the create-post composer (Continue button).
        """
        markers = (
            "Review audience",
            "Choose who can see this and future posts and reels",
            "All video posts are now reels",
            "Post and reel default audience merge",
        )
        deadline = time.time() + timeout
        seen = False
        while time.time() < deadline:
            page = ""
            try:
                page = self.driver.page_source
            except Exception:
                pass
            if any(m in page for m in markers):
                seen = True
                break
            # Also detect by visible heading
            try:
                for el in self.driver.find_elements(
                    By.XPATH, "//*[contains(text(), 'Review audience')]"
                ):
                    if el.is_displayed():
                        seen = True
                        break
            except Exception:
                pass
            if seen:
                break
            time.sleep(0.4)

        if not seen:
            return False

        logger.info("Review audience dialog detected — clicking Continue")
        for xpath in (
            "//*[@role='dialog']//*[@role='button' and normalize-space()='Continue']",
            "//*[@role='dialog']//div[@aria-label='Continue']",
            "//*[@role='button' and normalize-space()='Continue']",
            "//span[normalize-space()='Continue']/ancestor::*[@role='button'][1]",
            "//div[@role='button' and .//span[normalize-space()='Continue']]",
        ):
            try:
                for btn in self.driver.find_elements(By.XPATH, xpath):
                    if not btn.is_displayed():
                        continue
                    try:
                        btn.click()
                    except Exception:
                        self.driver.execute_script("arguments[0].click();", btn)
                    time.sleep(1.5)
                    logger.info("Dismissed Review audience dialog")
                    return True
            except Exception:
                continue

        logger.warning("Review audience dialog present but Continue not clicked")
        return False

    def _find_post_composer(self, timeout: int = 15):
        """Find the contenteditable box inside the create-post dialog."""
        # Audience merge dialog often appears right as composer opens
        self._dismiss_review_audience_dialog(timeout=5)

        labels = (
            "What's on your mind",
            "Create a post",
            "Write something",
            "Say something",
            "Write a post",
        )
        deadline = time.time() + timeout
        while time.time() < deadline:
            self._dismiss_review_audience_dialog(timeout=1)
            for label in labels:
                xpaths = (
                    f"//*[@role='dialog']//*[@role='textbox' and contains(@aria-label, \"{label}\")]",
                    f"//*[@role='dialog']//*[@contenteditable='true' and contains(@aria-label, \"{label}\")]",
                    f"//*[@role='textbox' and contains(@aria-label, \"{label}\")]",
                    f"//*[@contenteditable='true' and contains(@aria-label, \"{label}\")]",
                    f"//*[@role='dialog']//*[@role='textbox' and @contenteditable='true']",
                    f"//*[@role='dialog']//div[@contenteditable='true']",
                )
                for xpath in xpaths:
                    try:
                        for el in self.driver.find_elements(By.XPATH, xpath):
                            if el.is_displayed() and el.is_enabled():
                                return el
                    except Exception:
                        continue
            time.sleep(0.5)
        return None

    def _type_post(self, box, text: str) -> bool:
        try:
            # Make sure blocking dialogs are gone before interacting
            self._dismiss_review_audience_dialog(timeout=3)

            self.driver.execute_script("arguments[0].scrollIntoView({block:'center'});", box)
            time.sleep(0.2)
            # Prefer JS focus — Selenium click is often intercepted by overlays
            try:
                box.click()
            except Exception:
                logger.info("Direct click intercepted — focusing composer via JS")
                self.driver.execute_script(
                    "arguments[0].focus(); arguments[0].click();", box
                )
            time.sleep(0.3)

            # insertText is reliable for long multi-line posts in contenteditable
            ok = self.driver.execute_script(
                """
                const el = arguments[0];
                const text = arguments[1];
                el.focus();
                el.textContent = '';
                const inserted = document.execCommand('insertText', false, text);
                el.dispatchEvent(new InputEvent('input', {bubbles: true, data: text}));
                return inserted || (el.textContent || '').length > 0;
                """,
                box,
                text,
            )
            time.sleep(0.8)
            current = (box.text or box.get_attribute("textContent") or "").strip()
            if not current:
                logger.info("JS insert left composer empty — typing via ActionChains")
                actions = ActionChains(self.driver)
                actions.click(box)
                chunk = 40
                for i in range(0, len(text), chunk):
                    actions.send_keys(text[i : i + chunk])
                    actions.pause(random.uniform(0.05, 0.12))
                actions.perform()
                time.sleep(0.5)
                current = (box.text or box.get_attribute("textContent") or "").strip()

            if not current:
                logger.error("Failed to put text into post composer")
                return False
            logger.info("Composer has %d characters", len(current))
            return True
        except Exception as exc:
            logger.error("Failed to type post: %s", exc)
            return False

    def _click_post_button(self) -> bool:
        """Click the blue Post button in the create dialog."""
        xpaths = (
            "//*[@role='dialog']//*[@role='button' and (@aria-label='Post' or normalize-space()='Post')]",
            "//*[@role='dialog']//div[@aria-label='Post' and @role='button']",
            "//*[@role='button' and @aria-label='Post']",
            "//span[text()='Post']/ancestor::*[@role='button'][1]",
            "//div[@role='button' and .//span[text()='Post']]",
        )
        for xpath in xpaths:
            try:
                for el in self.driver.find_elements(By.XPATH, xpath):
                    if not el.is_displayed():
                        continue
                    disabled = el.get_attribute("aria-disabled")
                    if disabled in ("true", "True"):
                        # Wait for enable after text entered
                        for _ in range(15):
                            time.sleep(0.3)
                            if el.get_attribute("aria-disabled") not in ("true", "True"):
                                break
                        if el.get_attribute("aria-disabled") in ("true", "True"):
                            continue
                    try:
                        el.click()
                    except Exception:
                        self.driver.execute_script("arguments[0].click();", el)
                    logger.info("Clicked Post button")
                    return True
            except Exception:
                continue
        return False

    def _dialog_still_open(self) -> bool:
        try:
            dialogs = self.driver.find_elements(By.XPATH, "//*[@role='dialog']")
            for d in dialogs:
                if d.is_displayed():
                    # create-post dialog usually mentions Post / audience
                    txt = (d.text or "").lower()
                    if "post" in txt or "what's on your mind" in txt or "create" in txt:
                        return True
        except Exception:
            pass
        return False

    def confirm_post_published(self, snippet: str) -> bool:
        """Wait for dialog to close and optionally find the text on the feed."""
        ts = time.strftime("%Y%m%d_%H%M%S")
        screenshot = f"post_confirmed_{ts}.png"

        for _ in range(20):
            time.sleep(0.5)
            if not self._dialog_still_open():
                break

        # Give the feed a moment to refresh
        time.sleep(2)
        try:
            self.driver.get(HOME_URL)
            time.sleep(4)
            self.handle_cookie_consent()
        except Exception:
            pass

        found = False
        needle = " ".join(snippet.split())[:60]
        if needle:
            try:
                for el in self.driver.find_elements(
                    By.XPATH, f"//*[contains(normalize-space(.), {self._xpath_literal(needle)})]"
                ):
                    if el.is_displayed() and needle in " ".join((el.text or "").split()):
                        try:
                            self.driver.execute_script(
                                """
                                arguments[0].scrollIntoView({block:'center'});
                                arguments[0].style.outline = '3px solid #2563eb';
                                """,
                                el,
                            )
                        except Exception:
                            pass
                        found = True
                        logger.info("Found published post text on feed")
                        break
            except Exception as exc:
                logger.warning("Feed search error: %s", exc)

        try:
            self.driver.save_screenshot(screenshot)
            logger.info("Saved confirmation screenshot: %s", screenshot)
        except Exception as exc:
            logger.warning("Screenshot failed: %s", exc)

        if not found:
            logger.warning(
                "Could not locate post text on feed yet — check %s (dialog closed=%s)",
                screenshot,
                not self._dialog_still_open(),
            )
            # Still treat as success if the composer dialog closed after Post
            return not self._dialog_still_open()
        time.sleep(2)
        return True

    @staticmethod
    def _xpath_literal(s: str) -> str:
        if "'" not in s:
            return f"'{s}'"
        if '"' not in s:
            return f'"{s}"'
        parts = s.split("'")
        return "concat(" + ", \"'\", ".join(f"'{p}'" for p in parts) + ")"

    def create_post(self, text: str) -> bool:
        logger.info("Opening Facebook home...")
        self.driver.get(HOME_URL)
        time.sleep(4)
        self.handle_cookie_consent()
        time.sleep(1)

        if self._is_login_page():
            logger.error("Not logged in — cannot create post")
            return False

        if not self._click_create_post_entry():
            logger.error("Could not open create-post composer")
            try:
                self.driver.save_screenshot("post_composer_missing.png")
            except Exception:
                pass
            return False

        # Facebook often shows "Review audience" over the composer
        self._dismiss_review_audience_dialog(timeout=8)
        time.sleep(0.5)

        box = self._find_post_composer(timeout=15)
        if not box:
            logger.error("Post text box not found")
            try:
                self.driver.save_screenshot("post_textbox_missing.png")
            except Exception:
                pass
            return False

        logger.info("Typing post content (%d chars)...", len(text))
        if not self._type_post(box, text):
            return False

        time.sleep(1)
        if not self._click_post_button():
            logger.error("Post button not found / not clickable")
            try:
                self.driver.save_screenshot("post_button_missing.png")
            except Exception:
                pass
            return False

        logger.info("Post clicked — confirming publish...")
        return self.confirm_post_published(text[:80])

    def run_post(
        self,
        message_file: str = DEFAULT_MESSAGE_FILE,
        account_id: Optional[str] = None,
        email: Optional[str] = None,
        index: int = 0,
        use_proxy: bool = True,
        headless: bool = False,
        keep_open: bool = True,
        auto_verify: bool = True,
        profile_dir: Optional[str] = None,
    ) -> bool:
        loaded = self.load_post_file(message_file)
        if not loaded:
            return False
        _title, text = loaded

        if not self.load_data():
            return False
        account = self.find_account(account_id=account_id, email=email, index=index)
        if not account:
            return False

        proxy = account.get("proxy") if use_proxy else None
        if profile_dir is None:
            safe = re.sub(r"[^a-zA-Z0-9_.-]", "_", account["gmail"])
            profile_dir = os.path.join(os.getcwd(), "chrome_profiles", safe)

        logger.info("Starting browser for %s ...", account["gmail"])
        self.driver = self.create_driver(
            proxy_string=proxy,
            headless=headless,
            user_data_dir=profile_dir,
        )
        if not self.driver:
            return False

        login_result = self.ensure_logged_in(account, auto_verify=auto_verify)
        if login_result not in ("success", "checkpoint", "verification_needed"):
            logger.error("Cannot post — login failed (%s)", login_result)
            ok = False
        else:
            self.handle_cookie_consent()
            if self._is_login_page():
                logger.error("Still on login page after ensure_logged_in")
                ok = False
            else:
                ok = self.create_post(text)

        if keep_open and not headless:
            logger.info("Browser left open — press Ctrl+C when done.")
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

        return ok


def main() -> int:
    parser = argparse.ArgumentParser(description="Publish a Facebook post from a text file.")
    parser.add_argument(
        "--message-file",
        default=DEFAULT_MESSAGE_FILE,
        help=f"Text file to post (default: {DEFAULT_MESSAGE_FILE})",
    )
    parser.add_argument("--id", dest="account_id", help="Account id from accounts.csv")
    parser.add_argument("--email", help="Account email from accounts.csv")
    parser.add_argument("--index", type=int, default=0)
    parser.add_argument("--no-proxy", action="store_true")
    parser.add_argument("--headless", action="store_true")
    parser.add_argument("--no-keep-open", action="store_true")
    parser.add_argument("--no-verify", action="store_true")
    parser.add_argument("--profile-dir", default=None)
    args = parser.parse_args()

    bot = FacebookPoster()
    ok = bot.run_post(
        message_file=args.message_file,
        account_id=args.account_id,
        email=args.email,
        index=args.index,
        use_proxy=not args.no_proxy,
        headless=args.headless,
        keep_open=not args.no_keep_open,
        auto_verify=not args.no_verify,
        profile_dir=args.profile_dir,
    )
    return 0 if ok else 1


if __name__ == "__main__":
    raise SystemExit(main())
