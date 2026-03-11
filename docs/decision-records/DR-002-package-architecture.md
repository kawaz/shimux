# DR-002: パッケージ構成

## 問題/背景

shimux は「tmux CLI の受け口」「ターミナルアプリの操作」「両者を繋ぐ抽象層」という 3 つの関心事を持つ。
これらを適切に分離しつつ、MoonBit native backend のプロジェクト構成として自然な形にする必要がある。

## 決定内容

以下の 3 層 + エントリーポイントのフラットな構成を採用する。
`internal/` のようなネストは掘らず、フラットに配置する。

```
src/
  core/             # 抽象 API レイヤ
  shim/
    tmux/           # tmux CLI パース -> core API 呼び出し
  terminal/
    ghostty/        # Ghostty AppleScript バックエンド（core のインターフェース実装）
cmd/
  shimux/           # エントリーポイント（wrapper モード + shim ディスパッチ）
ffi/                # Rust FFI クレート（osascript 実行等）
```

### 依存関係

```
cmd/shimux -> src/core, src/shim/tmux, src/terminal/ghostty
src/shim/tmux -> src/core
src/terminal/ghostty -> src/core
src/core -> (外部依存なし)
```

**shim と terminal は core のみに依存し、互いを知らない。**

### 各パッケージの責務

| パッケージ | 責務 |
|---|---|
| `src/core` | Split, SendKeys, Focus, ListPanes 等のインターフェース定義 + ペイン状態管理 |
| `src/shim/tmux` | tmux CLI 引数のパース、core API への変換 |
| `src/terminal/ghostty` | core インターフェースの Ghostty AppleScript 実装 |
| `cmd/shimux` | エントリーポイント。wrapper モードと shim ディスパッチ |
| `ffi/` | Rust クレート。osascript 実行、ファイル I/O 等の最小限の FFI |

## 選択理由/トレードオフ

- **shim/terminal の分離**: 将来的に tmux 以外の shim（screen 等）や Ghostty 以外の terminal backend（iTerm2, WezTerm 等）を追加可能
- **core の独立性**: テスト容易性を確保。モック実装で shim のテストが可能
- **フラット構成**: `internal/` 等のネストを避け、パッケージパスを短く保つ
- **ffi/ の分離**: Rust コードを MoonBit プロジェクト外に置くことで、ビルドシステムの独立性を維持

## 影響

- `moon build cmd/shimux --target native` でメインパッケージのみをビルドする（パッケージ未指定の `moon build --target native` は lib パッケージの `cc-link-flags` で `_main` 未定義エラーになる）
- Rust FFI は `ffi/` ディレクトリで `cargo build --release` し、生成物を MoonBit のリンクフラグで参照
