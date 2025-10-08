#!/usr/bin/env bash

set -eo pipefail
dir=$(dirname "$0")

RED="\033[31m"
GREEN="\033[32m"
NOCOLOR="\033[0m"


for ((i=1; i<=CLUSTERS; i++)); do
    query="sum(count_over_time({namespace_name=\"shoot--logging--dev-${i}\"}[24h]))"
    echo "Querying logs for cluster dev-${i}..."

    # Safe parsing - handle null/empty results
    result=$(logcli query "$query" --quiet --output=jsonl 2>/dev/null || echo "")
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

    if [[ "$count" -ne "$((JOBS * LOGS))" ]]; then
        printf "got %b%d%b logs for cluster dev-%d\n" "${RED}" "$count" "${NOCOLOR}" "$i" >&2
        continue
    fi
    printf "got %b%d%b logs for cluster dev-%d\n" "${GREEN}" "$count" "${NOCOLOR}" "$i"
done