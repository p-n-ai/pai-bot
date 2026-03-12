#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

if [[ -f "${ROOT_DIR}/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "${ROOT_DIR}/.env"
  set +a
fi

POSTGRES_USER="${POSTGRES_USER:-pai}"
POSTGRES_DB="${POSTGRES_DB:-pai}"
PAI_DB_URL="${PAI_DB_URL:-${LEARN_DATABASE_URL:-}}"
DAYS=7
XLSX_OUTPUT=""
EXAMPLE_XLSX_OUTPUT=""

usage() {
  echo "Usage: scripts/analytics.sh [days] [--xlsx path] [--example-xlsx path]"
  echo "Examples:"
  echo "  scripts/analytics.sh 7"
  echo "  scripts/analytics.sh 14 --xlsx output/spreadsheet/pai-analytics.xlsx"
  echo "  scripts/analytics.sh --example-xlsx output/spreadsheet/pai-analytics-example.xlsx"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --xlsx)
      [[ $# -ge 2 ]] || { usage; exit 1; }
      XLSX_OUTPUT="$2"
      shift 2
      ;;
    --example-xlsx)
      [[ $# -ge 2 ]] || { usage; exit 1; }
      EXAMPLE_XLSX_OUTPUT="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      if [[ "$1" =~ ^[0-9]+$ ]] && [[ "$1" -gt 0 ]]; then
        DAYS="$1"
        shift
        continue
      fi
      usage
      exit 1
      ;;
  esac
done

if [[ -n "${XLSX_OUTPUT}" && -n "${EXAMPLE_XLSX_OUTPUT}" ]]; then
  echo "Choose either --xlsx or --example-xlsx, not both."
  exit 1
fi

run_psql() {
  local command="$1"

  if command -v psql >/dev/null 2>&1 && [[ -n "${PAI_DB_URL}" ]]; then
    psql "${PAI_DB_URL}" -X -v ON_ERROR_STOP=1 -P pager=off -c "${command}"
    return
  fi

  if command -v docker >/dev/null 2>&1; then
    docker compose exec -T postgres psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -X -v ON_ERROR_STOP=1 -P pager=off -c "${command}"
    return
  fi

  echo "Cannot run analytics: provide PAI_DB_URL with psql installed, or run with Docker Compose available."
  exit 1
}

run_sql_table() {
  run_psql "$1"
}

run_sql_csv() {
  run_psql "COPY ($1) TO STDOUT WITH CSV HEADER"
}

run_excel_builder() {
  go run ./cmd/analyticsxlsx "$@"
}

sql_dau() {
  cat <<'SQL'
SELECT
  TO_CHAR(day, 'YYYY-MM-DD') AS day,
  dau
FROM (
  SELECT
    DATE_TRUNC('day', m.created_at)::date AS day,
    COUNT(DISTINCT c.user_id) AS dau
  FROM messages m
  JOIN conversations c ON c.id = m.conversation_id
  WHERE m.created_at >= NOW() - INTERVAL '__DAYS__ days'
  GROUP BY 1
) d
ORDER BY day DESC
SQL
}

sql_messages_per_session() {
  cat <<'SQL'
WITH session_counts AS (
  SELECT
    c.id AS conversation_id,
    COUNT(m.*) FILTER (WHERE m.role IN ('user', 'assistant')) AS message_count
  FROM conversations c
  LEFT JOIN messages m ON m.conversation_id = c.id
  WHERE c.started_at >= NOW() - INTERVAL '__DAYS__ days'
  GROUP BY c.id
)
SELECT
  COUNT(*) AS sessions,
  ROUND(AVG(message_count)::numeric, 2) AS avg_messages_per_session,
  MAX(message_count) AS max_messages_in_session
FROM session_counts
SQL
}

sql_latency() {
  cat <<'SQL'
WITH latency_pairs AS (
  SELECT
    EXTRACT(
      EPOCH FROM (
        a.created_at - (
          SELECT u.created_at
          FROM messages u
          WHERE u.conversation_id = a.conversation_id
            AND u.role = 'user'
            AND u.created_at <= a.created_at
          ORDER BY u.created_at DESC
          LIMIT 1
        )
      )
    ) * 1000 AS latency_ms
  FROM messages a
  WHERE a.role = 'assistant'
    AND a.created_at >= NOW() - INTERVAL '__DAYS__ days'
)
SELECT
  COUNT(*) FILTER (WHERE latency_ms IS NOT NULL) AS samples,
  ROUND(AVG(latency_ms)::numeric, 2) AS avg_latency_ms,
  ROUND(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms)::numeric, 2) AS p95_latency_ms
FROM latency_pairs
SQL
}

sql_tokens_by_model() {
  cat <<'SQL'
SELECT
  COALESCE(NULLIF(model, ''), 'unknown') AS model,
  COUNT(*) AS responses,
  COALESCE(SUM(input_tokens), 0) AS input_tokens,
  COALESCE(SUM(output_tokens), 0) AS output_tokens,
  COALESCE(SUM(input_tokens + output_tokens), 0) AS total_tokens
FROM messages
WHERE role = 'assistant'
  AND created_at >= NOW() - INTERVAL '__DAYS__ days'
GROUP BY 1
ORDER BY total_tokens DESC, responses DESC
SQL
}

sql_returning_users() {
  cat <<'SQL'
WITH active_days AS (
  SELECT
    c.user_id,
    DATE_TRUNC('day', m.created_at)::date AS active_day
  FROM messages m
  JOIN conversations c ON c.id = m.conversation_id
  WHERE m.created_at >= NOW() - INTERVAL '__DAYS__ days'
  GROUP BY c.user_id, DATE_TRUNC('day', m.created_at)::date
),
user_active_day_counts AS (
  SELECT
    user_id,
    COUNT(*) AS active_days
  FROM active_days
  GROUP BY user_id
),
summary AS (
  SELECT
    COUNT(*) AS active_users,
    COUNT(*) FILTER (WHERE active_days >= 2) AS returning_users
  FROM user_active_day_counts
)
SELECT
  active_users,
  returning_users,
  CASE
    WHEN active_users = 0 THEN 0
    ELSE ROUND((returning_users::numeric / active_users) * 100, 2)
  END AS returning_rate_percent
FROM summary
SQL
}

sql_ratings_summary() {
  cat <<'SQL'
WITH ratings AS (
  SELECT
    e.id::text AS event_id,
    (e.data->>'rating')::int AS rating,
    NULLIF(e.data->>'rated_message_id', '') AS rated_message_id,
    COALESCE(NULLIF(e.data->>'source', ''), 'unknown') AS source
  FROM events e
  WHERE e.event_type = 'answer_rating_submitted'
    AND e.created_at >= NOW() - INTERVAL '__DAYS__ days'
),
summary AS (
  SELECT
    COUNT(*) AS ratings_submitted,
    COUNT(DISTINCT COALESCE(rated_message_id, event_id)) AS unique_rated_messages,
    ROUND(AVG(rating)::numeric, 2) AS avg_rating
  FROM ratings
)
SELECT
  ratings_submitted,
  unique_rated_messages,
  avg_rating
FROM summary
SQL
}

sql_ratings_by_source() {
  cat <<'SQL'
SELECT
  source,
  COUNT(*) AS submissions,
  ROUND(AVG(rating)::numeric, 2) AS avg_rating
FROM (
  SELECT
    COALESCE(NULLIF(e.data->>'source', ''), 'unknown') AS source,
    (e.data->>'rating')::int AS rating
  FROM events e
  WHERE e.event_type = 'answer_rating_submitted'
    AND e.created_at >= NOW() - INTERVAL '__DAYS__ days'
) r
GROUP BY source
ORDER BY submissions DESC, source ASC
SQL
}

sql_conversations() {
  cat <<'SQL'
SELECT
  c.id::text AS conversation_id,
  u.external_id AS user_id,
  TO_CHAR(c.started_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SSOF') AS started_at,
  COALESCE(TO_CHAR(c.ended_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SSOF'), '') AS ended_at,
  c.state,
  COALESCE(c.topic_id, '') AS topic_id,
  COUNT(m.id) FILTER (WHERE m.role IN ('user', 'assistant')) AS message_count,
  COALESCE(c.metadata->>'summary', '') AS summary,
  COALESCE(c.metadata->>'compacted_at', '') AS compacted_at,
  COALESCE(c.metadata::text, '{}') AS metadata
FROM conversations c
JOIN users u ON u.id = c.user_id
LEFT JOIN messages m ON m.conversation_id = c.id
WHERE c.started_at >= NOW() - INTERVAL '__DAYS__ days'
   OR EXISTS (
     SELECT 1
     FROM messages recent
     WHERE recent.conversation_id = c.id
       AND recent.created_at >= NOW() - INTERVAL '__DAYS__ days'
   )
GROUP BY c.id, u.external_id, c.started_at, c.ended_at, c.state, c.topic_id, c.metadata
ORDER BY c.started_at DESC
SQL
}

sql_conversation_messages() {
  cat <<'SQL'
SELECT
  c.id::text AS conversation_id,
  m.id::text AS message_id,
  m.role,
  REPLACE(REPLACE(m.content, E'\n', ' '), E'\r', ' ') AS content,
  COALESCE(m.model, '') AS model,
  COALESCE(m.input_tokens, 0) AS input_tokens,
  COALESCE(m.output_tokens, 0) AS output_tokens,
  TO_CHAR(m.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SSOF') AS created_at
FROM messages m
JOIN conversations c ON c.id = m.conversation_id
WHERE m.created_at >= NOW() - INTERVAL '__DAYS__ days'
ORDER BY m.created_at DESC
LIMIT 500
SQL
}

render_sql() {
  local sql="$1"
  printf '%s' "${sql//__DAYS__/${DAYS}}"
}

print_section() {
  local title="$1"
  local sql="$2"

  echo "=== ${title} ==="
  run_sql_table "$(render_sql "${sql}")"
  echo
}

export_csv_bundle() {
  local output_dir="$1"

  run_sql_csv "$(render_sql "$(sql_dau)")" >"${output_dir}/dau.csv"
  run_sql_csv "$(render_sql "$(sql_messages_per_session)")" >"${output_dir}/messages_per_session.csv"
  run_sql_csv "$(render_sql "$(sql_latency)")" >"${output_dir}/latency.csv"
  run_sql_csv "$(render_sql "$(sql_tokens_by_model)")" >"${output_dir}/tokens_by_model.csv"
  run_sql_csv "$(render_sql "$(sql_returning_users)")" >"${output_dir}/returning_users.csv"
  run_sql_csv "$(render_sql "$(sql_ratings_summary)")" >"${output_dir}/ratings_summary.csv"
  run_sql_csv "$(render_sql "$(sql_ratings_by_source)")" >"${output_dir}/ratings_by_source.csv"
  run_sql_csv "$(render_sql "$(sql_conversations)")" >"${output_dir}/conversations.csv"
  run_sql_csv "$(render_sql "$(sql_conversation_messages)")" >"${output_dir}/conversation_messages.csv"
}

if [[ -n "${EXAMPLE_XLSX_OUTPUT}" ]]; then
  run_excel_builder --example --days "${DAYS}" --output "${EXAMPLE_XLSX_OUTPUT}"
  echo "Excel example written to ${EXAMPLE_XLSX_OUTPUT}"
  exit 0
fi

echo "P&AI Analytics (last ${DAYS} day(s))"
echo "Generated at: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
echo

print_section "DAU" "$(sql_dau)"
print_section "Messages Per Session" "$(sql_messages_per_session)"
print_section "AI Latency (message -> assistant response)" "$(sql_latency)"
print_section "Tokens By Model" "$(sql_tokens_by_model)"
print_section "Returning Users" "$(sql_returning_users)"
print_section "Ratings" "$(sql_ratings_summary)"
print_section "Ratings By Source" "$(sql_ratings_by_source)"

if [[ -n "${XLSX_OUTPUT}" ]]; then
  mkdir -p "${ROOT_DIR}/tmp"
  EXPORT_DIR="$(mktemp -d "${ROOT_DIR}/tmp/analytics-xlsx.XXXXXX")"
  trap 'rm -rf "${EXPORT_DIR}"' EXIT
  export_csv_bundle "${EXPORT_DIR}"
  run_excel_builder --input-dir "${EXPORT_DIR}" --days "${DAYS}" --output "${XLSX_OUTPUT}"
  echo "Excel report written to ${XLSX_OUTPUT}"
fi
