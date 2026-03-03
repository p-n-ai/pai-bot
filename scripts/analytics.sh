#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

DAYS="${1:-7}"
if ! [[ "${DAYS}" =~ ^[0-9]+$ ]] || [[ "${DAYS}" -le 0 ]]; then
  echo "Usage: scripts/analytics.sh [days]"
  echo "Example: scripts/analytics.sh 7"
  exit 1
fi

POSTGRES_USER="${POSTGRES_USER:-pai}"
POSTGRES_DB="${POSTGRES_DB:-pai}"

run_sql() {
  local sql="$1"

  if command -v psql >/dev/null 2>&1 && [[ -n "${PAI_DB_URL:-}" ]]; then
    psql "${PAI_DB_URL}" -X -v ON_ERROR_STOP=1 -P pager=off -c "${sql}"
    return
  fi

  if command -v docker >/dev/null 2>&1; then
    docker compose exec -T postgres psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -X -v ON_ERROR_STOP=1 -P pager=off -c "${sql}"
    return
  fi

  echo "Cannot run analytics: provide PAI_DB_URL with psql installed, or run with Docker Compose available."
  exit 1
}

echo "P&AI Analytics (last ${DAYS} day(s))"
echo "Generated at: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
echo

echo "=== DAU ==="
run_sql "
SELECT
  TO_CHAR(day, 'YYYY-MM-DD') AS day,
  dau
FROM (
  SELECT
    DATE_TRUNC('day', m.created_at)::date AS day,
    COUNT(DISTINCT c.user_id) AS dau
  FROM messages m
  JOIN conversations c ON c.id = m.conversation_id
  WHERE m.created_at >= NOW() - INTERVAL '${DAYS} days'
  GROUP BY 1
) d
ORDER BY day DESC;
"
echo

echo "=== Messages Per Session ==="
run_sql "
WITH session_counts AS (
  SELECT
    c.id AS conversation_id,
    COUNT(m.*) FILTER (WHERE m.role IN ('user', 'assistant')) AS message_count
  FROM conversations c
  LEFT JOIN messages m ON m.conversation_id = c.id
  WHERE c.started_at >= NOW() - INTERVAL '${DAYS} days'
  GROUP BY c.id
)
SELECT
  COUNT(*) AS sessions,
  ROUND(AVG(message_count)::numeric, 2) AS avg_messages_per_session,
  MAX(message_count) AS max_messages_in_session
FROM session_counts;
"
echo

echo "=== AI Latency (message -> assistant response) ==="
run_sql "
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
    AND a.created_at >= NOW() - INTERVAL '${DAYS} days'
)
SELECT
  COUNT(*) FILTER (WHERE latency_ms IS NOT NULL) AS samples,
  ROUND(AVG(latency_ms)::numeric, 2) AS avg_latency_ms,
  ROUND(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms)::numeric, 2) AS p95_latency_ms
FROM latency_pairs;
"
echo

echo "=== Tokens By Model ==="
run_sql "
SELECT
  COALESCE(NULLIF(model, ''), 'unknown') AS model,
  COUNT(*) AS responses,
  COALESCE(SUM(input_tokens), 0) AS input_tokens,
  COALESCE(SUM(output_tokens), 0) AS output_tokens,
  COALESCE(SUM(input_tokens + output_tokens), 0) AS total_tokens
FROM messages
WHERE role = 'assistant'
  AND created_at >= NOW() - INTERVAL '${DAYS} days'
GROUP BY 1
ORDER BY total_tokens DESC, responses DESC;
"
echo

echo "=== Returning Users ==="
run_sql "
WITH active_days AS (
  SELECT
    c.user_id,
    DATE_TRUNC('day', m.created_at)::date AS active_day
  FROM messages m
  JOIN conversations c ON c.id = m.conversation_id
  WHERE m.created_at >= NOW() - INTERVAL '${DAYS} days'
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
FROM summary;
"
echo

echo "=== Ratings ==="
run_sql "
WITH ratings AS (
  SELECT
    e.id::text AS event_id,
    (e.data->>'rating')::int AS rating,
    NULLIF(e.data->>'rated_message_id', '') AS rated_message_id,
    COALESCE(NULLIF(e.data->>'source', ''), 'unknown') AS source
  FROM events e
  WHERE e.event_type = 'answer_rating_submitted'
    AND e.created_at >= NOW() - INTERVAL '${DAYS} days'
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
FROM summary;
"
echo

echo "=== Ratings By Source ==="
run_sql "
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
    AND e.created_at >= NOW() - INTERVAL '${DAYS} days'
) r
GROUP BY source
ORDER BY submissions DESC, source ASC;
"
