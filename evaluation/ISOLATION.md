# Running the Blind Phase in Isolation

The blind phase MUST run from a directory that does not contain the reference
implementation or the informed phase.

## Recommended: copy to isolated directory

```bash
# From the repo root
mkdir ~/kurabiye-blind-workspace
cp -r kurabiye-blind-go/ ~/kurabiye-blind-workspace/
cd ~/kurabiye-blind-workspace/kurabiye-blind-go

# Optional: create CLAUDE.md symlink for Claude Code auto-loading
ln -s AGENT.md CLAUDE.md

# Start the agent
claude  # or: opencode, aider, etc.
```

Initial prompt (same for every tool):

> Read AGENT.md in this directory and follow its instructions completely.

## After completion

```bash
# Copy results back
cp -r ~/kurabiye-blind-workspace/kurabiye-blind-go/ ./kurabiye-blind-go/

# Save the session transcript
cp <transcript> evaluation/session-logs/kurabiye-blind-go.md
```

## Pre-flight checklist

Before starting, verify from the agent's working directory:

- [ ] `ls ..` shows NO `reference/`, `kurabiye-informed-go/`, or `sweet-cookie/`
- [ ] No `.git` history containing sweet-cookie source
- [ ] Only `kurabiye-blind-go/` contents are accessible
