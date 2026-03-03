# 設計決定の経緯

## poc3 設計書

poc/poc2/Go PoC の知見を統合した最終設計: `poc3/DESIGN.md`

## poc vs poc2 比較

### 背景

- **poc**: 単一プロセス PTY プロキシ。動作確認済み。FFI 11 関数
- **poc2**: server/client 分離の全体設計文書。FFI 約 50 関数。コード 0 行

### 評価

| 観点 | poc | poc2 |
|---|---|---|
| 動作実績 | tmux 透過操作確認済み | コード 0 行 |
| FFI 粒度 | 統合的（shimux_proxy_io に全部入り） | 細粒度（約 50 関数） |
| 拡張性 | イベントループが Rust 内で閉じている | MoonBit 側で制御可能 |
| 設計の網羅性 | shimux-agent 相当のみ | 全体アーキテクチャ |

poc2 の問題点:
- 50 関数は「フル tmux クローン」に近く、shimux の目的に対して過剰
- fork 前提の記述が残っている（MoonBit GC 非互換）

### 設計決定: 段階的融合アプローチ

poc の実装をベースに、poc2 の設計思想を段階的に取り込む。

```
Phase 1: shimux-agent 基盤（poll ループ MoonBit 化 + Unix socket）
    │
    ├── Path A: 1 → 3 → 2
    │   Phase 3: observer null実装（完全透過プロキシ）
    │   Phase 2: tmux 互換 CLI インターフェース
    │
    └── Path B: 1 → 4
        Phase 4: server/client 分離
```

Phase 1 完了後、Path A と Path B は並行して進められる。

#### Phase 1: shimux-agent 基盤

`shimux_proxy_io` を解体し、poll ループを MoonBit 側に持ち上げる。Unix socket で外部コマンドを受け付ける口を追加。

追加 FFI: socket 関連 5〜6 関数 + poll ラッパー

#### Phase 3: observer null 実装（完全透過プロキシ）

observer のインターフェースだけ定義し、実装は完全透過（何も監視しない）。capture-pane は空文字列を返す。実際の ANSI パース・画面バッファは後付けで昇格させる。

Design rationale: observer の実装は重い（ANSI パーサ + 仮想画面バッファ）ため、null 実装で先に進め、server/client 分離やCLI の構造を優先する。

#### Phase 4: server/client 分離

shimux-server（daemon）と shimux-client に分離。Ghostty 再起動後のセッション永続化。

- daemonize + PID ファイル管理
- Unix socket による client → server 接続
- 複数クライアント broadcast

#### Phase 2: tmux 互換 CLI インターフェース

tmux 互換コマンドを受け付ける CLI 層。Phase 4 の server/client 構造の上に構築。

- `shimux new-session` / `shimux attach-session`
- `shimux send-keys` / `shimux capture-pane` / `shimux split-window`
- Claude Code Agent Teams との結合テスト
