# Kurabiye Implementation Plan

## Architecture

### Package Structure

```
kurabiye-blind-go/
├── kurabiye.go          # Public API: Cookie, GetCookiesOptions, GetCookiesResult, GetCookies, ToCookieHeader
├── chromium.go          # Shared Chromium logic: read DB, decrypt values, query cookies
├── chrome.go            # Chrome-specific: paths, keychain service name
├── edge.go              # Edge-specific: paths, keychain service name
├── firefox.go           # Firefox: profile discovery, cookies.sqlite reading
├── safari.go            # Safari: binarycookies parser (macOS only, build-tagged)
├── safari_stub.go       # Stub for non-macOS (build-tagged)
├── paths_darwin.go      # macOS path resolution
├── paths_windows.go     # Windows path resolution
├── paths_linux.go       # Linux path resolution
├── crypto_darwin.go     # macOS: Keychain retrieval + PBKDF2/AES-128-CBC
├── crypto_windows.go    # Windows: DPAPI/Local State + AES-256-GCM
├── crypto_linux.go      # Linux: secret-tool/kwallet + PBKDF2/AES-128-CBC
├── domain.go            # Domain matching logic
├── domain_test.go       # Unit tests for domain matching
├── cookie_test.go       # Unit tests for ToCookieHeader, filtering
├── safari_test.go       # Unit tests for binarycookies parsing
├── firefox_test.go      # Unit tests for Firefox profile discovery
├── chromium_test.go     # Unit tests for Chromium decryption
├── integration_test.go  # Integration tests (build tag: integration)
├── cmd/
│   └── kurabiye/
│       └── main.go      # CLI entry point
├── go.mod
├── go.sum
├── AGENT.md
├── PLAN.md
├── DECISIONS.md
└── STATUS.md
```

### Design Principles

1. **Browser as interface**: Each browser implements a common internal interface
   that returns `[]Cookie` for a given domain filter.
2. **Build tags for OS-specific code**: Use `//go:build darwin`, `windows`, `linux`
   to isolate platform-specific file paths and crypto.
3. **Chromium code sharing**: Chrome and Edge share 95% of logic. Each just
   provides its specific paths and keychain service name to the shared Chromium
   backend.
4. **Fail soft**: Every browser extraction returns `([]Cookie, error)`. Errors
   become warnings in the aggregated result; we never stop on a single failure.

## Browser-Specific Details

### Chromium (Chrome + Edge)

**Cookie database**: SQLite file, table `cookies` with key columns:
- `host_key` (domain), `name`, `value`, `encrypted_value` (BLOB)
- `path`, `expires_utc` (microseconds since 1601-01-01), `is_secure`,
  `is_httponly`, `samesite` (0=unspecified, 1=Lax, 2=Strict, -1=None... need to verify)
- `has_expires`

**Database location** (Default profile):
- macOS Chrome: `~/Library/Application Support/Google/Chrome/Default/Cookies`
- macOS Edge: `~/Library/Application Support/Microsoft Edge/Default/Cookies`
- Windows Chrome: `%LOCALAPPDATA%\Google\Chrome\User Data\Default\Network\Cookies`
- Windows Edge: `%LOCALAPPDATA%\Microsoft\Edge\User Data\Default\Network\Cookies`
- Linux Chrome: `~/.config/google-chrome/Default/Cookies`
- Linux Edge: `~/.config/microsoft-edge/Default/Cookies`

**Encryption by platform:**

| Platform | Key source | KDF | Cipher | IV | Prefix |
|----------|-----------|-----|--------|-----|--------|
| macOS | Keychain (`security find-generic-password -s "<service>" -w`) | PBKDF2(1003 iter, salt="saltysalt", keylen=16) | AES-128-CBC | 16 × 0x20 | `v10` |
| Linux | GNOME Keyring (`secret-tool lookup application <app>`) or fallback "peanuts" | PBKDF2(1 iter, salt="saltysalt", keylen=16) | AES-128-CBC | 16 × 0x20 | `v10`/`v11` |
| Windows | `Local State` JSON → `os_crypt.encrypted_key` → base64 decode → strip "DPAPI" prefix → DPAPI decrypt | None (key used directly) | AES-256-GCM | 12-byte nonce from ciphertext | `v10` |

For macOS/Linux CBC decryption: strip 3-byte prefix, decrypt AES-128-CBC, remove PKCS#7 padding.
For Windows GCM decryption: strip 3-byte prefix, first 12 bytes = nonce, rest = ciphertext+tag.
Windows fallback: if no `v10` prefix, try raw DPAPI on the entire blob.

**Locking**: The browser holds a WAL lock on the DB. Copy to a temp file before opening.

### Firefox

**Cookie database**: `cookies.sqlite` in the profile directory, table `moz_cookies`:
- `host`, `name`, `value` (plaintext!), `path`, `expiry` (Unix seconds),
  `isSecure`, `isHttpOnly`, `sameSite` (0=None, 1=Lax, 2=Strict)

**Profile discovery**:
- Read `profiles.ini` from the Firefox config directory
- Look for `[Install*]` section → `Default=` gives the relative profile path
- Fallback: look for `[Profile*]` sections where `Default=1`
- Fallback: glob for `*.default-release` directories

**Profile directory**:
- macOS: `~/Library/Application Support/Firefox/`
- Windows: `%APPDATA%\Mozilla\Firefox\`
- Linux: `~/.mozilla/firefox/`

**Locking**: Same WAL issue. Copy `cookies.sqlite` to temp.

### Safari (macOS only)

**File**: `~/Library/Cookies/Cookies.binarycookies`

**Binary format**:
1. Header: magic `"cook"` (4 bytes) + big-endian uint32 page count
2. Page sizes: array of big-endian uint32
3. Pages sequentially:
   - Page header: little-endian uint32 (0x00000100)
   - Cookie count: little-endian uint32
   - Cookie offsets: array of little-endian uint32
   - Cookie records at those offsets
4. Cookie record:
   - size (4B LE), flags (4B LE), padding (4B), url_offset (4B LE),
     name_offset (4B LE), path_offset (4B LE), value_offset (4B LE),
     comment (8B), expiry (float64 LE), creation (float64 LE)
   - Strings at offsets are null-terminated
   - Dates: seconds since Apple epoch (2001-01-01T00:00:00Z)
   - Flags: 0x1 = Secure, 0x4 = HttpOnly

## Dependencies

- `modernc.org/sqlite` — pure-Go SQLite driver (no CGO)
- Standard library: `crypto/aes`, `crypto/cipher`, `crypto/sha1`,
  `golang.org/x/crypto/pbkdf2`, `database/sql`, `encoding/binary`,
  `encoding/json`, `os/exec`, `flag`, `net/url`

## Uncertainties & Resolution Strategy

1. **Chrome samesite integer mapping**: Will verify by reading actual DB values
   or checking Chromium docs. Fallback: map 0→"", 1→"Lax", 2→"Strict", -1→"None".
2. **Windows DPAPI without CGO**: Will shell out to PowerShell
   `[Security.Cryptography.ProtectedData]::Unprotect()`.
3. **Safari binarycookies exact field offsets**: Will test with a hex dump of a
   real file if available, otherwise rely on documented format.
4. **Chrome cookie DB path**: Newer Chrome may use `Network/Cookies` subfolder.
   Will try both paths.

## Testing Strategy

1. **Unit tests** (no browser needed):
   - Domain matching: various subdomain/parent-domain cases
   - ToCookieHeader formatting and deduplication
   - Safari binary parsing with crafted test data
   - Firefox profile.ini parsing
   - Chromium AES-CBC decryption with known key/ciphertext
2. **Integration tests** (`//go:build integration`):
   - Actual cookie extraction from installed browsers
   - End-to-end CLI test
3. **Error path tests**:
   - Missing browser (no DB file)
   - Corrupted/empty DB
   - Expired cookie filtering

## Execution Order

1. Initialize Go module and install dependencies
2. Implement domain matching (`domain.go` + tests)
3. Implement public API types and ToCookieHeader (`kurabiye.go` + tests)
4. Implement Chromium backend (crypto + DB reading)
5. Implement Firefox backend
6. Implement Safari backend
7. Wire up GetCookies orchestration
8. Build CLI
9. Run all tests, fix issues
10. Write DECISIONS.md and STATUS.md
