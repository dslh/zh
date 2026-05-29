BASEDIR=$(dirname "$0")

PROMPT="$BASEDIR/$1.md"

while grep '^- \[ \] ' -q "$BASEDIR/../checklist.md"
do
  cat $PROMPT |\
    claude --print --verbose \
           --output-format=stream-json \
           --settings "$BASEDIR/test-hook.settings.json" \
           --allowed-tools=Bash,Edit,Write,Read,Search,mcp__github__execute_graphql_query,mcp__zenhub__execute_graphql_query |\
    claude-stream-formatter
done
