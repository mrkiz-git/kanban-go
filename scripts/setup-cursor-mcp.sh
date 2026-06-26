#!/bin/zsh
# Generate project-local .cursor/mcp.json from gitignored secrets.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SECRETS="$ROOT/scripts/.kanban-go-secrets.sh"
MCP_JSON="$ROOT/.cursor/mcp.json"

if [[ ! -f "$SECRETS" ]]; then
  echo "Missing $SECRETS — create it and add GITHUB_PERSONAL_ACCESS_TOKEN." >&2
  exit 1
fi

# shellcheck source=/dev/null
source "$SECRETS"

if [[ -z "${GITHUB_PERSONAL_ACCESS_TOKEN:-}" ]]; then
  echo "GITHUB_PERSONAL_ACCESS_TOKEN is not set in $SECRETS" >&2
  exit 1
fi

mkdir -p "$ROOT/.cursor"

MCP_JSON="$MCP_JSON" GITHUB_PERSONAL_ACCESS_TOKEN="$GITHUB_PERSONAL_ACCESS_TOKEN" python3 - <<'PY'
import json
import os

path = os.environ["MCP_JSON"]
token = os.environ["GITHUB_PERSONAL_ACCESS_TOKEN"]

config = {
    "mcpServers": {
        "github": {
            "url": "https://api.githubcopilot.com/mcp/",
            "headers": {
                "Authorization": f"Bearer {token}",
            },
        }
    }
}

with open(path, "w", encoding="utf-8") as f:
    json.dump(config, f, indent=2)
    f.write("\n")

print(f"Wrote {path}")
PY
