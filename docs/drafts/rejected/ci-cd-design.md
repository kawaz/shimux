# CI/CD 設計書

## パイプライン構成

```
                          push (main) / PR
                               |
                         .github/workflows/ci.yml
                               |
                    +----------+----------+
                    |    macos-latest      |
                    |                      |
                    |  1. checkout         |
                    |  2. setup-go         |
                    |  3. go build ./...   |
                    |  4. go vet ./...     |
                    |  5. golangci-lint    |
                    |  6. go test -race    |
                    |  7. coverage         |
                    +---------------------+


                          push tag (v*)
                               |
                     .github/workflows/release.yml
                               |
                    +----------+----------+
                    |    macos-latest      |
                    |                      |
                    |  1. checkout         |
                    |  2. setup-go         |
                    |  3. goreleaser       |
                    |     - gmux           |
                    |     - gmux-agent     |
                    |     darwin/amd64     |
                    |     darwin/arm64     |
                    +---------------------+
                               |
                        GitHub Release
                    (バイナリ + チェックサム)
```

## CI ワークフロー (`ci.yml`)

### トリガー

- `push` to `main`
- `pull_request` to `main`

### ステップ

| # | ステップ | 目的 |
|---|---------|------|
| 1 | `go build ./...` | コンパイルエラーの検出 |
| 2 | `go vet ./...` | 静的解析（標準ツール） |
| 3 | `golangci-lint` | 追加の静的解析（errcheck, staticcheck等） |
| 4 | `go test -race -count=1 ./...` | テスト実行 + データ競合検出 |
| 5 | `go test -cover -coverprofile=coverage.out ./...` | カバレッジ計測 |

### Runner

`macos-latest` を使用。gmuxはmacOS専用プロジェクトのため、Linux runnerでは不十分。

## Release ワークフロー (`release.yml`)

### トリガー

- `push` tag matching `v*`

### GoReleaser 設定

- **バイナリ**: `gmux`（`cmd/gmux`）, `gmux-agent`（`cmd/gmux-agent`）
- **ターゲット**: `darwin/amd64`, `darwin/arm64`
- **ldflags**: `-s -w -X main.version={{.Version}}`
- **アーカイブ**: tar.gz、両バイナリを同梱
- **チェックサム**: `checksums.txt`

## golangci-lint 設定 (`.golangci.yml`)

最小限のlinter構成:

- `errcheck`: エラー戻り値の未チェック検出
- `gosimple`: コード簡素化の提案
- `govet`: `go vet` と同等
- `ineffassign`: 未使用の代入検出
- `staticcheck`: 高度な静的解析
- `unused`: 未使用コード検出

テストファイル (`_test.go`) では `errcheck` を緩和。

## macOS 固有の考慮事項

### CI環境でのテスト互換性

全テストがCI互換であることを確認済み:

- **keysim テスト**: `mockExecutor` を使用。実際の `osascript` は呼ばない
- **controller テスト**: `MockController` を使用。System Events不要
- **detect テスト**: 環境変数のみ使用。OS依存なし
- **agent テスト**: `MockPTYOpener` を使用。パイプでPTYを模擬
- **integration テスト**: 全てモック/パイプベース

Accessibility権限やGhosttyプロセスの存在を前提とするテストは現時点で存在しない。

### macOS Runner のコスト

GitHub Actions の macOS runner は Linux runner の約10倍のコスト。対策:

- CIジョブは1つに集約（並列化しない）
- 不要なステップの重複を避ける
- キャッシュは `actions/setup-go` が自動管理

## リリースフロー

```
1. git tag v0.1.0
2. git push origin v0.1.0
3. GitHub Actions が release.yml を実行
4. GoReleaser がバイナリをビルド
5. GitHub Release が自動作成
   - gmux_0.1.0_darwin_amd64.tar.gz
   - gmux_0.1.0_darwin_arm64.tar.gz
   - checksums.txt
```

## 将来の拡張

### Codecov 連携

カバレッジレポートをCodecovに送信してPRにカバレッジ変動を表示:

```yaml
- uses: codecov/codecov-action@v4
  with:
    files: coverage.out
    token: ${{ secrets.CODECOV_TOKEN }}
```

### Dependabot

依存関係の自動更新:

```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: gomod
    directory: /
    schedule:
      interval: weekly
  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
```

### Homebrew Formula

GoReleaser の `brews` セクションで Homebrew tap への自動公開:

```yaml
brews:
  - repository:
      owner: kawaz
      name: homebrew-tap
    homepage: https://github.com/kawaz/gmux
    description: tmux-compatible CLI wrapper for Ghostty terminal
```

### pre-commit / commit lint

コミットメッセージの規約チェックやコードフォーマットの自動チェック。
