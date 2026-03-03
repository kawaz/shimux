# gmux テスト計画書

## 1. テスト戦略

### 1.1 テストピラミッド

```
        /   E2E   \           少量 — ghostty実機での結合テスト
       / 統合テスト  \         中量 — ラッパーモード、コマンドフロー
      / ユニットテスト \       多量 — パーサ、変換、状態管理、ログ
```

| レベル | 対象 | 実行環境 | 実行頻度 |
|--------|------|----------|----------|
| ユニット | 各パッケージの関数・メソッド | CI/ローカル | コミットごと |
| 統合 | パッケージ間連携、ラッパーモード | CI/ローカル | コミットごと |
| E2E | ghostty実機操作、Claude Code互換 | ローカル（macOS） | リリース前 |

### 1.2 テストID命名規則

テストケースIDは以下のプレフィックスで分類する:

| プレフィックス | 対象領域 |
|-------------|---------|
| `TC-P` | パーサ（Parser）のユニットテスト |
| `TC-C` | コマンド変換（Command conversion）のユニットテスト |
| `TC-AG` | gmux-agent（PTYプロキシ）のユニットテスト |
| `TC-PM` | ペイン管理（Pane Manager）のユニットテスト |
| `TC-L` | ログ（Logger）のユニットテスト |
| `TC-W` | ラッパーモード（Wrapper）のユニットテスト |
| `TC-GC` | Ghostty Controllerのユニットテスト |
| `TC-I` | 統合テスト（Integration） |
| `TC-CC` | Claude Code互換テスト |
| `TC-WI` | ラッパーモード統合テスト（Wrapper Integration） |
| `TC-TX` | tmux互換性テスト |
| `TC-E` | エッジケース・異常系テスト |
| `TC-PF` | パフォーマンステスト |

各IDの末尾にはアルファベット小文字のサフィックス（a, b, c, ...）を付与してバリエーションを区別する。

### 1.3 テスト命名規約

```go
// ファイル名: <対象>_test.go
// 関数名: Test<対象>_<シナリオ>_<期待結果>
// サブテスト名: 日本語可

func TestExpandSpecialKey_Enter_ReturnsCR(t *testing.T) { ... }

func TestCmdSplitWindow(t *testing.T) {
    t.Run("水平分割フラグ -h", func(t *testing.T) { ... })
    t.Run("垂直分割フラグ -v", func(t *testing.T) { ... })
}
```

テーブル駆動テストを基本とする。

### 1.4 モック/スタブ方針

| 対象 | 方式 | 理由 |
|------|------|------|
| `ghostty.Controller` | インターフェースモック | 外部プロセス呼び出しを回避 |
| ファイルシステム | `t.TempDir()` | テスト間の独立性確保 |
| 環境変数 | `t.Setenv()` | テスト終了時に自動復元 |
| osascript / ghostty | `exec.Command` 差し替え | `execCommandContext` をDIするか、Controllerモックで代替 |
| `gmux-agent` | 内部PTY + Unixソケットのテストハーネス | PTYプロキシのsend-keysテスト用 |
| `net.Conn` | テスト用Unixソケット接続 | ソケットプロトコルテスト用 |

Controllerモックの定義:

```go
// internal/ghostty/mock_controller.go (テスト用)
type MockController struct {
    Calls       []MockCall
    SplitErr    error
    GotoErr     error
    // ...
}

type MockCall struct {
    Method string
    Args   []interface{}
}

func (m *MockController) NewSplit(d SplitDirection) error {
    m.Calls = append(m.Calls, MockCall{"NewSplit", []interface{}{d}})
    return m.SplitErr
}
// ... 他メソッドも同様
```

### 1.5 テストヘルパー

```go
// assertNoError は err が nil でなければテスト失敗
func assertNoError(t *testing.T, err error) { t.Helper(); ... }

// assertError は err が nil ならテスト失敗
func assertError(t *testing.T, err error) { t.Helper(); ... }

// assertContains は s に substr が含まれなければテスト失敗
func assertContains(t *testing.T, s, substr string) { t.Helper(); ... }

// captureStdout は関数実行中の stdout 出力をキャプチャ
func captureStdout(t *testing.T, fn func()) string { t.Helper(); ... }

// captureStderr は関数実行中の stderr 出力をキャプチャ
func captureStderr(t *testing.T, fn func()) string { t.Helper(); ... }

// captureOutput は関数実行中の stdout と stderr の両方をキャプチャ
func captureOutput(t *testing.T, fn func()) (stdout, stderr string) { t.Helper(); ... }
```

検討事項: アサーションライブラリとして `testify/assert` または `go-cmp` の導入を検討する。テーブル駆動テストのアサーション記述が簡潔になり、差分表示も改善される。導入判断はPhase 1実装開始時に行う。

---

## 2. ユニットテスト

### 2.1 tmux引数パーサ (`internal/tmux/`)

#### TC-P001: グローバルオプション `-L <socket>` のパース

| ID | 入力 | 期待結果 |
|----|------|----------|
| TC-P001a | `["-L", "claude-swarm", "split-window", "-h"]` | socket=`"claude-swarm"`, command=`"split-window"`, args=`["-h"]` |
| TC-P001b | `["-L", "my-socket", "send-keys", "-t", "%0", "ls", "Enter"]` | socket=`"my-socket"`, command=`"send-keys"`, args=`["-t", "%0", "ls", "Enter"]` |
| TC-P001c | `["split-window", "-h"]`（-Lなし） | socket=`""`, command=`"split-window"`, args=`["-h"]` |
| TC-P001d | `["-L"]`（値なし） | エラー: `-L` に値が必要 |
| TC-P001e | `["-V"]`（コマンドなし） | バージョン表示（グローバル `-V` として解釈） |
| TC-P001f | `["split-window", "-v"]` | command=`"split-window"`, args=`["-v"]`（コマンド固有の垂直分割フラグとして解釈） |
| TC-P001g | `["-V", "split-window"]` | バージョン表示（`-V` がグローバルオプションとしてコマンド名より先に出現するため、バージョン表示として処理） |
| TC-P001h | `["-S", "/tmp/my.sock", "split-window"]` | socketPath=`"/tmp/my.sock"`, command=`"split-window"` |
| TC-P001i | `["-S"]`（値なし） | エラー: `-S` に値が必要 |

#### TC-P002: ターゲット指定 `-t` のパース

| ID | 入力 | 期待結果 |
|----|------|----------|
| TC-P002a | `["-t", "%0"]` | target=`"%0"` |
| TC-P002b | `["-t", "%123"]` | target=`"%123"` |
| TC-P002c | `["-t", "session_name"]` | target=`"session_name"` |
| TC-P002d | `["-t", "gmux_worktree-feature_new"]` | target=`"gmux_worktree-feature_new"` |
| TC-P002e | `["-t"]`（値なし） | エラー: `-t` に値が必要 |
| TC-P002f | `["-t", "gmux:0.1"]` | target=`"gmux:0.1"`（session:window.pane形式） |
| TC-P002g | `["-t", "gmux:0"]` | target=`"gmux:0"`（session:window形式） |
| TC-P002h | `["-t", ":0.1"]` | target=`":0.1"`（sessionなし、window.pane形式） |

#### TC-P003: `-P -F` 出力フラグのパース

| ID | 入力 | 期待結果 |
|----|------|----------|
| TC-P003a | `["-P", "-F", "#{pane_id}"]` | printAfter=true, format=`"#{pane_id}"` |
| TC-P003b | `["-P", "-F", "#{pane_id}:#{pane_width}x#{pane_height}"]` | printAfter=true, format=（複合） |
| TC-P003c | `["-P"]`（-Fなし） | printAfter=true, format=デフォルト |
| TC-P003d | `["-F", "#{pane_id}"]`（-Pなし） | printAfter=false, format=`"#{pane_id}"` |

#### TC-P004: フォーマット変数 `#{...}` の展開

| ID | 入力 | 期待結果 |
|----|------|----------|
| TC-P004a | `"#{session_name}"` | `"gmux"` |
| TC-P004b | `"#{window_index}"` | `"0"` |
| TC-P004c | `"#{pane_id}"` | `"%0"`（アクティブペインのID） |
| TC-P004d | `"#{pane_pid}"` | 数値文字列（プロセスID） |
| TC-P004e | `"#{pane_current_path}"` | 有効なディレクトリパス |
| TC-P004f | `"#{pane_width}"` | 数値文字列（ioctl TIOCGWINSZ利用可能時は実サイズ、不可時はデフォルト`"80"`） |
| TC-P004g | `"#{pane_height}"` | 数値文字列（ioctl TIOCGWINSZ利用可能時は実サイズ、不可時はデフォルト`"24"`） |
| TC-P004h | `"#{pane_active}"` | `"1"` または `"0"` |
| TC-P004i | `"#{window_id}"` | `"@0"` |
| TC-P004j | `"#{session_id}"` | `"$0"` |
| TC-P004k | `"#{window_name}"` | `"ghostty"` |
| TC-P004l | `"#{pane_id}:#{pane_width}x#{pane_height}:#{pane_active}"` | `"%0:80x24:1"` 形式 |
| TC-P004m | `"session=#{session_name}"` | `"session=gmux"` |
| TC-P004n | `"プレーンテキスト"` | `"プレーンテキスト"`（変数なし） |
| TC-P004o | `"#{unknown_variable}"` | `"#{unknown_variable}"`（未知変数はそのまま） |
| TC-P004p | `""` | `""`（空文字列） |

#### TC-P005: 特殊キー名展開（send-keys用）

| ID | 入力 | 期待結果 | 説明 |
|----|------|----------|------|
| TC-P005a | `"Enter"` | `"\r"` | CR（PTYではEnterキーはCR=0x0Dとして送信される） |
| TC-P005b | `"Escape"` | `"\x1b"` | ESCキー |
| TC-P005c | `"Tab"` | `"\t"` | タブ |
| TC-P005d | `"Space"` | `" "` | スペース |
| TC-P005e | `"BSpace"` | `"\x7f"` | バックスペース |
| TC-P005f | `"C-c"` | `"\x03"` | Ctrl+C |
| TC-P005g | `"C-d"` | `"\x04"` | Ctrl+D |
| TC-P005h | `"C-z"` | `"\x1a"` | Ctrl+Z |
| TC-P005i | `"C-l"` | `"\x0c"` | Ctrl+L |
| TC-P005j | `"C-a"` | `"\x01"` | Ctrl+A |
| TC-P005k | `"C-e"` | `"\x05"` | Ctrl+E |
| TC-P005l | `"C-k"` | `"\x0b"` | Ctrl+K |
| TC-P005m | `"C-u"` | `"\x15"` | Ctrl+U |
| TC-P005n | `"C-w"` | `"\x17"` | Ctrl+W |
| TC-P005o | `"Up"` | `"\x1b[A"` | 上矢印 |
| TC-P005p | `"Down"` | `"\x1b[B"` | 下矢印 |
| TC-P005q | `"Right"` | `"\x1b[C"` | 右矢印 |
| TC-P005r | `"Left"` | `"\x1b[D"` | 左矢印 |
| TC-P005s | `"hello"` | `"hello"` | 通常テキスト（変換なし） |
| TC-P005t | `"BTab"` | `"\x1b[Z"` | Shift+Tab |
| TC-P005u | `"DC"` | `"\x1b[3~"` | Delete |
| TC-P005v | `"End"` | `"\x1b[F"` | End |
| TC-P005w | `"Home"` | `"\x1b[H"` | Home |
| TC-P005x | `"IC"` | `"\x1b[2~"` | Insert |
| TC-P005y | `"NPage"` | `"\x1b[6~"` | Page Down |
| TC-P005z | `"PPage"` | `"\x1b[5~"` | Page Up |
| TC-P005aa | `"F1"` | `"\x1bOP"` | F1 |
| TC-P005ab | `"F2"` | `"\x1bOQ"` | F2 |
| TC-P005ac | `"F3"` | `"\x1bOR"` | F3 |
| TC-P005ad | `"F4"` | `"\x1bOS"` | F4 |
| TC-P005ae | `"F5"` | `"\x1b[15~"` | F5 |
| TC-P005af | `"F6"` | `"\x1b[17~"` | F6 |
| TC-P005ag | `"F7"` | `"\x1b[18~"` | F7 |
| TC-P005ah | `"F8"` | `"\x1b[19~"` | F8 |
| TC-P005ai | `"F9"` | `"\x1b[20~"` | F9 |
| TC-P005aj | `"F10"` | `"\x1b[21~"` | F10 |
| TC-P005ak | `"F11"` | `"\x1b[23~"` | F11 |
| TC-P005al | `"F12"` | `"\x1b[24~"` | F12 |
| TC-P005am | `""` | `""` | 空文字列 |

#### TC-P006: エッジケース

| ID | 入力 | 期待結果 |
|----|------|----------|
| TC-P006a | `[""]`（空引数） | コマンド名が空としてエラーまたは無視 |
| TC-P006b | `["split-window", "-h", "-v"]`（相反フラグ） | 後勝ち: `-v`（垂直分割）。設計書パーサ仕様の後勝ちルール（last-wins）に基づく |
| TC-P006c | `["send-keys", "echo", "'hello world'"]`（引用符含む） | テキストとしてそのまま渡す |
| TC-P006d | `["send-keys", "echo", "\"nested\""]`（ダブルクォート） | テキストとしてそのまま渡す |
| TC-P006e | `["send-keys", "-t", "%0", "ls -la", "Enter"]`（-t付きsend-keys） | target=%0, keys=["ls -la", "Enter"] |
| TC-P006f | `["send-keys", ""]`（空テキスト） | 空文字列の送信 |
| TC-P006g | `["select-pane", "-t", "%0", "-P", "bg=#282828", "-T", "agent-1"]`（複合引数） | target=%0, スタイル・タイトルの処理 |
| TC-P006h | ペインID体系 | ペインIDが`%0`始まり（pane-base-index=0相当）であることの検証 |

#### TC-P007: `new-window` の `-n` オプションパース

| ID | 入力 | 期待結果 |
|----|------|----------|
| TC-P007a | `["new-window", "-n", "work"]` | windowName=`"work"` |
| TC-P007b | `["new-window", "-n", "teammate-agent1", "-P", "-F", "#{pane_id}"]` | windowName=`"teammate-agent1"`, printAfter=true |
| TC-P007c | `["new-window", "-n"]`（値なし） | エラー: `-n` に値が必要 |

---

### 2.2 tmuxコマンド変換 (`cmd/gmux/` + `internal/tmux/`)

#### TC-C001: `split-window` -> `NewSplit` 変換

| ID | tmux引数 | 期待されるController呼び出し | 期待される出力 |
|----|----------|------|------|
| TC-C001a | `[]`（引数なし） | `NewSplit(SplitDown)`（tmuxデフォルトは垂直分割=下方向、`-v`相当） | なし |
| TC-C001b | `["-h"]` | `NewSplit(SplitRight)` | なし |
| TC-C001c | `["-v"]` | `NewSplit(SplitDown)` | なし |
| TC-C001d | `["--left"]` | `NewSplit(SplitLeft)` | なし。gmux独自拡張フラグ（tmuxには存在しない） |
| TC-C001e | `["--right"]` | `NewSplit(SplitRight)` | なし。gmux独自拡張フラグ（tmuxには存在しない） |
| TC-C001f | `["--up"]` | `NewSplit(SplitUp)` | なし。gmux独自拡張フラグ（tmuxには存在しない） |
| TC-C001g | `["--down"]` | `NewSplit(SplitDown)` | なし。gmux独自拡張フラグ（tmuxには存在しない） |
| TC-C001h | `["-t", "%0", "-h"]` | `NewSplit(SplitRight)`, target=%0 | なし |
| TC-C001i | `["-t", "%0", "-h", "-l", "70%", "-P", "-F", "#{pane_id}"]` | `NewSplit(SplitRight)` | `"%N"` 形式のペインID出力 |
| TC-C001j | `["-h", "-v"]`（相反フラグ） | `NewSplit(SplitDown)`（後勝ち） | なし |
| TC-C001k | `["--help"]` | ヘルプ出力、Controller呼び出しなし | ヘルプテキスト |

#### TC-C002: `send-keys` のバイト列組み立てとPTYプロキシ送信

tmuxのsend-keysは各引数を個別に送信し、引数間にスペースを入れない。スペースが必要な場合は明示的にスペース文字を含む引数として渡す必要がある。ただしClaude Codeの実際のパターンは `send-keys -t %0 "cd /path && command" Enter` のようにテキスト全体を1つの引数として渡す形式が主流。

| ID | tmux引数 | 期待される動作 |
|----|----------|------|
| TC-C002a | `["ls", "Enter"]` | ソケット経由でPTYマスターに `"ls\r"` を書き込み（引数間スペースなし） |
| TC-C002b | `["echo hello"]` | ソケット経由でPTYマスターに `"echo hello"` を書き込み |
| TC-C002c | `["C-c"]` | ソケット経由でPTYマスターに `"\x03"` を書き込み |
| TC-C002d | `["-t", "%0", "ls -la", "Enter"]` | ソケット経由でペイン%0のPTYマスターに `"ls -la\r"` を書き込み |
| TC-C002e | `["echo", " ", "hello", " ", "world", "Enter"]` | ソケット経由でPTYマスターに `"echo hello world\r"` を書き込み |
| TC-C002f | `[]`（引数なし） | no-op, exit code 0（tmux互換: 引数なしはエラーにならず空操作） |
| TC-C002g | `["--help"]` | ヘルプ出力、送信なし |
| TC-C002h | `["cd /path/to/dir", "Enter"]` | ソケット経由でPTYマスターに `"cd /path/to/dir\r"` を書き込み |
| TC-C002i | `["Up", "Enter"]`（ヒストリ操作） | ソケット経由でPTYマスターに `"\x1b[A\r"` を書き込み |
| TC-C002j | `["C-c", "C-c"]`（連続Ctrl+C） | ソケット経由でPTYマスターに `"\x03\x03"` を書き込み |
| TC-C002k | `["-l", "Enter"]`（リテラルモード） | ソケット経由でPTYマスターに `"Enter"` を書き込み（特殊キー展開せずリテラル送信） |
| TC-C002l | `["-l", "-t", "%0", "Enter"]`（リテラル+ターゲット指定） | ソケット経由でペイン%0のPTYマスターに `"Enter"` を書き込み（特殊キー展開せずリテラル送信） |
| TC-C002m | `["-t", "%0", "-l", "Enter"]`（-tが先、-lが後） | ソケット経由でペイン%0のPTYマスターに `"Enter"` を書き込み（オプション順序不問で同じ結果） |

#### TC-C003: `select-pane` -> `GotoSplit` 変換

| ID | tmux引数 | 期待されるController呼び出し |
|----|----------|------|
| TC-C003a | `["-U"]` | `GotoSplit(GotoUp)` |
| TC-C003b | `["-D"]` | `GotoSplit(GotoDown)` |
| TC-C003c | `["-L"]` | `GotoSplit(GotoLeft)` |
| TC-C003d | `["-R"]` | `GotoSplit(GotoRight)` |
| TC-C003e | `[]`（引数なし） | no-op, exit code 0（現在のペインを維持） |
| TC-C003f | `["-t", "%0"]` | ペイン%0へのフォーカス移動 |
| TC-C003g | `["-t", "%0", "-P", "bg=#282828"]` | スタイル設定（将来対応。現状はフォーカス移動のみ）。stderrが空であること（警告なし） |
| TC-C003h | `["-t", "%0", "-T", "agent-1"]` | タイトル設定（将来対応。現状はフォーカス移動のみ）。stderrが空であること（警告なし） |
| TC-C003i | `["--help"]` | ヘルプ出力 |

注記: macOS keysim環境では`-U`/`-D`/`-L`/`-R`の方向指定について、ghosttyが`goto_split:up`等の方向指定をサポートしていない場合、`goto_split:next`/`goto_split:previous`の繰り返しで代替する実装方針とする。

##### TC-C003サブグループ: フォーカス移動アルゴリズム

`select-pane -t %N` でのgoto_split:next/previous繰り返しによるフォーカス移動の回数計算を検証する。

| ID | ペイン構成 | アクティブ | ターゲット | 期待される移動 |
|----|-----------|-----------|-----------|---------------|
| TC-C003j | 3ペイン [%0, %1, %2] | %0 | %1 | next x 1 |
| TC-C003k | 3ペイン [%0, %1, %2] | %0 | %2 | previous x 1（next x 2 より短い） |
| TC-C003l | 5ペイン [%0, %1, %2, %3, %4] | %0 | %3 | previous x 2（next x 3 より短い） |
| TC-C003m | 5ペイン [%0, %1, %2, %3, %4] | %2 | %2 | 移動なし（アクティブ==ターゲット、steps=0） |
| TC-C003n | 1ペイン [%0] | %0 | %0 | 移動なし（ペイン数=1） |
| TC-C003o | 4ペイン [%0, %1, %2, %3] | %0 | %2 | next x 2 または previous x 2（等距離の場合はnextを選択） |

#### TC-C004: `display-message` のフォーマット展開

| ID | tmux引数 | 期待出力 |
|----|----------|----------|
| TC-C004a | `["-p", "#{session_name}"]` | `"gmux"` |
| TC-C004b | `["-p", "#{window_index}"]` | `"0"` |
| TC-C004c | `["-p", "#{pane_id}"]` | `"%0"` |
| TC-C004d | `["-p", "session=#{session_name}"]` | `"session=gmux"` |
| TC-C004e | `["-p", "#{pane_width}x#{pane_height}"]` | `"80x24"` |
| TC-C004f | `["#{session_name}"]`（-pなし） | no-op, exit code 0 |

Design rationale: `-p`なしの`display-message`はtmuxではステータスラインに表示するが、gmuxにステータスラインの概念がないため無視する。

#### TC-C005: `list-panes -F` のフォーマット展開

| ID | tmux引数 | 期待出力 |
|----|----------|----------|
| TC-C005a | `["-F", "#{pane_id}"]` | `"%0"` |
| TC-C005b | `["-F", "#{pane_id}:#{pane_width}x#{pane_height}:#{pane_active}"]` | `"%0:80x24:1"` 形式 |
| TC-C005c | `[]`（引数なし） | デフォルト出力 `"0: [80x24] ..."` |
| TC-C005d | `["-t", "gmux"]` | セッションフィルタ（現状は無視、全ペイン出力） |

#### TC-C006: `list-windows -F` のフォーマット展開

| ID | tmux引数 | 期待出力 |
|----|----------|----------|
| TC-C006a | `["-F", "#{window_id}"]` | `"@0"` |
| TC-C006b | `[]`（引数なし） | デフォルト出力 |

#### TC-C007: `list-sessions -F` のフォーマット展開

| ID | tmux引数 | 期待出力 |
|----|----------|----------|
| TC-C007a | `["-F", "#{session_name}"]` | `"gmux"` |
| TC-C007b | `[]`（引数なし） | `"gmux: 1 windows ..."` |

#### TC-C008: `has-session` のセッション存在確認

| ID | tmux引数 | 期待結果 |
|----|----------|----------|
| TC-C008a | `["-t", "gmux"]`（一致するセッション名） | exit code 0 |
| TC-C008b | `["-t", "nonexistent"]`（不一致のセッション名） | exit code 1 |
| TC-C008c | `[]`（引数なし） | exit code 0。Design rationale: gmuxはラッパーモード内で常に単一セッションが存在するためexit 0を返す |
| TC-C008d | `["-t", "テスト_セッション"]`（マルチバイト文字） | セッション名照合が正しく動作すること |

#### TC-C009: `new-session` のセッション管理

| ID | tmux引数 | 期待結果 |
|----|----------|----------|
| TC-C009a | `["-A", "-s", "gmux_worktree-feature"]` | セッション作成/アタッチ（no-op相当） |
| TC-C009b | `["-A", "-s", "test", "-c", "/path/to/dir", "--", "claude", "--args"]` | コマンド付きセッション（将来対応） |
| TC-C009c | `["-A", "-s", "existing_session"]`（セッション存在時） | フォーカス移動（no-op相当）, exit code 0 |
| TC-C009d | `["-A", "-s", "new_session"]`（セッション非存在時） | 新規ウィンドウ作成, exit code 0 |
| TC-C009e | `["-A", "-s", "test", "-c", "/path/to/workdir"]` | ワーキングディレクトリを記録のみ, exit code 0 |
| TC-C009f | `["-A", "-s", "test", "--", "claude", "--resume"]` | `--` 以降のコマンド引数はPhase 2以降で対応。Phase 1ではno-op, exit code 0 |
| TC-C009g | `["--", "-s", "session"]` | `--`が先頭: `-s`はコマンド引数として扱われる（オプションとしてパースしない）。セッション名はデフォルトの`"gmux"` |
| TC-C009h | `["-A", "-s", "test", "--"]` | `--`の後に引数なし: セッション名は`"test"`、コマンド引数なし。exit code 0 |
| TC-C009i | `["-A", "-s", "テスト_セッション"]`（マルチバイト文字） | セッション名にマルチバイト文字が正しく記録されること |

#### TC-C010: `show-options` のレスポンス

| ID | tmux引数 | 期待出力 |
|----|----------|----------|
| TC-C010a | `["-g", "prefix"]` | `"prefix C-b\n"`（tmuxデフォルト互換） |
| TC-C010b | `["-gv", "prefix"]` | `"C-b\n"`（値のみ） |
| TC-C010c | `["-g", "unknown-option"]` | 空出力, exit code 0 |
| TC-C010d | `["prefix"]`（-gなし） | `"prefix C-b\n"`, exit code 0 |
| TC-C010e | `["-gv", "prefix"]`（フラグ結合形式） | `"C-b\n"`（`-gv`が`-g` + `-v`に分離パースされる） |
| TC-C010f | `["-s", "prefix"]` | `"prefix C-b\n"`, exit code 0 |
| TC-C010g | `["-w", "prefix"]` | `"prefix C-b\n"`, exit code 0 |

Design rationale: Claude Codeは `show-options -gv prefix` でprefix keyを取得する。`-gv` のようなフラグ結合形式は `-g` + `-v` に分離パースされる必要がある。gmuxではtmuxのデフォルト値を返す。未知のオプションに対しては空出力で安全にフォールバックする。

#### TC-C011: `resize-pane` の変換

| ID | tmux引数 | 期待されるController呼び出し |
|----|----------|------|
| TC-C011a | `["-U", "5"]` | `ResizeSplit(ResizeUp, 5)` |
| TC-C011b | `["-D", "10"]` | `ResizeSplit(ResizeDown, 10)` |
| TC-C011c | `["-L", "3"]` | `ResizeSplit(ResizeLeft, 3)` |
| TC-C011d | `["-R"]`（量なし） | `ResizeSplit(ResizeRight, 1)` |
| TC-C011e | `["-t", "%0", "-x", "30%"]`（パーセント指定） | 将来対応。現状はベストエフォート |
| TC-C011f | `["--help"]` | ヘルプ出力 |

#### TC-C012: `kill-pane` の変換

| ID | tmux引数 | 期待されるController呼び出し |
|----|----------|------|
| TC-C012a | `[]`（引数なし） | `CloseSurface()` |
| TC-C012b | `["-t", "%0"]` | 指定ペイン%0へフォーカス移動後に`CloseSurface()`（Phase 1対応） |

#### TC-C013: 未知コマンドのハンドリング

| ID | コマンド | 期待結果 |
|----|----------|----------|
| TC-C013a | `"break-pane"` | stderr に警告出力、exit code 0 |
| TC-C013b | `"join-pane"` | stderr に警告出力、exit code 0 |
| TC-C013c | `"completely-unknown-cmd"` | stderr に警告出力、exit code 0 |

#### TC-C014: `new-window` の変換

| ID | tmux引数 | 期待されるController呼び出し |
|----|----------|------|
| TC-C014a | `[]`（引数なし） | `NewWindow()` |
| TC-C014b | `["-t", "gmux", "-n", "work", "-P", "-F", "#{pane_id}"]` | `NewWindow()` + ペインID出力 |
| TC-C014c | `["--help"]` | ヘルプ出力 |
| TC-C014d | `["-n", "mywindow"]` | `NewTab()`（ウィンドウ名は記録して無視） |
| TC-C014e | `["-t", "%0"]` | ターゲット指定（gmuxではペイン概念で処理） |

#### TC-C015: `switch-client` の変換

| ID | tmux引数 | 期待されるController呼び出し | 期待結果 |
|----|----------|------|----------|
| TC-C015a | `["-t", "session_name"]` | なし（no-op相当） | exit code 0 |
| TC-C015b | `["-t", "gmux_worktree-feature"]` | なし（no-op相当） | exit code 0 |
| TC-C015c | `[]`（引数なし） | なし（no-op相当） | exit code 0 |

Design rationale: gmuxは単一セッション前提のため、switch-clientは常にno-opで成功する。Claude Codeのfast pathで `has-session` + `switch-client` のセットで呼ばれる。

#### TC-C016: `select-layout` のハンドリング

| ID | tmux引数 | 期待されるController呼び出し | 期待結果 |
|----|----------|------|----------|
| TC-C016a | `["-t", "window", "main-vertical"]` | なし | stderr に警告出力, exit code 0 |
| TC-C016b | `["-t", "window", "even-horizontal"]` | なし | stderr に警告出力, exit code 0 |
| TC-C016c | `["tiled"]` | なし | stderr に警告出力, exit code 0 |

Design rationale: select-layoutはPhase 2コマンド。Phase 1では未対応コマンドとして警告出力しexit 0で安全に通過させる。Phase 2で `equalize_splits` 等へのマッピングを実装する。

#### TC-C017: attach-session（Phase 3）

| ID | 入力 | 期待結果 | 備考 |
|----|------|----------|------|
| TC-C017a | `["attach-session", "-t", "mysession"]` | Phase 3で実装予定 | Claude Codeの `--tmux` フラグ利用時に使用される可能性あり |

#### TC-C018: set-option（Phase 3）

| ID | 入力 | 期待結果 | 備考 |
|----|------|----------|------|
| TC-C018a | `["set-option", "-g", "prefix", "C-a"]` | Phase 3で実装予定 | gmuxではtmuxデフォルト値を返すため、設定変更はno-op |

#### TC-C019: バージョン出力

| ID | tmux引数 | 期待出力 | 期待結果 |
|----|----------|----------|----------|
| TC-C019a | `["-V"]` | `"tmux 3.5a\n"` | exit code 0 |

---

### 2.3 gmux-agent PTYプロキシ (`cmd/gmux-agent/`)

> **goroutineリーク検出**: gmux-agentのテストでは `go.uber.org/goleak` を使用してgoroutineリークを検出する。`TestMain` に `goleak.VerifyTestMain(m)` を設置する。

#### TC-AG001: gmux-agent起動モード分岐

| ID | 環境 | 期待結果 |
|----|------|----------|
| TC-AG001a | `GMUX=1` 設定時 | PTYプロキシモードで起動。内部PTY作成、Unixソケット作成 |
| TC-AG001b | `GMUX` 未設定時 | デフォルトシェルを`exec`（プロセス置換、gmux-agentプロセスは消滅） |
| TC-AG001c | `GMUX=` (空文字列)時 | `GMUX`未設定と同じ扱い（デフォルトシェルをexec） |
| TC-AG001d | `SHELL=/bin/bash` 設定時、`GMUX`未設定 | `/bin/bash` を exec |
| TC-AG001e | `GMUX=1`, `GMUX_PANE_ID=5` 設定時 | PTYプロキシモードで起動。ペインID=`"5"` を使用してソケットパスを生成 |
| TC-AG001f | `GMUX=1`, `GMUX_PANE_ID` 未設定時 | エラー終了（設定漏れ検出。初期ペインはGMUX_PANE_ID=0、split時は新IDが伝達される前提のため、未設定は異常） |
| TC-AG001g | `GMUX=1`, `GMUX_SESSION=test_session`, `GMUX_PANE_ID=0` 設定時 | ソケットパスが `<tmpdir>/gmux/test_session/pane-0.sock` に生成される |
| TC-AG001h | `GMUX=1`, `GMUX_SESSION` が極端に長い（200文字）場合 | ソケットパス長制限に基づくエラーハンドリング（TC-AG003h参照） |
| TC-AG001i | ソケットディレクトリ作成失敗（パーミッション不足） | エラーメッセージ出力 + exit 1 |

#### TC-AG002: 内部PTY作成と管理

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-AG002a | PTYプロキシモード起動 | master/slaveペアが作成される |
| TC-AG002b | シェルの接続 | slave側がシェルのstdin/stdout/stderrに接続される |
| TC-AG002c | I/Oプロキシ | stdin→master(Write)、master(Read)→stdoutの双方向プロキシが動作 |
| TC-AG002d | ユーザ入力の透過 | ghostty PTY経由のキー入力がシェルに到達する |
| TC-AG002e | シェル出力の透過 | シェルの出力がghostty PTYに表示される |

#### TC-AG003: Unixソケットサーバ

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-AG003a | ソケット作成 | `<tmpdir>/gmux/<session>/pane-<id>.sock` にソケットが作成される |
| TC-AG003b | ソケットディレクトリ権限 | ディレクトリが `0700` で作成される |
| TC-AG003c | ソケットファイル権限 | ソケットが `0600` で作成される |
| TC-AG003d | 接続受付 | Unixソケットへのダイヤルが成功する |
| TC-AG003e | 複数接続 | 複数のクライアントが同時接続可能（goroutine per connectionで並行処理）。注記: 上限値の定義は設計書§3のパラメータテーブルを参照 |
| TC-AG003f | ソケット未準備時のリトライ成功 | gmux-agentのソケット準備前にsend-keys接続を試み、100ms間隔のリトライで接続成功する（遅延起動シミュレーション） |
| TC-AG003g | リトライ回数超過時のエラー返却 | ソケットが存在しない状態で10回リトライ後、`"pane %N not reachable"` エラーが返る |
| TC-AG003h | セッション名が極端に長い場合（ソケットパスが104バイト超過） | ソケット作成時にパス長制限エラーが返る。macOSの `sun_path` 制限は104バイト |
| TC-AG003i | 同一UID接続の許可 | 同一ユーザのプロセスからのソケット接続が `getpeereid()` 検証を通過して許可される。注記: 異なるUIDからの接続拒否テストはroot権限が必要なため、CIでは実施しない |
| TC-AG003j | シャットダウン時のクリーンアップ | gmux-agent終了時にソケットファイルが削除される |
| TC-AG003k | 起動時のstaleソケット検出・削除 | gmux-agent起動時に同一パスにソケットファイルが残存しており接続不能（stale）な場合、そのソケットを削除してから新規作成する |
| TC-AG003l | staleソケット検出時に接続成功（別プロセス稼働中） | `"socket is already in use by another process"` エラーで終了 |

#### TC-AG004: send-keysデータ受信と書き込み

注記: `SendKeysRequest.Data` はstring型（UTF-8文字列）。Base64エンコードではなく、そのまま文字列としてJSON送信する。制御文字はJSONエスケープ（`\n`, `\x03` 等）で表現される。

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-AG004a | `{"data":"echo hello\r"}` 送信 | 内部PTYマスターに "echo hello\r" が書き込まれ、シェルに入力される |
| TC-AG004b | 特殊キー（`\x03` = Ctrl+C）送信 | シェルにSIGINTが送られる |
| TC-AG004c | 空データ送信 | no-op、`{"ok":true}` 応答 |
| TC-AG004d | バイナリデータ送信 | そのまま内部PTYに書き込まれる |
| TC-AG004e | 大量データ（1MB）送信 | 正常送信またはグレースフルエラー |
| TC-AG004f | レスポンス形式 | `{"ok":true}` または `{"ok":false,"error":"..."}` |
| TC-AG004g | 不正なJSON送信 | エラーレスポンス、接続は切断される |
| TC-AG004h | 同時接続数制限超過（50同時接続） | 設計上の上限16を超える50同時接続を試行し、超過分がリジェクトされることを検証。既存接続は影響を受けない |
| TC-AG004i | 巨大リクエスト（1MB超のデータ） | `{"ok":false,"error":"..."}` で拒否。メモリ枯渇しないこと |
| TC-AG004j | 接続後にデータ送信なしでタイムアウト | 5秒後に接続がクローズされる。他のクライアントの接続をブロックしないこと |
| TC-AG004k | `GMUX_PANE_ID=3` でPTYプロキシモード起動後に子シェルの環境変数を確認 | `TMUX_PANE=%3` が設定されている |
| TC-AG004l | `GMUX_PANE_ID=0` でPTYプロキシモード起動後に子シェルの環境変数を確認 | `TMUX_PANE=%0` が設定されている |

#### TC-AG005: SIGWINCH転送

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-AG005a | ターミナルリサイズ（SIGWINCH受信） | 内部PTYのウィンドウサイズが更新される |
| TC-AG005b | リサイズ後のシェル動作 | シェルが新しいサイズを認識する（$COLUMNSの変化等） |
| TC-AG005c | 起動直後の初期サイズ同期 | 内部PTYのサイズが外側（ghostty PTY）のサイズと一致する。TIOCGWINSZ で取得した行数・列数が外側PTYと同一であること |
| TC-AG005d | 短時間に複数回のSIGWINCH（50ms間隔で5回連続リサイズ） | 最終的な内部PTYサイズが最後のリサイズ値と一致する。中間サイズへの収束ではなく最終値が反映されること |
| TC-AG005e | TIOCGWINSZ による具体的なサイズ値検証（80x24 → 120x40） | ioctl TIOCGWINSZ で内部PTYを問い合わせ、cols=120, rows=40 が返ること |

#### TC-AG006: シェル終了処理

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-AG006a | シェルが正常終了（exit 0） | gmux-agentも正常終了。ソケットファイルが削除される |
| TC-AG006b | シェルがexit 42で終了 | gmux-agentもexit 42で終了 |
| TC-AG006c | シェルがSIGKILLで死亡 | gmux-agentが検知して終了 |

---

### 2.4 ペイン管理 (`internal/pane/`)

#### TC-PM001: ペインID生成・割り当て

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-PM001a | 初回ペイン登録 | ID=`"%0"`, nextID=1 |
| TC-PM001b | 2回目ペイン登録 | ID=`"%1"`, nextID=2 |
| TC-PM001c | 10回連続登録 | ID=`"%0"`~`"%9"`, nextID=10 |
| TC-PM001d | ペイン削除後に新規登録 | IDは連番で増加し続ける（再利用しない） |

#### TC-PM002: ペイン登録・削除

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-PM002a | `Register("%0", "/dev/ttys001")` | panes["%0"]が登録される |
| TC-PM002b | `Unregister("%0")` | panes["%0"]が削除される |
| TC-PM002c | 存在しないID削除 | エラーまたはno-op |
| TC-PM002d | 同じIDに重複登録 | 上書き（既存ペインの情報を更新）。Design rationale: 同一IDへの再登録はペイン情報の更新として扱う。エラーにするとリカバリが困難になるため |

#### TC-PM003: ペインIDによるルックアップ

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-PM003a | `Get("%0")` （存在するID） | 対応するPane構造体 |
| TC-PM003b | `Get("%999")` （存在しないID） | nil + エラー |
| TC-PM003c | `List()` | 登録済み全ペインのスライス |
| TC-PM003d | `GetActive()` | Active=trueのペイン |

#### TC-PM004: 状態ファイルの永続化・読み込み

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-PM004a | `Save()` → `Load()` | 保存前と読み込み後で同一データ |
| TC-PM004b | 空のManager → `Save()` | 空JSONの書き込み |
| TC-PM004c | 存在しない状態ファイル → `Load()` | エラーなし、空のManager |
| TC-PM004d | 不正なJSONファイル → `Load()` | エラー返却 |
| TC-PM004e | 状態ファイルパスに `~/.local/state/gmux/` | XDG準拠パス |
| TC-PM004f | セッションごとのファイル分離 `panes-<session>.json` | 別セッションと混在しない |
| TC-PM004g | 状態ファイルディレクトリ作成失敗（パーミッション不足） | Save()がエラーを返す |

#### TC-PM005: 同時アクセス（ファイルロック）

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-PM005a | 2つのプロセスが同時にSave | flockにより逐次処理 |
| TC-PM005b | Save中にLoadが発生 | 一貫したデータが読める |
| TC-PM005c | ロック取得タイムアウト | エラー返却（デッドロックしない） |

#### TC-PM006: PTY差分検出

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-PM006a | split前後のTTYデバイス差分 | 新しいTTYが1つ検出される |
| TC-PM006b | split後にTTYが増えない（タイミング問題） | リトライ後にエラーまたは検出 |
| TC-PM006c | 無関係なプロセスのTTYが同時に出現 | ghosttyプロセスツリーでフィルタリング |
| TC-PM006d | close後のTTYデバイス消失検出 | stat失敗でペイン消失を検知 |

---

### 2.5 ログ蓄積 (`internal/logger/`)

#### TC-L001: JSONLフォーマットの正確性

| ID | 入力 | 期待結果 |
|----|------|----------|
| TC-L001a | `Entry{Command:"split-window", Args:["-h"]}` | 有効なJSONL行。全フィールド含む |
| TC-L001b | `Entry{Command:"send-keys", Args:["-t","%0","ls","Enter"]}` | Argsが正しくJSON配列に |
| TC-L001c | `Entry{Error:"some error"}` | `"error"` フィールドが含まれる |
| TC-L001d | `Entry{Error:""}` | `"error"` フィールドが省略される（omitempty） |
| TC-L001e | `Entry{Unsupported:true}` | `"unsupported":true` が含まれる |
| TC-L001f | `Entry{Unsupported:false}` | `"unsupported"` フィールドが省略される（omitempty） |
| TC-L001g | Timestamp | ISO 8601形式（RFC 3339）。タイムスタンプはUTC（末尾`Z`）で出力すること |

#### TC-L002: 呼び出し元プロセス情報の取得

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-L002a | `getCallerInfo()` | CallerPID=親プロセスのPID |
| TC-L002b | `getCallerInfo()` | CallerNameが空でない |
| TC-L002c | 親プロセスが終了済み | エラーにならない（グレースフル） |

#### TC-L003: 未対応コマンドのフラグ付け

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-L003a | 未知コマンド `"break-pane"` 実行時のログ | `Unsupported:true` |
| TC-L003b | 既知コマンド `"split-window"` 実行時のログ | `Unsupported:false` |

#### TC-L004: ログローテーション

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-L004a | 日付が変わった場合 | 新しい日別ファイルに書き込み |
| TC-L004b | ファイル名形式 | `gmux-YYYY-MM-DD.jsonl` |
| TC-L004c | ログディレクトリが存在しない | 自動作成 |
| TC-L004d | ログ書き込み失敗 | stderrに警告、メイン処理は続行 |

#### TC-L005: ログ書き込みの並行安全性

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-L005a | 複数goroutineから同時書き込み | 各行が壊れない（行単位のアトミック性） |
| TC-L005b | ファイルアペンドモード | 既存ログを上書きしない |

#### TC-L006: send-keysログのデータマスク

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-L001h | send-keysログにデータバイト列が含まれないこと | argsにテキスト引数が記録されるが、結合後のバイト列は除外 |
| TC-L001i | GMUX_LOG_DATA=1でデータバイト列がログに記録されること | デバッグ用の有効化テスト |

---

### 2.6 ラッパーモード (`internal/wrapper/`)

#### TC-W001: PATH偽装

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-W001a | `buildEnv("/tmp/gmux-wrapper-XXX")` | PATHの先頭が`"/tmp/gmux-wrapper-XXX"` |
| TC-W001b | 元のPATHが存在 | 元のPATHが偽装ディレクトリの後ろに続く |
| TC-W001c | 元のPATHが空 | PATHが偽装ディレクトリのみ |
| TC-W001d | symlinkの存在確認 | `/tmp/gmux-wrapper-XXX/tmux` が存在 |
| TC-W001e | symlinkの参照先 | gmux自身のバイナリを指す |

#### TC-W002: TMUX環境変数の偽装フォーマット

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-W002a | `buildEnv(tmpDir)` | `TMUX=<tmpDir>/gmux,<PID>,0` 形式 |
| TC-W002b | TMUX値のパース互換性 | カンマ区切り3要素: ソケットパス, PID, ウィンドウインデックス |
| TC-W002c | PIDフィールド | 現在のプロセスのPID（`os.Getpid()`との一致を検証） |
| TC-W002d | ウィンドウインデックス | `"0"` |

#### TC-W002サブグループ: セッション名生成ルール

Claude Codeが生成するセッション名 `{repoName}_worktree-{worktreeName}` の `[^a-zA-Z0-9-]` → `_` 置換ルールを検証する。

| ID | 入力（リポジトリ名, worktree名） | 期待されるセッション名 |
|----|------|----------|
| TC-W002e | `"gmux"`, `"feature/new-cmd"` | `"gmux_worktree-feature_new-cmd"` |
| TC-W002f | `"my.app"`, `"fix/v2.0"` | `"my_app_worktree-fix_v2_0"` |
| TC-W002g | `"repo"`, `"日本語branch"` | マルチバイト文字が `_` に置換される |
| TC-W002h | `"repo"`, `""` （空worktree名） | `"repo_worktree-"` |
| TC-W002i | `"a!@#$%"`, `"b^&*()"` | `"a_____worktree-b_____"` |

#### TC-W003: GMUX環境変数の設定

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-W003a | `buildEnv(tmpDir)` | `GMUX=1` が含まれる |
| TC-W003b | 子プロセスから確認 | `$GMUX` が `"1"` |
| TC-W003c | `buildEnv(tmpDir)` | `GMUX_SESSION=<生成したセッション名>` が含まれる |
| TC-W003d | セッション名が `gmux_worktree-feature_new` の場合 | `$GMUX_SESSION` が `"gmux_worktree-feature_new"` と一致 |
| TC-W003g | ラッパーモード起動時のGMUX_PANE_ID | `GMUX_PANE_ID=0` が設定されている |
| TC-W003h | ラッパーモード起動時のTMUX_PANE | `TMUX_PANE=%0` が設定されている |

#### TC-W004: argv[0]分岐

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-W004a | argv[0]=`"tmux"` | tmux互換モードで動作 |
| TC-W004b | argv[0]=`"gmux"` | gmuxネイティブモードで動作 |
| TC-W004c | argv[0]=`"/usr/local/bin/gmux"` | gmuxネイティブモード（パス含む） |
| TC-W004d | argv[0]=`"/tmp/gmux-wrapper-XXX/tmux"` | tmux互換モード（symlinkパス） |
| TC-W004e | argv[0]がsymlink chain（多段symlink）の場合 | symlink解決してベース名で判定 |

#### TC-W005: 既存TMUX/GMUXのフィルタリング

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-W005a | 既存 `TMUX=/old/path,999,0` | 古い値が除去され新しい値のみ |
| TC-W005b | 既存 `GMUX` なし | `GMUX=1` が追加 |
| TC-W005c | TMUX が1つだけ存在 | 重複しない |

#### TC-W006: 子プロセスのシグナル転送

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-W006a | gmuxにSIGINT送信 | 子プロセスにSIGINTが転送される |
| TC-W006b | gmuxにSIGTERM送信 | 子プロセスにSIGTERMが転送される |
| TC-W006c | gmuxにSIGHUP送信 | 子プロセスにSIGHUPが転送される |
| TC-W006d | 子プロセスの終了コード | gmuxの終了コードとして返る |
| TC-W006e | 子プロセスがexit 42で終了 | gmuxもexit 42で終了 |

#### TC-W007: 一時ディレクトリのクリーンアップ

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-W007a | 正常終了後 | 一時ディレクトリが削除されている |
| TC-W007b | エラー終了後 | 一時ディレクトリが削除されている |
| TC-W007c | 子プロセスがシグナルで終了 | 一時ディレクトリが削除されている |

#### TC-W008: エラーケース

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-W008a | `Run([]string{})` | エラー: コマンド未指定 |
| TC-W008b | `Run([]string{"nonexistent-command"})` | エラー: コマンド起動失敗 |

#### TC-W009: XDG_STATE_HOMEオーバーライドのログパス

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-W009a | `XDG_STATE_HOME=/custom/path` 設定時 | ログが `/custom/path/gmux/logs/` に書き込まれる |
| TC-W009b | `XDG_STATE_HOME` 未設定時 | ログが `~/.local/state/gmux/logs/` に書き込まれる |

---

### 2.7 Controllerインターフェース (`internal/ghostty/`)

#### TC-GC001: CLIController

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-GC001a | `NewCLIController()` | GhosttyBinが`"ghostty"`（デフォルト） |
| TC-GC001b | `GHOSTTY_BIN=custom` 設定後 `NewCLIController()` | GhosttyBinが`"custom"` |
| TC-GC001c | `NewSplit(SplitRight)` | `ghostty +new-split --direction=right` を実行 |
| TC-GC001d | `GotoSplit(GotoNext)` | `ghostty +goto-split --direction=next` を実行 |
| TC-GC001e | `ResizeSplit(ResizeUp, 5)` | `ghostty +resize-split --direction=up --amount=5` を実行 |
| TC-GC001f | Class指定あり | `--class=<class>` が引数に含まれる |
| TC-GC001g | ghosttyが見つからない場合 | エラー返却 |

#### TC-GC002: KeySimController

> **CI実行可能性**: KeySimControllerのテストは `osascript` の実行を伴い、macOSのアクセシビリティ権限が必要。CI環境（ヘッドレス）では `t.Skip("requires macOS accessibility permissions")` でスキップする。`//go:build darwin` ビルドタグを使用し、非macOS環境ではコンパイル対象外とする。

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-GC002a | `NewKeySimController()` | 初期化成功 |
| TC-GC002b | `sendKeyCombo("super+d")` | osascript でキーストローク送信を実行 |
| TC-GC002c | osascriptが失敗した場合 | エラー返却 |
| TC-GC002d | `NewWindow()` | osascript で `keystroke "n" using {command down}` を実行 |
| TC-GC002e | `NewTab()` | osascript で `keystroke "t" using {command down}` を実行 |

ビルドタグ方針: `//go:build darwin` またはruntime.GOOSチェックでmacOS以外のプラットフォームではビルドスキップまたはスタブに切り替え。

#### TC-GC003: AutoController

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-GC003a | `NewAutoController()` | CLI, KeySimの両方が初期化される |
| TC-GC003b | CLIアクション成功 | CLIの結果が返る |
| TC-GC003c | CLIController失敗時のKeySim自動フォールバック | CLIが`ghostty +new-split`でエラー → KeySimControllerで再試行・成功 |
| TC-GC003d | CLI/KeySim両方失敗 | 適切なエラーメッセージを返す |

#### TC-GC004: IsInsideGhostty判定

| ID | 環境変数 `TERM_PROGRAM` | 期待結果 |
|----|--------------------------|----------|
| TC-GC004a | `"ghostty"` | true |
| TC-GC004b | `"Ghostty"` | true |
| TC-GC004c | `"GHOSTTY"` | true |
| TC-GC004d | `"xterm"` | false |
| TC-GC004e | `""` | false |
| TC-GC004f | 未設定 | false |
| TC-GC004g | `GHOSTTY_RESOURCES_DIR` が設定されている場合 | ghosttyの存在をフォールバック検出（TERM_PROGRAMが未設定でもghostty環境と判定） |

---

## 3. 統合テスト

### 3.1 tmuxコマンド -> ghosttyアクション 統合テスト

これらのテストはControllerモックを使い、tmuxコマンドラインからController呼び出しまでの全フローを検証する。

#### TC-I001: split-window E2E

| ID | tmuxコマンドライン | 期待されるController呼び出し | 期待出力 |
|----|-------------------|------|----------|
| TC-I001a | `tmux split-window -h` | `NewSplit(SplitRight)` | なし |
| TC-I001b | `tmux split-window -v` | `NewSplit(SplitDown)` | なし |
| TC-I001c | `tmux split-window -t %0 -h -l 70% -P -F "#{pane_id}"` | `NewSplit(SplitRight)` | `"%N"` 形式 |

#### TC-I002: send-keys E2E

| ID | tmuxコマンドライン | 期待される動作 |
|----|-------------------|------|
| TC-I002a | `tmux send-keys -t %0 "ls" Enter` | ソケット経由でペイン%0のPTYマスターに `"ls\r"` を書き込み |
| TC-I002b | `tmux send-keys -t %0 C-c` | ソケット経由でペイン%0のPTYマスターに `"\x03"` を書き込み |
| TC-I002c | `tmux send-keys "echo hello" Enter` | ソケット経由でPTYマスターに `"echo hello\r"` を書き込み |

#### TC-I003: kill-pane E2E

| ID | tmuxコマンドライン | 期待されるController呼び出し |
|----|-------------------|------|
| TC-I003a | `tmux kill-pane -t %0` | `CloseSurface()` |
| TC-I003b | `tmux kill-pane` | `CloseSurface()` |

#### TC-I004: list-panes E2E

| ID | tmuxコマンドライン | 期待出力 |
|----|-------------------|----------|
| TC-I004a | `tmux list-panes -F "#{pane_id}"` | `"%0"` |
| TC-I004b | `tmux list-panes` | デフォルトフォーマット出力 |

#### TC-I005: 連続操作シナリオ

前提条件: 初期状態でペイン%0のみが存在し、アクティブである。

| ID | 操作シーケンス | 期待結果 |
|----|---------------|----------|
| TC-I005a | split -> send-keys -> kill | 各操作が順序通り実行。最終的にペイン数が元に戻る |
| TC-I005b | split -> split -> list-panes | 3ペイン分のID一覧（将来の動的管理で対応） |
| TC-I005c | split -> select-pane -> send-keys | フォーカス移動後にテキスト送信 |

#### TC-I005d: split-window後の新ペインでのTMUX_PANE検証

| ID | 操作シーケンス | 期待結果 |
|----|---------------|----------|
| TC-I005d | `split-window -h -P -F "#{pane_id}"` → 新ペインで `echo $TMUX_PANE` | 出力されたペインID（例: `%1`）と `$TMUX_PANE` の値が一致する |

> **注記**: GMUX_PANE_IDの伝達（split-window時に新ペインIDが環境変数経由でgmux-agentに渡されること）はPhase 1 PoCでの検証項目とする。TC-AG001e, TC-AG004k, TC-AG004lも参照。

#### TC-I006: `-L` ソケット指定の透過

| ID | tmuxコマンドライン | 期待結果 |
|----|-------------------|----------|
| TC-I006a | `tmux -L claude-swarm split-window -h` | `-L` を無視して `NewSplit(SplitRight)` |
| TC-I006b | `tmux -L my-socket send-keys "test" Enter` | `-L` を無視してソケット経由でPTYマスターに書き込み |
| TC-I006c | `tmux -L socket has-session -t "name"` | `-L` を無視してexit code 0 |

---

### 3.2 Claude Code互換テスト

Claude Codeが `--tmux` で実行するコマンドシーケンスを模擬する。

#### TC-CC001: Claude Code fast path コマンドシーケンス

| ID | シーケンス | 期待結果 |
|----|-----------|----------|
| TC-CC001a | `has-session -t "gmux_worktree-feature"` → `switch-client -t "gmux_worktree-feature"` | セッション確認 → 切替（exit 0） |
| TC-CC001b | `display-message -p "#{session_name}"` | `"gmux"` 出力 |
| TC-CC001c | `show-options -gv prefix` | `"C-b"` 出力 |

#### TC-CC002: Agent Teams ペイン操作シーケンス

| ID | シーケンス | 期待結果 |
|----|-----------|----------|
| TC-CC002a | `list-panes -F "#{pane_id}"` → `split-window -t %0 -h -l 70% -P -F "#{pane_id}"` | ペインID取得 → 分割 → 新ペインID出力 |
| TC-CC002b | `send-keys -t %1 "cd /path && claude --resume" Enter` | テキスト＋改行送信 |
| TC-CC002c | `select-pane -t %1 -P "bg=#282828" -T "agent-1"` | フォーカス移動（スタイルは将来対応） |
| TC-CC002d | `resize-pane -t %0 -x 30%` | リサイズ（パーセント指定は将来対応）。注記: Phase 2コマンド（resize-pane）を含む。Phase 1では未対応コマンドとして安全にexit 0する |
| TC-CC002e | `kill-pane -t %1` | ペイン閉じる |

#### TC-CC003: 完全なエージェント起動フロー

前提条件: 初期状態でペイン%0のみが存在し、アクティブである。

```
1. has-session -t "gmux_worktree-feature"    → exit 0
2. switch-client -t "gmux_worktree-feature"  → exit 0
3. display-message -p "#{session_name}"      → "gmux"
4. show-options -gv prefix                   → "C-b"
5. list-panes -F "#{pane_id}:#{pane_width}x#{pane_height}:#{pane_active}"
                                             → "%0:80x24:1"
6. split-window -t %0 -h -l 70% -P -F "#{pane_id}"
                                             → "%1"
7. send-keys -t %1 "cd /worktree && claude --worktree --resume abc123" Enter
                                             → テキスト送信
8. select-pane -t %1 -P "bg=#282828" -T "agent-1"
                                             → フォーカス移動
```

全コマンドがエラーなく順序通りに実行されることを検証する。

#### TC-CC004: list-windows/list-sessions の Claude Code互換出力

| ID | シーケンス | 期待結果 |
|----|-----------|----------|
| TC-CC004a | `list-windows -F "#{window_id}:#{window_name}"` | Claude Codeが期待するフォーマットで出力 |
| TC-CC004b | `list-sessions -F "#{session_name}"` | セッション名リスト出力 |
| TC-CC004c | `list-sessions` → `list-windows` → `list-panes` の連続実行 | 各コマンドの出力が互いに整合している |

---

### 3.3 ラッパーモード統合テスト

#### TC-WI001: tmux偽装の検証

| ID | コマンド | 期待結果 |
|----|---------|----------|
| TC-WI001a | `gmux -- sh -c "which tmux"` | 偽装ディレクトリのtmuxパスを返す |
| TC-WI001b | `gmux -- sh -c "echo $TMUX"` | `<tmpDir>/gmux,<PID>,0` 形式 |
| TC-WI001c | `gmux -- sh -c "echo $GMUX"` | `"1"` |
| TC-WI001d | `gmux -- sh -c "echo $PATH"` | PATHの先頭に偽装ディレクトリ |

#### TC-WI002: tmux呼び出しの透過

| ID | コマンド | 期待結果 |
|----|---------|----------|
| TC-WI002a | `gmux -- sh -c "tmux has-session -t test"` | exit code 0 |
| TC-WI002b | `gmux -- sh -c "tmux display-message -p '#{session_name}'"` | `"gmux"` |
| TC-WI002c | `gmux -- sh -c "tmux list-panes -F '#{pane_id}'"` | `"%0"` |

#### TC-WI003: ネストしたコマンド実行

| ID | コマンド | 期待結果 |
|----|---------|----------|
| TC-WI003a | `gmux -- bash -c "tmux split-window -h"` | ghosttyへの分割命令が発行される |
| TC-WI003b | `gmux -- sh -c "tmux send-keys 'echo test' Enter"` | テキスト送信が実行される |

---

## 4. tmux互換性テスト

本物のtmuxが利用可能な場合に実施する出力互換性テスト。
`go test -tags tmuxcompat` で実行条件を制御する。

#### TC-TX001: `display-message -p` の出力フォーマット

| ID | 引数 | tmux出力との比較 |
|----|------|-----------------|
| TC-TX001a | `"#{session_name}"` | 文字列型の値が返る |
| TC-TX001b | `"#{window_index}"` | 数値文字列が返る |
| TC-TX001c | `"#{pane_id}"` | `%` + 数値の形式 |

#### TC-TX002: `list-panes -F` の出力フォーマット

| ID | 引数 | tmux出力との比較 |
|----|------|-----------------|
| TC-TX002a | `"#{pane_id}:#{pane_width}x#{pane_height}"` | `%N:WxH` 形式 |
| TC-TX002b | `"#{pane_id}:#{pane_active}"` | `%N:0` または `%N:1` |

#### TC-TX003: `list-sessions -F` の出力フォーマット

| ID | 引数 | tmux出力との比較 |
|----|------|-----------------|
| TC-TX003a | `"#{session_name}"` | セッション名文字列 |

#### TC-TX004: `has-session` の終了コード

| ID | 引数 | tmux出力との比較 |
|----|------|-----------------|
| TC-TX004a | `-t "存在するセッション"` | exit code 0 |
| TC-TX004b | `-t "存在しないセッション"` | tmux: exit code 1 / gmux: exit code 1（セッション名不一致でexit 1。tmuxと同じ挙動） |

#### TC-TX005: デフォルト出力フォーマット

| ID | コマンド | 検証内容 |
|----|---------|----------|
| TC-TX005a | `list-panes`（-Fなし） | tmuxのデフォルト出力フォーマットとの互換性（historyフィールドのフォーマットを含む） |
| TC-TX005b | `list-windows`（-Fなし） | tmuxのデフォルト出力フォーマットとの互換性 |
| TC-TX005c | `list-sessions`（-Fなし） | tmuxのデフォルト出力フォーマットとの互換性 |

---

## 5. エッジケース・異常系テスト

#### TC-E001: ghosttyが起動していない状態

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-E001a | `split-window` 実行 | エラーメッセージ（ghosttyが見つからない等） |
| TC-E001b | `send-keys "test"` 実行 | エラーメッセージ |
| TC-E001c | `display-message -p "#{session_name}"` | 正常（ghostty不要の情報取得） |
| TC-E001d | `has-session` | 正常（ghostty不要） |
| TC-E001e | `list-panes` | 正常（ghostty不要、静的応答） |

#### TC-E002: アクセシビリティ権限なし（macOS）

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-E002a | System Events経由のキーシミュレーション | 権限不足エラー、分かりやすいメッセージ |
| TC-E002b | 権限案内メッセージ | システム環境設定への誘導テキスト |

#### TC-E003: 存在しないペインIDへの操作

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-E003a | `send-keys -t %999 "test"` | エラー（ペインが見つからない） |
| TC-E003b | `kill-pane -t %999` | エラー（ペインが見つからない） |
| TC-E003c | `select-pane -t %999` | エラー（ペインが見つからない） |

#### TC-E004: 同時に複数のgmuxプロセスが動作

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-E004a | 2つのgmuxラッパーモードが並行実行 | 各々の一時ディレクトリが独立 |
| TC-E004b | 同時にペイン状態を更新 | flockで競合を回避 |
| TC-E004c | 同時にログ書き込み | ログ行が壊れない |

#### TC-E005: 非常に長いテキストのsend-keys

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-E005a | 1KB のテキスト | 正常送信 |
| TC-E005b | 100KB のテキスト | 正常送信またはグレースフルエラー |
| TC-E005c | 1MB のテキスト | エラーまたはチャンク分割 |

#### TC-E006: バイナリデータ/制御文字のsend-keys

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-E006a | NULLバイト(`\x00`)を含むテキスト | 適切にハンドリング |
| TC-E006b | ANSIエスケープシーケンス(`\x1b[31m`)を含むテキスト | そのまま送信 |
| TC-E006c | マルチバイト文字（日本語）を含むテキスト | 正常送信 |
| TC-E006d | 絵文字を含むテキスト | 正常送信 |

#### TC-E007: ネストしたgmux

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-E007a | `gmux -- sh -c "gmux -- echo hello"` | 二重偽装で正常動作 |
| TC-E007b | ネスト時のTMUX環境変数 | 外側のTMUXが内側で上書きされる |
| TC-E007c | ネスト時のPATH | 内側の偽装ディレクトリがPATH最先頭 |

#### TC-E008: 引数なしの実行

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-E008a | `gmux`（引数なし） | Usage出力、exit code 1 |
| TC-E008b | `gmux --`（コマンドなし） | エラーメッセージ |
| TC-E008c | tmuxモードで引数なし | Usage出力、exit code 1 |

#### TC-E009: ghosttyバックグラウンド時の挙動

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-E009a | ghosttyがバックグラウンドの状態で `split-window` | エラーメッセージ、またはghosttyをフォアグラウンド化してからリトライ |
| TC-E009b | ghosttyがバックグラウンドの状態で `send-keys "test" Enter` | 正常送信（PTYプロキシ方式ではghosttyのフォーカス状態に依存しない） |

#### TC-E010: キーバインドカスタマイズ検出

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-E010a | ghosttyのデフォルトキーバインドが変更されている場合 | stderrに警告メッセージ出力（ghostty設定ファイルの検出可能な場合） |
| TC-E010b | ghostty設定ファイルが存在しない場合 | 警告なし（デフォルトキーバインド前提で動作） |

#### TC-E012: gmux-agent未設定時の警告

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-E012a | gmux-agentがghostty configに設定されていない状態でsplit-window | 警告メッセージ出力 + エラー |

#### TC-E011: マルチウィンドウの挙動

| ID | 操作 | 期待結果 |
|----|------|----------|
| TC-E011a | 複数のghosttyウィンドウが存在する状態で `split-window` | フォアグラウンドウィンドウに対して操作が実行される |
| TC-E011b | 複数ウィンドウ存在時の `send-keys` | フォアグラウンドウィンドウの対象ペインに送信 |

---

## 6. パフォーマンステスト

`go test -bench` で計測する。

#### TC-PF001: gmuxの起動時間

| ID | 測定内容 | 許容値 | 測定方法 |
|----|---------|--------|----------|
| TC-PF001a | `gmux version` の実行時間 | 10ms以内 | `time gmux version` を100回繰り返して平均 |
| TC-PF001b | `gmux has-session -t test` の実行時間 | 10ms以内 | 同上 |
| TC-PF001c | `gmux display-message -p "#{session_name}"` の実行時間 | 10ms以内 | 同上 |
| TC-PF001d | `gmux list-panes -F "#{pane_id}"` の実行時間 | 10ms以内 | 同上 |

許容値の根拠: 10ms はGoバイナリの標準的な起動時間（5-10ms）に収まる閾値。Claude Codeのfast pathで頻繁にtmuxが呼ばれるため、情報取得系コマンド（外部プロセス不要）はこの範囲に収まる必要がある。

#### TC-PF002: ghosttyアクションの応答時間

| ID | 測定内容 | 許容値 | 測定方法 |
|----|---------|--------|----------|
| TC-PF002a | `split-window -h` の応答時間 | 200ms以内 | ベンチマーク（ghostty実機） |
| TC-PF002b | `send-keys "test" Enter` の応答時間 | 200ms以内 | ベンチマーク（ghostty実機） |
| TC-PF002c | `select-pane -R` の応答時間 | 200ms以内 | ベンチマーク（ghostty実機） |

許容値の根拠: 200ms はosascript実行時間（数十ms）を含む上限値。osascript/ghostty IPC呼び出し自体に数十msかかるため、200ms以内であれば実用上問題ない。

#### TC-PF003: ログ書き込みのオーバーヘッド

| ID | 測定内容 | 許容値 | 測定方法 |
|----|---------|--------|----------|
| TC-PF003a | ログ書き込み1回あたりの時間 | 1ms以内 | `testing.B` ベンチマーク |
| TC-PF003b | ログ有効/無効でのコマンド実行時間差 | 2ms以内 | 同一コマンドをログ有無で比較 |

#### TC-PF004: ラッパーモードのオーバーヘッド

| ID | 測定内容 | 許容値 | 測定方法 |
|----|---------|--------|----------|
| TC-PF004a | `gmux -- echo hello` の実行時間 | 50ms以内 | 100回繰り返して平均 |
| TC-PF004b | 一時ディレクトリ作成+symlink作成 | 5ms以内 | ベンチマーク |

---

## 7. テスト実行方法

### 7.1 全テスト実行

```bash
go test ./...
```

### 7.2 ユニットテストのみ

```bash
go test -short ./...
```

### 7.3 統合テスト（ラッパーモード含む）

```bash
go test -run "TestIntegration" ./...
```

### 7.4 tmux互換性テスト

```bash
go test -tags tmuxcompat ./...
```

### 7.5 パフォーマンステスト

```bash
go test -bench=. -benchtime=10s ./...
```

### 7.6 E2Eテスト（ghostty実機、手動確認含む）

```bash
go test -tags e2e -timeout 60s ./...
```

前提条件:
- ghostty起動済み
- アクセシビリティ権限付与済み（System Events経由のキーシミュレーションに必要）
- ghosttyウィンドウがフォアグラウンド状態

TestMainで環境チェックを行い、条件未達の場合は `t.Skip("ghostty not running or not in foreground")` でスキップする。

**クリーンアップ手順**:
- テスト前: テスト開始時にペイン状態をリセットする（状態ファイルの初期化）
- テスト後: `t.Cleanup()` で作成したペインを全て削除し、初期状態（ペイン%0のみ）に戻す
- ペイン削除は義務: `t.Cleanup()` でのペイン削除を全E2Eテストで義務付ける。テスト失敗時もCleanupが実行されるため、後続テストへの影響を防止する

```go
func setupE2ETest(t *testing.T) *TestEnv {
    t.Helper()
    env := &TestEnv{initialPanes: listCurrentPanes()}
    t.Cleanup(func() {
        // テストで作成したペインを全て削除
        currentPanes := listCurrentPanes()
        for _, p := range currentPanes {
            if !contains(env.initialPanes, p) {
                closePaneByID(p)
            }
        }
    })
    return env
}
```

### 7.7 カバレッジ

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

目標カバレッジ: ユニットテスト80%以上、パーサ/変換層は95%以上。

---

## 8. テスト優先度

### Phase 1 (MVP) -- 最優先

| 対象 | テストID群 |
|------|-----------|
| 特殊キー展開 | TC-P005全体 |
| -t ターゲット指定 | TC-P002全体, TC-E003 |
| -P -F 出力 | TC-P003全体, TC-C001i |
| フォーマット変数展開 | TC-P004全体 |
| split-window変換 | TC-C001全体 |
| send-keys変換 | TC-C002全体 |
| select-pane変換 | TC-C003全体 |
| display-message | TC-C004全体 |
| list-panes | TC-C005全体 |
| has-session | TC-C008全体 |
| new-session | TC-C009全体 |
| show-options | TC-C010全体 |
| switch-client | TC-C015全体 |
| 未知コマンド | TC-C013全体 |
| ペインID管理（生成・登録・削除・ルックアップ・永続化） | TC-PM001 ~ TC-PM004 |
| gmux-agent PTYプロキシ | TC-AG001 ~ TC-AG006 |
| ラッパーモードbuildEnv | TC-W001 ~ TC-W005 |
| ラッパーモードRun | TC-W006 ~ TC-W008 |
| Claude Code互換 | TC-CC001 ~ TC-CC004 |

### Phase 2 -- 動的ペイン管理

| 対象 | テストID群 |
|------|-----------|
| ペインID管理（同時アクセス・PTY差分検出） | TC-PM005 ~ TC-PM006 |
| -L ソケット指定 | TC-P001, TC-I006 |
| select-layout | TC-C016全体 |
| resize-pane | TC-C011全体 |
| list-windows | TC-C006全体 |
| list-sessions | TC-C007全体 |

### Phase 3 -- 完全対応

| 対象 | テストID群 |
|------|-----------|
| ログ蓄積 | TC-L001 ~ TC-L006 |
| パフォーマンス | TC-PF001 ~ TC-PF004 |
| tmux互換性 | TC-TX001 ~ TC-TX005 |
| エッジケース全般 | TC-E001 ~ TC-E012 |
| attach-session | TC-C017全体 |
| set-option | TC-C018全体 |
