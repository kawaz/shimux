# moonbit-community ライブラリ調査

調査日: 2026-02-26

shimux poc3 で活用できるか検討するため、moonbit-community の 2 ライブラリを調査した。

## illusory0x0/native (v0.2.1)

https://github.com/moonbit-community/native

MoonBit native backend 用の FFI ユーティリティ。`moon add illusory0x0/native` で導入可能。

### 提供する型

| 型 | 用途 |
|---|---|
| `Ptr[T]` / `ConstPtr[T]` | 型安全なネイティブポインタ（加算・デリファレンス・reinterpret） |
| `CStr` | null 終端 C 文字列 |
| `Rc[T]` | GC オブジェクトの参照カウント管理 |

### ゼロコピーの仕組み

核心は `%identity` intrinsic（コード生成なし、型変換のみ）と `#external type`。

```moonbit
// Bytes → ポインタ変換がゼロコスト
Rc::scope_bytes(my_bytes, fn(ptr, len) {
  // ptr は GC ヒープ上の Bytes データを直接指す。コピーなし。
  some_c_function(ptr, len)
})
// スコープ終了で自動 release（moonbit_decref）
```

`CStr::from_bytes(bytes)` で Bytes から null 終端 C 文字列をゼロコピー構築可能（Bytes の末尾 `\0` 保証を利用）。

### poc3 での活用可能性

| 現状（poc3） | native で改善可能 |
|---|---|
| Rust `bytes_to_cstring()` でコピー+null追加 | `CStr::from_bytes()` でゼロコピー。Rust 側ヘルパー不要に |
| Rust `bytes_length()` で meta header パース | MoonBit 側で長さが取れる。Rust は `*const u8` + `len` を受け取るだけ |
| `pollfd_pack` の手動 LE バイトパッキング | `Ptr[T]` + `Getter/Setter` で型安全に |

### 判断

**将来の活用候補**。poc3 は既に動いているので即時導入は不要だが、イベントループ実装やリファクタリング時に検討する価値あり。特に `CStr` と `Rc::scope_bytes` は Rust 側のボイラープレート削減に有効。

### 注意点

- native target 専用（poc3 も native 専用なので問題なし）
- C スタブ (`stub.c`) を含む。Rust staticlib との共存は `moon.pkg.json` の設定で可能
- コミュニティライブラリのため API 安定性は未保証

## illusory0x0/posix

https://github.com/moonbit-community/posix

MoonBit 用 POSIX バインディング。C スタブ方式。`illusory0x0/native` に依存。

### カバー範囲

unistd.h 中心（read/write/close/fork/getpid 等）。`*at` 系関数優先の設計。

### shimux には使えない

shimux が必要とする以下が**全て欠けている**:
- pty (openpty, posix_spawn)
- termios (tcgetattr, tcsetattr, cfmakeraw)
- socket (socket, bind, listen, accept, connect)
- poll
- sigaction
- ioctl

その他の問題:
- C スタブ方式（poc3 は Rust staticlib）
- WIP 状態、macOS 対応が不完全
- errno 定数が Linux 固有値でハードコード
- writev の再帰呼び出しバグあり

### 参考になる点

- newtype パターン: `type Fd Int`, `type Pid Int` 等
- `#define` 定数のランタイム取得パターン
- `moon.pkg.json` の `native-stub` 設定
