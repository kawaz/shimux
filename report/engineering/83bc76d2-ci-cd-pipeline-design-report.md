# CI/CD Pipeline Design Report

## Task: 83bc76d2-ci-cd-pipeline-design

## Summary

GitHub Actions CI/CDパイプラインを構築。CI（PR/push時）、Release（タグ時）、GoReleaser、golangci-lint設定を作成。

## Created Files

| File | Description |
|------|-------------|
| `.github/workflows/ci.yml` | build, vet, lint, test(-race), coverage |
| `.github/workflows/release.yml` | GoReleaser v2 リリース自動化 |
| `.goreleaser.yml` | gmux + gmux-agent, darwin arm64/amd64 |
| `.golangci.yml` | errcheck, staticcheck等の最小設定 |
| `docs/drafts/ci-cd-design.md` | CI/CD設計書 |

## CI Compatibility

全テストがCI環境で問題なく実行可能（keysim: mock化済み、agent: MockPTYOpener使用）。ビルドタグ追加不要。

## Commit

```
fb62839 ci: GitHub Actions CI/CD パイプライン構築
```
