# Kurabiye — Status Report

## What Works

### Chrome (macOS) — Fully Working
- Cookie database discovery (Default and Profile 1, both legacy and Network/ paths)
- Keychain password retrieval via `security find-generic-password`
- PBKDF2 key derivation (1003 iterations, SHA1, "saltysalt")
- AES-128-CBC decryption with fixed IV (16 × 0x20)
- Chrome 130+ SHA256(domain) prefix detection and stripping
- Domain matching and filtering
- Expired cookie exclusion
- Cookie name filtering
- Proper timestamp conversion (Chromium epoch → Go time.Time)
- SameSite integer-to-string mapping

### Edge (macOS) — Expected Working
- Shares all Chromium logic with Chrome
- Uses "Microsoft Edge Safe Storage" keychain name
- Not tested (Edge not installed on test machine)

### Firefox (macOS) — Working (Parser Verified)
- Profile discovery via profiles.ini parsing
- Fallback to glob-based profile discovery
- Cookie database reading (plaintext values, no decryption needed)
- Not tested with real Firefox installation (not installed on test machine)

### Safari (macOS) — Parser Working
- Binary cookies file parser implemented and unit-tested
- Parses magic header, pages, cookie records
- Handles Secure/HttpOnly flags
- Apple epoch timestamp conversion
- Not tested with real Safari cookies (file not found on test machine)

### CLI — Fully Working
- `--url`, `--browsers`, `--names`, `--mode`, `--header` flags
- JSON output (default) and Cookie header output
- Environment variables: `KURABIYE_BROWSERS`, `KURABIYE_MODE`
- Warnings printed to stderr

### Cross-Platform Code — Compiles
- Windows implementation (DPAPI via PowerShell, AES-256-GCM, Local State parsing)
- Linux implementation (secret-tool, PBKDF2 with 1 iteration, "peanuts" fallback)
- Platform-specific code isolated via build tags

## What Does Not Work / Untested

- **Windows**: DPAPI decryption via PowerShell is implemented but untested.
  Windows Chrome also uses AES-256-GCM (not CBC), which is implemented but
  not integration-tested.
- **Linux**: secret-tool integration is implemented but untested.
- **Multi-profile support**: Only checks Default and Profile 1. Users with
  multiple Chrome profiles may not get cookies from all profiles.

## Known Limitations

1. **Browser must have been run at least once** to create the cookie database.
2. **Keychain access on macOS** may prompt the user for permission the first
   time. Subsequent calls are typically allowed.
3. **Chrome 130+ domain binding**: The SHA256 domain hash is stripped
   automatically, but if Chrome changes this format again, decryption may
   produce garbled values.
4. **WAL mode**: We copy WAL and SHM files alongside the database, but if the
   browser is actively writing, there's a small window where the copy may be
   inconsistent.
5. **Safari cookie format changes**: Safari's binarycookies format is
   reverse-engineered. Future macOS versions may change it.
6. **No SameSite in Safari**: The binarycookies format doesn't expose SameSite
   attribute information.

## Test Results

- 22 unit tests: all passing
- Integration tests (Chrome on macOS): all passing
- Successfully extracted and decrypted 19 Google cookies from Chrome
- CLI produces valid JSON and Cookie header output

## Dependencies

- `modernc.org/sqlite` v1.47.0 — pure Go SQLite
- `golang.org/x/crypto` v0.49.0 — PBKDF2
- Go standard library for everything else
