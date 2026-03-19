# Aviary — X/Twitter CLI Tool (Go)

## Instructions for AI Agent

You are building this project **fully autonomously with zero human guidance**.
You will receive no follow-up instructions, corrections, or hints. Everything
you need to know is in this file.

Follow this process exactly:

1. **Read this entire file first.**

2. **Generate a detailed implementation plan.** Write it to `PLAN.md`. Cover:
   - How you will discover Twitter/X's API (endpoints, auth, headers)
   - Architecture for the CLI
   - How you will integrate with kurabiye for auth
   - Which features you will implement in what order (prioritize core
     functionality first)
   - What risks you foresee and how you will mitigate them

3. **Execute the plan.** Build the CLI. Do NOT stop to ask for feedback or
   clarification. If something is ambiguous or an API call fails, debug it
   yourself, try alternative approaches, and document what you tried in
   `DECISIONS.md`.

4. **Test against the real Twitter/X API.** Use your kurabiye integration to
   get real cookies and make real API calls. Fix what breaks.

5. **When finished**, write `STATUS.md` summarizing: which features work,
   which failed and why, what you learned about the API, and total time/effort.

---

## Project Goal

Build a command-line tool called **aviary** that lets users read and interact
with X/Twitter from the terminal using their existing browser session for
authentication. No API keys or OAuth required.

## Features (in priority order)

Build as many as you can, in this order:

1. **Read timeline** — fetch and display the user's home timeline
2. **Read tweet** — display a single tweet by URL or ID
3. **Post tweet** — post a new text tweet
4. **Reply** — reply to an existing tweet
5. **Search** — search tweets by keyword
6. **View profile** — display a user's profile and recent tweets
7. **Bookmarks** — list the user's bookmarks
8. **Media upload** — attach images to tweets (stretch goal)

## Authentication

Use the **kurabiye** library at `../kurabiye-informed-go/` for cookie
extraction. Add a replace directive in `go.mod`:

```
replace github.com/youruser/kurabiye => ../kurabiye-informed-go
```

You need at minimum:
- `auth_token` cookie from `.twitter.com` / `.x.com`
- `ct0` cookie (CSRF token) from `.twitter.com` / `.x.com`

Authentication flow:
1. Extract cookies from the user's browser via kurabiye
2. Use cookies in HTTP requests
3. Set appropriate headers (the `ct0` cookie value typically must also be sent
   as an `x-csrf-token` header)

## Twitter/X API

Twitter/X's web client uses an undocumented GraphQL API. You will need to
figure out how it works. Key things to discover:

- The base URL for API requests
- What authorization headers are needed (there is typically a static bearer
  token used by the web client in addition to cookies)
- The GraphQL endpoint structure and how query/mutation IDs work
- Request payload format for each operation
- Response structure and how to extract tweet/user data
- Pagination mechanisms

**You may search the internet freely** for blog posts, reverse engineering
notes, documentation, or any public information about Twitter's web API,
GraphQL endpoints, or authentication mechanisms.

## CLI Interface

```
aviary timeline [--count N]
aviary tweet <text>
aviary reply <tweet-id> <text>
aviary search <query> [--count N]
aviary profile <username>
aviary bookmarks [--count N]
aviary read <tweet-url-or-id>
```

Default output: human-readable formatted text. `--json` flag for JSON output.

## Technical Constraints

- Go 1.22+
- Depend on `../kurabiye-informed-go/` via replace directive
- Minimal external dependencies beyond kurabiye
- Handle rate limiting gracefully (back off, warn the user)
- Clear error messages when auth fails or cookies are expired

## Success Criteria

A user with Chrome open and logged into twitter.com/x.com can run
`aviary timeline` and see their home feed. That is the minimum bar.
Everything else is bonus.
