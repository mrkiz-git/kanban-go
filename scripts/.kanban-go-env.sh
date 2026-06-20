#!/bin/zsh
# Public session environment for kanban-go project.
# Loaded automatically by ~/.zshrc. Safe to commit — no secrets here.
# Actual secrets live in scripts/.kanban-go-secrets.sh (gitignored).

SECRETS_FILE="$(dirname "$0")/.kanban-go-secrets.sh"
if [[ -f "$SECRETS_FILE" ]]; then
  source "$SECRETS_FILE"
else
  echo "Warning: $SECRETS_FILE not found. Run: cp scripts/.kanban-go-env.sh scripts/.kanban-go-secrets.sh and fill in values." >&2
fi
