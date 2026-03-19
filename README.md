# Aviary Project

This repo is a multi-phase experiment testing if AI can port a deleted software project.

## The Experiment

A useful open-source Twitter/X CLI tool ([bird](https://github.com/steipete/bird)) was recently removed from GitHub. It relies on a cookie extraction library ([sweet-cookie](https://github.com/steipete/sweet-cookie)) which still exists.

### Phase 1: kurabiye-blind-go

Port sweet-cookie to Go using only a functional specification. The AI cannot
see the original source code. Clean-room implementation.

### Phase 2: kurabiye-informed-go

Port sweet-cookie to Go with full access to the original TypeScript source.
Same goal, maximum information.

### Phase 3: aviary

Rebuild the bird CLI from scratch. Fully autonomous — one prompt, zero human
intervention. The AI must discover how to interact with Twitter/X's
undocumented GraphQL API on its own.

## Structure

```
├── kurabiye-blind-go/      # Phase 1: clean-room Go port
├── kurabiye-informed-go/   # Phase 2: informed Go port (reference in sweet-cookie/)
├── aviary/                 # Phase 3: autonomous bird CLI replacement
├── reference/              # Original source snapshots
├── evaluation/             # Comparison tools, session logs, similarity analysis
├── blog/                   # Blog post drafts
├── METHODOLOGY.md          # Rules of engagement
└── README.md               # This file
```

## Running

See `METHODOLOGY.md` for detailed instructions on how to run each phase,
including isolation requirements for the blind phase.
