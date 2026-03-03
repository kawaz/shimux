package tmux

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/kawaz/shimux/internal/ghostty/ghosttytest"
	"github.com/kawaz/shimux/internal/pane"
)

// newBenchContext creates a CommandContext for benchmarking with mock controller.
func newBenchContext(session string) *CommandContext {
	mock := &ghosttytest.MockController{}
	pm := pane.NewManager(filepath.Join(os.TempDir(), "shimux-bench-panes.json"))
	return &CommandContext{
		Controller:  mock,
		PaneManager: pm,
		Session:     session,
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
	}
}

// resetBenchContext resets the stdout/stderr buffers for reuse.
func resetBenchContext(ctx *CommandContext) {
	ctx.Stdout.(*bytes.Buffer).Reset()
	ctx.Stderr.(*bytes.Buffer).Reset()
}

// --- Execute dispatch benchmarks ---

func BenchmarkExecute_HasSession(b *testing.B) {
	ctx := newBenchContext("test-session")
	result := &ParseResult{
		Command: "has-session",
		Args:    []string{"-t", "test-session"},
	}
	for b.Loop() {
		resetBenchContext(ctx)
		Execute(ctx, result)
	}
}

func BenchmarkExecute_DisplayMessage(b *testing.B) {
	ctx := newBenchContext("test-session")
	ctx.PaneManager.Register("")
	result := &ParseResult{
		Command: "display-message",
		Args:    []string{"-p", "#{session_name}:#{pane_id}"},
	}
	for b.Loop() {
		resetBenchContext(ctx)
		Execute(ctx, result)
	}
}

func BenchmarkExecute_ListPanes(b *testing.B) {
	ctx := newBenchContext("test-session")
	// Pre-register panes
	for i := 0; i < 10; i++ {
		ctx.PaneManager.Register("")
	}
	result := &ParseResult{
		Command: "list-panes",
		Args:    []string{"-F", "#{pane_id}:#{pane_active}"},
	}
	for b.Loop() {
		resetBenchContext(ctx)
		Execute(ctx, result)
	}
}

func BenchmarkExecute_ShowOptions(b *testing.B) {
	ctx := newBenchContext("test-session")
	result := &ParseResult{
		Command: "show-options",
		Args:    []string{"-g", "prefix"},
	}
	for b.Loop() {
		resetBenchContext(ctx)
		Execute(ctx, result)
	}
}

func BenchmarkExecute_SplitWindow(b *testing.B) {
	ctx := newBenchContext("test-session")
	result := &ParseResult{
		Command: "split-window",
		Args:    []string{"-h"},
	}
	for b.Loop() {
		resetBenchContext(ctx)
		Execute(ctx, result)
	}
}

func BenchmarkExecute_SelectPane(b *testing.B) {
	ctx := newBenchContext("test-session")
	result := &ParseResult{
		Command: "select-pane",
		Args:    []string{"-t", "%0"},
	}
	for b.Loop() {
		resetBenchContext(ctx)
		Execute(ctx, result)
	}
}

func BenchmarkExecute_Unsupported(b *testing.B) {
	ctx := newBenchContext("test-session")
	result := &ParseResult{
		Command: "resize-pane",
	}
	for b.Loop() {
		resetBenchContext(ctx)
		Execute(ctx, result)
	}
}

// --- extractPaneID benchmarks ---

func BenchmarkExtractPaneID_Simple(b *testing.B) {
	for b.Loop() {
		extractPaneID("%0")
	}
}

func BenchmarkExtractPaneID_SessionWindowPane(b *testing.B) {
	for b.Loop() {
		extractPaneID("session:0.%5")
	}
}

func BenchmarkExtractPaneID_Empty(b *testing.B) {
	for b.Loop() {
		extractPaneID("")
	}
}
