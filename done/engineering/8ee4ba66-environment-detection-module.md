# 環境検出モジュールの設計と実装

## 指示
承認不要の完全オートモードで以下を行うこと。
この指示による新規ドキュメント作成は `docs/drafts/` 配下に限定。既存ファイル編集・コード新規作成は通常通り。

## 概要
gmuxが起動時に実行環境を正確に検出するモジュールを設計・実装する。Ghosttyの有無、バージョン、macOS固有の機能（System Events権限）、既存のgmux-agentプロセスなどを検出し、利用可能な機能セットを判定する。

## 設計項目

1. **Ghostty検出**
   - `TERM_PROGRAM=ghostty` のチェック（既存実装の確認）
   - Ghosttyバージョン取得（`TERM_PROGRAM_VERSION` or `ghostty --version`）
   - Ghosttyが直接の親かネストされた環境か（tmux内のghostty等）
   - `GHOSTTY_RESOURCES_DIR` の存在確認

2. **macOS検出**
   - `runtime.GOOS == "darwin"` の確認
   - macOSバージョン取得（`sw_vers`）
   - System Events権限の確認方法（osascriptの簡易テスト）
   - Accessibility権限の有無

3. **gmux-agent検出**
   - `command = gmux-agent` の設定確認方法
   - 既存のUnixソケットの検出
   - gmux-agentプロセスの生存確認
   - gmuxセッションの自動検出

4. **機能セット判定**
   - 検出結果から利用可能なコマンドセットを判定
   - `gmux info` コマンドの設計（環境情報の表示）
   - JSON出力対応（`--json`フラグ）
   - 不足機能のユーザーフレンドリーなガイダンス

## テスト設計
- 各環境変数の組み合わせテスト（テーブル駆動）
- 現在の `IsInsideGhostty` テストの拡張
- モック環境での機能セット判定テスト

## 期待する成果物
- 環境検出モジュール設計書（`docs/drafts/environment-detection.md`）
- 既存コードへの改善提案
- テストケース追加
