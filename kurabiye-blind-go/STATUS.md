# Status

## What Works

### Chrome (macOS) -- Fully Tested
- Cookie extraction from Chrome on macOS: **working**
- Decryption of encrypted cookie values: **working**
- Handles both modern (Chrome v127+) and legacy encrypted value formats
- Keychain password retrieval via `security` command
- PBKDF2 key derivation with correct parameters (1003 iterations)
- Domain and path filtering: **working**
- Name filtering: **working**
- Expired cookie filtering: **working**
- Header output mode: **working**
- JSON output mode: **working**
- Database copy to avoid WAL locking: **working**

### Edge (macOS) -- Implemented, Not Tested
- Shares Chromium backend with Chrome
- Uses "Microsoft Edge Safe Storage" Keychain entry
- Expected to work if Edge is installed

### Firefox -- Implemented, Not Tested on This Machine
- Profile discovery via `profiles.ini` parsing
- Glob-based fallback for profile directories
- SQLite cookie extraction (plaintext values)
- Tested: profile INI parsing, timestamp conversion, SameSite mapping

### Safari (macOS) -- Implemented, Not Tested on This Machine
- Binary cookies file parser (`Cookies.binarycookies`)
- Handles the multi-page binary format
- Mac absolute time conversion
- Tested: binary format parsing with synthetic data, timestamp conversion

### Windows -- Implemented, Not Tested
- Chrome and Edge cookie extraction
- DPAPI decryption via syscall to `crypt32.dll`
- AES-256-GCM decryption for v10-prefixed values
- `Local State` file parsing for encrypted master key

### Linux -- Implemented, Not Tested
- Chrome and Edge cookie extraction
- GNOME Keyring password retrieval via `secret-tool`
- Fallback to hardcoded "peanuts" password
- PBKDF2 key derivation (1 iteration)

### CLI
- `--url` flag (required): **working**
- `--browsers` flag: **working**
- `--names` flag: **working**
- `--mode` flag: **working**
- `--header` flag: **working**
- `KURABIYE_BROWSERS` env var: **implemented**
- `KURABIYE_MODE` env var: **implemented**

### Unit Tests
- 30 tests, all passing:
  - Domain matching (15 cases)
  - Path matching (11 cases)
  - Cookie header formatting (4 cases)
  - API error handling (3 cases)
  - Chromium epoch conversion (3 cases)
  - Chromium SameSite mapping (5 cases)
  - Firefox expiry conversion (2 cases)
  - Firefox SameSite mapping (4 cases)
  - Firefox profiles.ini parsing (2 cases)
  - Safari binary format parsing (4 cases)
  - Mac absolute time conversion (6 cases)
  - Null-terminated string parsing (6 cases)

## What Does Not Work / Known Limitations

1. **Safari on modern macOS (Sonoma+):** Safari may have moved or restricted
   access to `Cookies.binarycookies`. The file may not exist at the traditional
   path. Sandboxing may prevent reading.

2. **Chrome profiles beyond Default:** Only reads from the "Default" profile.
   Users with multiple Chrome profiles would need to specify the profile directory
   manually (not currently supported via API).

3. **Windows DPAPI testing:** The Windows implementation uses direct syscalls to
   `crypt32.dll` which can only be tested on Windows. The implementation follows
   standard DPAPI patterns.

4. **KWallet on Linux:** Only GNOME Keyring (`secret-tool`) is supported. KDE's
   KWallet is not implemented.

5. **Brave/Opera/Vivaldi:** Only Chrome and Edge are supported as Chromium-based
   browsers. Other Chromium forks would need additional path definitions.

6. **Session-only cookies:** Session cookies (no expiry) are included in results.
   There's no way to filter them out separately.

7. **Concurrent browser access:** If the browser writes new cookies while we're
   copying the database, we might get a slightly stale snapshot. This is by design
   — the copy approach avoids lock contention.

## Platform-Specific Caveats

### macOS
- Requires Keychain access — may prompt the user for permission the first time
- `security find-generic-password` may require Full Disk Access in some cases
- Modern Chrome (v127+) uses a different encryption format with a 32-byte header
  before the CBC ciphertext

### Windows
- DPAPI decryption requires running in the user's login session
- Chrome's master key is encrypted with DPAPI in `Local State`
- Both v10 (AES-256-GCM) and legacy (direct DPAPI) formats are supported

### Linux
- `secret-tool` must be installed for GNOME Keyring access
- Falls back to hardcoded password "peanuts" if keyring is unavailable
- Uses 1 PBKDF2 iteration (vs 1003 on macOS)
