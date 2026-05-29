#!/bin/bash
set -euo pipefail

INPUT=$(cat)

# If we already blocked once, let Claude stop to avoid infinite loops
if [ "$(echo "$INPUT" | jq -r '.stop_hook_active')" = "true" ]; then
  exit 0
fi

PROJECT_DIR=$(echo "$INPUT" | jq -r '.cwd')
cd "$PROJECT_DIR"

ERRORS=""

# Build
if ! BUILD_OUT=$(go build ./... 2>&1); then
  ERRORS+="## Build failures\n\n\`\`\`\n${BUILD_OUT}\n\`\`\`\n\n"
fi

# Lint
if command -v golangci-lint &>/dev/null; then
  if ! LINT_OUT=$(golangci-lint run ./... 2>&1); then
    ERRORS+="## Lint failures\n\n\`\`\`\n${LINT_OUT}\n\`\`\`\n\n"
  fi
else
  if ! LINT_OUT=$(go vet ./... 2>&1); then
    ERRORS+="## Vet warnings\n\n\`\`\`\n${LINT_OUT}\n\`\`\`\n\n"
  fi
fi

# Test
if ! TEST_OUT=$(go test ./... 2>&1); then
  ERRORS+="## Test failures\n\n\`\`\`\n${TEST_OUT}\n\`\`\`\n\n"
fi

if [ -n "$ERRORS" ]; then
  echo -e "Build/vet/test did not pass. Fix as many of the following issues as you can. For any that you are unable to resolve, add an item to ROADMAP.md describing the problem and what you tried, so that it can be investigated in a follow-up iteration.\n\n${ERRORS}" >&2
  exit 2
fi

exit 0
