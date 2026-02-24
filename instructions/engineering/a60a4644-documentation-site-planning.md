# ドキュメントサイト構築計画

## 指示
承認不要の完全オートモードで以下を行うこと。マルチペルソナでアイデアを出し合うこと。

## 概要
gmuxのユーザードキュメントとAPIリファレンスを公開するための静的サイトを計画する。

## ペルソナ

1. **テクニカルライター**: 情報の構造化と読みやすさ
2. **初心者ユーザー**: 「何もわからない」状態からの学習パス
3. **パワーユーザー**: リファレンスの網羅性と検索性
4. **OSS貢献者**: コントリビューションガイドの充実

## 検討事項

### サイトジェネレーター選定
- mdBook（Rust製、シンプル）
- Hugo（Go製、高速）
- VitePress（Vue.js、モダン）
- Docusaurus（React、機能豊富）
- 評価基準: Go projectとの親和性、メンテコスト、検索機能

### コンテンツ構成
- Getting Started（5分で動かす）
- Installation Guide（Homebrew, go install, binary）
- Configuration Guide（ghostty config設定）
- Command Reference（tmux互換コマンド一覧）
- Architecture Guide（開発者向け）
- Troubleshooting
- FAQ

### 自動化
- GoDocからのAPI reference自動生成
- コマンドヘルプからの usage 自動生成
- CHANGELOGの自動反映
- GitHub Pages / Cloudflare Pages でのデプロイ

## 期待する成果物
- サイト構成案（IA: Information Architecture）
- 各ページのアウトライン
- 推奨ジェネレーターとその理由
- 初期セットアップ手順
