#!/usr/bin/env bash
# Copyright 2025 SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-License-Identifier: Apache-2.0


set -eo pipefail
dir=$(dirname "$0")
QUERY_WAIT=${QUERY_WAIT:-1}       # seconds to wait before first query
QUERY_RETRIES=${QUERY_RETRIES:-10}  # number of query attempts
QUERY_INTERVAL=${QUERY_INTERVAL:-5} # seconds between query attempts

RED="\033[31m"
GREEN="\033[32m"
NOCOLOR="\033[0m"

# VictoriaLogs endpoint
VLOGS_ADDR="${VLOGS_ADDR:-http://localhost:9428/select/logsql/query}"

run_log_query() {
  # VictoriaLogs LogsQL query with count() pipe - efficient counting without fetching all logs
  local q="_time:24h k8s.container.name:logger | extract_regexp \".+id.: .(?P<id>([a-z]+|[0-9]+|-)+)\" from _msg | count_uniq(id)"
  local attempt=0
  while (( attempt < QUERY_RETRIES )); do
    # Query VictoriaLogs using curl with count() stats
    if result=$(curl -s --max-time 10 "${VLOGS_ADDR}" --data-urlencode "query=${q}"  2>/dev/null); then
      if [[ -n "$result" ]]; then
        count=$(printf '%s' "$result" | jq -r '."count_uniq(id)"')
        echo "Total logs found: ${count}"
        exit 0
      fi
    fi
    attempt=$((attempt+1))
    echo "No data yet (attempt ${attempt}/${QUERY_RETRIES}), retrying in ${QUERY_INTERVAL}s..."
    sleep "${QUERY_INTERVAL}"
  done
  echo "Failed to obtain data for query after ${QUERY_RETRIES} attempts" >&2
  return 1
}

run_log_query
