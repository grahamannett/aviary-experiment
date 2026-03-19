# Methodology

## Overview

This project tests how well AI coding agents replicate existing software under
varying levels of information access. Three phases, each with strict rules
about what the AI can see.

## Phases

### Phase 1: kurabiye-blind-go (Clean-Room)

**Goal**: Build a Go cookie extraction library from a functional spec only.

**Rules**:
- AI receives ONLY `kurabiye-blind-go/AGENT.md`
- AI may NOT access any directory outside `kurabiye-blind-go/`
- AI may NOT search for existing cookie extraction library source code
- AI MAY search for OS/browser documentation (Keychain, DPAPI, SQLite schemas,
  binary formats, etc.)

**Isolation**: Run from a directory containing ONLY `kurabiye-blind-go/`. The
`reference/`, `kurabiye-informed-go/`, and `aviary/` directories must NOT be
on disk. See `evaluation/ISOLATION.md`.

**Recording**: Save full session transcript to
`evaluation/session-logs/kurabiye-blind-go.md`

### Phase 2: kurabiye-informed-go (Reference Port)

**Goal**: Port sweet-cookie to Go with full source access.

**Rules**:
- AI receives `kurabiye-informed-go/AGENT.md` AND the full sweet-cookie source
  in `kurabiye-informed-go/sweet-cookie/`
- AI MAY read and reference any file in the sweet-cookie repo
- AI should aim for behavioral parity

**Recording**: Save transcript to
`evaluation/session-logs/kurabiye-informed-go.md`

### Phase 3: aviary (Autonomous Application)

**Goal**: Build a Twitter/X CLI with feature parity to bird, zero human input.

**Rules**:
- AI receives ONLY `aviary/AGENT.md`
- Uses `kurabiye-informed-go` as cookie dependency
- ONE prompt, fully autonomous — no corrections, no follow-ups
- MAY search the internet freely

**Recording**: Save transcript to `evaluation/session-logs/aviary.md`

## Running Each Phase

### Agent startup command

For each phase, the initial human message to the agent should be identical:

> Read AGENT.md in this directory and follow its instructions completely.

For Claude Code specifically, you can symlink `AGENT.md` → `CLAUDE.md` so it
auto-loads:

```bash
cd kurabiye-blind-go && ln -s AGENT.md CLAUDE.md
```

For opencode or other tools, point them at `AGENT.md` per their docs.

### Recording sessions

Each tool has its own transcript mechanism:
- **Claude Code**: conversation history is saved automatically; export it
- **opencode**: check its logging/export options

Copy the transcript to the corresponding file in `evaluation/session-logs/`.

## Evaluation

### Cookie Correctness (Phases 1 & 2)

1. Run sweet-cookie (TypeScript) to extract cookies from twitter.com
2. Run kurabiye-blind-go against the same browsers
3. Run kurabiye-informed-go against the same browsers
4. Compare: same cookies? same count? decryption failures?

### Code Similarity (Phase 1 vs Phase 2)

Run JPlag, MOSS, or diff-based analysis on kurabiye-blind-go vs
kurabiye-informed-go:
- Structural similarity (file layout, function signatures)
- Algorithmic similarity (same approaches or different?)
- Architectural divergence

### Application Completeness (Phase 3)

Compare aviary against bird's feature list:
- [ ] Read timeline
- [ ] Post tweet
- [ ] Reply
- [ ] Search
- [ ] View profile
- [ ] Bookmarks
- [ ] Media upload
- [ ] Cookie-based auth

## Timing & Cost

Record per phase:
- Wall-clock time
- API cost (tokens in/out)
- Number of agent turns/iterations
- Number of web searches performed

## Environment

Fill in before starting:

- OS:
- Agent tool + version:
- Model:
- Go version:
- Node version: (for sweet-cookie reference)
- Browsers installed:
