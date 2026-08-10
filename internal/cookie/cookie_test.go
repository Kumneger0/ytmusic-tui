package cookie

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kumneger0/ytmusic-tui/internal/config"
)

func TestConvertToNetscapeCookies_Empty(t *testing.T) {
	result := ConvertToNetscapeCookies("")
	if !strings.HasPrefix(result, "# Netscape HTTP Cookie File") {
		t.Fatalf("expected header, got: %q", result)
	}
}

func TestConvertToNetscapeCookies_AlreadyNetscape(t *testing.T) {
	input := "# Netscape HTTP Cookie File\n.youtube.com\tTRUE\t/\tTRUE\t2147483647\tSID\t12345"
	result := ConvertToNetscapeCookies(input)
	if result != input {
		t.Fatalf("expected %q, got %q", input, result)
	}
}

func TestConvertToNetscapeCookies_JSONArray(t *testing.T) {
	jsonInput := `[
		{
			"name": "VISITOR_INFO1_LIVE",
			"value": "xyz123",
			"domain": ".youtube.com",
			"path": "/",
			"secure": true,
			"httpOnly": true,
			"expirationDate": 1750000000
		}
	]`
	result := ConvertToNetscapeCookies(jsonInput)
	if !strings.Contains(result, "#HttpOnly_.youtube.com\tTRUE\t/\tTRUE\t1750000000\tVISITOR_INFO1_LIVE\txyz123") {
		t.Fatalf("unexpected netscape content: %s", result)
	}
}

func TestConvertToNetscapeCookies_JSONObject(t *testing.T) {
	jsonInput := `{"Cookie": "GPS=1; YSC=abc"}`
	result := ConvertToNetscapeCookies(jsonInput)
	if !strings.Contains(result, ".youtube.com\tTRUE\t/\tTRUE\t2147483647\tGPS\t1") || !strings.Contains(result, ".youtube.com\tTRUE\t/\tTRUE\t2147483647\tYSC\tabc") {
		t.Fatalf("unexpected netscape content: %s", result)
	}
}

func TestConvertToNetscapeCookies_RawHeaderString(t *testing.T) {
	rawInput := "Cookie: SID=abc1234; HSID=xyz5678"
	result := ConvertToNetscapeCookies(rawInput)
	if !strings.Contains(result, "SID\tabc1234") || !strings.Contains(result, "HSID\txyz5678") {
		t.Fatalf("unexpected netscape content: %s", result)
	}
}

func TestEnsureCookieFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cookie_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("HOME", tmpDir)
	ResetCookieHeaderCache()

	configDir := config.GetConfigDir(runtime.GOOS)
	browserPath := filepath.Join(configDir, "browser.json")
	if err := os.MkdirAll(filepath.Dir(browserPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(browserPath, []byte(`{"Cookie": "TEST=123"}`), 0600); err != nil {
		t.Fatal(err)
	}

	cookiePath := EnsureCookieFile()
	if cookiePath == "" {
		t.Fatal("expected non-empty cookie path")
	}

	content, err := os.ReadFile(cookiePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "TEST\t123") {
		t.Fatalf("unexpected cookie file content: %s", string(content))
	}

	header := GetCookieHeader()
	if header != "TEST=123" {
		t.Fatalf("expected 'TEST=123', got %q", header)
	}
}
