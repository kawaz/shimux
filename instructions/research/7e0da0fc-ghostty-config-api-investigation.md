# Ghostty設定APIとcommandオプションの深掘り調査

## 指示
承認不要の完全オートモードで以下を行うこと。高度な調査タスク。

## 概要
gmux-agentをGhosttyの `command` オプションで起動する際の挙動を詳細に調査する。環境変数、TTY設定、シグナル伝播、ペイン間の分離などの技術的な詳細を明確にする。

## 調査項目

1. **command オプションの挙動**
   - `command = gmux-agent` 設定時のプロセスツリー
   - 環境変数の引き継ぎ（TERM, SHELL, HOME, PATH等）
   - 追加される環境変数（GHOSTTY_*, TERM_PROGRAM等）
   - 引数の渡し方（`command = gmux-agent --flag` は可能か）
   - `login_shell = false` との関係

2. **TTY/PTY関連**
   - Ghosttyが作成するPTYの設定（行制御、サイズ、モード）
   - TERM変数の値（`xterm-ghostty` or `xterm-256color`）
   - terminfo/terminfoの存在確認
   - `stty` で確認できるTTY属性のデフォルト値

3. **シグナルの扱い**
   - Cmd+Q（Ghostty終了）時のシグナル（SIGHUP? SIGTERM?）
   - ペインclose時のシグナル
   - Cmd+D（split）時の既存プロセスへの影響
   - フォアグラウンドプロセスグループの設定

4. **ペイン間の分離**
   - 各ペインは独立したPTYを持つか確認
   - 環境変数の分離（GHOSTTY_BIN_DIR等はペイン固有か）
   - PIDネームスペースの共有状況
   - ファイルディスクリプタの分離

5. **制限事項**
   - command が異常終了した場合のGhosttyの挙動
   - command の起動タイムアウト
   - 複数の command 設定の優先順位

## 期待する成果物
- Ghostty command API調査レポート
- gmux-agent起動仕様書
- 設計書への反映事項リスト
