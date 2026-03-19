# Decisions Log

## D1: Chrome 130+ SHA256 Domain Hash Prefix

**Context**: Chrome 130+ prepends SHA256(domain) to the cookie value plaintext
before encrypting. This was discovered during testing when decrypted values had
32 bytes of binary data at the start.

**Decision**: After decryption, check if the first 32 bytes match
SHA256(host_key). If so, strip them. This maintains backward compatibility with
older Chrome versions that don't include this prefix.

## D2: Chromium SameSite Integer Mapping

**Context**: Chrome's cookies table stores samesite as an integer. The mapping
was determined empirically and from browser documentation.

**Decision**: Map as follows:
- `-1` → `"None"`
- `0` → `""` (unspecified)
- `1` → `"Lax"`
- `2` → `"Strict"`

## D3: Firefox SameSite Integer Mapping

**Decision**: Firefox uses different integer values:
- `0` → `"None"`
- `1` → `"Lax"`
- `2` → `"Strict"`

## D4: Database Locking Strategy

**Context**: Browsers hold WAL locks on their SQLite databases while running.

**Decision**: Copy the database file (and WAL/SHM files if present) to a
temporary location before opening. This avoids lock contention without requiring
the browser to be closed.

## D5: Chromium Cookie Path Resolution

**Context**: Chrome has moved the Cookies database between versions. Older
versions stored it directly in the profile directory; newer versions use a
`Network/` subdirectory.

**Decision**: Try multiple paths for each browser, preferring newer locations.
Currently check both `Default/Cookies` and `Default/Network/Cookies`, plus
`Profile 1` variants.

## D6: Firefox Profile Discovery Order

**Context**: Firefox has multiple ways to determine the default profile.

**Decision**: Priority order:
1. Parse `profiles.ini` → look for `[Install*]` section's `Default=` key
2. Parse `profiles.ini` → look for `[Profile*]` section with `Default=1`
3. Parse `profiles.ini` → use first profile with a valid path
4. Glob for `*.default-release` in Profiles directory
5. Glob for `*.default` in Profiles directory

## D7: Safari Build Tags

**Context**: Safari is macOS-only. The binarycookies parser and Safari backend
only compile on macOS.

**Decision**: Use `//go:build darwin` for Safari implementation and provide a
stub (`safari_stub.go` with `//go:build !darwin`) that returns an error on other
platforms.

## D8: Linux Chromium Keyring Fallback

**Context**: On Linux, if GNOME Keyring is not available (no `secret-tool`),
Chromium falls back to a hardcoded password.

**Decision**: Try secret-tool with the v2 schema
(`chrome_libsecret_os_crypt_password_v2`), then v1, then a simple
`application` lookup. If all fail, fall back to the password `"peanuts"` which
is Chromium's default when no keyring is available.

## D9: Safari Binarycookies Record Format

**Context**: The Safari cookie record has a 56-byte fixed header. After the
string offsets come 8 bytes of comment, then 4 bytes of unknown/padding, then
the expiry and creation timestamps.

**Decision**: Cookie record header layout (all little-endian):
size(4) + flags(4) + padding(4) + urlOffset(4) + nameOffset(4) +
pathOffset(4) + valueOffset(4) + comment(8) + unknown(4) + expiry(8) +
creation(8) = 56 bytes total. Safari also checks both the legacy path
(`~/Library/Cookies/`) and the sandboxed path
(`~/Library/Containers/com.apple.Safari/Data/Library/Cookies/`).

## D10: Pure Go SQLite

**Context**: The requirement is no CGO.

**Decision**: Use `modernc.org/sqlite` which is a pure Go translation of SQLite.
This works on all platforms without requiring a C compiler.

## D11: Edge Shares Chromium Backend

**Context**: Edge is Chromium-based and uses the same cookie storage format.

**Decision**: Chrome and Edge share the `chromiumBrowser` struct and all
decryption/reading logic. They only differ in:
- Database file paths
- macOS Keychain service name ("Chrome Safe Storage" vs "Microsoft Edge Safe Storage")
- Linux secret-tool application name
- Windows Local State file path
