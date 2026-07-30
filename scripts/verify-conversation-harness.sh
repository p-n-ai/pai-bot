#!/bin/sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(dirname -- "$script_dir")
temp_root=$(mktemp -d "${TMPDIR:-/tmp}/pai-conversation-harness.XXXXXX")
binary="$temp_root/conversation-harness"
harness_cache="${TMPDIR:-/tmp}/pai-bot-conversation-harness-go-cache"

cleanup() {
  case "$temp_root" in
    */pai-conversation-harness.*) rm -R -- "$temp_root" ;;
  esac
}
trap cleanup EXIT HUP INT TERM

require_text() {
  result_file=$1
  expected_text=$2
  if ! grep -F -- "$expected_text" "$result_file" >/dev/null; then
    echo "missing expected verifier output: $expected_text" >&2
    echo "result file: $result_file" >&2
    exit 1
  fi
}

cd "$repo_root"
GOCACHE="$harness_cache" go build -o "$binary" ./cmd/conversation-harness

queue_result="$temp_root/queue.jsonl"
"$binary" \
  --case Q12 \
  --mock-response "You can read 3x as three copies of x." \
  --jsonl >"$queue_result"
require_text "$queue_result" '"passed":true'
require_text "$queue_result" '"delivered":3'
queue_normalized="$temp_root/queue-normalized.jsonl"
sed -E 's/"duration_ms":[0-9]+/"duration_ms":0/g' "$queue_result" >"$queue_normalized"
require_text "$queue_normalized" '"outcomes":[{"turn":1,"delivery":"wait","status":"delivered","duration_ms":0},{"turn":2,"delivery":"queue","status":"delivered","duration_ms":0},{"turn":3,"delivery":"queue","status":"delivered","duration_ms":0}]'

interrupt_result="$temp_root/interrupt.jsonl"
"$binary" \
  --case Q13 \
  --mock-response "You can read 4x as four copies of x." \
  --jsonl >"$interrupt_result"
require_text "$interrupt_result" '"passed":true'
require_text "$interrupt_result" '"interrupted":1'
require_text "$interrupt_result" '"turn":1,"delivery":"wait","status":"interrupted"'
require_text "$interrupt_result" '"turn":2,"delivery":"interrupt","status":"delivered"'

terse_result="$temp_root/terse.jsonl"
"$binary" \
  --case Q14 \
  --mock-response "Kita baca 2x sebagai dua kali x." \
  --jsonl >"$terse_result"
require_text "$terse_result" '"passed":true'
require_text "$terse_result" '"delivered":3'

long_reply="Kita baca 2x sebagai dua kali x. Setiap x mewakili satu nilai yang sama, jadi dua salinan x ditambah bersama. Bayangkan dua kotak yang beratnya sama, dengan setiap kotak bernilai x. Apabila kedua-duanya digabungkan, jumlahnya ialah 2x. Penerangan ini sengaja terlalu panjang untuk semakan had balasan ringkas."
negative_result="$temp_root/negative.jsonl"
if "$binary" \
  --case Q14 \
  --mock-response "$long_reply" \
  --jsonl >"$negative_result" 2>"$temp_root/negative.stderr"; then
  echo "negative verifier probe unexpectedly passed" >&2
  exit 1
fi
require_text "$negative_result" '"passed":false'
require_text "$negative_result" 'turn 2: response has'
require_text "$negative_result" 'max 260'

invalid_fixture="$temp_root/invalid-fixture.yaml"
printf '%s\n' \
  'version: 2' \
  'provider: mock' \
  'conversations:' \
  '  - id: INVALID01' \
  '    title: Unknown field must fail' \
  '    turns:' \
  '      - user: hello' \
  '        delivry: queue' >"$invalid_fixture"
if "$binary" \
  --fixture "$invalid_fixture" \
  --mock-response "unused" \
  --jsonl >"$temp_root/invalid.stdout" 2>"$temp_root/invalid.stderr"; then
  echo "invalid fixture unexpectedly passed" >&2
  exit 1
fi
require_text "$temp_root/invalid.stderr" 'field delivry not found'

echo "conversation harness verifier: PASS"
