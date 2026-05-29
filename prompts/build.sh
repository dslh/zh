BASEDIR=$(dirname "$0")

PROMPT="$BASEDIR/build.md"

while grep '^- \[ \] ' -q "$BASEDIR/../ROADMAP.md"
do
  cat $PROMPT |\
    claude --print --verbose \
           --output-format=stream-json \
           --settings "$BASEDIR/test-hook.settings.json" \
           --allowed-tools=Bash,Edit,Write,Read,Search,mcp__github__execute_graphql_query,mcp__zenhub__execute_graphql_query |\
    claude-stream-formatter --stats="$BASEDIR/build.stats.json"
done
