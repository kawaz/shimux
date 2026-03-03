# 環境検出モジュール設計書

## 概要

`internal/ghostty/detect.go` の環境検出モジュール。
gmux の実行環境を判定し、適切な動作モード選択・デバッグ情報提供を行う。

## 検出項目一覧

| フィールド | 型 | 検出元 | 説明 |
|---|---|---|---|
| InsideGhostty | bool | `TERM_PROGRAM`, `GHOSTTY_RESOURCES_DIR` | Ghosttyターミナル内か |
| GhosttyVersion | string | `TERM_PROGRAM_VERSION` | Ghosttyバージョン（Ghostty外では空） |
| NestedTerminal | bool | `TMUX` | tmux等にネストされているか |
| OS | string | `runtime.GOOS` | OS名 |
| Arch | string | `runtime.GOARCH` | CPUアーキテクチャ |
| GmuxSession | string | `GMUX_SESSION` | gmuxセッション名 |
| GmuxActive | bool | `GMUX` | gmuxラッパー経由で起動されたか（`GMUX=1`） |
| ResourcesDir | string | `GHOSTTY_RESOURCES_DIR` | Ghosttyリソースディレクトリ |

## 環境変数の意味と信頼性

### 高信頼性

| 変数 | 設定元 | 備考 |
|---|---|---|
| `TERM_PROGRAM` | ターミナルエミュレータ | Ghosttyは `ghostty` を設定。ネスト時に上書きされる可能性あり |
| `GHOSTTY_RESOURCES_DIR` | Ghostty | Ghostty固有。他ターミナルでは設定されない |
| `TMUX` | tmux | `ソケットパス,PID,ウィンドウ番号` 形式。tmux内でのみ存在 |
| `runtime.GOOS` / `runtime.GOARCH` | Goランタイム | コンパイル時確定。偽装不可 |

### gmux管理変数

| 変数 | 設定元 | 備考 |
|---|---|---|
| `GMUX` | gmux wrapper | `1` でgmux経由の起動を示す |
| `GMUX_SESSION` | gmux wrapper | セッション識別子。ペイン管理に使用 |

### 補助的

| 変数 | 設定元 | 備考 |
|---|---|---|
| `TERM_PROGRAM_VERSION` | ターミナルエミュレータ | バージョン文字列。Ghostty環境外では意味をなさない |

## Ghostty検出ロジック

```
IsInsideGhostty = (TERM_PROGRAM == "ghostty") OR (GHOSTTY_RESOURCES_DIR != "")
```

2条件のOR判定とする理由:
- `TERM_PROGRAM` はネストされたシェルで上書きされることがある
- `GHOSTTY_RESOURCES_DIR` はGhostty固有のフォールバック

## API

### 関数

```go
func IsInsideGhostty() bool      // Ghostty環境判定
func GhosttyVersion() string     // バージョン取得（非Ghosttyでは空文字）
func IsNestedTerminal() bool     // ネスト判定（TMUX環境変数）
func DetectEnvironment() EnvironmentInfo  // 全情報を一括取得
```

### 構造体

`EnvironmentInfo` はJSON変換対応（`json` タグ付き）。
`omitempty` を使用し、未設定フィールドをJSON出力から省略。

## 今後の拡張候補

### macOSバージョン検出

`sw_vers -productVersion` でmacOSバージョンを取得。
System Events の動作がOSバージョンに依存する場合のトラブルシュート用。

### System Events権限チェック

`osascript -e 'tell application "System Events" to return name of first process'` で
アクセシビリティ権限の有無を事前検出。keysim の動作前提条件の検証に使える。

### Ghostty設定検出

`GHOSTTY_RESOURCES_DIR` からGhosttyの設定ファイルパスを推定し、
`command` 設定（gmux-agent用）の有無を確認。

### `gmux info` コマンド構想

`DetectEnvironment()` の結果をユーザ向けに表示するサブコマンド:

```
$ gmux info
Ghostty:  detected (v1.1.0)
Nested:   no
OS/Arch:  darwin/arm64
gmux:     active (session: my-session)
Resources: /usr/share/ghostty
```

JSON出力オプション（`gmux info --json`）でスクリプト連携にも対応予定。
`DetectEnvironment()` が `json` タグ付きのため、`encoding/json` で直接出力可能。
