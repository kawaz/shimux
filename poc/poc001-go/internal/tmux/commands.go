package tmux

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/kawaz/shimux/internal/agent"
	"github.com/kawaz/shimux/internal/ghostty"
	"github.com/kawaz/shimux/internal/pane"
)

// CommandContext はコマンド実行に必要な依存関係。
type CommandContext struct {
	Controller  ghostty.Controller
	PaneManager *pane.Manager
	Session     string // セッション名
	Stdout      io.Writer
	Stderr      io.Writer
}

// Execute はパーサ結果に基づいてコマンドを実行する。
// 戻り値は (終了コード, エラー)。
func Execute(ctx *CommandContext, result *ParseResult) (int, error) {
	switch result.Command {
	case "has-session":
		opts, err := ParseHasSession(result.Args)
		if err != nil {
			return 1, err
		}
		return handleHasSession(ctx, opts)
	case "switch-client":
		return handleSwitchClient(ctx)
	case "new-session":
		return handleNewSession(ctx)
	case "show-options":
		opts, err := ParseShowOptions(result.Args)
		if err != nil {
			return 1, err
		}
		return handleShowOptions(ctx, opts)
	case "split-window":
		opts, err := ParseSplitWindow(result.Args)
		if err != nil {
			return 1, err
		}
		return handleSplitWindow(ctx, opts)
	case "send-keys":
		opts, err := ParseSendKeys(result.Args)
		if err != nil {
			return 1, err
		}
		return handleSendKeys(ctx, opts)
	case "select-pane":
		opts, err := ParseSelectPane(result.Args)
		if err != nil {
			return 1, err
		}
		return handleSelectPane(ctx, opts)
	case "kill-pane":
		opts, err := ParseKillPane(result.Args)
		if err != nil {
			return 1, err
		}
		return handleKillPane(ctx, opts)
	case "list-panes":
		opts, err := ParseListPanes(result.Args)
		if err != nil {
			return 1, err
		}
		return handleListPanes(ctx, opts)
	case "display-message":
		opts, err := ParseDisplayMessage(result.Args)
		if err != nil {
			return 1, err
		}
		return handleDisplayMessage(ctx, opts)
	default:
		return handleUnsupported(ctx, result.Command)
	}
}

// handleHasSession はセッション名一致チェック。
func handleHasSession(ctx *CommandContext, opts *HasSessionOptions) (int, error) {
	if opts.Target == ctx.Session {
		return 0, nil
	}
	return 1, nil
}

// handleSwitchClient は no-op。
func handleSwitchClient(_ *CommandContext) (int, error) {
	return 0, nil
}

// handleNewSession はセッション名更新はスコープ外のため、no-op + exit 0。
func handleNewSession(_ *CommandContext) (int, error) {
	return 0, nil
}

// handleShowOptions は prefix のみ対応。
func handleShowOptions(ctx *CommandContext, opts *ShowOptionsOptions) (int, error) {
	if opts.Global && opts.Option == "prefix" {
		fmt.Fprintln(ctx.Stdout, "prefix C-b")
	}
	return 0, nil
}

// handleSplitWindow は ghostty でペイン分割。
func handleSplitWindow(ctx *CommandContext, opts *SplitWindowOptions) (int, error) {
	// 1. 分割方向決定
	var direction ghostty.SplitDirection
	switch opts.Direction {
	case "right":
		direction = ghostty.SplitRight
	case "left":
		direction = ghostty.SplitLeft
	case "up":
		direction = ghostty.SplitUp
	default: // "down" or unspecified
		direction = ghostty.SplitDown
	}

	// 2. controller.NewSplit
	if err := ctx.Controller.NewSplit(direction); err != nil {
		return 1, fmt.Errorf("split-window: %w", err)
	}

	// 3. ペイン登録
	p, err := ctx.PaneManager.Register("")
	if err != nil {
		return 1, fmt.Errorf("split-window: register pane: %w", err)
	}

	// 4. -P フラグがある場合はフォーマット出力
	if opts.PrintAfter {
		format := opts.Format
		if format == "" {
			format = "#{pane_id}" // デフォルトフォーマット
		}
		fctx := buildFormatContext(ctx, p)
		expanded := ExpandFormat(format, fctx)
		fmt.Fprintln(ctx.Stdout, expanded)
	}

	return 0, nil
}

// handleSendKeys はソケット経由で PTY に書き込み。
func handleSendKeys(ctx *CommandContext, opts *SendKeysOptions) (int, error) {
	// 1. テキスト構築
	text := BuildSendKeysData(opts.Keys, opts.Literal)

	// 2. データが空の場合: no-op + exit 0
	if text == "" {
		return 0, nil
	}

	// 3. ターゲットペインID取得
	paneID := extractPaneID(opts.Target)
	if paneID == "" {
		// デフォルト: アクティブペイン
		active := ctx.PaneManager.GetActive()
		if active != nil {
			paneID = active.ID
		}
	}
	if paneID == "" {
		return 1, fmt.Errorf("send-keys: no target pane specified")
	}

	// 4. ソケットパス
	socketPath := agent.SafeSocketPath(ctx.Session, paneID)

	// 5. agent.SendKeys
	if err := agent.SendKeys(socketPath, text); err != nil {
		return 1, fmt.Errorf("send-keys: %w", err)
	}

	return 0, nil
}

// handleSelectPane はフォーカス移動。
// Design rationale: goto_split:next/previousの巡回順序がペイン作成順と
// 一致するか未検証のため、Phase 1ではベストエフォートで no-op。
func handleSelectPane(_ *CommandContext, _ *SelectPaneOptions) (int, error) {
	return 0, nil
}

// handleKillPane はペイン削除。
func handleKillPane(ctx *CommandContext, opts *KillPaneOptions) (int, error) {
	// 1. controller.CloseSurface()
	if err := ctx.Controller.CloseSurface(); err != nil {
		return 1, fmt.Errorf("kill-pane: %w", err)
	}

	// 2. paneManager.Unregister(paneID)
	paneID := extractPaneID(opts.Target)
	if paneID != "" {
		if err := ctx.PaneManager.Unregister(paneID); err != nil {
			return 1, fmt.Errorf("kill-pane: unregister: %w", err)
		}
	}

	return 0, nil
}

// handleListPanes はペイン一覧を出力。
func handleListPanes(ctx *CommandContext, opts *ListPanesOptions) (int, error) {
	panes := ctx.PaneManager.List()
	if len(panes) == 0 {
		return 0, nil
	}

	format := opts.Format
	if format == "" {
		format = "#{pane_id}" // デフォルトフォーマット
	}

	for _, p := range panes {
		fctx := buildFormatContext(ctx, p)
		expanded := ExpandFormat(format, fctx)
		fmt.Fprintln(ctx.Stdout, expanded)
	}

	return 0, nil
}

// handleDisplayMessage はフォーマット変数展開して出力。
func handleDisplayMessage(ctx *CommandContext, opts *DisplayMessageOptions) (int, error) {
	if !opts.Print {
		// -p なしの場合はステータスラインに表示するが、shimuxにはステータスラインがないので no-op。
		return 0, nil
	}

	// アクティブペインの情報を使ってフォーマット展開
	active := ctx.PaneManager.GetActive()
	var fctx *FormatContext
	if active != nil {
		fctx = buildFormatContext(ctx, active)
	} else {
		fctx = &FormatContext{
			SessionName: ctx.Session,
		}
	}

	expanded := ExpandFormat(opts.Format, fctx)
	fmt.Fprintln(ctx.Stdout, expanded)

	return 0, nil
}

// handleUnsupported は未対応コマンドの警告。
func handleUnsupported(ctx *CommandContext, cmd string) (int, error) {
	fmt.Fprintf(ctx.Stderr, "shimux: unsupported tmux command: %s\n", cmd)
	return 0, nil
}

// extractPaneID はターゲット文字列からペインIDを抽出する。
// "%N" -> "N"
// "session:window.%N" -> "N"
// "" -> "" (デフォルト=現在のペイン)
func extractPaneID(target string) string {
	if target == "" {
		return ""
	}

	// %N パターンを探す
	idx := strings.LastIndex(target, "%")
	if idx < 0 {
		return ""
	}

	// % の後の数字部分を取得
	rest := target[idx+1:]
	if rest == "" {
		return ""
	}

	// 数字のみを抽出
	var b strings.Builder
	for _, ch := range rest {
		if ch >= '0' && ch <= '9' {
			b.WriteRune(ch)
		} else {
			break
		}
	}

	return b.String()
}

// buildFormatContext はペイン情報から FormatContext を構築する。
func buildFormatContext(ctx *CommandContext, p *pane.Pane) *FormatContext {
	idx := ctx.PaneManager.IndexOf(p.ID)
	paneIndex := "0"
	if idx >= 0 {
		paneIndex = strconv.Itoa(idx)
	}
	return &FormatContext{
		SessionName: ctx.Session,
		SessionID:   "$0",
		WindowIndex: "0",
		WindowID:    "@0",
		WindowName:  ctx.Session,
		PaneID:      "%" + p.ID,
		PaneIndex:   paneIndex,
		PanePID:     p.PID,
		PaneTTY:     p.TTY,
		PanePath:    p.CurrentPath,
		PaneWidth:   p.Width,
		PaneHeight:  p.Height,
		PaneActive:  p.Active,
	}
}
