// Package wrapper provides the tmux impersonation wrapper for shimux.
//
// The Wrapper launches a child process (e.g., Claude Code) with PATH modified
// to include a tmux→shimux symlink, making the child believe tmux is available.
package wrapper

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kawaz/shimux/internal/ghostty"
	"github.com/kawaz/shimux/internal/pane"
)

// Wrapper はtmux偽装ラッパー。
type Wrapper struct {
	controller  ghostty.Controller
	paneManager *pane.Manager
	session     string // セッション名
	tmpDir      string // 一時ディレクトリ（tmux symlink置き場）
	shimuxPath    string // shimux実行ファイルのパス
}

// Config はWrapper設定。
type Config struct {
	Controller  ghostty.Controller // 必須
	PaneManager *pane.Manager      // 必須
	Session     string             // 空→自動生成
	ShimuxPath    string             // 空→os.Executable()
}

// New はWrapperを作成する。
func New(cfg Config) (*Wrapper, error) {
	if cfg.Controller == nil {
		return nil, fmt.Errorf("controller is required")
	}
	if cfg.PaneManager == nil {
		return nil, fmt.Errorf("pane manager is required")
	}

	session := cfg.Session
	if session == "" {
		session = fmt.Sprintf("shimux-%d", os.Getpid())
	}
	session = GenerateSessionName(session)

	shimuxPath := cfg.ShimuxPath
	if shimuxPath == "" {
		exe, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("resolve executable path: %w", err)
		}
		shimuxPath = exe
	}

	return &Wrapper{
		controller:  cfg.Controller,
		paneManager: cfg.PaneManager,
		session:     session,
		shimuxPath:    shimuxPath,
	}, nil
}

// Setup は一時ディレクトリとシンボリックリンクを作成する。
func (w *Wrapper) Setup() error {
	tmpDir, err := os.MkdirTemp("", "shimux-wrapper-")
	if err != nil {
		return err
	}
	w.tmpDir = tmpDir

	if err := os.Symlink(w.shimuxPath, filepath.Join(tmpDir, "tmux")); err != nil {
		return err
	}

	return nil
}

// Env は子プロセスに設定する環境変数を返す。
func (w *Wrapper) Env() []string {
	return []string{
		fmt.Sprintf("PATH=%s:%s", w.tmpDir, os.Getenv("PATH")),
		fmt.Sprintf("TMUX=%s/shimux,%d,0", w.tmpDir, os.Getpid()),
		"SHIMUX=1",
		fmt.Sprintf("SHIMUX_SESSION=%s", w.session),
		"SHIMUX_PANE_ID=0",
		"TMUX_PANE=%0",
	}
}

// Run は子プロセスを起動して完了を待つ。
func (w *Wrapper) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	// Setup
	if err := w.Setup(); err != nil {
		return fmt.Errorf("setup: %w", err)
	}
	defer w.Cleanup()

	// 初期ペイン登録
	if _, err := w.paneManager.Register(""); err != nil {
		return fmt.Errorf("register initial pane: %w", err)
	}

	// 子プロセス起動
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 環境変数設定
	env := os.Environ()
	for _, e := range w.Env() {
		key := strings.SplitN(e, "=", 2)[0]
		env = replaceEnv(env, key, e)
	}
	cmd.Env = env

	return cmd.Run()
}

// Cleanup は一時ディレクトリを削除する。
func (w *Wrapper) Cleanup() error {
	if w.tmpDir != "" {
		return os.RemoveAll(w.tmpDir)
	}
	return nil
}

// GenerateSessionName はセッション名を生成する。
// 規則: [^a-zA-Z0-9-] → _ に置換
func GenerateSessionName(name string) string {
	var buf strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			buf.WriteRune(r)
		} else {
			buf.WriteByte('_')
		}
	}
	return buf.String()
}

// replaceEnv は環境変数スライス内の指定キーの値を置換する。
// 既存のキーがあれば上書き、なければ追加。
func replaceEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = value
			return env
		}
	}
	return append(env, value)
}
