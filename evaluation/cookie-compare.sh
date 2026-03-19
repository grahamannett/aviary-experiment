#!/usr/bin/env bash
# cookie-compare.sh — Run all three implementations and compare output
#
# Prerequisites:
#   - Node 22+ with @steipete/sweet-cookie installed
#   - kurabiye-blind-go built:    (cd ../kurabiye-blind-go && go build ./cmd/kurabiye)
#   - kurabiye-informed-go built: (cd ../kurabiye-informed-go && go build ./cmd/kurabiye)
#
# Usage: ./cookie-compare.sh https://twitter.com auth_token,ct0

set -euo pipefail

URL="${1:?Usage: $0 <url> <cookie-names>}"
NAMES="${2:?Usage: $0 <url> <cookie-names>}"
OUTDIR="$(dirname "$0")/comparison-$(date +%Y%m%d-%H%M%S)"

mkdir -p "$OUTDIR"

echo "=== Comparing cookie extraction for $URL ==="
echo "=== Cookie names: $NAMES ==="
echo "=== Output: $OUTDIR ==="
echo

# 1. Reference: sweet-cookie (TypeScript)
echo "[1/3] Running sweet-cookie (reference)..."
node -e "
const { getCookies } = require('@steipete/sweet-cookie');
getCookies({
  url: '$URL',
  names: '${NAMES}'.split(','),
  browsers: ['chrome', 'firefox', 'safari'],
}).then(r => console.log(JSON.stringify(r, null, 2)));
" > "$OUTDIR/sweet-cookie.json" 2> "$OUTDIR/sweet-cookie.stderr" || true

# 2. kurabiye-blind-go
echo "[2/3] Running kurabiye-blind-go..."
../kurabiye-blind-go/kurabiye \
  --url "$URL" \
  --names "$NAMES" \
  --browsers chrome,firefox,safari \
  > "$OUTDIR/kurabiye-blind-go.json" 2> "$OUTDIR/kurabiye-blind-go.stderr" || true

# 3. kurabiye-informed-go
echo "[3/3] Running kurabiye-informed-go..."
../kurabiye-informed-go/kurabiye \
  --url "$URL" \
  --names "$NAMES" \
  --browsers chrome,firefox,safari \
  > "$OUTDIR/kurabiye-informed-go.json" 2> "$OUTDIR/kurabiye-informed-go.stderr" || true

# Compare
echo
echo "=== Results ==="
for f in "$OUTDIR"/*.json; do
  name=$(basename "$f" .json)
  count=$(jq '.cookies // .Cookies | length' "$f" 2>/dev/null || echo "PARSE_ERROR")
  echo "  $name: $count cookies"
done

echo
echo "=== Cookie names found ==="
for f in "$OUTDIR"/*.json; do
  name=$(basename "$f" .json)
  names=$(jq -r '(.cookies // .Cookies)[] | .name // .Name' "$f" 2>/dev/null | sort | tr '\n' ', ' || echo "PARSE_ERROR")
  echo "  $name: $names"
done

echo
echo "Full output in $OUTDIR/ — cookie values are sensitive, do not commit."
echo
echo "=== Diff: blind vs informed ==="
diff <(jq -r '(.cookies // .Cookies)[] | "\(.name // .Name)=\(.domain // .Domain)"' "$OUTDIR/kurabiye-blind-go.json" 2>/dev/null | sort) \
     <(jq -r '(.cookies // .Cookies)[] | "\(.name // .Name)=\(.domain // .Domain)"' "$OUTDIR/kurabiye-informed-go.json" 2>/dev/null | sort) \
     || true
