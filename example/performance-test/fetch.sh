#!/usr/bin/env bash

set -eo pipefail
dir=$(dirname "$0")

RED="\033[31m"
GREEN="\033[32m"
NOCOLOR="\033[0m"

# check to see if vali-shoot is port-forwarded
if ! curl --silent --max-time 2 http://localhost:3100/metrics >/dev/null; then
    echo "Please port-forward vali-shoot to localhost:3100"
    echo "kubectl port-forward -n fluent-bit pod/logging-vali-shoot-0 3100:3100"
    exit 1
fi

for ((i=1; i<=CLUSTERS; i++)); do
    query="sum(count_over_time({namespace_name=\"shoot--logging--dev-${i}\"}[24h]))"
    echo "Querying logs for cluster dev-${i}..."

    # Safe parsing - handle null/empty results
    result=$(${dir}/bin/logcli query "$query" --quiet --output=jsonl 2>/dev/null || echo "")
    if [[ -n "$result" ]]; then
        out=$(printf '%s' "$result" | jq -r 'if .[0].values | length > 0 then (.[0].values | last | .[1]) else "0" end')
    else
        out="0"
    fi

    # Convert to number safely
    count=${out:-0}
    if ! [[ "$count" =~ ^[0-9]+$ ]]; then
        count=0
    fi

    if (( count < ((JOBS * LOGS)) )); then
        printf "got %b%d%b logs for cluster dev-%d\n" "${RED}" "$count" "${NOCOLOR}" "$i" >&2
        continue
    fi
    printf "got %b%d%b logs for cluster dev-%d\n" "${GREEN}" "$count" "${NOCOLOR}" "$i"
done