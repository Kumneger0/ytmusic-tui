import hashlib
import sys
import time
from http.cookiejar import CookieJar
from typing import Callable, cast

import browser_cookie3  # pyright: ignore[reportMissingTypeStubs]

from grpc_server.src.auth import (  # pyright: ignore[reportImplicitRelativeImport]
    process_raw_headers_via_ytmusicapi,
)

SUPPORTED_BROWSERS: dict[str, Callable[..., CookieJar]] = {
    "chrome": cast(Callable[..., CookieJar], browser_cookie3.chrome),
    "firefox": cast(Callable[..., CookieJar], browser_cookie3.firefox),
    "brave": cast(Callable[..., CookieJar], browser_cookie3.brave),
    "edge": cast(Callable[..., CookieJar], browser_cookie3.edge),
    "chromium": cast(Callable[..., CookieJar], browser_cookie3.chromium),
    "opera": cast(Callable[..., CookieJar], browser_cookie3.opera),
    "opera_gx": cast(Callable[..., CookieJar], browser_cookie3.opera_gx),
    "vivaldi": cast(Callable[..., CookieJar], browser_cookie3.vivaldi),
    "librewolf": cast(Callable[..., CookieJar], browser_cookie3.librewolf),
    "safari": cast(Callable[..., CookieJar], browser_cookie3.safari),
}

YOUTUBE_ORIGIN = "https://music.youtube.com"


def _extract_cookies(browser_name: str) -> CookieJar:
    """
    Extracts YouTube cookies from the given browser using browser-cookie3.
    Returns a CookieJar containing matching cookies.
    """
    browser_fn = SUPPORTED_BROWSERS.get(browser_name)
    if browser_fn is None:
        supported = ", ".join(sorted(SUPPORTED_BROWSERS.keys()))
        print(f"[Error] Unsupported browser: '{browser_name}'", file=sys.stderr)
        print(f"  Supported browsers: {supported}", file=sys.stderr)
        sys.exit(1)

    try:
        return  browser_fn(domain_name=".youtube.com")
    except Exception as e:
        print(f"[Error] Failed to extract cookies from {browser_name}: {e}", file=sys.stderr)
        print(
            f"\n  Hint: Make sure {browser_name} is closed before running this command.",
            file=sys.stderr,
        )
        print(
            "  browser-cookie3 cannot read cookies while the browser holds a lock on its database.",
            file=sys.stderr,
        )
        sys.exit(1)


def _cookies_to_header(cj: CookieJar) -> str:
    """
    Converts a CookieJar into a single Cookie header value string.
    Example: "name1=value1; name2=value2"
    """
    parts: list[str] = []
    for cookie in cj:
        parts.append(f"{cookie.name}={cookie.value}")
    return "; ".join(parts)


def _get_sapisid(cj: CookieJar) -> str | None:
    """
    Extracts the SAPISID or __Secure-3PAPISID cookie value from the CookieJar.
    YouTube uses these for SAPISIDHASH authorization.
    Prefers SAPISID, falls back to __Secure-3PAPISID.
    """
    sapisid = None
    secure_3papisid = None
    for cookie in cj:
        if cookie.name == "SAPISID":
            sapisid = cookie.value
        elif cookie.name == "__Secure-3PAPISID":
            secure_3papisid = cookie.value
    return sapisid or secure_3papisid


def _compute_sapisidhash(sapisid: str, origin: str) -> str:
    """
    Computes the SAPISIDHASH authorization header value.
    Format: SAPISIDHASH <timestamp>_<SHA1(timestamp + " " + origin + " " + SAPISID)>
    This is required by YouTube's API for browser-based authentication.
    """
    timestamp = str(int(time.time()))
    hash_input = f"{timestamp} {origin} {sapisid}"
    sha1_hash = hashlib.sha1(hash_input.encode("utf-8")).hexdigest()
    return f"SAPISIDHASH {timestamp}_{sha1_hash}"


def _build_raw_headers(cookie_header: str, authorization: str) -> str:
    """
    Builds the raw HTTP headers that ytmusicapi.setup() expects.
    Includes the Cookie header and the computed SAPISIDHASH Authorization header.
    """
    lines = [
        "User-Agent: Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0",
        "Accept: */*",
        "Accept-Language: en-US,en;q=0.5",
        "Content-Type: application/json",
        "X-Goog-AuthUser: 0",
        f"X-Origin: {YOUTUBE_ORIGIN}",
        f"Origin: {YOUTUBE_ORIGIN}",
        f"Authorization: {authorization}",
        f"Cookie: {cookie_header}",
    ]
    return "\n".join(lines)


def run_cookie_extraction(browser_name: str) -> None:
    """
    Main entry point for cookie extraction.
    Extracts YouTube cookies from the specified browser, computes the
    SAPISIDHASH authorization header, and saves credentials as browser.json
    via ytmusicapi.
    """
    print("  COOKIE EXTRACTION")
    print(f"  Browser: {browser_name}")
    print()
    print(f"  ⚠️  Please make sure {browser_name} is CLOSED before continuing.")
    print("  browser-cookie3 cannot read cookies while the browser is running.")
    print()

    cj = _extract_cookies(browser_name)

    cookie_count = sum(1 for _ in cj)
    if cookie_count == 0:
        print(f"[Error] No YouTube cookies found in {browser_name}.", file=sys.stderr)
        print(
            "  Make sure you are logged into YouTube Music in this browser.",
            file=sys.stderr,
        )
        sys.exit(1)

    print(f"  ✓ Found {cookie_count} YouTube cookie(s) in {browser_name}")

    sapisid = _get_sapisid(cj)
    if sapisid is None:
        print("[Error] Could not find SAPISID or __Secure-3PAPISID cookie.", file=sys.stderr)
        print(
            "  This cookie is required for YouTube Music authentication.",
            file=sys.stderr,
        )
        print(
            "  Make sure you are logged into YouTube Music in this browser.",
            file=sys.stderr,
        )
        sys.exit(1)

    authorization = _compute_sapisidhash(sapisid, YOUTUBE_ORIGIN)

    cookie_header = _cookies_to_header(cj)
    raw_headers = _build_raw_headers(cookie_header, authorization)

    print("  ⏳ Validating credentials with YouTube Music API...")
    print()

    success, err_msg, browser_json_path = process_raw_headers_via_ytmusicapi(raw_headers)

    if success:
        print(f"  Saved credentials to: {browser_json_path}")
        print("  You can now run clispot normally.")
    else:
        print(f"\n[Error] Authentication failed: {err_msg}", file=sys.stderr)
        print(
            "  The cookies were extracted but YouTube Music API rejected them.",
            file=sys.stderr,
        )
        print(
            "  Make sure you are logged into YouTube Music in this browser.",
            file=sys.stderr,
        )
        sys.exit(1)
