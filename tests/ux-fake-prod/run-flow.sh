#!/usr/bin/env bash
set -u

ROOT="/repo"
BASE="/tmp/usync-fake-prod"
HOME_DIR="$BASE/home"
WORKSPACE_DIR="$BASE/workspace"
BIN_DIR="$BASE/bin"
ARTIFACTS="${UX_ARTIFACTS:-$ROOT/artifacts/ux-fake-prod}"
FLOW_DIR="$ARTIFACTS/flows"
PLAN_DIR="$ARTIFACTS/plans"
CASE_FILTER="${1:-all}"

mkdir -p "$FLOW_DIR" "$PLAN_DIR" "$ARTIFACTS/home-after"

setup_fake_prod() {
  rm -rf "$BASE"
  mkdir -p "$HOME_DIR" "$WORKSPACE_DIR" "$BIN_DIR"
  mkdir -p "$HOME_DIR/.gemini/antigravity-cli" "$HOME_DIR/.gemini/config" "$HOME_DIR/.gemini/antigravity"
  mkdir -p "$HOME_DIR/.claude" "$HOME_DIR/.cursor" "$HOME_DIR/.config/Code/User" "$WORKSPACE_DIR/.vscode"

  for cmd in node npx docker claude codex agy; do
    cat > "$BIN_DIR/$cmd" <<'SCRIPT'
#!/usr/bin/env bash
case "$1" in
  --version|-v|version) echo "fake 1.0.0" ;;
  *) echo "fake runtime" ;;
esac
exit 0
SCRIPT
    chmod +x "$BIN_DIR/$cmd"
  done

  cat > "$HOME_DIR/.gemini/antigravity-cli/mcp_config.json" <<'JSON'
{"mcpServers":{"context7":{"url":"https://mcp.context7.com/mcp"}}}
JSON
  cat > "$HOME_DIR/.gemini/config/mcp_config.json" <<'JSON'
{"mcpServers":{"exa":{"serverUrl":"https://mcp.exa.ai/mcp?exaApiKey=REDACT_ME"}}}
JSON
  cat > "$HOME_DIR/.gemini/antigravity/mcp_config.json" <<'JSON'
{"mcpServers":{}}
JSON
  cat > "$HOME_DIR/.claude.json" <<'JSON'
{"mcpServers":{"playwright":{"command":"npx","args":["@playwright/mcp@latest"]}}}
JSON
  cat > "$HOME_DIR/.cursor/mcp.json" <<'JSON'
{"mcpServers":{"playwright":{"command":"npx","args":["@playwright/mcp@latest"]}}}
JSON
  cat > "$HOME_DIR/.config/Code/User/mcp.json" <<'JSON'
{"servers":{"playwright":{"command":"npx","args":["@playwright/mcp@latest"]}}}
JSON
  cat > "$WORKSPACE_DIR/.vscode/mcp.json" <<'JSON'
{"servers":{"context7":{"url":"https://mcp.context7.com/mcp"}}}
JSON
}

strip_ansi() {
  perl -pe 's/\e\[[0-9;?]*[ -\/]*[@-~]//g; s/\e\][^\a]*(\a|\e\\)//g; s/\r//g' "$1" > "$2"
  if [[ ! -s "$2" ]]; then
    cp "$1" "$2"
  fi
}

run_doctor_json() {
  PATH="$BIN_DIR:$PATH" usync doctor \
    --home-dir "$HOME_DIR" \
    --workspace "$WORKSPACE_DIR" \
    --json \
    > "$ARTIFACTS/doctor.json" \
    2> "$ARTIFACTS/doctor.stderr"
}

run_plan_json() {
  local home_plan="$HOME_DIR/.usync/plans/fake-prod-plan.json"
  mkdir -p "$(dirname "$home_plan")"
  PATH="$BIN_DIR:$PATH" usync plan \
    --provider exa \
    --keys "11111111-1111-1111-1111-111111111111" \
    --home-dir "$HOME_DIR" \
    --workspace "$WORKSPACE_DIR" \
    --all-detected \
    --include-workspace \
    --out "$home_plan" \
    > "$ARTIFACTS/plan.stdout" \
    2> "$ARTIFACTS/plan.stderr"
  cp "$home_plan" "$PLAN_DIR/fake-prod-plan.json"
}

run_tui_dm_p31() {
  local ansi="$FLOW_DIR/DM-P31.ansi"
  local text="$FLOW_DIR/DM-P31.txt"

  if ! command -v script >/dev/null 2>&1; then
    echo "script command not available; cannot capture TUI PTY transcript" > "$text"
    return 2
  fi

  timeout 8s bash -c "
    {
      sleep 0.4
      printf 'p'
      sleep 0.4
      printf '\r'
      sleep 0.4
      printf '\r'
      sleep 0.4
      printf 'q'
    } | PATH=\"$BIN_DIR:\$PATH\" script -q -e -c \"usync --home-dir '$HOME_DIR'\" \"$ansi\" >/dev/null 2>&1
  "
  strip_ansi "$ansi" "$text"
}

run_teatest_matrix() {
  cd "$ROOT" || return 1
  USYNC_UX_MATRIX=1 go test ./pkg/tui -run TestDashboardFlowMatrix -v \
    > "$ARTIFACTS/teatest-matrix.txt" \
    2>&1
}

run_fake_prod_matrix() {
  cd "$ROOT" || return 1
  PATH="$BIN_DIR:$PATH" \
    USYNC_UX_FAKE_PROD=1 \
    USYNC_UX_HOME="$HOME_DIR" \
    USYNC_UX_WORKSPACE="$WORKSPACE_DIR" \
    go test ./pkg/tui -run TestDashboardFakeProdMatrix -v \
    > "$ARTIFACTS/fake-prod-matrix.txt" \
    2>&1
}

json_escape() {
  sed 's/\\/\\\\/g; s/"/\\"/g' "$1" | awk '{printf "%s\\n", $0}'
}

write_matrix_json() {
  local teatest_status="$1"
  local dm_p31_status="$2"
  local doctor_status="$3"
  local plan_status="$4"
  local fake_prod_status="$5"

  cat > "$ARTIFACTS/matrix.json" <<JSON
{
  "runner": "ux-fake-prod",
  "cases": [
    {
      "id": "DM-P31",
      "kind": "real-tui-pty",
      "status": "$dm_p31_status",
      "artifacts": {
        "ansi": "flows/DM-P31.ansi",
        "text": "flows/DM-P31.txt"
      }
    },
    {
      "id": "FAKE-PROD-MATRIX",
      "kind": "docker-fake-prod-teatest",
      "status": "$fake_prod_status",
      "artifacts": {
        "text": "fake-prod-matrix.txt"
      }
    },
    {
      "id": "UX-MATRIX",
      "kind": "teatest",
      "status": "$teatest_status",
      "artifacts": {
        "text": "teatest-matrix.txt"
      }
    },
    {
      "id": "DOCTOR-JSON",
      "kind": "real-cli",
      "status": "$doctor_status",
      "artifacts": {
        "json": "doctor.json",
        "stderr": "doctor.stderr"
      }
    },
    {
      "id": "PLAN-ALL-DETECTED",
      "kind": "real-cli",
      "status": "$plan_status",
      "artifacts": {
        "plan": "plans/fake-prod-plan.json",
        "stdout": "plan.stdout",
        "stderr": "plan.stderr"
      }
    }
  ]
}
JSON
}

write_issues_json() {
  local issues="$ARTIFACTS/issues.json"
  local first=1
  printf '{\n  "issues": [\n' > "$issues"

  if grep -q "DM-P31/docker" "$ARTIFACTS/fake-prod-matrix.txt" 2>/dev/null; then
    if [[ "$first" != "1" ]]; then
      printf ',\n' >> "$issues"
    fi
    first=0
    cat >> "$issues" <<JSON
    {
      "id": "DM-P31",
      "severity": "critical",
      "source": "docker-fake-prod",
      "title": "Missing credentials reach a dead-end plan error",
      "invariants": ["I-03", "I-04", "I-13"],
      "artifact": "fake-prod-matrix.txt"
    }
JSON
  fi

  if grep -q "DM-P12:" "$ARTIFACTS/teatest-matrix.txt" 2>/dev/null; then
    if [[ "$first" != "1" ]]; then
      printf ',\n' >> "$issues"
    fi
    first=0
    cat >> "$issues" <<JSON
    {
      "id": "DM-P12",
      "severity": "critical",
      "source": "docker-teatest",
      "title": "Deselected target is still planned",
      "invariants": ["I-06", "I-07"],
      "artifact": "teatest-matrix.txt"
    }
JSON
  fi

  if grep -q "DM-P10:" "$ARTIFACTS/teatest-matrix.txt" 2>/dev/null; then
    if [[ "$first" != "1" ]]; then
      printf ',\n' >> "$issues"
    fi
    cat >> "$issues" <<JSON
    {
      "id": "DM-P10",
      "severity": "high",
      "source": "docker-teatest",
      "title": "Hidden r advances from provider readiness without conflicts",
      "invariants": ["I-02"],
      "artifact": "teatest-matrix.txt"
    }
JSON
  fi

  printf '\n  ]\n}\n' >> "$issues"
}
main() {
  setup_fake_prod

  local doctor_status="skipped"
  local plan_status="skipped"
  local dm_p31_status="skipped"
  local teatest_status="skipped"
  local fake_prod_status="skipped"

  if [[ "$CASE_FILTER" == "all" || "$CASE_FILTER" == "doctor" ]]; then
    run_doctor_json
    status=$?
    if [[ "$status" == "0" ]]; then
      doctor_status="pass"
    elif [[ "$status" == "2" ]]; then
      doctor_status="findings"
    else
      doctor_status="fail"
    fi
  fi

  if [[ "$CASE_FILTER" == "all" || "$CASE_FILTER" == "plan" ]]; then
    run_plan_json
    status=$?
    if [[ "$status" == "0" ]]; then
      plan_status="pass"
    else
      plan_status="fail"
    fi
  fi

  if [[ "$CASE_FILTER" == "all" || "$CASE_FILTER" == "DM-P31" ]]; then
    run_tui_dm_p31
    status=$?
    if [[ "$status" == "0" ]]; then
      dm_p31_status="captured"
    else
      dm_p31_status="fail"
    fi
  fi

  if [[ "$CASE_FILTER" == "all" || "$CASE_FILTER" == "teatest" ]]; then
    run_teatest_matrix
    status=$?
    if [[ "$status" == "0" ]]; then
      teatest_status="pass"
    else
      teatest_status="fail"
    fi
  fi

  if [[ "$CASE_FILTER" == "all" || "$CASE_FILTER" == "fake-prod-matrix" ]]; then
    run_fake_prod_matrix
    status=$?
    if [[ "$status" == "0" ]]; then
      fake_prod_status="pass"
    else
      fake_prod_status="fail"
    fi
  fi

  cp -R "$HOME_DIR/." "$ARTIFACTS/home-after/" 2>/dev/null || true
  write_matrix_json "$teatest_status" "$dm_p31_status" "$doctor_status" "$plan_status" "$fake_prod_status"
  write_issues_json

  echo "UX fake-prod artifacts:"
  echo "  $ARTIFACTS/matrix.json"
  echo "  $ARTIFACTS/issues.json"
  echo "  $FLOW_DIR"
  echo "  $PLAN_DIR"
  echo "  $ARTIFACTS/fake-prod-matrix.txt"
  echo "  $ARTIFACTS/teatest-matrix.txt"
}

main "$@"
