# Kurabiye Implementation Plan

## Architecture and File/Package Structure

Single Go package `kurabiye` at the root, with a CLI in `cmd/kurabiye/`.

```
kurabiye-blind-go/
├── go.mod                  # module github.com/kurabiye
├── go.sum
├── kurabiye.go             # Public API: GetCookies, ToCookieHeader, types
├── chrome.go               # Chrome cookie extraction
├── chromium.go             # Shared Chromium logic (Chrome + Edge)
├── edge.go                 # Edge cookie extraction (thin wrapper over chromium)
├── firefox.go              # Firefox cookie extraction
├── safari.go               # Safari binary cookies parser (macOS only)
├── paths.go                # OS-specific profile path resolution
├── paths_darwin.go         # macOS paths
├── paths_windows.go        # Windows paths
├── paths_linux.go          # Linux paths
├── crypto.go               # Common crypto types
├── crypto_darwin.go         # macOS Keychain + AES-CBC decryption
├── crypto_windows.go        # DPAPI decryption
├── crypto_linux.go          # GNOME Keyring / KWallet / hardcoded key
├── domain.go               # Domain matching logic
├── domain_test.go          # Domain matching tests
├── header_test.go          # ToCookieHeader tests
├── safari_test.go          # Safari binary format parsing tests
├── firefox_test.go         # Firefox parsing tests
├── chromium_test.go        # Chromium parsing/decryption tests
├── integration_test.go     # Integration tests (build tag: integration)
├── cmd/
│   └── kurabiye/
│       └── main.go         # CLI entry point
├── AGENT.md
├── PLAN.md
├── DECISIONS.md
└── STATUS.md
```

## Browser-Specific Details

### Chromium-Based (Chrome, Edge)

**Cookie Database Location:**
- macOS: `~/Library/Application Support/Google/Chrome/Default/Cookies`
- Windows: `%LOCALAPPDATA%\Google\Chrome\User Data\Default\Cookies`  
  (or `Network/Cookies` in newer versions)
- Linux: `~/.config/google-chrome/Default/Cookies`

Edge uses similar paths with `Microsoft/Edge` instead of `Google/Chrome`.

**SQLite Schema (Chromium cookies table):**
```sql
CREATE TABLE cookies (
    creation_utc     INTEGER NOT NULL,
    host_key         TEXT NOT NULL,
    top_frame_site_key TEXT NOT NULL,
    name             TEXT NOT NULL,
    value            TEXT NOT NULL,
    encrypted_value  BLOB NOT NULL,
    path             TEXT NOT NULL,
    expires_utc      INTEGER NOT NULL,
    is_secure        INTEGER NOT NULL,
    is_httponly       INTEGER NOT NULL,
    last_access_utc  INTEGER NOT NULL,
    has_expires      INTEGER NOT NULL,
    is_persistent    INTEGER NOT NULL,
    priority         INTEGER NOT NULL,
    samesite         INTEGER NOT NULL,
    source_scheme    INTEGER NOT NULL,
    source_port      INTEGER NOT NULL,
    last_update_utc  INTEGER NOT NULL,
    source_type      INTEGER NOT NULL,
    has_cross_site_ancestor INTEGER NOT NULL
);
```

Key columns: `host_key`, `name`, `encrypted_value` (or `value` if unencrypted), `path`, `expires_utc`, `is_secure`, `is_httponly`, `samesite`.

**SameSite mapping:** 0=None (unspecified/Lax pre-default), 1=Lax, 2=Strict. Actually: -1=Unspecified, 0=None (explicitly set), 1=Lax, 2=Strict.

**Chromium Epoch:** Chromium timestamps are microseconds since January 1, 1601 (Windows epoch). To convert to Unix time: `(chromium_time / 1_000_000) - 11644473600`.

**Encryption:**
- macOS: AES-128-CBC with PBKDF2 key derived from a password stored in macOS Keychain. The Keychain entry is for service "Chrome Safe Storage" (or "Chromium Safe Storage"). Password is derived with PBKDF2-SHA1, 1003 iterations, 16-byte key, salt = "saltysalt". IV is 16 bytes of space (0x20). Encrypted values are prefixed with `v10`.
- Windows: DPAPI. In newer Chrome (v80+), there's an AES-256-GCM layer with a key stored in `Local State` JSON file, itself encrypted with DPAPI. Encrypted values prefixed with `v10`. Older values are just DPAPI-encrypted directly.
- Linux: Similar to macOS — PBKDF2 key derivation. Password retrieved from GNOME Keyring (`secret-tool lookup application chrome`) or KWallet, or falls back to hardcoded "peanuts". 1 iteration for PBKDF2 on Linux (not 1003). Salt = "saltysalt", 16-byte key, AES-128-CBC.

### Firefox

**Cookie Database Location:**
- macOS: `~/Library/Application Support/Firefox/Profiles/*.default-release/cookies.sqlite`
- Windows: `%APPDATA%\Mozilla\Firefox\Profiles\*.default-release\cookies.sqlite`
- Linux: `~/.mozilla/firefox/*.default-release/cookies.sqlite`

Need to parse `profiles.ini` or glob for profile directories.

**SQLite Schema:**
```sql
CREATE TABLE moz_cookies (
    id               INTEGER PRIMARY KEY,
    originAttributes TEXT NOT NULL DEFAULT '',
    name             TEXT,
    value            TEXT,
    host             TEXT,
    path             TEXT,
    expiry           INTEGER,     -- Unix timestamp
    lastAccessed     INTEGER,
    creationTime     INTEGER,
    isSecure         INTEGER,
    isHttpOnly       INTEGER,
    inBrowserElement INTEGER DEFAULT 0,
    sameSite         INTEGER DEFAULT 0,
    rawSameSite      INTEGER DEFAULT 0,
    schemeMap         INTEGER DEFAULT 0
);
```

Values are plaintext — no decryption needed.

### Safari (macOS only)

**File Location:** `~/Library/Cookies/Cookies.binarycookies`

**Binary Format:**
1. Header: magic bytes "cook" (4 bytes)
2. Number of pages: 4 bytes big-endian uint32
3. Page sizes: array of uint32 big-endian values, one per page
4. Pages: each page contains cookies
   - Page header: 4 bytes (0x00000100 little-endian)
   - Number of cookies in page: uint32 little-endian
   - Cookie offsets: array of uint32 little-endian
   - Cookie records (each at given offset within page):
     - Size: uint32 LE
     - Flags: uint32 LE (1=Secure, 4=HttpOnly, 5=Secure+HttpOnly)
     - Unknown: uint32 LE
     - URL offset: uint32 LE
     - Name offset: uint32 LE
     - Path offset: uint32 LE
     - Value offset: uint32 LE
     - Comment: uint32 LE (often 0)
     - Expiry date: float64 LE (Mac absolute time: seconds since 2001-01-01)
     - Creation date: float64 LE
     - String fields are null-terminated at their offsets within the cookie record
5. Checksum at end (can be ignored for reading)

**Mac Absolute Time Epoch:** January 1, 2001. To convert: add 978307200 to get Unix timestamp.

## Encryption/Decryption Approach

### macOS (Chromium)
1. Shell out to `security find-generic-password -s "Chrome Safe Storage" -w` to get the Keychain password
2. Derive AES key: PBKDF2-HMAC-SHA1(password, salt="saltysalt", iterations=1003, keyLen=16)
3. Strip the `v10` prefix from encrypted_value
4. Decrypt with AES-128-CBC, IV = 16 bytes of 0x20 (space character)
5. Remove PKCS#7 padding

### Windows (Chromium)
1. Read `Local State` file, parse JSON, extract `os_crypt.encrypted_key`
2. Base64-decode the key, strip the "DPAPI" prefix (5 bytes)
3. Decrypt with DPAPI via `powershell` or `certutil` — actually, will need to shell out to a PowerShell script that calls `[System.Security.Cryptography.ProtectedData]::Unprotect()`
4. For cookie values prefixed with `v10`: strip prefix, extract 12-byte nonce, ciphertext, 16-byte tag, decrypt with AES-256-GCM using the master key
5. For values without `v10` prefix: decrypt directly with DPAPI

### Linux (Chromium)
1. Try `secret-tool lookup application chrome` for GNOME Keyring
2. Fall back to hardcoded password "peanuts"
3. Derive AES key: PBKDF2-HMAC-SHA1(password, salt="saltysalt", iterations=1, keyLen=16)
4. Same AES-128-CBC decryption as macOS (IV = 16 x 0x20), strip `v10` prefix

### Firefox
No decryption needed — values stored in plaintext.

### Safari
No decryption needed — binary format but values are plaintext strings.

## Go Dependencies

- `modernc.org/sqlite` — pure-Go SQLite driver (no CGO)
- Standard library: `crypto/aes`, `crypto/cipher`, `crypto/sha1`, `golang.org/x/crypto/pbkdf2`, `encoding/json`, `os/exec`, `database/sql`, `flag`, `encoding/binary`, `math`, `net/url`
- `golang.org/x/crypto` — for PBKDF2

## Uncertainties and Resolutions

1. **Chromium cookie DB may be locked (WAL mode) while browser is running.**
   Resolution: Copy the DB file (and -wal, -shm files) to a temp directory before opening.

2. **Newer Chromium versions may use `Network/Cookies` instead of `Cookies`.**
   Resolution: Try `Default/Network/Cookies` first, fall back to `Default/Cookies`.

3. **Safari binary cookie format may have variations across OS versions.**
   Resolution: Implement based on the well-documented format. Handle parse errors gracefully.

4. **Firefox profile directory name varies (*.default, *.default-release, etc.).**
   Resolution: Parse `profiles.ini` to find the default profile, or glob for directories.

5. **DPAPI access on Windows requires the user's session.**
   Resolution: Document this requirement. Our library runs in the user's context.

6. **Edge on Linux may store its password under a different keyring entry.**
   Resolution: Try "Chromium Safe Storage" and "Microsoft Edge Safe Storage" entries.

7. **The `samesite` integer mapping in Chromium.**
   Resolution: Use -1=unspecified (map to ""), 0=None, 1=Lax, 2=Strict.

## Testing Strategy

1. **Unit tests (no build tags needed):**
   - `domain_test.go`: domain matching — exact match, subdomain match, parent domain cookies, path matching, edge cases
   - `header_test.go`: ToCookieHeader formatting, deduplication
   - `safari_test.go`: Parse a crafted binary cookies buffer
   - `chromium_test.go`: Test decryption with known key/ciphertext pairs, test Chromium timestamp conversion
   - `firefox_test.go`: Test Firefox timestamp conversion, SameSite mapping

2. **Integration tests (`//go:build integration`):**
   - `integration_test.go`: Actually extract cookies from installed browsers, requires real browser data on the machine

3. **Error path tests:**
   - Missing browser / missing cookie file
   - Locked database simulation
   - Expired cookie filtering
   - Invalid/corrupt data handling
