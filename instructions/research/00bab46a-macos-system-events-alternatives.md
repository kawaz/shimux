# macOS System Events以外のキーシミュレーション手法の調査

## 指示
承認不要の完全オートモードで以下を行うこと。高度な調査タスク。
この指示による新規ドキュメント作成は `docs/drafts/` 配下に限定。既存ファイル編集・コード新規作成は通常通り。

## 概要
gmuxは現在System Events（osascript）経由でキーシミュレーションを行っているが、他にmacOSで利用可能な方法がないか調査する。レイテンシ改善やAccessibility権限回避の可能性を探る。

## 調査対象

1. **CGEvent (Core Graphics Events)**
   - `CGEventCreateKeyboardEvent` によるキーイベント生成
   - Accessibility権限は必要？（System Events vs CGEvent の違い）
   - Goからの呼び出し方法（cgo or syscall）
   - レイテンシの比較（osascript起動コスト vs CGEvent直接呼び出し）

2. **Accessibility API (AXUIElement)**
   - `AXUIElementPerformAction` でのアクション実行
   - Ghosttyウィンドウ/ペインへのUI要素アクセス
   - メニューアイテムの直接呼び出し
   - Goからの呼び出し方法

3. **NSEvent (AppKit)**
   - `NSEvent.keyEvent` によるキーイベント投入
   - 対象アプリへの送付方法
   - cgoでのAppKit呼び出しの実績

4. **hidutil / IOHIDManager**
   - HID (Human Interface Device) レベルでのキー入力
   - ルート権限の要否
   - macOS Venturaでの制限

5. **AppleScript vs JXA (JavaScript for Automation)**
   - osascript -l JavaScript の方が速い？
   - System Eventsの直接呼び出しとの比較
   - エラーハンドリングの差異

## 評価基準
- レイテンシ（目標: 現状のosascriptより速い）
- 権限要件（Accessibility権限は避けたい？それとも許容？）
- Goとの親和性（cgoの有無）
- macOSバージョン互換性
- 安定性・将来性

## 期待する成果物
- 各手法の比較表（`docs/drafts/keysim-alternatives.md`）
- 推奨手法とその理由
- PoC（最も有望な手法のみ）
