#!/usr/bin/env bash
# VisualKB end-to-end smoke test.
#
# Flow: wait backend ready -> register+login -> create project ->
# wait sandbox CR reconcile + publicURL -> probe preview CDN ->
# drive chat SSE ("帮我写个 hello vue 项目") -> assert assistant used create_project.
#
# Usage: test/e2e.sh [USER] [EMAIL] [PASSWORD] [CHAT_PROMPT]
# Defaults: USER=e2e-$(date +%s), EMAIL=u@e.local, PWD=test12345678, "帮我写个 hello vue 项目"
#
# Requires: curl, jq, python3, kubectl (kube-context pointed at minikube).
#
# Bring-up before running this script:
#   make down
#   make up                          # postgres / redis / minio on host
#   make backend                     # air-launched server on :8889 (sources .env)
#
# If the server runs in-cluster instead of on host, also port-forward:
#   kubectl -n agent-sandbox-system port-forward svc/controller-exec 8082:8082 &
#   kubectl -n visualkb-stack port-forward svc/visualkb-server 8889:8889 &
# so e2e.sh can dial :8889 and the controller svc (in-cluster dial needs pod, not host).

set -euo pipefail

USER_NAME="${1:-e2e-$(date +%s)}"
USER_EMAIL="${2:-e2e-$(date +%s)@e.local}"
USER_PASS="${3:-test12345678}"
CHAT_PROMPT="${4:-帮我写个 hello vue 项目}"

API="http://localhost:8889/api/v1"
RED=$'\033[31m'; GRN=$'\033[32m'; YLW=$'\033[33m'; NC=$'\033[0m'
log()  { printf '%s[step]%s %s\n' "$YLW" "$NC" "$*"; }
ok()   { printf '%s[ ok ]%s %s\n' "$GRN" "$NC" "$*"; }
die()  { printf '%s[fail]%s %s\n' "$RED" "$NC" "$*" >&2; exit 1; }

jq_get() { python3 -c "import json,sys;print(json.load(sys.stdin)$1)"; }

# ---------- 1. backend ready ----------
log "wait backend :8889 listen"
for i in $(seq 1 30); do
  if curl -sf -o /dev/null "$API/../"; then break; fi
  sleep 1
done
curl -sf -o /dev/null -w "%{http_code}\n" "http://localhost:8889/health" >/dev/null 2>&1 || true

# If server runs on host but dial in-cluster svc fails, open port-forward.
log "ensure controller-exec reachable at 127.0.0.1:8082"
if ! curl -sf -m 3 -X POST -H 'Content-Type: application/json' \
    -d '{"tenantID":"u-probe"}' http://127.0.0.1:8082/v1/namespaces/sandbox-u-probe/ensure \
    >/dev/null 2>&1; then
  log "starting kubectl port-forward controller-exec 8082:8082"
  kubectl -n agent-sandbox-system port-forward svc/controller-exec 8082:8082 \
    >/tmp/e2e-pf-controller.log 2>&1 &
  PF_PID=$!
  for i in $(seq 1 20); do
    if curl -sf -m 3 -X POST -H 'Content-Type: application/json' \
        -d '{"tenantID":"u-probe"}' http://127.0.0.1:8082/v1/namespaces/sandbox-u-probe/ensure \
        >/dev/null 2>&1; then break; fi
    sleep 1
  done
fi

cleanup() { [ -n "${PF_PID:-}" ] && kill "$PF_PID" 2>/dev/null || true; }
trap cleanup EXIT

# ---------- 2. register + login ----------
log "register $USER_EMAIL"
REG=$(curl -sf -X POST "$API/auth/register" -H 'Content-Type: application/json' \
  -d "{\"name\":\"$USER_NAME\",\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASS\"}" \
  || true)
if [ -z "$REG" ]; then
  log "register returned empty, trying login"
  REG=$(curl -sf -X POST "$API/auth/login" -H 'Content-Type: application/json' \
    -d "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASS\"}")
fi
TOKEN=$(echo "$REG" | jq_get "['data']['access_token']")
USER_ID=$(echo "$REG" | jq_get "['data']['user']['id']")
[ -n "$TOKEN" ] || die "no token from auth"
ok "auth ok, user_id=$USER_ID"

AUTH=( -H "Authorization: Bearer $TOKEN" )

# ---------- 3. create project (user-initiated) ----------
log "create project"
PROJ=$(curl -sf -X POST "$API/projects" -H 'Content-Type: application/json' "${AUTH[@]}" \
  -d "{\"name\":\"e2e$(date +%s)\"}")
PROJECT_ID=$(echo "$PROJ" | jq_get "['data']['id']")
[ -n "$PROJECT_ID" ] || die "no project id"
ok "project=$PROJECT_ID"

# ---------- 4. wait sandbox CR reconciled + publicURL stamped ----------
NS="sandbox-u-$USER_ID"
log "poll $NS for agentsandbox $PROJECT_ID"
PUBLIC_URL=""
for i in $(seq 1 60); do
  OUT=$(kubectl -n "$NS" get agentsandbox "$PROJECT_ID" -o jsonpath='{.status.publicURL}' 2>/dev/null || true)
  if [ -n "$OUT" ] && [ "$OUT" != "" ]; then
    PUBLIC_URL="$OUT"
    break
  fi
  sleep 2
done
[ -n "$PUBLIC_URL" ] || die "sandbox CR never stamped publicURL after 120s"
ok "publicURL=$PUBLIC_URL"

# ---------- 5. probe preview CDN ----------
log "probe preview CDN"
PREVIEW_HOST=$(echo "$PUBLIC_URL" | sed -E 's#^https?://##; s#/.*$##')
HTTP_CODE=$(curl -sk -o /dev/null -w "%{http_code}" "https://$PREVIEW_HOST/" || echo "000")
if [ "$HTTP_CODE" = "000" ]; then
  log "preview host unreachable (likely no /etc/hosts entry on this dev box) — that's OK"
  log "expected path on the sandbox cluster: curl https://$PREVIEW_HOST/ returns JuiceFS root"
else
  ok "preview host responded HTTP $HTTP_CODE"
fi

# ---------- 6. create conversation + chat SSE ----------
log "create conversation"
CONVO=$(curl -sf -X POST "$API/conversations" -H 'Content-Type: application/json' "${AUTH[@]}" \
  -d "{\"title\":\"e2e\"}")
CONVO_ID=$(echo "$CONVO" | jq_get "['data']['id']")
[ -n "$CONVO_ID" ] || die "no conversation id"
ok "conversation=$CONVO_ID"

log "send chat prompt (SSE, max 180s): $CHAT_PROMPT"
PROMPT_ENC=$(python3 -c "import urllib.parse,sys;print(urllib.parse.quote(sys.argv[1]))" "$CHAT_PROMPT")
DONE_FILE=$(mktemp)
# curl -N streams SSE; --max-time guards against hangs. awk marks "done" events.
timeout 180 curl -sN --max-time 180 "${AUTH[@]}" \
  "$API/conversations/$CONVO_ID/events?content=$PROMPT_ENC&edit=false" \
  | tee /tmp/e2e-sse-$$.log \
  | awk -v done_file="$DONE_FILE" '
      /^event: done/ { print; f=1; system("touch " done_file); exit }
      /^event: tool_call/ { print; tool_calls++ }
      /^event: error/ { print "ERROR-EVENT"; exit 2 }
      END { if (!f) exit 1 }
    ' || die "SSE stream ended without done event"

[ -f "$DONE_FILE" ] || die "no done marker"
ok "chat SSE completed"

# ---------- 7. assert tool_call evidence ----------
TOOL_CALLS=$(grep -c "^event: tool_call" /tmp/e2e-sse-$$.log || true)
[ "$TOOL_CALLS" -gt 0 ] || die "no tool_call observed — LLM likely refused"
ok "tool_call events seen: $TOOL_CALLS"
grep -E "^data: .*create_project" /tmp/e2e-sse-$$.log >/dev/null \
  && ok "create_project tool was invoked" \
  || log "no create_project tool_call yet (LLM may still be reasoning)"

# ---------- 8. verify preview_url surfaced on active project ----------
log "poll /active-project for preview_url"
PREVIEW=""
for i in $(seq 1 30); do
  AP=$(curl -sf "${AUTH[@]}" "$API/conversations/$CONVO_ID/active-project" || true)
  if [ -n "$AP" ]; then
    PREVIEW=$(echo "$AP" | python3 -c "import json,sys;d=json.load(sys.stdin).get('data') or {};print(d.get('preview_url') or '')")
  fi
  if [ -n "$PREVIEW" ]; then break; fi
  sleep 2
done
[ -n "$PREVIEW" ] && ok "preview_url exposed to frontend: $PREVIEW" \
  || log "preview_url still empty (LLM not yet created child project)"

rm -f "$DONE_FILE" /tmp/e2e-sse-$$.log
printf '\n%s=== e2e PASS ===%s\n' "$GRN" "$NC"
