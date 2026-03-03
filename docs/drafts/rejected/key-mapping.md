# gmux キーマッピング表

## サポート済みキー名

### 基本キー

| Key Name | Escape Sequence | Notes |
|----------|----------------|-------|
| Enter    | `\r` (0x0D)    | CR |
| Escape   | `\x1b` (0x1B)  | ESC |
| Tab      | `\t` (0x09)    | HT |
| Space    | ` ` (0x20)     | |
| BSpace   | `\x7f` (0x7F)  | DEL (Backspace) |
| BTab     | `ESC [ Z`      | Shift+Tab (Backtab) |

### 矢印キー

| Key Name | Escape Sequence | Notes |
|----------|----------------|-------|
| Up       | `ESC [ A`      | |
| Down     | `ESC [ B`      | |
| Right    | `ESC [ C`      | |
| Left     | `ESC [ D`      | |

### ナビゲーションキー

| Key Name | Escape Sequence | Notes |
|----------|----------------|-------|
| Home     | `ESC [ H`      | |
| End      | `ESC [ F`      | |
| IC       | `ESC [ 2 ~`    | Insert Character |
| Insert   | `ESC [ 2 ~`    | IC のエイリアス |
| DC       | `ESC [ 3 ~`    | Delete Character |
| Delete   | `ESC [ 3 ~`    | DC のエイリアス |
| PPage    | `ESC [ 5 ~`    | Page Up |
| PageUp   | `ESC [ 5 ~`    | PPage のエイリアス |
| PgUp     | `ESC [ 5 ~`    | PPage のエイリアス |
| NPage    | `ESC [ 6 ~`    | Page Down |
| PageDown | `ESC [ 6 ~`    | NPage のエイリアス |
| PgDn     | `ESC [ 6 ~`    | NPage のエイリアス |

### ファンクションキー

| Key Name | Escape Sequence | Notes |
|----------|----------------|-------|
| F1       | `ESC O P`      | SS3 形式 |
| F2       | `ESC O Q`      | SS3 形式 |
| F3       | `ESC O R`      | SS3 形式 |
| F4       | `ESC O S`      | SS3 形式 |
| F5       | `ESC [ 15 ~`   | CSI tilde 形式 |
| F6       | `ESC [ 17 ~`   | |
| F7       | `ESC [ 18 ~`   | |
| F8       | `ESC [ 19 ~`   | |
| F9       | `ESC [ 20 ~`   | |
| F10      | `ESC [ 21 ~`   | |
| F11      | `ESC [ 23 ~`   | |
| F12      | `ESC [ 24 ~`   | |

### C0 制御文字（tmux `[XXX]` 記法）

| Key Name | Code   | Notes |
|----------|--------|-------|
| [NUL]    | 0x00   | Null |
| [SOH]    | 0x01   | Start of Heading |
| [STX]    | 0x02   | Start of Text |
| [ETX]    | 0x03   | End of Text |
| [EOT]    | 0x04   | End of Transmission |
| [ENQ]    | 0x05   | Enquiry |
| [ASC]    | 0x06   | Acknowledge (tmux表記) |
| [BEL]    | 0x07   | Bell |
| [BS]     | 0x08   | Backspace |
| [LF]     | 0x0A   | Line Feed |
| [VT]     | 0x0B   | Vertical Tab |
| [FF]     | 0x0C   | Form Feed |
| [SO]     | 0x0E   | Shift Out |
| [SI]     | 0x0F   | Shift In |
| [DLE]    | 0x10   | Data Link Escape |
| [DC1]    | 0x11   | Device Control 1 |
| [DC2]    | 0x12   | Device Control 2 |
| [DC3]    | 0x13   | Device Control 3 |
| [DC4]    | 0x14   | Device Control 4 |
| [NAK]    | 0x15   | Negative Acknowledge |
| [SYN]    | 0x16   | Synchronous Idle |
| [ETB]    | 0x17   | End of Transmission Block |
| [CAN]    | 0x18   | Cancel |
| [EM]     | 0x19   | End of Medium |
| [SUB]    | 0x1A   | Substitute |
| [FS]     | 0x1C   | File Separator |
| [GS]     | 0x1D   | Group Separator |
| [RS]     | 0x1E   | Record Separator |
| [US]     | 0x1F   | Unit Separator |

### テンキー

| Key Name | Output | Notes |
|----------|--------|-------|
| KP0-KP9  | `0`-`9` | 数字キー |
| KP/      | `/`    | 除算 |
| KP*      | `*`    | 乗算 |
| KP-      | `-`    | 減算 |
| KP+      | `+`    | 加算 |
| KP.      | `.`    | 小数点 |
| KPEnter  | `\r`   | テンキー Enter |

### 修飾キー

| Prefix | Modifier | xterm パラメータ値 |
|--------|----------|-------------------|
| C-     | Control  | +4 |
| M-     | Meta/Alt | +2 |
| S-     | Shift    | +1 |

修飾キーは組み合わせ可能: `C-M-x`, `C-S-x`, `M-S-x`, `C-M-S-x`

xterm 修飾パラメータ = 1 + 修飾値の合計:

| 修飾             | パラメータ値 |
|-----------------|------------|
| Shift           | 2          |
| Meta            | 3          |
| Shift+Meta      | 4          |
| Ctrl            | 5          |
| Shift+Ctrl      | 6          |
| Meta+Ctrl       | 7          |
| Shift+Meta+Ctrl | 8          |

### Control+文字 (C-x)

| Key Name | Sequence   | Notes |
|----------|-----------|-------|
| C-a      | 0x01      | C-A も同一 |
| C-b      | 0x02      | |
| ...      | ...       | C-a(0x01) 〜 C-z(0x1A) |
| C-z      | 0x1A      | |
| C-@      | 0x00      | NUL |

大文字・小文字ともに同一の制御文字を生成。

### Meta+文字 (M-x)

`ESC` を前置して送信: `M-a` = `ESC a` (0x1B 0x61)

### 修飾キー付き特殊キー

xterm スタイルのエスケープシーケンスを生成:

| 例            | Sequence           | 形式 |
|--------------|-------------------|------|
| C-Up         | `ESC [ 1 ; 5 A`   | CSI letter |
| S-Up         | `ESC [ 1 ; 2 A`   | CSI letter |
| M-Up         | `ESC [ 1 ; 3 A`   | CSI letter |
| C-Home       | `ESC [ 1 ; 5 H`   | CSI letter |
| C-End        | `ESC [ 1 ; 5 F`   | CSI letter |
| C-PPage      | `ESC [ 5 ; 5 ~`   | CSI tilde |
| C-NPage      | `ESC [ 6 ; 5 ~`   | CSI tilde |
| C-F1         | `ESC [ 1 ; 5 P`   | SS3→CSI 変換 |
| C-F5         | `ESC [ 15 ; 5 ~`  | CSI tilde |
| C-M-S-Up     | `ESC [ 1 ; 8 A`   | 全修飾 |

## Claude Code 使用キー

| Key Name | 使用頻度 | 用途 |
|----------|---------|------|
| Enter    | 高      | コマンド実行、確認 |
| C-c      | 高      | プロセス割り込み (SIGINT) |
| C-d      | 高      | EOF送信、シェル終了 |
| C-z      | 中      | プロセス一時停止 (SIGTSTP) |
| C-l      | 中      | 画面クリア |
| C-a      | 中      | 行頭移動 |
| C-e      | 中      | 行末移動 |
| C-k      | 中      | カーソル以降削除 |
| C-u      | 中      | カーソル以前削除 |
| C-w      | 中      | 直前の単語削除 |
| Tab      | 高      | 補完 |
| BTab     | 低      | 逆方向補完 (Shift+Tab) |
| Up       | 高      | ヒストリ前方 |
| Down     | 高      | ヒストリ後方 |
| Left     | 中      | カーソル左移動 |
| Right    | 中      | カーソル右移動 |
| Home     | 低      | 行頭移動 |
| End      | 低      | 行末移動 |
| Escape   | 低      | キャンセル、vi モード切替 |
| Space    | 高      | スペース入力 |
| BSpace   | 中      | 文字削除 (Backspace) |
| DC       | 低      | カーソル位置の文字削除 |
| M-b      | 低      | 単語単位で左移動 |
| M-f      | 低      | 単語単位で右移動 |
| C-Right  | 低      | 単語単位で右移動（代替） |
| C-Left   | 低      | 単語単位で左移動（代替） |

## tmux 互換性

### カバー率

gmux は tmux の `send-keys` で使用される主要なキー名を網羅的にサポート。

| カテゴリ           | tmux 定義数 | gmux サポート | カバー率 |
|-------------------|------------|--------------|---------|
| 基本キー           | 6          | 6            | 100%    |
| 矢印キー           | 4          | 4            | 100%    |
| ナビゲーション      | 10         | 10           | 100%    |
| ファンクションキー   | 12         | 12           | 100%    |
| C0 制御文字        | 29         | 29           | 100%    |
| テンキー           | 16         | 16           | 100%    |
| C-x (a-z, A-Z, @) | 53         | 53           | 100%    |
| M-x               | 任意       | 任意          | 100%    |
| S-x               | 任意       | 任意          | 100%    |
| 複合修飾キー        | 任意       | 任意          | 100%    |
| 修飾付き特殊キー    | 任意       | 任意          | 100%    |

### 未サポートキー（スコープ外）

以下はgmuxのCLIラッパーとしての用途では不要なため、意図的に未サポート:

| カテゴリ | 理由 |
|---------|------|
| マウスイベント (MouseDown, MouseUp, WheelUp 等) | PTYプロキシ経由の send-keys では不要 |
| F13-F20 | 一般的なターミナルでは使用されない |
| FocusIn/FocusOut | ターミナルイベント、send-keys のスコープ外 |
| PasteStart/PasteEnd | ブラケットペースト制御、send-keys では不要 |
| User0-UserN | tmux 内部用カスタムキー |
