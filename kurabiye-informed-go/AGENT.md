# Kurabiye (Informed) — Go Port of sweet-cookie

## Instructions for AI Agent

You are porting an existing TypeScript library to Go **fully autonomously**.
Follow this process exactly:

1. **Read this entire file first.**

2. **Study the reference implementation.** The sweet-cookie source is in
   `./sweet-cookie/` (cloned from https://github.com/steipete/sweet-cookie).
   Read through `./sweet-cookie/packages/core/src/` thoroughly before writing
   any Go code. Understand:
   - The overall architecture and module structure
   - How each browser backend works
   - How cookie decryption works per OS
   - How domain matching and origin filtering work
   - How inline cookie sources work
   - How errors/warnings are handled
   - The test suite and what it covers

3. **Generate a detailed implementation plan.** Write it to `PLAN.md`. Cover:
   - How you will map the TypeScript architecture to Go idioms
   - Which patterns translate directly vs. need rethinking
   - Your dependency choices and why
   - What the reference does that may be tricky in Go
   - Testing approach (porting existing tests + new ones)

4. **Execute the plan.** Build the complete library and CLI. Do NOT stop to ask
   for feedback, clarification, or approval. If something is ambiguous, check
   the reference implementation for the answer. If still ambiguous, decide,
   document in `DECISIONS.md`, and continue.

5. **Test thoroughly.** Port relevant tests from the reference. Write
   additional Go-idiomatic tests. Run them and fix failures.

6. **When finished**, write `STATUS.md` covering: what works, what does not,
   behavioral parity with the reference, and known limitations.

---

## Project Goal

Port **sweet-cookie** (`@steipete/sweet-cookie`) to Go. The Go library is
called **kurabiye** and must provide equivalent functionality: extracting HTTP
cookies from locally installed web browsers for programmatic HTTP requests.

## Reference Implementation

**Local path**: `./sweet-cookie/` — the full sweet-cookie repository.

Key locations in the reference:
- `packages/core/src/` — main library source
- `packages/core/src/__tests__/` or test files — test suite
- `docs/` — specification documents
- `apps/extension/` — Chrome extension (understand but do not port)

## Target API

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
    SameSite string    `json:"sameSite"`
    Source   string    `json:"source"`
}

type GetCookiesOptions struct {
    URL      string   // required — base URL for domain filtering
    Origins  []string // additional origins (OAuth/SSO, multi-domain auth)
    Names    []string // optional name allowlist
    Browsers []string // "chrome", "edge", "firefox", "safari"
    Mode     string   // "merge" (default) or "first"

    // Browser-specific profile overrides
    ChromeProfile  string // profile name, dir path, or Cookies DB path
    EdgeProfile    string
    FirefoxProfile string
    SafariCookiesFile string

    // Chromium variant targeting (e.g. "arc", "brave", "chrome")
    ChromiumBrowser string

    // Inline cookie sources (match sweet-cookie inline-first behavior)
    InlineCookiesJSON   string // raw JSON: Cookie[] or {cookies: Cookie[]}
    InlineCookiesBase64 string // base64-encoded JSON
    InlineCookiesFile   string // path to JSON/base64 file

    TimeoutMs      int  // max time for OS helper calls
    IncludeExpired bool
    Debug          bool
}

type GetCookiesResult struct {
    Cookies  []Cookie `json:"cookies"`
    Warnings []string `json:"warnings"`
}

func GetCookies(opts GetCookiesOptions) (*GetCookiesResult, error)

// ToCookieHeader formats cookies as an HTTP Cookie header string.
// When dedupeByName is true, keep only the first occurrence of each name.
func ToCookieHeader(cookies []Cookie, dedupeByName bool) string
```

### Browser Support — match sweet-cookie exactly

| Browser  | macOS | Windows | Linux |
|----------|-------|---------|-------|
| Chrome   | ✓     | ✓       | ✓     |
| Edge     | ✓     | ✓       | ✓     |
| Firefox  | ✓     | ✓       | ✓     |
| Safari   | ✓     | —       | —     |

### CLI

Build at `cmd/kurabiye/main.go`:

```
kurabiye --url https://twitter.com --browsers chrome,firefox --names auth_token,ct0
kurabiye --url https://twitter.com --browsers chrome --header
```

JSON output by default. `--header` for `Cookie: k=v; k2=v2`.

## Behavioral Parity Checklist

Verify each against the reference:

- [ ] Inline cookies short-circuit browser reads when they yield results
- [ ] Domain matching handles parent-domain cookies
      (`.google.com` matches `gemini.google.com`)
- [ ] `mode: merge` combines across all backends
- [ ] `mode: first` stops at first browser that yields cookies
- [ ] Expired cookies excluded by default, included with `IncludeExpired`
- [ ] Warnings emitted for inaccessible browsers; never panic
- [ ] `ToCookieHeader` deduplication keeps first occurrence
- [ ] Environment variable overrides: `KURABIYE_BROWSERS`, `KURABIYE_MODE`,
      `KURABIYE_CHROME_PROFILE`, `KURABIYE_EDGE_PROFILE`,
      `KURABIYE_FIREFOX_PROFILE`
- [ ] Linux keyring support matches reference
      (`KURABIYE_LINUX_KEYRING=gnome|kwallet|basic`)
- [ ] ChromiumBrowser targeting works (arc, brave, chrome, edge)

## Technical Constraints

- Go 1.22+
- `modernc.org/sqlite` (pure Go, no CGO)
- Minimal dependencies
- Shell out to OS tools for key retrieval (match sweet-cookie's approach)

## Testing

- Port or adapt test cases from the reference `./sweet-cookie/`
- Unit tests: parsing, domain matching, decryption, header formatting
- Integration tests: `//go:build integration` — real cookie extraction
- Compare output against the TypeScript version for identical inputs

## Success Criteria

Given the same browser state and options, kurabiye and sweet-cookie produce
the same cookies. Output from both should be comparable field-by-field.
