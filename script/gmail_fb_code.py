#!/usr/bin/env python3
"""Fetch a Facebook confirmation code from Gmail over IMAP."""

from __future__ import annotations

import argparse
import email
import imaplib
import json
import re
import sys
import time
from datetime import timezone
from email.header import decode_header
from email.utils import parsedate_to_datetime


CODE_PATTERNS = (
    re.compile(r"(?:confirmation code|security code|code is|code:)\s*(\d{5})", re.I),
    re.compile(r"(\d{5})\s+is your", re.I),
    re.compile(r"\b(\d{5})\b"),
)


def decode_part(value: str) -> str:
    chunks = []
    for part, enc in decode_header(value or ""):
        if isinstance(part, bytes):
            chunks.append(part.decode(enc or "utf-8", errors="replace"))
        else:
            chunks.append(part)
    return "".join(chunks)


def message_date(msg: email.message.Message) -> float:
    raw = msg.get("Date")
    if not raw:
        return 0
    try:
        dt = parsedate_to_datetime(raw)
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=timezone.utc)
        return dt.timestamp()
    except Exception:
        return 0


def extract_text(msg: email.message.Message) -> str:
    parts = [decode_part(msg.get("Subject", ""))]
    if msg.is_multipart():
        for part in msg.walk():
            ctype = part.get_content_type()
            if ctype not in {"text/plain", "text/html"}:
                continue
            payload = part.get_payload(decode=True) or b""
            charset = part.get_content_charset() or "utf-8"
            parts.append(payload.decode(charset, errors="replace"))
    else:
        payload = msg.get_payload(decode=True) or b""
        charset = msg.get_content_charset() or "utf-8"
        parts.append(payload.decode(charset, errors="replace"))
    return "\n".join(parts)


def extract_code(text: str) -> str:
    for pattern in CODE_PATTERNS:
        match = pattern.search(text)
        if match:
            return match.group(1)
    return ""


def fetch_code(address: str, password: str, timeout: int, since: int) -> str:
    deadline = time.time() + max(1, timeout)
    password = password.replace(" ", "").strip()
    last_error = ""
    while time.time() < deadline:
        try:
            client = imaplib.IMAP4_SSL("imap.gmail.com", 993)
            try:
                client.login(address, password)
                client.select("INBOX")
                status, data = client.search(None, '(FROM "facebookmail.com")')
                if status != "OK":
                    status, data = client.search(None, '(SUBJECT "Facebook")')
                ids = (data[0] or b"").split()
                for msg_id in reversed(ids[-20:]):
                    status, raw = client.fetch(msg_id, "(RFC822)")
                    if status != "OK" or not raw or not raw[0]:
                        continue
                    msg = email.message_from_bytes(raw[0][1])
                    if since and message_date(msg) and message_date(msg) < since:
                        continue
                    code = extract_code(extract_text(msg))
                    if code:
                        return code
            finally:
                try:
                    client.logout()
                except Exception:
                    pass
        except Exception as exc:
            last_error = str(exc)
        time.sleep(5)
    if last_error:
        print(last_error, file=sys.stderr)
    return ""


def main() -> int:
    parser = argparse.ArgumentParser(description="Read a Facebook confirmation code from Gmail.")
    parser.add_argument("--email", required=True)
    parser.add_argument("--password", required=True)
    parser.add_argument("--timeout", type=int, default=45)
    parser.add_argument("--since", type=int, default=0, help="Unix timestamp; ignore older mail")
    args = parser.parse_args()
    code = fetch_code(args.email, args.password, args.timeout, args.since)
    json.dump({"code": code}, sys.stdout)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
