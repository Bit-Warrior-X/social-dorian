#!/usr/bin/env python3
"""
Report a Facebook post using your own logged-in account.

Facebook's report UI is multi-step. List every built-in path:

  python fb_report.py --list-reasons

report.txt examples:

  reason=Spam
  reason=Scam, fraud or false information > Sharing false information
  reason=Bullying, harassment or abuse
  reason=Adult content > Nudity or sexual activity

Examples:
  python fb_report.py --id 1
  python fb_report.py --list-reasons
  python fb_report.py --reason "Spam" --dry-run
"""

from __future__ import annotations

import argparse
import logging
import os
import re
import time
from typing import List, Optional, Tuple

from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.common.action_chains import ActionChains

from fb_login import FacebookLogin

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

DEFAULT_CONFIG_FILE = "report.txt"
DEFAULT_POST_URL = "https://www.facebook.com/share/p/1BucEVSSC9/?mibextid=wwXIfr"
DEFAULT_REASON = "Spam"

# ---------------------------------------------------------------------------
# Real report tree (from live Facebook UI walkthrough).
# Values:
#   "Submit" — click Submit, then "Thanks for reporting…" → Next → "More actions" → Done
#   "Done"   — often skips Submit; still finish with Next/Done screens if shown
#   "SKIP"   — needs typed input (name / page URL); script will refuse this path
# Nested dicts = more dialog steps.
# ---------------------------------------------------------------------------
REPORT_TREE = {
    "Problem involving someone under 18": {
        "Threatening to share my nude images": "Submit",
        "Seems like sexual exploitation": "Submit",
        "Sharing someone's nude images": "Submit",
        "Bullying or harassment": "Submit",
        "Physical abuse": "Submit",
    },
    "Bullying, harassment or abuse": {
        "Threatening to share my nude images": "Submit",
        "Seems like sexual exploitation": "Submit",
        "Seems like human trafficking": "Submit",
        "Bullying or harassment": {
            "Me": "Submit",
            "A friend": "SKIP",  # requires typing a name
            "I don't know them": "Submit",
        },
    },
    "Suicide or self-harm": {
        "Suicide or self-injury": "Submit",
        "Eating disorder": "Submit",
    },
    "Violent, hateful or disturbing content": {
        "Credible threat to safety": "Submit",
        "Seems like terrorism": "Submit",
        "Calling for violence": "Submit",
        "Seems like organised crime": {
            "It represents an organised hate group": "Submit",
            "Posting hateful speech": "Submit",
        },
        "Promoting hate": {
            "It represents an organised hate group": "Submit",
            "Posting hateful speech": "Submit",
        },
        "Showing violence, death or severe injury": "Submit",
        "Animal abuse": "Submit",
    },
    "Selling or promoting restricted items": {
        "Drugs": {
            "Highly addictive drugs, such as cocaine, heroin or fentanyl": "Submit",
            "Other drugs": "Submit",
        },
        "Weapons": "Submit",
        "Alcohol": "Submit",
        "Tobacco": "Submit",
        "Gambling": "Submit",
        "Animals": "Submit",
    },
    "Adult content": {
        "Threatening to share my nude images": "Submit",
        "Seems like prostitution": "Submit",
        "My nude images have been shared": "Submit",
        "Seems like sexual exploitation": "Submit",
        "Nudity or sexual activity": "Submit",
    },
    "Scam, fraud or false information": {
        "Fraud or scam": "Submit",
        "Sharing false information": "Done",
        "Spam": "Done",
        "Pretending to be a business": "SKIP",  # requires page number/URL
    },
    "Intellectual property": "Submit",
    "I don't want to see this": "Submit",
}


def _flatten_tree(node, prefix=None, out=None):
    """Flatten nested REPORT_TREE into list of {path, finish, skip}."""
    if out is None:
        out = []
    if prefix is None:
        prefix = []

    if isinstance(node, str):
        out.append({
            "path": list(prefix),
            "finish": "SKIP" if node == "SKIP" else node,
            "skip": node == "SKIP",
        })
        return out

    if isinstance(node, dict):
        for key, val in node.items():
            _flatten_tree(val, prefix + [key], out)
        return out

    return out


REPORT_LEAVES = _flatten_tree(REPORT_TREE)

# Short aliases → path list
REASON_PATHS: dict[str, list[str]] = {
    "spam": ["Scam, fraud or false information", "Spam"],
    "scam": ["Scam, fraud or false information", "Fraud or scam"],
    "fraud": ["Scam, fraud or false information", "Fraud or scam"],
    "fraud or scam": ["Scam, fraud or false information", "Fraud or scam"],
    "false information": ["Scam, fraud or false information", "Sharing false information"],
    "misinformation": ["Scam, fraud or false information", "Sharing false information"],
    "sharing false information": ["Scam, fraud or false information", "Sharing false information"],
    "bullying": ["Bullying, harassment or abuse", "Bullying or harassment", "Me"],
    "harassment": ["Bullying, harassment or abuse", "Bullying or harassment", "Me"],
    "suicide": ["Suicide or self-harm", "Suicide or self-injury"],
    "self-harm": ["Suicide or self-harm", "Suicide or self-injury"],
    "terrorism": ["Violent, hateful or disturbing content", "Seems like terrorism"],
    "hate speech": ["Violent, hateful or disturbing content", "Promoting hate", "Posting hateful speech"],
    "drugs": ["Selling or promoting restricted items", "Drugs", "Other drugs"],
    "weapons": ["Selling or promoting restricted items", "Weapons"],
    "nudity": ["Adult content", "Nudity or sexual activity"],
    "adult content": ["Adult content", "Nudity or sexual activity"],
    "dont want to see": ["I don't want to see this"],
    "i don't want to see this": ["I don't want to see this"],
    "intellectual property": ["Intellectual property"],
}

# Register every leaf path + last-step alias (first wins for ambiguous last labels)
_LEAF_BY_PATH: dict[tuple[str, ...], dict] = {}
for _leaf in REPORT_LEAVES:
    _key = tuple(_leaf["path"])
    _LEAF_BY_PATH[_key] = _leaf
    _path_str = " > ".join(_leaf["path"])
    REASON_PATHS.setdefault(_path_str.lower(), list(_leaf["path"]))
    REASON_PATHS.setdefault(_leaf["path"][-1].lower(), list(_leaf["path"]))
    # Also register top-level alone when it's a leaf
    if len(_leaf["path"]) == 1:
        REASON_PATHS.setdefault(_leaf["path"][0].lower(), list(_leaf["path"]))


def lookup_leaf(steps: list[str]) -> dict | None:
    """Return leaf metadata for an exact path, if known."""
    return _LEAF_BY_PATH.get(tuple(steps))


def print_reason_catalog() -> None:
    """Print the real report tree (for --list-reasons)."""
    print("Facebook report reason catalog (real UI tree)")
    print("=" * 70)
    print("After the last option, Facebook shows:")
    print("  Submit/Done → \"Thanks for reporting this post\" → Next")
    print("               → \"More actions\" → Done")
    print()

    def _walk(node, indent=0):
        if isinstance(node, str):
            return
        for key, val in node.items():
            pad = "  " * indent
            if isinstance(val, dict):
                print(f"{pad}{key}")
                _walk(val, indent + 1)
            else:
                flag = val
                if flag == "SKIP":
                    print(f"{pad}{key}  [SKIP — needs typed input]")
                else:
                    print(f"{pad}{key}  → {flag}")
                    # show ready-to-copy reason= path
                    # rebuild path by scanning leaves
        # print ready paths at end of top-level handled below

    for top, body in REPORT_TREE.items():
        print(f"\n{top}")
        print("-" * 70)
        if isinstance(body, str):
            print(f"  (leaf) → {body}")
            print(f"  reason={top}")
        else:
            _walk(body, 1)
            # list copy-paste paths under this top
            for leaf in REPORT_LEAVES:
                if leaf["path"][0] != top:
                    continue
                if leaf["skip"]:
                    print(f"  # SKIP: {' > '.join(leaf['path'])}")
                else:
                    print(f"  reason={' > '.join(leaf['path'])}")

    print("\nShort aliases:")
    print("-" * 70)
    for alias, steps in sorted(REASON_PATHS.items()):
        if " > " in alias or len(alias) > 40:
            continue
        leaf = lookup_leaf(steps)
        if leaf and leaf.get("skip"):
            continue
        print(f"  reason={alias}")
        print(f"         → {' > '.join(steps)}")


def resolve_reason_path(reason: str) -> List[str]:
    """
    Turn a reason string into an ordered list of dialog clicks.

    Accepts:
      - alias: "Spam"
      - path:  "Scam, fraud or false information > Spam"
      - single top-level label: "I don't want to see this"
    """
    raw = (reason or "").strip()
    if not raw:
        return list(REASON_PATHS["spam"])

    if ">" in raw:
        return [s.strip() for s in raw.split(">") if s.strip()]

    alias = REASON_PATHS.get(raw.lower())
    if alias:
        return list(alias)

    return [raw]


class FacebookReport(FacebookLogin):
    def load_report_config(self, path: str) -> Optional[dict]:
        """Parse report.txt: url=, reason=, note= lines."""
        cfg = {"url": None, "reason": DEFAULT_REASON, "note": ""}
        try:
            with open(path, "r", encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line or line.startswith("#"):
                        continue
                    if "=" in line:
                        key, val = line.split("=", 1)
                        key = key.strip().lower()
                        val = val.strip()
                        if key == "url":
                            cfg["url"] = val
                        elif key == "reason":
                            cfg["reason"] = val
                        elif key in ("note", "details", "comment"):
                            cfg["note"] = val
            if not cfg["url"]:
                cfg["url"] = DEFAULT_POST_URL
            path_steps = resolve_reason_path(cfg["reason"])
            cfg["reason_path"] = path_steps
            logger.info(
                "Report config: url=%s path=%s",
                cfg["url"][:60] + ("..." if len(cfg["url"]) > 60 else ""),
                " > ".join(path_steps),
            )
            return cfg
        except FileNotFoundError:
            logger.error("Config file not found: %s", path)
            return None
        except Exception as exc:
            logger.error("Could not read %s: %s", path, exc)
            return None

    def ensure_logged_in(self, account: dict, auto_verify: bool = True) -> str:
        logger.info("Ensuring login for %s ...", account["gmail"])
        result = self.login(account, auto_verify=auto_verify)
        logger.info("Login result: %s", result)
        return result

    def _dialog_root(self):
        """Prefer the visible Report dialog if present."""
        try:
            dialogs = self.driver.find_elements(By.XPATH, "//*[@role='dialog']")
            visible = [d for d in dialogs if d.is_displayed()]
            if visible:
                # Prefer dialog that mentions Report / Why are you reporting
                for d in visible:
                    txt = (d.text or "").lower()
                    if "report" in txt or "why are you reporting" in txt or "which best describes" in txt:
                        return d
                return visible[-1]
        except Exception:
            pass
        return None

    def _click_option_in_dialog(self, text: str, timeout: float = 12) -> bool:
        """
        Click a report menu row (chevron list item) inside the Report dialog.
        Prefer exact text match, then contains.
        """
        deadline = time.time() + timeout
        while time.time() < deadline:
            root = self._dialog_root()
            scopes = [root] if root is not None else [None]

            for scope in scopes:
                search_root = scope if scope is not None else self.driver
                xpaths = (
                    f".//*[@role='button' and normalize-space()='{text}']",
                    f".//*[@role='menuitem' and normalize-space()='{text}']",
                    f".//*[@role='radio' and normalize-space()='{text}']",
                    f".//span[normalize-space()='{text}']/ancestor::*[@role='button'][1]",
                    f".//span[normalize-space()='{text}']/ancestor::*[@role='menuitem'][1]",
                    f".//div[@role='button' and .//span[normalize-space()='{text}']]",
                    f".//*[normalize-space()='{text}']",
                    f".//*[@role='button' and contains(., '{text}')]",
                    f".//span[contains(text(), '{text}')]/ancestor::*[@role='button'][1]",
                    f".//*[contains(text(), '{text}')]",
                )
                for xpath in xpaths:
                    try:
                        els = search_root.find_elements(By.XPATH, xpath)
                    except Exception:
                        continue
                    for el in els:
                        try:
                            if not el.is_displayed():
                                continue
                            # Skip the dialog title / close / back chrome
                            label = (el.text or "").strip()
                            aria = (el.get_attribute("aria-label") or "").lower()
                            if aria in ("close", "back", "go back"):
                                continue
                            if label in ("Report", "Close", "Back"):
                                # Only skip bare "Report" title; allow longer labels
                                if label == "Report" and len(label) < 10:
                                    continue

                            target = el
                            tag = el.tag_name.lower()
                            if tag in ("span", "div") and el.get_attribute("role") not in (
                                "button",
                                "menuitem",
                                "radio",
                                "option",
                            ):
                                try:
                                    anc = el.find_elements(
                                        By.XPATH,
                                        "./ancestor::*[@role='button' or @role='menuitem' or @role='radio'][1]",
                                    )
                                    if anc:
                                        target = anc[0]
                                except Exception:
                                    pass

                            self.driver.execute_script(
                                "arguments[0].scrollIntoView({block:'center'});", target
                            )
                            time.sleep(0.25)
                            try:
                                target.click()
                            except Exception:
                                self.driver.execute_script("arguments[0].click();", target)
                            logger.info("Selected report step: %s", text)
                            time.sleep(1.5)
                            return True
                        except Exception:
                            continue
            time.sleep(0.4)
        return False

    def _click_by_texts(self, texts: tuple[str, ...], timeout: float = 10) -> bool:
        for text in texts:
            if self._click_option_in_dialog(text, timeout=timeout / max(1, len(texts))):
                return True
        # Broader page search (menus outside dialog)
        deadline = time.time() + timeout
        while time.time() < deadline:
            for text in texts:
                xpaths = (
                    f"//*[@role='menuitem' and contains(., '{text}')]",
                    f"//*[@role='button' and contains(., '{text}')]",
                    f"//span[normalize-space()='{text}']/ancestor::*[@role='button' or @role='menuitem'][1]",
                )
                for xpath in xpaths:
                    try:
                        for el in self.driver.find_elements(By.XPATH, xpath):
                            if el.is_displayed():
                                self.driver.execute_script("arguments[0].click();", el)
                                logger.info("Clicked: %s", text)
                                time.sleep(1.2)
                                return True
                    except Exception:
                        continue
            time.sleep(0.4)
        return False

    def _open_post_menu(self) -> bool:
        """Click the ⋯ menu on the post."""
        labels = (
            "Actions for this post",
            "More options for this post",
            "More",
            "Actions for this reel",
            "Actions for this photo",
        )
        for label in labels:
            xpaths = (
                f"//*[@aria-label='{label}' and (@role='button' or self::div)]",
                f"//*[contains(@aria-label, '{label}')]",
            )
            for xpath in xpaths:
                try:
                    for el in self.driver.find_elements(By.XPATH, xpath):
                        if el.is_displayed():
                            self.driver.execute_script("arguments[0].click();", el)
                            logger.info("Opened post menu via: %s", label)
                            time.sleep(1.5)
                            return True
                except Exception:
                    continue
        try:
            for article in self.driver.find_elements(By.XPATH, "//div[@role='article']"):
                if not article.is_displayed():
                    continue
                for el in article.find_elements(
                    By.XPATH,
                    ".//*[@role='button' and (contains(@aria-label,'Actions') or contains(@aria-label,'More'))]",
                ):
                    if el.is_displayed():
                        self.driver.execute_script("arguments[0].click();", el)
                        logger.info("Opened post menu (article fallback)")
                        time.sleep(1.5)
                        return True
        except Exception:
            pass
        return False

    def _page_has(self, *needles: str) -> bool:
        try:
            page = self.driver.page_source.lower()
            return any(n.lower() in page for n in needles)
        except Exception:
            return False

    def _is_thanks_report_screen(self) -> bool:
        """Screen after Submit: 'Thanks for reporting this post' (has Next)."""
        return self._page_has(
            "thanks for reporting this post",
            "report received",
            "awaiting review",
        )

    def _is_more_actions_screen(self) -> bool:
        """Final screen: 'More actions' / 'What else would you like to do?' (has Done)."""
        return self._page_has(
            "more actions",
            "what else would you like to do",
            "submitted to facebook for review",
        )

    def _is_thanks_screen(self) -> bool:
        """Any post-submit confirmation (old or new UI)."""
        if self._is_thanks_report_screen() or self._is_more_actions_screen():
            return True
        return self._page_has("thanks for letting us know")

    def _is_reason_picker(self) -> bool:
        """True when the multi-choice report reason dialog is open."""
        markers = (
            "Why are you reporting",
            "Which best describes the problem",
            "If someone is in immediate danger",
            "Who is being bullied",
        )
        try:
            page = self.driver.page_source
            return any(m in page for m in markers)
        except Exception:
            return False

    def _fill_report_note(self, note: str) -> None:
        if not note:
            return
        xpaths = (
            "//*[@role='dialog']//textarea",
            "//*[@role='dialog']//*[@role='textbox' and @contenteditable='true']",
            "//textarea",
        )
        for xpath in xpaths:
            try:
                for el in self.driver.find_elements(By.XPATH, xpath):
                    if el.is_displayed():
                        el.click()
                        time.sleep(0.2)
                        el.send_keys(note)
                        logger.info("Filled report note (%d chars)", len(note))
                        return
            except Exception:
                continue

    def _walk_reason_path(self, steps: List[str]) -> bool:
        """Click each step in the report reason tree."""
        for i, step in enumerate(steps, start=1):
            logger.info("Report path step %d/%d: %s", i, len(steps), step)
            if not self._click_option_in_dialog(step, timeout=12):
                logger.error("Could not select step %d: %s", i, step)
                try:
                    self.driver.save_screenshot(f"report_step_{i}_missing.png")
                except Exception:
                    pass
                return False
            if self._is_thanks_screen():
                logger.info("Reached confirmation after step %d", i)
                return True
        return True

    def _click_dialog_button(self, labels: tuple[str, ...], timeout: float = 10) -> bool:
        """Click a primary dialog button (Submit / Next / Done)."""
        deadline = time.time() + timeout
        while time.time() < deadline:
            for label in labels:
                xpaths = (
                    f"//*[@role='dialog']//*[@role='button' and normalize-space()='{label}']",
                    f"//*[@role='button' and normalize-space()='{label}']",
                    f"//button[normalize-space()='{label}']",
                    f"//*[@role='dialog']//span[normalize-space()='{label}']/ancestor::*[@role='button'][1]",
                    f"//span[normalize-space()='{label}']/ancestor::*[@role='button'][1]",
                )
                for xpath in xpaths:
                    try:
                        for el in self.driver.find_elements(By.XPATH, xpath):
                            if not el.is_displayed():
                                continue
                            if el.get_attribute("aria-disabled") in ("true", "True"):
                                continue
                            self.driver.execute_script(
                                "arguments[0].scrollIntoView({block:'center'});", el
                            )
                            time.sleep(0.2)
                            try:
                                el.click()
                            except Exception:
                                self.driver.execute_script("arguments[0].click();", el)
                            logger.info("Clicked button: %s", label)
                            time.sleep(1.5)
                            return True
                    except Exception:
                        continue
            time.sleep(0.4)
        return False

    def _finish_report(self, dry_run: bool, finish_action: str = "Submit") -> bool:
        """
        End of report flow (real UI):

          [optional Submit]
          → "Thanks for reporting this post"  → Next
          → "More actions"                    → Done
        """
        if dry_run:
            logger.info("Dry run — stopping before Submit/Next/Done")
            self.driver.save_screenshot("report_dry_run.png")
            return True

        # 1) Submit (or Done on some leaf paths like Spam)
        if not self._is_thanks_screen():
            wanted = ("Submit", "Done", "Next") if finish_action == "Done" else ("Submit", "Next", "Done")
            if not self._click_dialog_button(wanted, timeout=10):
                logger.warning("No Submit/Done button yet — checking for thanks screen")
            time.sleep(1.5)

        # 2) Thanks for reporting this post → Next
        for _ in range(12):
            if self._is_thanks_report_screen():
                logger.info("Thanks-for-reporting screen — clicking Next")
                if not self._click_dialog_button(("Next",), timeout=8):
                    logger.error("Next button not found on thanks screen")
                    self.driver.save_screenshot("report_next_missing.png")
                    return False
                time.sleep(1.5)
                break
            if self._is_more_actions_screen():
                break
            # Older UI
            if self._page_has("thanks for letting us know"):
                logger.info("Legacy thanks screen — clicking Done")
                self._click_dialog_button(("Done",), timeout=8)
                return True
            time.sleep(0.5)
        else:
            if not self._is_more_actions_screen():
                logger.warning("Did not see 'Thanks for reporting this post'")

        # 3) More actions → Done
        for _ in range(12):
            if self._is_more_actions_screen() or self._page_has("what else would you like to do"):
                logger.info("More-actions screen — clicking Done")
                if not self._click_dialog_button(("Done",), timeout=8):
                    logger.error("Done button not found on More actions screen")
                    self.driver.save_screenshot("report_done_missing.png")
                    return False
                time.sleep(1)
                logger.info("Report flow finished (Done)")
                return True
            time.sleep(0.5)

        # Last resort
        if self._click_dialog_button(("Done", "Next", "Submit"), timeout=5):
            return True

        logger.warning("Could not complete Next → Done finish sequence")
        return False

    def report_post(
        self,
        post_url: str,
        reason: str,
        note: str = "",
        dry_run: bool = False,
        reason_path: Optional[List[str]] = None,
    ) -> bool:
        steps = reason_path or resolve_reason_path(reason)
        leaf = lookup_leaf(steps)
        if leaf and leaf.get("skip"):
            logger.error(
                "This path requires typed input and is not automated: %s",
                " > ".join(steps),
            )
            logger.error(
                "Pick another path, or complete that screen manually. "
                "Examples: Bullying… > Me   (not 'A friend'); "
                "avoid 'Pretending to be a business'."
            )
            return False

        finish_action = (leaf or {}).get("finish", "Submit")
        logger.info("Opening post to report: %s", post_url)
        logger.info("Reason path: %s  (finish=%s)", " > ".join(steps), finish_action)

        self.driver.get(post_url)
        time.sleep(5)
        self.handle_cookie_consent()
        time.sleep(1)

        try:
            ActionChains(self.driver).send_keys(Keys.ESCAPE).perform()
            time.sleep(0.3)
        except Exception:
            pass

        if not self._open_post_menu():
            logger.error("Could not open post ⋯ menu")
            self.driver.save_screenshot("report_menu_missing.png")
            return False

        # Enter report flow (only once — do not re-click "Report" inside the dialog)
        if not self._click_by_texts(
            (
                "Find support or report post",
                "Find support or report",
                "Report post",
                "Report photo",
                "Report reel",
                "Give feedback or report",
            ),
            timeout=12,
        ):
            if not self._click_by_texts(("Report",), timeout=5):
                logger.error("Could not find Report menu item")
                self.driver.save_screenshot("report_entry_missing.png")
                return False

        time.sleep(1.5)

        for _ in range(15):
            if self._is_reason_picker() or self._dialog_root() is not None:
                break
            time.sleep(0.4)

        if not self._walk_reason_path(steps):
            return False

        self._fill_report_note(note)

        ok = self._finish_report(dry_run=dry_run, finish_action=finish_action)

        ts = time.strftime("%Y%m%d_%H%M%S")
        screenshot = f"report_confirmed_{ts}.png"
        try:
            self.driver.save_screenshot(screenshot)
            logger.info("Saved screenshot: %s", screenshot)
        except Exception:
            pass

        if ok:
            logger.info("Report completed: %s", " > ".join(steps))
            return True

        logger.warning("Could not confirm submission — check %s", screenshot)
        return False

    def run_report(
        self,
        post_url: str,
        reason: str,
        note: str = "",
        config_file: str = DEFAULT_CONFIG_FILE,
        account_id: Optional[str] = None,
        email: Optional[str] = None,
        index: int = 0,
        use_proxy: bool = True,
        headless: bool = False,
        keep_open: bool = True,
        auto_verify: bool = True,
        profile_dir: Optional[str] = None,
        dry_run: bool = False,
        use_config_file: bool = True,
    ) -> bool:
        reason_path = resolve_reason_path(reason)

        if use_config_file and os.path.exists(config_file):
            cfg = self.load_report_config(config_file)
            if not cfg:
                return False
            # CLI --url / --reason override file when provided via --no-config path;
            # when using config, file wins unless caller already passed explicit overrides
            # Here: prefer file values, then fall back to args
            post_url = cfg["url"] or post_url
            reason = cfg["reason"] or reason
            note = cfg.get("note") or note
            reason_path = cfg.get("reason_path") or resolve_reason_path(reason)

        if not self.load_data():
            return False
        account = self.find_account(account_id=account_id, email=email, index=index)
        if not account:
            return False

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
            return False

        login_result = self.ensure_logged_in(account, auto_verify=auto_verify)
        if login_result not in ("success", "checkpoint", "verification_needed"):
            logger.error("Login failed (%s)", login_result)
            ok = False
        else:
            self.handle_cookie_consent()
            ok = self.report_post(
                post_url,
                reason,
                note=note,
                dry_run=dry_run,
                reason_path=reason_path,
            )

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
        description="Report a Facebook post (multi-step reason path supported)."
    )
    parser.add_argument(
        "--list-reasons",
        action="store_true",
        help="Print all top-level report cases and known sub-paths, then exit",
    )
    parser.add_argument("--config", default=DEFAULT_CONFIG_FILE, help="Path to report.txt")
    parser.add_argument("--url", default=None, help="Post URL (used with --no-config)")
    parser.add_argument(
        "--reason",
        default=DEFAULT_REASON,
        help='Alias (Spam) or path: "Scam, fraud or false information > Spam"',
    )
    parser.add_argument("--note", default="", help="Optional extra detail")
    parser.add_argument("--id", dest="account_id", help="Account id from accounts.csv")
    parser.add_argument("--email", help="Account email from accounts.csv")
    parser.add_argument("--index", type=int, default=0)
    parser.add_argument("--no-proxy", action="store_true")
    parser.add_argument("--headless", action="store_true")
    parser.add_argument("--no-keep-open", action="store_true")
    parser.add_argument("--no-verify", action="store_true")
    parser.add_argument("--profile-dir", default=None)
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Walk menus but stop before Done/Submit",
    )
    parser.add_argument(
        "--no-config",
        action="store_true",
        help="Ignore report.txt; use only --url and --reason",
    )
    args = parser.parse_args()

    if args.list_reasons:
        print_reason_catalog()
        return 0

    url = args.url or DEFAULT_POST_URL
    reason = args.reason

    bot = FacebookReport()
    ok = bot.run_report(
        post_url=url,
        reason=reason,
        note=args.note,
        config_file=args.config,
        account_id=args.account_id,
        email=args.email,
        index=args.index,
        use_proxy=not args.no_proxy,
        headless=args.headless,
        keep_open=not args.no_keep_open,
        auto_verify=not args.no_verify,
        profile_dir=args.profile_dir,
        dry_run=args.dry_run,
        use_config_file=not args.no_config,
    )
    return 0 if ok else 1


if __name__ == "__main__":
    raise SystemExit(main())
