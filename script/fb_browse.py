#!/usr/bin/env python3
"""
Browse Facebook like a real person — scroll, pause, move the mouse, occasionally Like.

Uses your own account from accounts.csv. No comments, shares, or mass actions.
Good for keeping an account active when you don't log in often.

Examples:
  python fb_browse.py --id 1
  python fb_browse.py --duration 600 --max-likes 3
  python fb_browse.py --duration 300 --like-chance 0.05 --no-keep-open
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
from selenium.webdriver.common.action_chains import ActionChains
from selenium.webdriver.common.keys import Keys

from fb_login import FacebookLogin

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

HOME_URL = "https://www.facebook.com/"


class FacebookBrowser(FacebookLogin):
    def __init__(self):
        super().__init__()
        self.likes_this_session = 0
        self.posts_viewed = 0
        self.scrolls = 0

    def ensure_logged_in(self, account: dict, auto_verify: bool = True) -> str:
        logger.info("Ensuring login for %s ...", account["gmail"])
        result = self.login(account, auto_verify=auto_verify)
        logger.info("Login result: %s", result)
        return result

    def _human_pause(self, lo: float = 1.5, hi: float = 5.0) -> None:
        time.sleep(random.uniform(lo, hi))

    def _dismiss_overlays(self) -> None:
        """Cookie consent + common blocking dialogs."""
        self.handle_cookie_consent()
        # Review audience (from fb_post flow)
        try:
            page = self.driver.page_source
            if "Review audience" in page or "Choose who can see this and future posts" in page:
                for xpath in (
                    "//*[@role='button' and normalize-space()='Continue']",
                    "//span[normalize-space()='Continue']/ancestor::*[@role='button'][1]",
                ):
                    for btn in self.driver.find_elements(By.XPATH, xpath):
                        if btn.is_displayed():
                            self.driver.execute_script("arguments[0].click();", btn)
                            logger.info("Dismissed overlay (Continue)")
                            time.sleep(1)
                            return
        except Exception:
            pass
        try:
            ActionChains(self.driver).send_keys(Keys.ESCAPE).perform()
            time.sleep(0.3)
        except Exception:
            pass

    def _random_mouse_wiggle(self) -> None:
        """Small random mouse moves within the viewport."""
        try:
            body = self.driver.find_element(By.TAG_NAME, "body")
            actions = ActionChains(self.driver)
            actions.move_to_element_with_offset(
                body,
                random.randint(80, 400),
                random.randint(80, 400),
            )
            for _ in range(random.randint(2, 5)):
                actions.move_by_offset(
                    random.randint(-60, 60),
                    random.randint(-40, 40),
                )
                actions.pause(random.uniform(0.05, 0.2))
            actions.perform()
        except Exception:
            pass

    def _human_scroll(self) -> None:
        """Scroll down (usually) or occasionally up, in uneven steps."""
        direction = 1
        if random.random() < 0.12:
            direction = -1  # scroll back up sometimes

        total = random.randint(120, 520) * direction
        steps = random.randint(3, 8)
        chunk = total // steps
        for _ in range(steps):
            jitter = random.randint(-30, 30)
            amount = chunk + jitter
            self.driver.execute_script(f"window.scrollBy(0, {amount});")
            time.sleep(random.uniform(0.08, 0.35))
        self.scrolls += 1

    def _visible_feed_posts(self):
        """Rough feed post containers (articles in the main column)."""
        posts = []
        selectors = (
            "//div[@role='main']//div[@role='article']",
            "//div[@role='feed']//div[@role='article']",
            "//div[contains(@data-pagelet,'FeedUnit')]//div[@role='article']",
            "//div[@role='article']",
        )
        seen = set()
        for xpath in selectors:
            try:
                for el in self.driver.find_elements(By.XPATH, xpath):
                    if not el.is_displayed():
                        continue
                    key = el.id or el.location.get("y", 0)
                    if key in seen:
                        continue
                    seen.add(key)
                    rect = el.rect
                    if rect.get("height", 0) < 80:
                        continue
                    posts.append(el)
            except Exception:
                continue
        return posts

    def _scroll_post_into_view(self, post) -> None:
        try:
            self.driver.execute_script(
                "arguments[0].scrollIntoView({block: 'center', behavior: 'instant'});",
                post,
            )
            time.sleep(random.uniform(0.5, 1.2))
        except Exception:
            pass

    def _read_post(self, post) -> None:
        """Pause on a post as if reading."""
        self._scroll_post_into_view(post)
        self.posts_viewed += 1
        read_time = random.uniform(3, 12)
        logger.info("Reading a post (~%.0fs)...", read_time)
        # Small mouse move while "reading"
        if random.random() < 0.6:
            self._random_mouse_wiggle()
        time.sleep(read_time)

    def _find_like_button(self, post):
        """Like button inside a post that is not already active."""
        xpaths = (
            ".//*[@role='button' and @aria-label='Like']",
            ".//*[@role='button' and starts-with(@aria-label, 'Like')]",
            ".//div[@aria-label='Like' and @role='button']",
        )
        for xpath in xpaths:
            try:
                for btn in post.find_elements(By.XPATH, xpath):
                    if not btn.is_displayed():
                        continue
                    label = (btn.get_attribute("aria-label") or "").lower()
                    # Skip "Unlike", "Remove Like", reaction picker
                    if "unlike" in label or "remove" in label:
                        continue
                    if btn.get_attribute("aria-pressed") == "true":
                        continue
                    return btn
            except Exception:
                continue
        return None

    def _maybe_like_post(self, post, like_chance: float, max_likes: int) -> bool:
        if self.likes_this_session >= max_likes:
            return False
        if random.random() > like_chance:
            return False

        btn = self._find_like_button(post)
        if not btn:
            return False

        try:
            self.driver.execute_script("arguments[0].scrollIntoView({block:'center'});", btn)
            time.sleep(random.uniform(0.3, 0.8))
            self._random_mouse_wiggle()
            time.sleep(random.uniform(0.2, 0.5))
            try:
                btn.click()
            except Exception:
                self.driver.execute_script("arguments[0].click();", btn)
            self.likes_this_session += 1
            logger.info("Liked a post (%d/%d this session)", self.likes_this_session, max_likes)
            time.sleep(random.uniform(1, 2.5))
            return True
        except Exception as exc:
            logger.warning("Could not like post: %s", exc)
            return False

    def browse_feed(
        self,
        duration_sec: int = 300,
        like_chance: float = 0.07,
        max_likes: int = 4,
    ) -> None:
        logger.info("Opening home feed for ~%ds session...", duration_sec)
        self.driver.get(HOME_URL)
        time.sleep(4)
        self._dismiss_overlays()
        time.sleep(1)

        deadline = time.time() + duration_sec
        action_weights = ["scroll", "scroll", "scroll", "read", "read", "mouse", "idle"]

        while time.time() < deadline:
            self._dismiss_overlays()
            action = random.choice(action_weights)
            remaining = int(deadline - time.time())
            logger.info("Action: %s (%ds left)", action, remaining)

            if action == "scroll":
                self._human_scroll()
                self._human_pause(1.0, 3.0)

            elif action == "mouse":
                self._random_mouse_wiggle()
                self._human_pause(0.8, 2.0)

            elif action == "idle":
                # Just sit on the page
                self._human_pause(2.0, 6.0)

            elif action == "read":
                posts = self._visible_feed_posts()
                if posts:
                    post = random.choice(posts[: min(8, len(posts))])
                    self._read_post(post)
                    self._maybe_like_post(post, like_chance, max_likes)
                else:
                    self._human_scroll()
                    self._human_pause(1.0, 2.0)

            # Occasional extra scroll between actions
            if random.random() < 0.35:
                self._human_scroll()

        logger.info(
            "Session done — scrolls=%d, posts_viewed=%d, likes=%d",
            self.scrolls,
            self.posts_viewed,
            self.likes_this_session,
        )

    def run_browse(
        self,
        duration_sec: int = 300,
        like_chance: float = 0.07,
        max_likes: int = 4,
        account_id: Optional[str] = None,
        email: Optional[str] = None,
        index: int = 0,
        use_proxy: bool = True,
        headless: bool = False,
        keep_open: bool = True,
        auto_verify: bool = True,
        profile_dir: Optional[str] = None,
    ) -> bool:
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
            logger.error("Login failed (%s)", login_result)
            ok = False
        else:
            self._dismiss_overlays()
            try:
                self.browse_feed(
                    duration_sec=duration_sec,
                    like_chance=like_chance,
                    max_likes=max_likes,
                )
                ok = True
            except KeyboardInterrupt:
                logger.info("Browse interrupted by user")
                ok = True

        if keep_open and not headless:
            logger.info("Browser left open — press Ctrl+C when done.")
            try:
                while True:
                    time.sleep(1)
            except KeyboardInterrupt:
                logger.info("Closing browser...")
        else:
            time.sleep(2)

        self.close_driver()
        return ok


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Browse Facebook feed with human-like scrolling and occasional Likes."
    )
    parser.add_argument(
        "--duration",
        type=int,
        default=300,
        help="How long to browse in seconds (default: 300 = 5 minutes)",
    )
    parser.add_argument(
        "--like-chance",
        type=float,
        default=0.07,
        help="Probability of liking when reading a post (default: 0.07)",
    )
    parser.add_argument(
        "--max-likes",
        type=int,
        default=4,
        help="Maximum likes per session (default: 4)",
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

    bot = FacebookBrowser()
    ok = bot.run_browse(
        duration_sec=args.duration,
        like_chance=max(0.0, min(1.0, args.like_chance)),
        max_likes=max(0, args.max_likes),
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
