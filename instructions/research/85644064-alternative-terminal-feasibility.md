# 代替ターミナル対応の実現可能性調査

## 指示
承認不要の完全オートモードで以下を行うこと。高度な調査タスク。

## 概要
gmuxは現在Ghostty専用だが、他のモダンターミナルエミュレータへの対応可能性を調査する。各ターミナルのAPI/IPC能力を比較し、抽象化レイヤーの設計案を作成する。

## 調査対象

### 1. WezTerm
- Lua設定APIの能力（ペイン操作、send-keys相当）
- `wezterm cli` コマンドの一覧と能力
- Unix domain socket IPC
- ペインID体系

### 2. kitty
- `kitty @ ` リモートコントロールプロトコル
- `launch --type=window` によるペイン作成
- `send-text` コマンド
- ペインID体系とウィンドウ管理

### 3. Alacritty
- IPC能力の有無（結論: ほぼなし？）
- 外部ツールとの連携方法

### 4. iTerm2
- AppleScript API
- Python Runtime API
- ペイン操作能力

### 5. Windows Terminal
- JSON設定ベースの操作
- コマンドラインからのペイン操作

## 評価基準
- ペイン作成/破棄のAPI有無
- send-keys相当の機能
- ペインID管理
- macOS対応
- API安定性

## 期待する成果物
- 各ターミナルの能力比較表（`docs/drafts/terminal-comparison.md`）
- 抽象化レイヤー（`TerminalBackend` interface）の設計案
- 対応優先度の推奨と理由
- 必要な工数の概算

> **配置ルール**: この指示による新規ドキュメントは `docs/drafts/` 配下に配置すること。既存ファイルの編集は直接行ってよい。
