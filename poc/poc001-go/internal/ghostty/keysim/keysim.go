// Package keysim は macOS System Events を使って Ghostty のキーバインドをシミュレートする。
package keysim

import (
	"fmt"
	"os/exec"

	"github.com/kawaz/shimux/internal/ghostty"
)

// keystroke はosascriptに渡すキーストローク定義。
type keystroke struct {
	key       string
	modifiers string
}

// キーバインディングマップ（固定値、osascriptインジェクション防止）
var (
	keyNewWindow = keystroke{key: "n", modifiers: "{command down}"}
	keyNewTab    = keystroke{key: "t", modifiers: "{command down}"}
	keyClose     = keystroke{key: "w", modifiers: "{command down}"}

	// Design rationale: SplitRight = horizontal split = super+d,
	// SplitDown = vertical split = super+shift+d (tmux互換デフォルト方向)
	keySplit = map[ghostty.SplitDirection]keystroke{
		ghostty.SplitRight: {key: "d", modifiers: "{command down}"},
		ghostty.SplitDown:  {key: "D", modifiers: "{command down, shift down}"},
	}

	keyGoto = map[ghostty.GotoDirection]keystroke{
		ghostty.GotoNext:     {key: "]", modifiers: "{command down}"},
		ghostty.GotoPrevious: {key: "[", modifiers: "{command down}"},
	}
)

// buildScript はosascriptコマンド文字列を組み立てる。
func buildScript(ks keystroke) string {
	return fmt.Sprintf(`tell application "System Events" to tell process "ghostty" to keystroke "%s" using %s`, ks.key, ks.modifiers)
}

// KeySimController は System Events 経由で Ghostty を操作する Controller 実装。
type KeySimController struct {
	// osascript実行関数（テスト用に差し替え可能）
	execScript func(script string) error
}

// New は KeySimController を作成する。
func New() *KeySimController {
	return &KeySimController{
		execScript: defaultExecScript,
	}
}

// NewWithExecutor はカスタム実行関数を使う KeySimController を作成する（テスト用）。
func NewWithExecutor(exec func(script string) error) *KeySimController {
	return &KeySimController{
		execScript: exec,
	}
}

func (k *KeySimController) NewWindow() error {
	return k.execScript(buildScript(keyNewWindow))
}

func (k *KeySimController) NewTab() error {
	return k.execScript(buildScript(keyNewTab))
}

func (k *KeySimController) NewSplit(direction ghostty.SplitDirection) error {
	ks, ok := keySplit[direction]
	if !ok {
		return fmt.Errorf("unsupported split direction: %d", direction)
	}
	return k.execScript(buildScript(ks))
}

func (k *KeySimController) GotoSplit(direction ghostty.GotoDirection) error {
	ks, ok := keyGoto[direction]
	if !ok {
		return fmt.Errorf("unsupported goto direction: %d", direction)
	}
	return k.execScript(buildScript(ks))
}

func (k *KeySimController) CloseSurface() error {
	return k.execScript(buildScript(keyClose))
}

func defaultExecScript(script string) error {
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript failed: %w: %s", err, string(output))
	}
	return nil
}
