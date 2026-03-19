# Kurabiye — Browser Cookie Extraction Library (Go)

## Instructions for AI Agent

You are building this project **fully autonomously**. Follow this process
exactly:

1. **Read this entire file first.** Do not start coding until you have read
   every section.

2. **Generate a detailed implementation plan.** Write it to `PLAN.md` in this
   directory. The plan must cover:
   - Architecture and file/package structure
   - How you will handle each browser on each OS
   - What encryption/decryption approach you will use per platform
   - What Go dependencies you will need
   - What you are uncertain about and how you will resolve each uncertainty
   - Your testing strategy

3. **Execute the plan.** Build the complete library and CLI. Do NOT stop to ask
   for human feedback, clarification, or approval at any point. If something
   is ambiguous, make a reasonable engineering decision, document it in
   `DECISIONS.md`, and continue.

4. **Test thoroughly.** Write and run tests. Fix what breaks. Iterate until
   the core functionality works.

5. **When finished**, write `STATUS.md` summarizing: what works, what does not
   work, known limitations, and any platform-specific caveats.

### Search and Reference Rules

This is a **clean-room implementation**. You must follow these rules strictly.

**FORBIDDEN — do NOT do any of the following:**
- Search for, fetch, read, clone, or reference the source code of any existing
  browser cookie extraction library, including but not limited to:
  `sweet-cookie`, `browser_cookie3`, `kooky`, `go-browser-cookie`,
  `chromedp/cookie`, or similar projects
- Access any directory outside this project directory (no `ls ..`, no
  `cat ../`, no `find /` for source code)
- Search the web for queries like "sweet-cookie source", "browser cookie
  extraction library implementation", "cookie extraction golang github" or
  any query whose purpose is to find an existing implementation to copy from
- Reference any source code you may have seen in training data from these
  libraries — reason from OS/browser documentation instead

**ALLOWED — you MAY do the following:**
- Search for and read Go standard library and third-party package documentation
  on pkg.go.dev
- Search for how specific browsers store cookies on disk — file locations,
  SQLite schema, encryption methods. These are browser implementation details,
  not library code. Example good queries:
  - "Chrome cookie database SQLite schema"
  - "macOS Keychain security CLI extract password"
  - "Windows DPAPI decrypt data powershell"
  - "Firefox cookies.sqlite table structure"
  - "Safari Cookies.binarycookies file format specification"
  - "Linux GNOME Keyring secret-tool lookup"
- Search for Go patterns: "golang exec command timeout", "golang SQLite
  without CGO", etc.
- Read man pages, MSDN docs, Apple developer docs

**The rule of thumb:** you may research how browsers and operating systems
work. You may NOT look at how other developers have wrapped that knowledge
into a library.

---

## Project Goal

Build a Go library and CLI called **kurabiye** that extracts HTTP cookies from
a user's locally installed web browsers. The extracted cookies should be usable
for programmatic HTTP requests — authenticating against a site using the user's
existing browser session.

## Functional Requirements

### Core API

```go
package kurabiye

import "time"

type Cookie struct {
    Name     string    `json:"name"`
    Value    string    `json:"value"`
    Domain   string    `json:"domain"`
    Path     string    `json:"path"`
    Expires  time.Time `json:"expires"`
    Secure   bool      `json:"secure"`
    HTTPOnly bool      `json:"httpOnly"`
    SameSite string    `json:"sameSite"` // "Strict", "Lax", "None", or ""
    Source   string    `json:"source"`   // which browser produced this cookie
}

type GetCookiesOptions struct {
    URL      string   // required — base URL for origin/domain filtering
    Names    []string // optional — only return cookies matching these names
    Browsers []string // which browsers to try: "chrome", "edge", "firefox", "safari"
    Mode     string   // "merge" (default) or "first"
}

type GetCookiesResult struct {
    Cookies  []Cookie `json:"cookies"`
    Warnings []string `json:"warnings"`
}

func GetCookies(opts GetCookiesOptions) (*GetCookiesResult, error)
```

Helper:

```go
// ToCookieHeader formats cookies as an HTTP Cookie header string.
// When dedupeByName is true, keep only the first occurrence of each name.
func ToCookieHeader(cookies []Cookie, dedupeByName bool) string
```

### Browser Support

| Browser  | macOS | Windows | Linux |
|----------|-------|---------|-------|
| Chrome   | ✓     | ✓       | ✓     |
| Edge     | ✓     | ✓       | ✓     |
| Firefox  | ✓     | ✓       | ✓     |
| Safari   | ✓     | —       | —     |

**Chromium-based (Chrome, Edge):**
- Cookies stored in a SQLite database
- Cookie values are encrypted — decryption is OS-specific
- The database file may be locked while the browser is running; you may need
  to copy it to a temp location first

**Firefox:**
- Cookies in SQLite (`cookies.sqlite`) inside the user's profile directory
- Values stored in plaintext (not encrypted at rest)

**Safari (macOS only):**
- Proprietary binary format file: `Cookies.binarycookies`
- Located in `~/Library/Cookies/Cookies.binarycookies`

### Behavior

1. **Domain/origin filtering**: given `https://twitter.com/`, return only
   cookies whose domain matches (including parent-domain cookies like
   `.twitter.com` for subdomain `x.com`).

2. **Name filtering**: if `Names` is non-empty, only return matching cookies.

3. **Mode**:
   - `merge` (default): try all requested browsers, combine all results
   - `first`: return cookies from the first browser that yields any

4. **Expired cookies**: excluded by default.

5. **Resilience**: if a browser's cookie store can't be read (locked DB,
   missing browser, decryption failure), add a warning and continue to the
   next browser. Never panic.

6. **No CGO**: pure Go only. Use `modernc.org/sqlite` or equivalent for
   SQLite. Shell out to OS command-line tools for key retrieval.

### CLI

Build at `cmd/kurabiye/main.go`:

```
kurabiye --url https://twitter.com --browsers chrome,firefox --names auth_token,ct0
kurabiye --url https://twitter.com --browsers chrome --header
```

Default output: JSON to stdout. `--header` outputs `Cookie: k=v; k2=v2`.

### Suggested Project Structure

```
kurabiye-blind-go/
├── AGENT.md
├── PLAN.md           # generate this first
├── DECISIONS.md      # document ambiguity resolutions here
├── STATUS.md         # write this when done
├── go.mod
├── go.sum
├── kurabiye.go       # public API
├── chrome.go         # Chrome backend
├── edge.go           # Edge backend (shares Chromium logic with Chrome)
├── firefox.go        # Firefox backend
├── safari.go         # Safari backend
├── paths.go          # OS-specific profile path resolution
├── crypto.go         # OS-specific decryption
├── domain.go         # domain matching
├── cmd/
│   └── kurabiye/
│       └── main.go
└── *_test.go
```

Adapt as needed — this is a starting point, not a rigid requirement.

### Testing

- Unit tests for domain matching, header formatting, any parsing logic
- Integration tests behind `//go:build integration` for real cookie extraction
- Test error paths: missing browser, locked database, expired cookies

### Environment Variables

- `KURABIYE_BROWSERS`: comma-separated browser list
- `KURABIYE_MODE`: `merge` or `first`

## Success Criteria

The library can extract a valid authentication cookie from Chrome on the
current OS, and that cookie works in an HTTP request to the target site.
