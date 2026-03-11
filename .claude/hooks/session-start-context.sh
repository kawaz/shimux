#!/bin/bash
# SessionStart hook: 前回セッションの末尾を自動ロード
set -euo pipefail

CLI=~/.claude/plugins/cache/claude-session-analysis/claude-session-analysis/0.10.0/skills/claude-session-analysis/bin/claude-session-analysis

input=$(cat)
current_session_id=$(echo "$input" | jq -r '.session_id')
source_type=$(echo "$input" | jq -r '.source')

# 現在のセッションファイルのパスからプロジェクトディレクトリを特定
current_path=$("$CLI" resolve-session --path "$current_session_id" 2>/dev/null) || exit 0
project_dir=$(dirname "$current_path")

# 前回のセッションを探す（更新日時順、現在のセッション除外）
prev_session_id=$(ls -t "$project_dir"/*.jsonl 2>/dev/null | while read -r f; do
  sid=$(basename "$f" .jsonl)
  if [[ "$sid" != "$current_session_id" ]]; then
    echo "$sid"
    break
  fi
done)

if [[ -z "$prev_session_id" ]]; then
  exit 0
fi

echo "$prev_session_id"
say hell

echo "## 前回セッションの末尾 (source: ${source_type})"
echo ""
echo "session_id: \`${prev_session_id}\`"
echo ""
"$CLI" timeline "$prev_session_id" --md --last-turn 10 -t RUT 2>/dev/null || true
