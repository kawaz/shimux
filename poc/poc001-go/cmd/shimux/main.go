package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kawaz/shimux/internal/ghostty"
	"github.com/kawaz/shimux/internal/ghostty/keysim"
	"github.com/kawaz/shimux/internal/pane"
	"github.com/kawaz/shimux/internal/tmux"
	"github.com/kawaz/shimux/internal/wrapper"
)

var version = "dev"

func main() {
	exitCode := runMain(os.Args, os.Stdout, os.Stderr)
	os.Exit(exitCode)
}

// runMain はテスト可能なメインエントリーポイント。
// stdout/stderr を注入可能にすることでテスタビリティを確保。
func runMain(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "shimux: no arguments")
		return 1
	}

	// argv[0] でモード判定
	base := filepath.Base(args[0])

	switch base {
	case "tmux":
		return runTmuxMode(args[1:], stdout, stderr)
	case "shimux":
		return runShimuxMode(args[1:], stdout, stderr)
	default:
		// argv[0] が tmux でも shimux でもない場合
		// SHIMUX 環境変数がセットされていれば tmux モードとして扱う
		if os.Getenv("SHIMUX") == "1" {
			return runTmuxMode(args[1:], stdout, stderr)
		}
		return runShimuxMode(args[1:], stdout, stderr)
	}
}

func runTmuxMode(args []string, stdout, stderr io.Writer) int {
	// 引数なしの場合、Parse が "no command specified" エラーを返す
	if len(args) == 0 {
		fmt.Fprintln(stderr, "shimux: no command specified")
		return 1
	}

	// 1. パース
	result, err := tmux.Parse(args)
	if err != nil {
		fmt.Fprintf(stderr, "shimux: %v\n", err)
		return 1
	}

	// 2. -V (バージョン)
	if result.Global.Version {
		fmt.Fprintf(stdout, "shimux %s (tmux-compatible)\n", version)
		return 0
	}

	// 3. コマンド実行
	session := os.Getenv("SHIMUX_SESSION")
	stateDir := pane.DefaultStateDir()
	mgr := pane.NewWithDir(stateDir)

	// controller はkeysimを使用
	ctrl := keysim.New()

	ctx := &tmux.CommandContext{
		Controller:  ctrl,
		PaneManager: mgr,
		Session:     session,
		Stdout:      stdout,
		Stderr:      stderr,
	}

	exitCode, err := tmux.Execute(ctx, result)
	if err != nil {
		fmt.Fprintf(stderr, "shimux: %v\n", err)
	}
	return exitCode
}

func runShimuxMode(args []string, stdout, stderr io.Writer) int {
	// -V チェック（-- の前にある場合のみ）
	for _, arg := range args {
		if arg == "-V" {
			fmt.Fprintf(stdout, "shimux %s\n", version)
			return 0
		}
		if arg == "--" {
			break
		}
	}

	// -- の後ろを子プロセスコマンドとして取得
	var cmdArgs []string
	foundSep := false
	for _, arg := range args {
		if foundSep {
			cmdArgs = append(cmdArgs, arg)
		} else if arg == "--" {
			foundSep = true
		}
	}

	if len(cmdArgs) == 0 {
		fmt.Fprintln(stderr, "usage: shimux -- <command> [args...]")
		return 1
	}

	// ghostty 環境チェック
	if !ghostty.IsInsideGhostty() {
		fmt.Fprintln(stderr, "shimux: not running inside ghostty terminal")
		return 1
	}

	ctrl := keysim.New()
	stateDir := pane.DefaultStateDir()
	mgr := pane.NewWithDir(stateDir)

	w, err := wrapper.New(wrapper.Config{
		Controller:  ctrl,
		PaneManager: mgr,
	})
	if err != nil {
		fmt.Fprintf(stderr, "shimux: %v\n", err)
		return 1
	}

	ctx := context.Background()
	if err := w.Run(ctx, cmdArgs); err != nil {
		fmt.Fprintf(stderr, "shimux: %v\n", err)
		return 1
	}
	return 0
}
