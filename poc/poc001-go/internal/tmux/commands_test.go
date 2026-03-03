package tmux

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/kawaz/shimux/internal/agent"
	"github.com/kawaz/shimux/internal/ghostty"
	"github.com/kawaz/shimux/internal/ghostty/ghosttytest"
	"github.com/kawaz/shimux/internal/pane"
)

// helper to create a CommandContext with mock controller and fresh pane manager.
func newTestContext(session string) (*CommandContext, *ghosttytest.MockController) {
	mock := &ghosttytest.MockController{}
	pm := pane.NewManager(filepath.Join(os.TempDir(), "shimux-test-panes.json"))
	return &CommandContext{
		Controller:  mock,
		PaneManager: pm,
		Session:     session,
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
	}, mock
}

func stdout(ctx *CommandContext) string {
	return ctx.Stdout.(*bytes.Buffer).String()
}

func stderr(ctx *CommandContext) string {
	return ctx.Stderr.(*bytes.Buffer).String()
}

// --- TC-CMD-001: Execute dispatch ---

func TestCommandExecuteDispatch(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantErr bool
	}{
		{"has-session", "has-session", false},
		{"switch-client", "switch-client", false},
		{"new-session", "new-session", false},
		{"show-options", "show-options", false},
		{"split-window", "split-window", false},
		{"send-keys", "send-keys", false},
		{"select-pane", "select-pane", false},
		{"kill-pane", "kill-pane", false},
		{"list-panes", "list-panes", false},
		{"display-message", "display-message", false},
		{"unsupported", "resize-pane", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := newTestContext("test-session")
			result := &ParseResult{Command: tt.command}
			exitCode, err := Execute(ctx, result)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// All commands should succeed (exit 0) in basic dispatch
			// except has-session which defaults to exit 1 (no matching session with empty target)
			if tt.command == "has-session" {
				// Empty target doesn't match "test-session"
				if exitCode != 1 {
					t.Errorf("exit code = %d, want 1 for has-session with no matching target", exitCode)
				}
			} else {
				if exitCode != 0 {
					t.Errorf("exit code = %d, want 0", exitCode)
				}
			}
		})
	}
}

// --- TC-CMD-002: has-session ---

func TestCommandHasSession(t *testing.T) {
	tests := []struct {
		name     string
		session  string
		target   string
		wantExit int
	}{
		{"match", "my-session", "my-session", 0},
		{"no match", "my-session", "other-session", 1},
		{"empty target", "my-session", "", 1},
		{"sanitized match", "my_session", "my_session", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := newTestContext(tt.session)
			result := &ParseResult{
				Command: "has-session",
				Args:    []string{"-t", tt.target},
			}
			if tt.target == "" {
				result.Args = nil
			}
			exitCode, err := Execute(ctx, result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if exitCode != tt.wantExit {
				t.Errorf("exit code = %d, want %d", exitCode, tt.wantExit)
			}
		})
	}
}

// --- TC-CMD-003: split-window ---

func TestCommandSplitWindow(t *testing.T) {
	t.Run("default direction is SplitDown", func(t *testing.T) {
		ctx, mock := newTestContext("test-session")
		result := &ParseResult{
			Command: "split-window",
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		if len(mock.Calls) == 0 {
			t.Fatal("expected NewSplit call")
		}
		call := mock.Calls[0]
		if call.Method != "NewSplit" {
			t.Errorf("method = %q, want NewSplit", call.Method)
		}
		if call.Args[0] != ghostty.SplitDown {
			t.Errorf("direction = %v, want SplitDown", call.Args[0])
		}
	})

	t.Run("-h produces SplitRight", func(t *testing.T) {
		ctx, mock := newTestContext("test-session")
		result := &ParseResult{
			Command: "split-window",
			Args:    []string{"-h"},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		call := mock.Calls[0]
		if call.Args[0] != ghostty.SplitRight {
			t.Errorf("direction = %v, want SplitRight", call.Args[0])
		}
	})

	t.Run("registers pane after split", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		result := &ParseResult{
			Command: "split-window",
		}
		_, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		panes := ctx.PaneManager.List()
		if len(panes) != 1 {
			t.Fatalf("pane count = %d, want 1", len(panes))
		}
	})

	t.Run("-P -F outputs formatted pane info", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		result := &ParseResult{
			Command: "split-window",
			Args:    []string{"-P", "-F", "#{pane_id}"},
		}
		_, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := stdout(ctx)
		// Should contain the pane ID prefixed with %
		if !strings.Contains(out, "%") {
			t.Errorf("stdout = %q, want pane ID with %% prefix", out)
		}
	})

	t.Run("-P without -F uses default format", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		result := &ParseResult{
			Command: "split-window",
			Args:    []string{"-P"},
		}
		_, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out := stdout(ctx)
		// Default format is #{pane_id}, should output %N
		if !strings.Contains(out, "%") {
			t.Errorf("stdout = %q, want pane ID with %% prefix", out)
		}
	})

	t.Run("split error returns error", func(t *testing.T) {
		ctx, mock := newTestContext("test-session")
		mock.SplitErr = os.ErrPermission
		result := &ParseResult{
			Command: "split-window",
		}
		exitCode, err := Execute(ctx, result)
		if err == nil {
			t.Fatal("expected error from split failure")
		}
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("direction flags", func(t *testing.T) {
		tests := []struct {
			name    string
			args    []string
			wantDir ghostty.SplitDirection
		}{
			{"--left", []string{"--left"}, ghostty.SplitLeft},
			{"--right", []string{"--right"}, ghostty.SplitRight},
			{"--up", []string{"--up"}, ghostty.SplitUp},
			{"--down", []string{"--down"}, ghostty.SplitDown},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx, mock := newTestContext("test-session")
				result := &ParseResult{
					Command: "split-window",
					Args:    tt.args,
				}
				_, err := Execute(ctx, result)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				call := mock.Calls[0]
				if call.Args[0] != tt.wantDir {
					t.Errorf("direction = %v, want %v", call.Args[0], tt.wantDir)
				}
			})
		}
	})
}

// --- TC-CMD-004: send-keys ---

func TestCommandSendKeys(t *testing.T) {
	t.Run("sends text via socket", func(t *testing.T) {
		session := "test-sendkeys"
		paneID := "0"
		socketPath := agent.SafeSocketPath(session, paneID)

		// Clean up socket dir
		socketDir := filepath.Dir(socketPath)
		os.MkdirAll(socketDir, 0700)
		defer os.RemoveAll(socketDir)

		// Start mock socket server
		var mu sync.Mutex
		var received string
		srv, err := agent.ListenAndServe(socketPath, func(data string) error {
			mu.Lock()
			defer mu.Unlock()
			received = data
			return nil
		})
		if err != nil {
			t.Fatalf("failed to start server: %v", err)
		}
		defer srv.Close()

		ctx, _ := newTestContext(session)
		// Register a pane so we have a valid target
		ctx.PaneManager.Register("")

		result := &ParseResult{
			Command: "send-keys",
			Args:    []string{"-t", "%" + paneID, "hello", "Enter"},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		mu.Lock()
		got := received
		mu.Unlock()
		// "hello" + Enter (expanded to \r)
		want := "hello\r"
		if got != want {
			t.Errorf("received = %q, want %q", got, want)
		}
	})

	t.Run("literal mode sends without expansion", func(t *testing.T) {
		session := "test-sendkeys-literal"
		paneID := "0"
		socketPath := agent.SafeSocketPath(session, paneID)

		socketDir := filepath.Dir(socketPath)
		os.MkdirAll(socketDir, 0700)
		defer os.RemoveAll(socketDir)

		var mu sync.Mutex
		var received string
		srv, err := agent.ListenAndServe(socketPath, func(data string) error {
			mu.Lock()
			defer mu.Unlock()
			received = data
			return nil
		})
		if err != nil {
			t.Fatalf("failed to start server: %v", err)
		}
		defer srv.Close()

		ctx, _ := newTestContext(session)
		ctx.PaneManager.Register("")

		result := &ParseResult{
			Command: "send-keys",
			Args:    []string{"-l", "-t", "%" + paneID, "Enter"},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		mu.Lock()
		got := received
		mu.Unlock()
		// In literal mode, "Enter" should NOT be expanded to \r
		want := "Enter"
		if got != want {
			t.Errorf("received = %q, want %q", got, want)
		}
	})

	t.Run("empty keys is no-op", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		result := &ParseResult{
			Command: "send-keys",
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
	})

	t.Run("socket error returns exit 1", func(t *testing.T) {
		ctx, _ := newTestContext("test-sendkeys-noserver")
		ctx.PaneManager.Register("")

		result := &ParseResult{
			Command: "send-keys",
			Args:    []string{"-t", "%0", "hello"},
		}
		exitCode, err := Execute(ctx, result)
		if err == nil {
			t.Fatal("expected error when socket server not running")
		}
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
	})
}

// --- TC-CMD-005: select-pane ---

func TestCommandSelectPane(t *testing.T) {
	ctx, _ := newTestContext("test-session")
	result := &ParseResult{
		Command: "select-pane",
		Args:    []string{"-t", "%0"},
	}
	exitCode, err := Execute(ctx, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Phase 1: best-effort no-op
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
}

// --- TC-CMD-006: kill-pane ---

func TestCommandKillPane(t *testing.T) {
	t.Run("close surface and unregister pane", func(t *testing.T) {
		ctx, mock := newTestContext("test-session")
		// Register a pane first
		p, _ := ctx.PaneManager.Register("")

		result := &ParseResult{
			Command: "kill-pane",
			Args:    []string{"-t", "%" + p.ID},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		// Verify CloseSurface was called
		found := false
		for _, call := range mock.Calls {
			if call.Method == "CloseSurface" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected CloseSurface call")
		}

		// Verify pane was unregistered
		panes := ctx.PaneManager.List()
		if len(panes) != 0 {
			t.Errorf("pane count = %d, want 0", len(panes))
		}
	})

	t.Run("close error returns error", func(t *testing.T) {
		ctx, mock := newTestContext("test-session")
		mock.CloseErr = os.ErrPermission
		p, _ := ctx.PaneManager.Register("")

		result := &ParseResult{
			Command: "kill-pane",
			Args:    []string{"-t", "%" + p.ID},
		}
		exitCode, err := Execute(ctx, result)
		if err == nil {
			t.Fatal("expected error from close failure")
		}
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
	})
}

// --- TC-CMD-007: list-panes ---

func TestCommandListPanes(t *testing.T) {
	t.Run("default format outputs pane IDs", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		ctx.PaneManager.Register("")
		ctx.PaneManager.Register("")

		result := &ParseResult{
			Command: "list-panes",
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		out := stdout(ctx)
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != 2 {
			t.Fatalf("line count = %d, want 2, output=%q", len(lines), out)
		}
		// Each line should be the pane ID prefixed with %
		if lines[0] != "%0" {
			t.Errorf("line 0 = %q, want %%0", lines[0])
		}
		if lines[1] != "%1" {
			t.Errorf("line 1 = %q, want %%1", lines[1])
		}
	})

	t.Run("custom format", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		ctx.PaneManager.Register("")

		result := &ParseResult{
			Command: "list-panes",
			Args:    []string{"-F", "#{pane_id}:#{pane_active}"},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		out := stdout(ctx)
		line := strings.TrimSpace(out)
		// First pane is active
		if line != "%0:1" {
			t.Errorf("output = %q, want %%0:1", line)
		}
	})

	t.Run("empty pane list", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		result := &ParseResult{
			Command: "list-panes",
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		out := stdout(ctx)
		if out != "" {
			t.Errorf("stdout = %q, want empty", out)
		}
	})
}

// --- TC-CMD-008: display-message -p ---

func TestCommandDisplayMessage(t *testing.T) {
	t.Run("-p outputs formatted text", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		ctx.PaneManager.Register("")

		result := &ParseResult{
			Command: "display-message",
			Args:    []string{"-p", "#{session_name}"},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		out := stdout(ctx)
		if strings.TrimSpace(out) != "test-session" {
			t.Errorf("stdout = %q, want 'test-session'", strings.TrimSpace(out))
		}
	})

	t.Run("without -p is no-op", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		result := &ParseResult{
			Command: "display-message",
			Args:    []string{"#{session_name}"},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		out := stdout(ctx)
		if out != "" {
			t.Errorf("stdout = %q, want empty (no -p flag)", out)
		}
	})

	t.Run("-p with pane variables", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		ctx.PaneManager.Register("")

		result := &ParseResult{
			Command: "display-message",
			Args:    []string{"-p", "#{pane_id}"},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		out := strings.TrimSpace(stdout(ctx))
		if !strings.HasPrefix(out, "%") {
			t.Errorf("stdout = %q, want pane ID with %% prefix", out)
		}
	})
}

// --- TC-CMD-009: show-options ---

func TestCommandShowOptions(t *testing.T) {
	t.Run("-g prefix outputs prefix C-b", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		result := &ParseResult{
			Command: "show-options",
			Args:    []string{"-g", "prefix"},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		out := stdout(ctx)
		if strings.TrimSpace(out) != "prefix C-b" {
			t.Errorf("stdout = %q, want 'prefix C-b'", strings.TrimSpace(out))
		}
	})

	t.Run("without -g prefix outputs nothing", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		result := &ParseResult{
			Command: "show-options",
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		out := stdout(ctx)
		if out != "" {
			t.Errorf("stdout = %q, want empty", out)
		}
	})

	t.Run("-g with non-prefix option outputs nothing", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		result := &ParseResult{
			Command: "show-options",
			Args:    []string{"-g", "status-style"},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}

		out := stdout(ctx)
		if out != "" {
			t.Errorf("stdout = %q, want empty", out)
		}
	})
}

// --- TC-CMD-010: unsupported command ---

func TestCommandUnsupported(t *testing.T) {
	ctx, _ := newTestContext("test-session")
	result := &ParseResult{
		Command: "resize-pane",
	}
	exitCode, err := Execute(ctx, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}

	errOut := stderr(ctx)
	if !strings.Contains(errOut, "unsupported") {
		t.Errorf("stderr = %q, want message containing 'unsupported'", errOut)
	}
	if !strings.Contains(errOut, "resize-pane") {
		t.Errorf("stderr = %q, want message containing 'resize-pane'", errOut)
	}
}

// --- TC-CMD-011: switch-client and new-session are no-op ---

func TestCommandSwitchClient(t *testing.T) {
	ctx, _ := newTestContext("test-session")
	result := &ParseResult{
		Command: "switch-client",
	}
	exitCode, err := Execute(ctx, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
}

func TestCommandNewSession(t *testing.T) {
	ctx, _ := newTestContext("test-session")
	result := &ParseResult{
		Command: "new-session",
	}
	exitCode, err := Execute(ctx, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
}

// --- TC-CMD-012a: buildFormatContext maps PaneInfo fields ---

func TestBuildFormatContextNewFields(t *testing.T) {
	t.Run("pane_tty from Pane.TTY", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		p, _ := ctx.PaneManager.Register("/dev/ttys005")

		result := &ParseResult{
			Command: "display-message",
			Args:    []string{"-p", "#{pane_tty}"},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		out := strings.TrimSpace(stdout(ctx))
		if out != p.TTY {
			t.Errorf("pane_tty = %q, want %q", out, p.TTY)
		}
	})

	t.Run("pane_pid from Pane.PID", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		p, _ := ctx.PaneManager.Register("/dev/ttys005")
		p.PID = 54321

		result := &ParseResult{
			Command: "display-message",
			Args:    []string{"-p", "#{pane_pid}"},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		out := strings.TrimSpace(stdout(ctx))
		if out != "54321" {
			t.Errorf("pane_pid = %q, want %q", out, "54321")
		}
	})

	t.Run("pane_current_path from Pane.CurrentPath", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		p, _ := ctx.PaneManager.Register("/dev/ttys005")
		p.CurrentPath = "/home/user/project"

		result := &ParseResult{
			Command: "display-message",
			Args:    []string{"-p", "#{pane_current_path}"},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		out := strings.TrimSpace(stdout(ctx))
		if out != "/home/user/project" {
			t.Errorf("pane_current_path = %q, want %q", out, "/home/user/project")
		}
	})

	t.Run("pane_index from registration order", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		ctx.PaneManager.Register("/dev/ttys001")
		ctx.PaneManager.Register("/dev/ttys002")
		// Set second pane as active
		ctx.PaneManager.SetActive("1")

		result := &ParseResult{
			Command: "display-message",
			Args:    []string{"-p", "#{pane_index}"},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		out := strings.TrimSpace(stdout(ctx))
		if out != "1" {
			t.Errorf("pane_index = %q, want %q", out, "1")
		}
	})

	t.Run("list-panes with all new format variables", func(t *testing.T) {
		ctx, _ := newTestContext("test-session")
		p, _ := ctx.PaneManager.Register("/dev/ttys005")
		p.PID = 12345
		p.CurrentPath = "/tmp"
		p.Width = 120
		p.Height = 40

		result := &ParseResult{
			Command: "list-panes",
			Args:    []string{"-F", "#{pane_index}:#{pane_tty}:#{pane_pid}:#{pane_current_path}:#{pane_width}x#{pane_height}"},
		}
		exitCode, err := Execute(ctx, result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exitCode != 0 {
			t.Errorf("exit code = %d, want 0", exitCode)
		}
		out := strings.TrimSpace(stdout(ctx))
		want := "0:/dev/ttys005:12345:/tmp:120x40"
		if out != want {
			t.Errorf("output = %q, want %q", out, want)
		}
	})
}

// --- TC-CMD-012: extractPaneID ---

func TestExtractPaneID(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   string
	}{
		{"percent N", "%0", "0"},
		{"percent large", "%123", "123"},
		{"session:window.pane", "session:0.%5", "5"},
		{"just pane", "%42", "42"},
		{"empty", "", ""},
		{"no percent", "session:0", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPaneID(tt.target)
			if got != tt.want {
				t.Errorf("extractPaneID(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}
