#!/usr/bin/env bash
# Benchmark the dynamic compression server with `hey`.
#
# Reads experiments from a file (arg 1) or stdin, one per line:
#
#   <encoding> <level>
#
# e.g.
#   br   5
#   br   6
#   zstd 1

set -euo pipefail

TARGET="${BENCH_TARGET:-localhost:8080}"
DURATION="${BENCH_DURATION:-10s}"
CONCURRENCY="${BENCH_CONCURRENCY:-1}"

command -v hey >/dev/null 2>&1 || { echo "missing required tool: hey" >&2; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "missing required tool: curl" >&2; exit 1; }

n=0
while read -r encoding level _; do
  case "$encoding" in
    ''|'#'*) continue ;;  # skip blanks and comments
  esac
  if [[ ! "$level" =~ ^[0-9]+$ ]]; then
    echo "skipping malformed line (need '<encoding> <level>'): $encoding $level" >&2
    continue
  fi

  url="http://$TARGET/?level=$level"
  n=$((n + 1))
  out="/tmp/hey-$(printf '%03d' "$n")-${encoding}-${level}.txt"

  # Bytes per request: one probe, no --compressed so curl keeps the body as
  # sent (compressed) and size_download is the on-the-wire body size.
  size="$(curl -s -o /dev/null -w '%{size_download}' \
    -H "Accept-Encoding: $encoding" "$url")"

  hey -z "$DURATION" -c "$CONCURRENCY" -disable-compression \
    -H "Accept-Encoding: $encoding" "$url" >"$out" 2>&1

  rps="$(awk '/Requests\/sec:/ {print $2}' "$out")"

  if [[ -z "$size" || -z "$rps" ]]; then
    echo "could not measure run #$n; see $out" >&2
    continue
  fi
  printf '%s,%s,%s,%s\n' "$encoding" "$level" "$size" "$rps"
done < "${1:-/dev/stdin}"
