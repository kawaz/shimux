# ビルドの注意事項

## moon build --target native の制限

パッケージ未指定の `moon build --target native` は使えない。
lib パッケージにテスト用の `cc-link-flags` があると `_main` 未定義エラーになる。

メインパッケージを明示的に指定すること:

```bash
moon build cmd/shimux --target native
```

## Rust FFI ビルド

MoonBit ビルドの前に Rust FFI をビルドする:

```bash
(cd ffi && cargo build --release)
moon build cmd/shimux --target native
```

## extern "C" の利用

MoonBit native では `extern "C"` で C 関数を呼べる。
osascript の実行には libc の popen/pclose または posix_spawn を使う。

## ビルドコマンド

`just` で統合管理。個別コマンドを手打ちしない:

```bash
just        # check + test
just build  # FFI + native ビルド
just clean  # 全成果物クリーン
```
