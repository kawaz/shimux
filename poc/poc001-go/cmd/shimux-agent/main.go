package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/kawaz/shimux/internal/agent"
)

var version = "dev"

func main() {
	exitCode := run()
	os.Exit(exitCode)
}

func run() int {
	return runWithIO(os.Stdout, os.Stderr)
}

func runWithIO(stdout, stderr io.Writer) int {
	// 1. 環境変数取得
	paneID := os.Getenv("SHIMUX_PANE_ID")
	session := os.Getenv("SHIMUX_SESSION")

	if paneID == "" {
		fmt.Fprintln(stderr, "shimux-agent: SHIMUX_PANE_ID not set")
		fmt.Fprintln(stderr, "shimux-agent: this program should be launched by shimux")
		return 1
	}
	if session == "" {
		fmt.Fprintln(stderr, "shimux-agent: SHIMUX_SESSION not set")
		fmt.Fprintln(stderr, "shimux-agent: this program should be launched by shimux")
		return 1
	}

	// 2. ソケットパス生成
	socketPath := agent.SafeSocketPath(session, paneID)

	// 3. Agent作成
	a := agent.New(agent.Config{
		SocketPath: socketPath,
		PaneID:     paneID,
	})

	// 4. コンテキスト（SIGINT/SIGTERMでキャンセル）
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// 5. Agent起動
	if err := a.Run(ctx); err != nil {
		// シェル正常終了（exit 0）でもwait結果はnon-nilになりうる
		// exit status の場合はそのコードを返す
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(stderr, "shimux-agent: %v\n", err)
		return 1
	}
	return 0
}
