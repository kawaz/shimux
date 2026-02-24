# ウィンドウリサイズ伝播の設計

## 指示
承認不要の完全オートモードで以下を行うこと。

## 概要
Ghosttyのウィンドウやペインがリサイズされた際に、gmux-agent内部のPTYサイズを正しく伝播する仕組みを設計する。tmuxではペインリサイズは自動的に内部プロセスへ伝播されるが、PTYプロキシアーキテクチャではこの部分を自前で実装する必要がある。

## 設計項目

1. **リサイズ検出**
   - Ghostty側のPTY（外部PTY）にSIGWINCHが送られる仕組みの確認
   - gmux-agentでのSIGWINCHシグナルハンドリング
   - `TIOCGWINSZ` ioctl での現在のウィンドウサイズ取得

2. **内部PTYへの伝播**
   - 外部PTYのサイズ変更を検知
   - `TIOCSWINSZ` ioctl で内部PTY（マスター側）のサイズを設定
   - 内部プロセス（zsh等）へのSIGWINCH転送
   - 即時反映 vs debounce（高速リサイズ時の負荷対策）

3. **tmuxとの互換性**
   - tmuxの`resize-pane`コマンドとの関係
   - Claude Codeがresize-paneを使用するか調査
   - `#{pane_width}`, `#{pane_height}` フォーマット変数への影響

4. **エッジケース**
   - ゼロサイズへのリサイズ（最小化時）
   - 超高速連続リサイズ
   - リサイズ中のsend-keys

## テスト設計
- `TIOCSWINSZ` 設定後の `TIOCGWINSZ` 一致確認テスト
- SIGWINCHハンドラの呼び出し確認テスト
- リサイズ＋出力の並行テスト

## 期待する成果物
- リサイズ伝播の設計書
- Go実装のシグネチャとテスト
- 設計書への追記案
