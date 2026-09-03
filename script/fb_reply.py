#!/usr/bin/env python3
"""
Post a comment on a Facebook post using your own account.

Flow on every run:
  1) Start Chrome with your saved profile (chrome_profiles/<email>)
  2) Log in via accounts.csv (same logic as fb_login.py) if needed
  3) Open the post URL
  4) Post the message from reply.txt (or --message-file)

Examples:
  python fb_reply.py
  python fb_reply.py --url "https://www.facebook.com/share/p/1BucEVSSC9/" --message-file reply.txt
  python fb_reply.py --id 1 --no-proxy --no-keep-open
"""

from __future__ import annotations

import argparse
import logging
import os
import random
import re
import time
from typing import Optional

from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.common.action_chains import ActionChains
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait

from fb_login import FacebookLogin

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

DEFAULT_POST_URL = "https://www.facebook.com/share/p/1BucEVSSC9/?mibextid=wwXIfr"
DEFAULT_MESSAGE_FILE = "reply.txt"


class FacebookReply(FacebookLogin):
    def load_message(self, path: str) -> Optional[str]:
        try:
            with open(path, "r", encoding="utf-8") as f:
                text = f.read().strip()
            if not text:
                logger.error("Message file is empty: %s", path)
                return None
            logger.info("Loaded reply from %s (%d chars)", path, len(text))
            return text
        except FileNotFoundError:
            logger.error("Message file not found: %s", path)
            return None
        except Exception as exc:
            logger.error("Could not read %s: %s", path, exc)
            return None

    def _find_comment_box(self, timeout: int = 20):
        """Find the post comment composer (contenteditable / textbox)."""
        labels = (
            "Write a comment",
            "Write a comment…",
            "Write a comment...",
            "Comment",
            "Write a public comment",
            "Add a comment",
        )
        xpaths = []
        for label in labels:
            xpaths.extend(
                [
                    f"//*[@role='textbox' and @aria-label='{label}']",
                    f"//*[@contenteditable='true' and @aria-label='{label}']",
                    f"//*[@role='textbox' and contains(@aria-label, '{label}')]",
                    f"//*[@contenteditable='true' and contains(@aria-label, '{label}')]",
                    f"//div[@aria-label='{label}']",
                ]
            )
        xpaths.extend(
            [
                "//form//div[@role='textbox' and @contenteditable='true']",
                "//div[@role='textbox' and @contenteditable='true']",
                "//div[@contenteditable='true' and contains(@aria-placeholder, 'comment')]",
                "//div[@contenteditable='true' and contains(@aria-placeholder, 'Comment')]",
            ]
        )

        deadline = time.time() + timeout
        while time.time() < deadline:
            for xpath in xpaths:
                try:
                    for el in self.driver.find_elements(By.XPATH, xpath):
                        if el.is_displayed() and el.is_enabled():
                            return el
                except Exception:
                    continue
            time.sleep(0.5)
        return None

    def _click_comment_entry_point(self) -> bool:
        """Some posts need clicking 'Comment' before the box appears."""
        for text in ("Comment", "Write a comment", "Leave a comment"):
            xpaths = (
                f"//*[@role='button' and normalize-space()='{text}']",
                f"//div[@aria-label='{text}']",
                f"//*[@role='button' and contains(@aria-label, '{text}')]",
                f"//span[normalize-space()='{text}']/ancestor::*[@role='button'][1]",
            )
            for xpath in xpaths:
                try:
                    for el in self.driver.find_elements(By.XPATH, xpath):
                        if el.is_displayed():
                            self.driver.execute_script("arguments[0].click();", el)
                            logger.info("Opened comment composer via: %s", text)
                            time.sleep(1)
                            return True
                except Exception:
                    continue
        return False

    def _type_into_comment_box(self, box, message: str) -> bool:
        try:
            self.driver.execute_script("arguments[0].scrollIntoView({block:'center'});", box)
            time.sleep(0.3)
            box.click()
            time.sleep(0.3)

            # Prefer real key events so Facebook's composer registers input
            actions = ActionChains(self.driver)
            actions.click(box)
            for char in message:
                actions.send_keys(char)
                actions.pause(random.uniform(0.02, 0.08))
            actions.perform()
            time.sleep(0.5)

            # Fallback if the box stayed empty
            current = (box.text or box.get_attribute("textContent") or "").strip()
            if not current:
                logger.info("ActionChains left box empty — trying JS insert")
                self.driver.execute_script(
                    """
                    const el = arguments[0];
                    const text = arguments[1];
                    el.focus();
                    el.textContent = '';
                    document.execCommand('insertText', false, text);
                    el.dispatchEvent(new InputEvent('input', {bubbles: true, data: text}));
                    """,
                    box,
                    message,
                )
                time.sleep(0.5)
            return True
        except Exception as exc:
            logger.error("Failed to type comment: %s", exc)
            return False

    def _composer_root(self, box):
        """Closest container that holds the comment box + send button."""
        for xpath in (
            "./ancestor::form[1]",
            "./ancestor::div[@role='group'][1]",
            "./ancestor::div[@role='presentation'][1]",
        ):
            try:
                root = box.find_element(By.XPATH, xpath)
                if root:
                    return root
            except Exception:
                continue
        try:
            return box.find_element(By.XPATH, "./ancestor::div[4]")
        except Exception:
            return box

    def _dismiss_composer_popups(self) -> None:
        """Close avatar/sticker/emoji sheets that block sending."""
        try:
            ActionChains(self.driver).send_keys(Keys.ESCAPE).perform()
            time.sleep(0.4)
        except Exception:
            pass
        # Close "Make your avatar" / sticker dialogs if still open
        for text in ("Close", "Not now", "Cancel"):
            try:
                for el in self.driver.find_elements(
                    By.XPATH,
                    f"//*[@role='button' and (@aria-label='{text}' or normalize-space()='{text}')]",
                ):
                    if el.is_displayed():
                        # Only close overlays, not the main post X if possible
                        self.driver.execute_script("arguments[0].click();", el)
                        time.sleep(0.4)
                        break
            except Exception:
                continue
        try:
            ActionChains(self.driver).send_keys(Keys.ESCAPE).perform()
            time.sleep(0.3)
        except Exception:
            pass

    def _is_accessory_button(self, el) -> bool:
        """Emoji / GIF / sticker / photo buttons — never treat as Send."""
        label = (el.get_attribute("aria-label") or el.get_attribute("title") or "").lower()
        deny = (
            "sticker",
            "avatar",
            "gif",
            "emoji",
            "emoticon",
            "photo",
            "camera",
            "voice",
            "attach",
            "feeling",
            "make your",
        )
        return any(w in label for w in deny)

    def _find_send_button(self, box):
        """
        Find the blue paper-plane on the RIGHT of the comment box.
        Never pick left-side tools (avatar sticker / emoji / GIF).
        """
        # Prefer JS geometry: must sit to the right of the text box
        try:
            el = self.driver.execute_script(
                """
                const box = arguments[0];
                const br = box.getBoundingClientRect();
                const allow = new Set(['comment', 'send', 'post', 'reply']);
                const deny = /sticker|avatar|gif|emoji|emoticon|photo|camera|voice|attach|feeling|make your/i;
                const nodes = Array.from(document.querySelectorAll('[role="button"], div[aria-label]'));
                const scored = [];
                for (const b of nodes) {
                  const label = (b.getAttribute('aria-label') || '').trim();
                  const low = label.toLowerCase();
                  if (!label || deny.test(low)) continue;
                  if (!allow.has(low)) continue;
                  const r = b.getBoundingClientRect();
                  if (r.width < 1 || r.height < 1) continue;
                  // Paper plane is on the RIGHT side of the composer row
                  if (r.left < br.left + br.width * 0.55) continue;
                  if (Math.abs(r.top - br.top) > 70) continue;
                  if (r.width * r.height > 8000) continue;
                  scored.push({
                    el: b,
                    dx: r.left - br.left,
                    dy: Math.abs(r.top - br.top),
                    label
                  });
                }
                scored.sort((a, b) => a.dy - b.dy || b.dx - a.dx);
                return scored.length ? scored[0].el : null;
                """,
                box,
            )
            if el:
                logger.info(
                    "Found right-side send button aria-label=%r",
                    el.get_attribute("aria-label"),
                )
                return el
        except Exception as exc:
            logger.warning("JS send-button search failed: %s", exc)

        # Fallback: exact-label buttons to the right of the box
        exact_labels = ("Comment", "Send", "Post", "Reply")
        try:
            box_rect = box.rect
            box_mid_x = box_rect["x"] + box_rect["width"] * 0.55
            box_y = box_rect["y"]
        except Exception:
            return None

        scored = []
        for label in exact_labels:
            try:
                for el in self.driver.find_elements(
                    By.XPATH, f"//*[@role='button' and @aria-label='{label}']"
                ):
                    if not el.is_displayed() or self._is_accessory_button(el):
                        continue
                    r = el.rect
                    if r["x"] < box_mid_x:
                        continue
                    if abs(r["y"] - box_y) > 80:
                        continue
                    scored.append((abs(r["y"] - box_y), -r["x"], el))
            except Exception:
                continue
        if scored:
            scored.sort(key=lambda t: (t[0], t[1]))
            btn = scored[0][2]
            logger.info("Fallback send button aria-label=%r", btn.get_attribute("aria-label"))
            return btn
        return None

    def _composer_still_has_draft(self, box, message: str) -> bool:
        try:
            text = (box.text or box.get_attribute("textContent") or "").strip()
            snippet = " ".join(message.split())
            if not text:
                return False
            # Any leftover draft counts (even if partially corrupted)
            return True if snippet[:12] in " ".join(text.split()) or len(text) >= 3 else bool(text)
        except Exception:
            return False

    def _retype_message(self, box, message: str) -> bool:
        """Clear composer and type the message again (draft may have been corrupted)."""
        try:
            box.click()
            time.sleep(0.1)
            box.send_keys(Keys.CONTROL, "a")
            box.send_keys(Keys.BACKSPACE)
            time.sleep(0.2)
            return self._type_into_comment_box(box, message)
        except Exception as exc:
            logger.warning("Retype failed: %s", exc)
            return False

    def _click_paper_plane(self, box) -> bool:
        send_btn = self._find_send_button(box)
        if not send_btn:
            logger.warning("No right-side paper-plane / Comment send button found")
            return False
        label = send_btn.get_attribute("aria-label") or ""
        if self._is_accessory_button(send_btn):
            logger.warning("Refusing accessory button: %r", label)
            return False
        try:
            for _ in range(10):
                if send_btn.get_attribute("aria-disabled") in ("true", "True"):
                    time.sleep(0.25)
                    continue
                break
            self.driver.execute_script("arguments[0].scrollIntoView({block:'center'});", send_btn)
            time.sleep(0.2)
            try:
                send_btn.click()
            except Exception:
                self.driver.execute_script("arguments[0].click();", send_btn)
            logger.info("Clicked paper-plane send (aria-label=%r)", label)
            return True
        except Exception as exc:
            logger.warning("Paper-plane click failed: %s", exc)
            return False

    def _press_enter_to_send(self, box) -> None:
        box.click()
        time.sleep(0.1)
        self.driver.execute_script("arguments[0].focus();", box)
        # Move caret to end so Enter doesn't eat a character
        self.driver.execute_script(
            """
            const el = arguments[0];
            const range = document.createRange();
            range.selectNodeContents(el);
            range.collapse(false);
            const sel = window.getSelection();
            sel.removeAllRanges();
            sel.addRange(range);
            """,
            box,
        )
        time.sleep(0.1)
        box.send_keys(Keys.ENTER)
        logger.info("Pressed Enter to send comment")

    def _submit_comment(self, box, message: str = "") -> bool:
        """
        On this Facebook story/comment UI the blue paper-plane on the RIGHT
        is the real send control. Enter is only a fallback.
        """
        self._dismiss_composer_popups()

        # Ensure full message is still in the box (previous runs corrupted draft)
        try:
            current = " ".join((box.text or box.get_attribute("textContent") or "").split())
            expected = " ".join(message.split())
            if expected and expected not in current:
                logger.info("Draft incomplete (%r) — retyping message", current)
                if not self._retype_message(box, message):
                    return False
                time.sleep(0.4)
        except Exception:
            pass

        # 1) Primary for this UI: click right-side paper plane
        clicked = self._click_paper_plane(box)
        if clicked:
            for _ in range(15):
                time.sleep(0.35)
                if message and not self._composer_still_has_draft(box, message):
                    logger.info("Composer cleared after paper-plane click — comment sent")
                    return True

        # 2) Fallback: Enter
        self._dismiss_composer_popups()
        try:
            # Retype if needed then Enter
            if message and self._composer_still_has_draft(box, message):
                current = " ".join((box.text or "").split())
                if " ".join(message.split()) not in current:
                    self._retype_message(box, message)
            self._press_enter_to_send(box)
        except Exception as exc:
            logger.warning("Enter submit failed: %s", exc)

        for _ in range(12):
            time.sleep(0.35)
            if message and not self._composer_still_has_draft(box, message):
                logger.info("Composer cleared after Enter — comment sent")
                return True

        # 3) One more paper-plane attempt
        self._dismiss_composer_popups()
        if message and self._composer_still_has_draft(box, message):
            current = " ".join((box.text or box.get_attribute("textContent") or "").split())
            if " ".join(message.split()) not in current:
                self._retype_message(box, message)
            if self._click_paper_plane(box):
                for _ in range(12):
                    time.sleep(0.35)
                    if not self._composer_still_has_draft(box, message):
                        logger.info("Composer cleared on retry — comment sent")
                        return True

        if message and self._composer_still_has_draft(box, message):
            logger.error(
                "Draft still in comment box after send — Facebook did not accept the comment "
                "(this causes the 'Changes you made may not be saved' warning on close)"
            )
            try:
                self.driver.save_screenshot("reply_send_failed.png")
            except Exception:
                pass
            return False

        return True

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
        """
        Always run login logic when the script starts.

        Uses the same FacebookLogin.login() flow as fb_login.py:
          - saved profile may already be logged in → success
          - otherwise fills email/password and handles email confirmation
        """
        logger.info("Ensuring login for %s ...", account["gmail"])
        result = self.login(account, auto_verify=auto_verify)
        logger.info("Login result: %s", result)

        if result == "success":
            return result

        # Soft success: cookies/session may still work for commenting
        if result in ("checkpoint", "verification_needed"):
            logger.warning("Login unfinished (%s) — will try the post page anyway", result)
            return result

        return result

    def reply_to_post(self, post_url: str, message: str, account: Optional[dict] = None, auto_verify: bool = True) -> bool:
        logger.info("Opening post: %s", post_url)
        self.driver.get(post_url)
        time.sleep(5)
        self.handle_cookie_consent()
        time.sleep(1)

        logger.info("Current URL: %s", self.driver.current_url)

        # Session expired / redirected to login while opening the post
        if self._is_login_page() and account:
            logger.warning("Post page redirected to login — logging in again...")
            login_result = self.ensure_logged_in(account, auto_verify=auto_verify)
            if login_result not in ("success", "checkpoint", "verification_needed"):
                logger.error("Re-login failed (%s)", login_result)
                return False
            logger.info("Re-opening post after login...")
            self.driver.get(post_url)
            time.sleep(5)
            self.handle_cookie_consent()
            time.sleep(1)
            if self._is_login_page():
                logger.error("Still on login page after re-login — cannot comment")
                return False

        box = self._find_comment_box(timeout=8)
        if not box:
            self._click_comment_entry_point()
            box = self._find_comment_box(timeout=15)

        if not box:
            logger.error("Comment box not found — are you logged in? Is the post public?")
            try:
                self.driver.save_screenshot("reply_comment_box_missing.png")
            except Exception:
                pass
            return False

        logger.info("Typing comment...")
        if not self._type_into_comment_box(box, message):
            return False

        if not self._submit_comment(box, message=message):
            return False

        time.sleep(2)
        logger.info("Comment sent — confirming it appears on screen...")
        return self.confirm_reply_on_screen(message)

    def _expand_comments_if_needed(self) -> None:
        """Click common controls that reveal newly posted comments."""
        for text in (
            "View more comments",
            "Most relevant",
            "All comments",
            "Newest",
            "View previous comments",
        ):
            xpaths = (
                f"//*[@role='button' and contains(., '{text}')]",
                f"//span[contains(text(), '{text}')]/ancestor::*[@role='button'][1]",
                f"//div[@aria-label='{text}']",
            )
            for xpath in xpaths:
                try:
                    for el in self.driver.find_elements(By.XPATH, xpath):
                        if el.is_displayed():
                            self.driver.execute_script("arguments[0].click();", el)
                            logger.info("Clicked: %s", text)
                            time.sleep(1.2)
                            return
                except Exception:
                    continue

    def _find_visible_reply(self, message: str, timeout: int = 20):
        """Find a visible element that contains the posted reply text."""
        # Use a short unique snippet in case of whitespace differences
        snippet = " ".join(message.split())
        if not snippet:
            return None

        # XPath can't easily handle all quotes — escape by concat if needed
        def xpath_literal(s: str) -> str:
            if "'" not in s:
                return f"'{s}'"
            if '"' not in s:
                return f'"{s}"'
            parts = s.split("'")
            return "concat(" + ", \"'\", ".join(f"'{p}'" for p in parts) + ")"

        # Prefer full message match, then a shorter prefix
        needles = [snippet]
        if len(snippet) > 40:
            needles.append(snippet[:40])

        deadline = time.time() + timeout
        while time.time() < deadline:
            for needle in needles:
                lit_n = xpath_literal(needle)
                xpaths = (
                    f"//*[contains(normalize-space(.), {lit_n})]",
                    f"//div[@dir='auto' and contains(normalize-space(.), {lit_n})]",
                    f"//span[contains(normalize-space(.), {lit_n})]",
                )
                for xpath in xpaths:
                    try:
                        for el in self.driver.find_elements(By.XPATH, xpath):
                            if not el.is_displayed():
                                continue
                            text = (el.text or "").strip()
                            if needle in " ".join(text.split()):
                                # Prefer a reasonably sized node (not the whole page)
                                if len(text) < max(500, len(snippet) * 8):
                                    return el
                    except Exception:
                        continue
            time.sleep(0.6)
        return None

    def confirm_reply_on_screen(self, message: str) -> bool:
        """
        Scroll the posted reply into view, highlight it, and save a screenshot
        so you can confirm the comment landed successfully.
        """
        self._expand_comments_if_needed()
        time.sleep(1)

        el = self._find_visible_reply(message, timeout=20)
        if not el:
            # Soft refresh of comments area
            try:
                self.driver.execute_script("window.scrollBy(0, 400);")
            except Exception:
                pass
            time.sleep(1.5)
            self._expand_comments_if_needed()
            el = self._find_visible_reply(message, timeout=10)

        ts = time.strftime("%Y%m%d_%H%M%S")
        screenshot = f"reply_confirmed_{ts}.png"

        if not el:
            logger.warning("Could not find the reply text on the page yet")
            try:
                self.driver.save_screenshot(screenshot)
                logger.info("Saved page screenshot for manual check: %s", screenshot)
            except Exception as exc:
                logger.warning("Screenshot failed: %s", exc)
            return False

        try:
            self.driver.execute_script(
                """
                const el = arguments[0];
                el.scrollIntoView({block: 'center', behavior: 'instant'});
                el.style.outline = '3px solid #e11d48';
                el.style.outlineOffset = '4px';
                el.style.backgroundColor = 'rgba(225, 29, 72, 0.12)';
                """,
                el,
            )
            time.sleep(0.8)
        except Exception as exc:
            logger.warning("Could not highlight reply: %s", exc)

        try:
            self.driver.save_screenshot(screenshot)
            logger.info("Reply visible on screen — screenshot saved: %s", screenshot)
        except Exception as exc:
            logger.warning("Screenshot failed: %s", exc)

        logger.info("Confirmed reply text: %r", message)
        # Leave highlight briefly so the open browser shows it clearly
        time.sleep(2)
        return True

    def run_reply(
        self,
        post_url: str,
        message_file: str,
        account_id: Optional[str] = None,
        email: Optional[str] = None,
        index: int = 0,
        use_proxy: bool = True,
        headless: bool = False,
        keep_open: bool = True,
        auto_verify: bool = True,
        profile_dir: Optional[str] = None,
    ) -> bool:
        message = self.load_message(message_file)
        if not message:
            return False

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

        # Step 1: login every time this script runs
        login_result = self.ensure_logged_in(account, auto_verify=auto_verify)
        if login_result not in ("success", "checkpoint", "verification_needed"):
            logger.error("Cannot reply — login did not succeed (%s)", login_result)
            ok = False
        else:
            self.handle_cookie_consent()
            # Step 2: open post + comment (re-logins if Facebook kicks to login page)
            ok = self.reply_to_post(post_url, message, account=account, auto_verify=auto_verify)

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
    parser = argparse.ArgumentParser(description="Comment on a Facebook post from your account.")
    parser.add_argument(
        "--url",
        default=DEFAULT_POST_URL,
        help=f"Facebook post URL (default: {DEFAULT_POST_URL})",
    )
    parser.add_argument(
        "--message-file",
        default=DEFAULT_MESSAGE_FILE,
        help=f"Text file with the comment (default: {DEFAULT_MESSAGE_FILE})",
    )
    parser.add_argument("--id", dest="account_id", help="Account id from accounts.csv")
    parser.add_argument("--email", help="Account email from accounts.csv")
    parser.add_argument("--index", type=int, default=0, help="Row index if id/email not given")
    parser.add_argument("--no-proxy", action="store_true", help="Ignore proxy column")
    parser.add_argument("--headless", action="store_true")
    parser.add_argument("--no-keep-open", action="store_true")
    parser.add_argument("--no-verify", action="store_true")
    parser.add_argument("--profile-dir", default=None)
    args = parser.parse_args()

    bot = FacebookReply()
    ok = bot.run_reply(
        post_url=args.url,
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
