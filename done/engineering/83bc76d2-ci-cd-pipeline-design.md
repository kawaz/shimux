# CI/CDパイプライン設計

## 指示
承認不要の完全オートモードで以下を行うこと。

## 概要
gmuxプロジェクトのCI/CDパイプラインを設計する。GitHub Actionsを使用し、macOS固有の制約を考慮する。

## 設計内容

1. **CIパイプライン（PR時）**
   - `go build ./...` - ビルド確認
   - `go test ./...` - ユニットテスト
   - `go test -race ./...` - race condition 検出
   - `go vet ./...` - 静的解析
   - `golangci-lint run` - lintチェック
   - カバレッジレポート（Codecov連携）

2. **macOS固有の考慮事項**
   - System Events権限がCI環境で使えない
   - `go test -tags=integration` でのゲート
   - `GOOS=darwin` でのクロスコンパイル
   - macOS runnerのコスト最適化

3. **CDパイプライン（リリース時）**
   - GoReleaser設定ファイル
   - バイナリのクロスコンパイル（darwin/arm64, darwin/amd64）
   - Homebrew tap 自動更新
   - GitHub Release 自動作成
   - CHANGELOG自動生成

4. **Dependabot / Renovate**
   - Go module の自動更新設定
   - GitHub Actions のバージョン固定と自動更新

## 期待する成果物
- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `.goreleaser.yml`
- `.golangci.yml`
- `CONTRIBUTING.md`（ビルド・テスト手順）
