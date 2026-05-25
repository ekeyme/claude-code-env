#!/bin/bash

CLAUDE_DIR="${CCENV_CLAUDE_HOME:-$HOME/.claude}"

if [[ -f "$CLAUDE_DIR/ccenv.activate" ]]; then
    . "$CLAUDE_DIR/ccenv.deactivate"
    . "$CLAUDE_DIR/ccenv.activate"
fi

exec claude "$@"
