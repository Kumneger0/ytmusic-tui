import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import MagicMock, patch

from grpc_server.src.auth import (  # pyright: ignore[reportImplicitRelativeImport]
    extract_and_validate_headers,
    get_browser_json_path,
    get_config_dir,
    parse_raw_headers,
    save_browser_json,
    test_authentication,
)


class TestAuthModule(unittest.TestCase):

    def test_parse_raw_headers_standard(self):
        raw = """
POST /youtubei/v1/browse?key=123 HTTP/1.1
Host: music.youtube.com
User-Agent: Mozilla/5.0
Accept: */*
Authorization: SAPISIDHASH 123456_abcdef
Content-Type: application/json
X-Goog-AuthUser: 0
x-origin: https://music.youtube.com
Cookie: VISITOR_INFO1_LIVE=abc; SID=def
"""
        parsed = parse_raw_headers(raw)
        self.assertEqual(parsed["accept"], "*/*")
        self.assertEqual(parsed["authorization"], "SAPISIDHASH 123456_abcdef")
        self.assertEqual(parsed["content-type"], "application/json")
        self.assertEqual(parsed["x-goog-authuser"], "0")
        self.assertEqual(parsed["x-origin"], "https://music.youtube.com")
        self.assertEqual(parsed["cookie"], "VISITOR_INFO1_LIVE=abc; SID=def")

    def test_parse_raw_headers_origin_fallback(self):
        raw = """
Authorization: SAPISIDHASH 123456_abcdef
Cookie: SID=def
origin: https://music.youtube.com
"""
        parsed = parse_raw_headers(raw)
        self.assertEqual(parsed["x-origin"], "https://music.youtube.com")

    def test_parse_raw_headers_curl(self):
        raw = """
curl 'https://music.youtube.com/youtubei/v1/browse' \
  -H 'authorization: SAPISIDHASH 123456_curl' \
  -H 'cookie: SID=curl_cookie'
"""
        parsed = parse_raw_headers(raw)
        self.assertEqual(parsed["authorization"], "SAPISIDHASH 123456_curl")
        self.assertEqual(parsed["cookie"], "SID=curl_cookie")

    def test_parse_raw_headers_json(self):
        raw = '{"authorization": "SAPISIDHASH 123456_json", "cookie": "SID=json_cookie"}'
        parsed = parse_raw_headers(raw)
        self.assertEqual(parsed["authorization"], "SAPISIDHASH 123456_json")
        self.assertEqual(parsed["cookie"], "SID=json_cookie")

    def test_parse_raw_headers_chrome_devtools(self):
        raw = """
:authority
music.youtube.com
:method
POST
authorization
SAPISIDHASH 123456_chrome
cookie
SID=chrome_cookie
"""
        parsed = parse_raw_headers(raw)
        self.assertEqual(parsed["authorization"], "SAPISIDHASH 123456_chrome")
        self.assertEqual(parsed["cookie"], "SID=chrome_cookie")

    def test_extract_and_validate_headers_success(self):
        raw = """
Host: music.youtube.com
Accept: */*
Authorization: SAPISIDHASH test_token
Content-Type: application/json
X-Goog-AuthUser: 0
x-origin: https://music.youtube.com
Cookie: SID=test_cookie
User-Agent: ignore_me
"""
        extracted, missing = extract_and_validate_headers(raw)
        self.assertEqual(missing, [])
        self.assertEqual(
            extracted,
            {
                "Accept": "*/*",
                "Authorization": "SAPISIDHASH test_token",
                "Content-Type": "application/json",
                "X-Goog-AuthUser": "0",
                "x-origin": "https://music.youtube.com",
                "Cookie": "SID=test_cookie",
            },
        )
        # Verify non-required header was ignored
        self.assertNotIn("User-Agent", extracted)
        self.assertNotIn("Host", extracted)

    def test_extract_and_validate_headers_defaults(self):
        raw = """
Authorization: SAPISIDHASH test_token
Cookie: SID=test_cookie
"""
        extracted, missing = extract_and_validate_headers(raw)
        self.assertEqual(missing, [])
        self.assertEqual(extracted["Accept"], "*/*")
        self.assertEqual(extracted["Content-Type"], "application/json")
        self.assertEqual(extracted["X-Goog-AuthUser"], "0")
        self.assertEqual(extracted["x-origin"], "https://music.youtube.com")
        self.assertEqual(extracted["Authorization"], "SAPISIDHASH test_token")
        self.assertEqual(extracted["Cookie"], "SID=test_cookie")

    def test_extract_and_validate_headers_missing_required(self):
        raw = """
Accept: */*
Content-Type: application/json
"""
        extracted, missing = extract_and_validate_headers(raw)
        self.assertIn("Authorization", missing)
        self.assertIn("Cookie", missing)

    def test_save_browser_json(self):
        headers = {
            "Accept": "*/*",
            "Authorization": "SAPISIDHASH 123",
            "Content-Type": "application/json",
            "X-Goog-AuthUser": "0",
            "x-origin": "https://music.youtube.com",
            "Cookie": "SID=456",
        }
        with tempfile.TemporaryDirectory() as tmpdir:
            file_path = Path(tmpdir) / "browser.json"
            saved_path = save_browser_json(headers, file_path)
            self.assertEqual(saved_path, file_path)
            self.assertTrue(file_path.exists())

            with open(file_path, "r", encoding="utf-8") as f:
                data = json.load(f)

            self.assertEqual(data["Authorization"], "SAPISIDHASH 123")
            self.assertEqual(data["Cookie"], "SID=456")

    @patch("grpc_server.src.auth.YTMusic")
    def test_test_authentication_success(self, mock_ytmusic):  # pyright: ignore[reportUnknownParameterType]
        mock_instance = MagicMock()
        mock_instance.get_library_subscriptions.return_value = []  # pyright: ignore[reportAny]
        mock_ytmusic.return_value = mock_instance

        with tempfile.TemporaryDirectory() as tmpdir:
            file_path = Path(tmpdir) / "browser.json"
            _ = file_path.write_text("{}")
            success, err = test_authentication(file_path)
            self.assertTrue(success)
            self.assertEqual(err, "")

    @patch("grpc_server.src.auth.YTMusic")
    def test_test_authentication_failure(self, mock_ytmusic):
        mock_ytmusic.side_effect = Exception("Unauthorized 401")

        with tempfile.TemporaryDirectory() as tmpdir:
            file_path = Path(tmpdir) / "browser.json"
            _ = file_path.write_text("{}")
            success, err = test_authentication(file_path)
            self.assertFalse(success)
            self.assertIn("Unauthorized 401", err)

    def test_get_free_port(self):
        from grpc_server.src.auth import get_free_port  # pyright: ignore[reportImplicitRelativeImport]
        port = get_free_port(9900)
        self.assertGreaterEqual(port, 9900)


if __name__ == "__main__":
    _ = unittest.main()
