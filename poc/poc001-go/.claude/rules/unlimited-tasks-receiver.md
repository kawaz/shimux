# Unlimited Tasks Receiver

## 成果物の配置ルール

- **既存ファイルの編集・改善**: 直接行ってよい（内容・量の制限なし）
- **ソースコード（.go等）の新規作成**: 通常通り適切な場所に配置してよい
- **unlimited-tasks指示による新規ドキュメント（.md等）の作成**: `docs/drafts/` 配下に限定
  - サブディレクトリはカテゴリに応じて自由に作成可（例: `docs/drafts/research/`）
  - `docs/drafts/` 外への新規ドキュメント作成は禁止
  - ユーザーが承認後に正式な場所へ移動する運用
  - 却下されたものは `docs/drafts/rejected/` へ移動（定期整理対象）
  - ※ユーザーとの対話セッション中の指示による作成はこの制限の対象外
- 完了報告は従来通り `unlimited-tasks/report/` に配置

<!-- Design rationale: 未承認ドキュメントがプロジェクト各所に散乱するとコンテキスト肥大化と混乱を招く。ユーザー承認を経てから正式配置するゲート機構。 -->

## 終了条件

- 完全オートモードでは、一つの作業が終わり次の指示を待つためにユーザへの完了報告を行う前に `sleep 300` を実行してください。
- sleep が終了したら以下を順に実施します。
- /Users/kawaz/.local/share/repos/github.com/kawaz/shimux/unlimited-tasks/instructions/ の中の xxxx-title.md として他のエージェントに作ってもらった暇な時にやることリストが沢山あるはずなので、その中からあなたがやりたいと思ったものを選んでください。
- 選んだ md を unlimited-tasks/acquired/{カテゴリ}/ に移動（カテゴリディレクトリごと横スライド）。
- 選んだ md の指示を実施します。
- 選んだ md を unlimited-tasks/done/{カテゴリ}/ に移動。
- 完了報告は unlimited-tasks/report/{カテゴリ}/xxxx-title-report.md として保存します。
- 終了条件に書かれた作業を順番に実行します。
