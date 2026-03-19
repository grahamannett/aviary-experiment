# Decisions Log

This document records ambiguous design decisions made during implementation.

## 1. Modern Chrome Encryption Format (macOS)

**Problem:** Chrome v127+ on macOS changed the encrypted cookie value format. The
documented format (`v10` + AES-128-CBC with IV = `0x20 * 16`) no longer produced
correct decryption results. The first 28-32 bytes of decrypted values were garbled.

**Investigation:** By examining raw encrypted values from the SQLite database, I
found that modern Chrome prepends a 32-byte header after the `v10` prefix:
- Bytes 0-15: Unknown header (possibly key identifier/hash)
- Bytes 16-31: CBC Initialization Vector
- Bytes 32+: AES-128-CBC ciphertext (PKCS#7 padded)

The PBKDF2 key derivation (Keychain password, `saltysalt`, 1003 iterations, 16
bytes, SHA-1) remains unchanged.

**Decision:** Try the modern format first (16-byte header + 16-byte IV + ciphertext).
If the decrypted result contains non-printable ASCII characters, fall back to the
legacy format (fixed `0x20` IV over the entire data after `v10`). This handles both
old and new Chrome versions.

## 2. Chromium Cookie DB Location

**Problem:** Newer Chrome versions moved the cookie database from
`Default/Cookies` to `Default/Network/Cookies`.

**Decision:** Check `Default/Network/Cookies` first, fall back to `Default/Cookies`.

## 3. Database Locking

**Problem:** The SQLite cookie database may be locked (WAL mode) while the browser
is running.

**Decision:** Copy the database file (and associated `-wal` and `-shm` files) to a
temporary directory before opening. The temp directory is cleaned up after reading.

## 4. Firefox Profile Discovery

**Problem:** Firefox profile directories have unpredictable names like
`abc123.default-release` or `xyz.default`.

**Decision:** First try parsing `profiles.ini` to find the profile with `Default=1`.
If that fails, glob for `*.default-release` and `*.default` directories.

## 5. SameSite Integer Mapping

**Problem:** Chromium and Firefox use different integer mappings for SameSite.

**Decision:**
- Chromium: -1 = "" (unspecified), 0 = "None", 1 = "Lax", 2 = "Strict"
- Firefox: 0 = "None", 1 = "Lax", 2 = "Strict"

## 6. Safari Build Tags

**Problem:** Safari is only available on macOS. The Safari extraction code uses
macOS-specific binary format parsing.

**Decision:** Gate `safari.go` with `//go:build darwin` and provide a stub
`safari_other.go` with `//go:build !darwin` that returns an appropriate error.

## 7. Domain Matching

**Problem:** Cookie domain matching has edge cases (leading dots, subdomains).

**Decision:** Follow RFC 6265 semantics:
- Strip leading dots from cookie domains for comparison
- A cookie domain matches if it equals the request domain or the request domain
  is a subdomain
- Matching is case-insensitive

## 8. Expired Cookie Handling

**Problem:** Should expired cookies be returned?

**Decision:** Filter out expired cookies by default. Session cookies (zero expiry
time) are always included since they don't have an expiration.

## 9. No CGO Constraint

**Problem:** The requirement specifies pure Go only — no CGO.

**Decision:** Use `modernc.org/sqlite` for SQLite access (pure Go implementation).
For key retrieval, shell out to OS tools (`security` on macOS, `secret-tool` on
Linux). On Windows, use syscalls to `crypt32.dll` for DPAPI.

## 10. Linux PBKDF2 Iterations

**Problem:** Different platforms use different PBKDF2 iteration counts.

**Decision:**
- macOS: 1003 iterations
- Linux: 1 iteration
- Both use the same salt (`saltysalt`) and key length (16 bytes with SHA-1)

## 11. Edge on Linux Keyring Lookup

**Problem:** Edge on Linux may store its password under a different keyring entry
than Chrome.

**Decision:** For Edge on Linux, look up `application=chromium` in GNOME Keyring
(same as Chromium). For the Keychain on macOS, use `Microsoft Edge Safe Storage`.

## 12. Error Handling Strategy

**Problem:** How to handle failures for individual browsers?

**Decision:** Never panic. If a browser's cookie store can't be read, add a warning
to `GetCookiesResult.Warnings` and continue to the next browser. Only return a
hard error if the input parameters themselves are invalid (no URL, etc.).
