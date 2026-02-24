# Homebrew Formula作成とパッケージング

## 指示
承認不要の完全オートモードで以下を行うこと。
この指示による新規ドキュメント作成は `docs/drafts/` 配下に限定。既存ファイル編集・コード新規作成は通常通り。

## 概要
gmuxをHomebrewで配布するためのFormulaを作成する。`brew install kawaz/tap/gmux` で一発インストールできるようにする。

## 作業内容

1. **Homebrew tap リポジトリの設計**
   - リポジトリ名: `homebrew-tap` (github.com/kawaz/homebrew-tap)
   - Formula命名: `gmux.rb`
   - 既存のtapがあるか確認

2. **Formula作成**
   - Go buildのFormula（`go_resource` or `bottle`）
   - 依存関係の宣言（Go 1.22+）
   - `system "go", "build", ...` でのビルド
   - `bin.install "gmux"` でのインストール
   - `test do ... end` ブロック
   - HEAD install対応（`head "https://github.com/kawaz/gmux.git"`）

3. **gmux-agentのインストール**
   - gmux-agentも同じFormulaに含める
   - `bin.install "gmux-agent"` でのインストール
   - PATHに両方が入ることの確認

4. **Ghostty設定ヘルパー**
   - `gmux setup` コマンドの検討
   - ghostty configへの `command = gmux-agent` 追記支援
   - `brew info gmux` のcaveatsセクションに設定手順表示

5. **GoReleaserとの統合**
   - `.goreleaser.yml` 作成
   - `brews` セクションでのtap自動更新
   - darwin/arm64, darwin/amd64 のクロスビルド
   - GitHub Releases + Homebrew bottleの自動化

## テスト
- `brew install --build-from-source gmux` でのビルド確認
- `brew test gmux` でのテスト確認
- `brew audit --strict gmux` でのFormula品質確認

## 期待する成果物
- `gmux.rb` Formula（`docs/drafts/gmux-formula.rb`）
- `.goreleaser.yml` の設計
- インストール手順ドキュメント
