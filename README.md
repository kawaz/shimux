# shimux

Ghostty 上で Claude Code Agent Teams の tmux モードを動作させる tmux 互換 CLI ラッパー。

Ghostty の AppleScript API を使い、tmux コマンドを Ghostty のペイン操作に変換する。

## Status

開発初期段階

## Structure

```
shim/
  tmux/             # tmux CLI パース → core API 呼び出し
terminal/
  ghostty/          # Ghostty AppleScript バックエンド
core/               # 抽象 API（Split, SendKeys, Focus 等）
cmd/
  shimux/           # エントリーポイント
```

## License

MIT License
