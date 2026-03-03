package integration_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/kawaz/shimux/internal/agent"
	"github.com/kawaz/shimux/internal/ghostty"
	"github.com/kawaz/shimux/internal/ghostty/ghosttytest"
	"github.com/kawaz/shimux/internal/pane"
	"github.com/kawaz/shimux/internal/tmux"
	"github.com/kawaz/shimux/internal/wrapper"
)

// newIntegrationContext creates a CommandContext with MockController and a
// PaneManager backed by a temporary directory. Returns the context, mock
// controller, and a cleanup function.
func newIntegrationContext(t *testing.T, session string) (*tmux.CommandContext, *ghosttytest.MockController) {
	t.Helper()
	mock := &ghosttytest.MockController{}
	stateDir := t.TempDir()
	mgr := pane.NewWithDir(stateDir)
	return &tmux.CommandContext{
		Controller:  mock,
		PaneManager: mgr,
		Session:     session,
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
	}, mock
}

func stdout(ctx *tmux.CommandContext) string {
	return ctx.Stdout.(*bytes.Buffer).String()
}

func resetStdout(ctx *tmux.CommandContext) {
	ctx.Stdout.(*bytes.Buffer).Reset()
}

// parseAndExecute is a helper that parses tmux args and executes the command.
func parseAndExecute(t *testing.T, ctx *tmux.CommandContext, args []string) (int, error) {
	t.Helper()
	result, err := tmux.Parse(args)
	if err != nil {
		return 1, err
	}
	return tmux.Execute(ctx, result)
}

// --- Scenario 1: tmux command full pipeline ---

func TestIntegration_TmuxCommandPipeline(t *testing.T) {
	session := "integration-test"
	ctx, mock := newIntegrationContext(t, session)

	// 1. Register initial pane (%0)
	_, err := ctx.PaneManager.Register("")
	if err != nil {
		t.Fatalf("register initial pane: %v", err)
	}

	// 2. split-window -h -P -F "#{pane_id}" -> parse & execute
	//    Expect: MockController.NewSplit(SplitRight) called, %1 registered, stdout = "%1\n"
	exitCode, err := parseAndExecute(t, ctx, []string{"split-window", "-h", "-P", "-F", "#{pane_id}"})
	if err != nil {
		t.Fatalf("split-window: unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("split-window exit code = %d, want 0", exitCode)
	}

	// Verify MockController received NewSplit(SplitRight)
	if len(mock.Calls) == 0 {
		t.Fatal("expected at least one MockController call")
	}
	call := mock.Calls[0]
	if call.Method != "NewSplit" {
		t.Errorf("method = %q, want NewSplit", call.Method)
	}
	if call.Args[0] != ghostty.SplitRight {
		t.Errorf("direction = %v, want SplitRight", call.Args[0])
	}

	// Verify stdout contains "%1"
	out := strings.TrimSpace(stdout(ctx))
	if out != "%1" {
		t.Errorf("split-window output = %q, want %%1", out)
	}

	// 3. list-panes -F "#{pane_id}" -> "%0\n%1\n"
	resetStdout(ctx)
	exitCode, err = parseAndExecute(t, ctx, []string{"list-panes", "-F", "#{pane_id}"})
	if err != nil {
		t.Fatalf("list-panes: unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("list-panes exit code = %d, want 0", exitCode)
	}
	lines := strings.Split(strings.TrimSpace(stdout(ctx)), "\n")
	if len(lines) != 2 {
		t.Fatalf("list-panes lines = %d, want 2, output=%q", len(lines), stdout(ctx))
	}
	if lines[0] != "%0" || lines[1] != "%1" {
		t.Errorf("list-panes output = %v, want [%%0, %%1]", lines)
	}

	// 4. display-message -p "#{session_name}" -> session name
	resetStdout(ctx)
	exitCode, err = parseAndExecute(t, ctx, []string{"display-message", "-p", "#{session_name}"})
	if err != nil {
		t.Fatalf("display-message: unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("display-message exit code = %d, want 0", exitCode)
	}
	dmOut := strings.TrimSpace(stdout(ctx))
	if dmOut != session {
		t.Errorf("display-message output = %q, want %q", dmOut, session)
	}

	// 5. has-session -t <session> -> exit 0
	resetStdout(ctx)
	exitCode, err = parseAndExecute(t, ctx, []string{"has-session", "-t", session})
	if err != nil {
		t.Fatalf("has-session (match): unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("has-session (match) exit code = %d, want 0", exitCode)
	}

	// 6. has-session -t <wrong> -> exit 1
	exitCode, err = parseAndExecute(t, ctx, []string{"has-session", "-t", "wrong-session"})
	if err != nil {
		t.Fatalf("has-session (no match): unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Errorf("has-session (no match) exit code = %d, want 1", exitCode)
	}

	// 7. kill-pane -t %1 -> CloseSurface called + %1 deleted
	resetStdout(ctx)
	callCountBefore := len(mock.Calls)
	exitCode, err = parseAndExecute(t, ctx, []string{"kill-pane", "-t", "%1"})
	if err != nil {
		t.Fatalf("kill-pane: unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("kill-pane exit code = %d, want 0", exitCode)
	}

	// Verify CloseSurface called
	closeSurfaceCalled := false
	for _, c := range mock.Calls[callCountBefore:] {
		if c.Method == "CloseSurface" {
			closeSurfaceCalled = true
			break
		}
	}
	if !closeSurfaceCalled {
		t.Error("expected CloseSurface call for kill-pane")
	}

	// Verify %1 is removed, only %0 remains
	panes := ctx.PaneManager.List()
	if len(panes) != 1 {
		t.Fatalf("pane count after kill = %d, want 1", len(panes))
	}
	if panes[0].ID != "0" {
		t.Errorf("remaining pane ID = %q, want '0'", panes[0].ID)
	}
}

// --- Scenario 2: send-keys with socket server ---

func TestIntegration_SendKeysWithSocketServer(t *testing.T) {
	session := "integ-sendkeys"
	paneID := "0"
	socketPath := agent.SafeSocketPath(session, paneID)

	// Ensure socket directory exists
	socketDir := filepath.Dir(socketPath)
	if err := os.MkdirAll(socketDir, 0700); err != nil {
		t.Fatalf("create socket dir: %v", err)
	}
	defer os.RemoveAll(socketDir)

	// Clean up stale socket if any
	os.Remove(socketPath)

	// Start socket server to capture received data
	var mu sync.Mutex
	var received string
	srv, err := agent.ListenAndServe(socketPath, func(data string) error {
		mu.Lock()
		defer mu.Unlock()
		received = data
		return nil
	})
	if err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer srv.Close()

	// Set up integration context
	ctx, _ := newIntegrationContext(t, session)
	ctx.PaneManager.Register("") // register pane %0

	// Parse and execute: send-keys -t %0 'echo hello' Enter
	exitCode, err := parseAndExecute(t, ctx, []string{"send-keys", "-t", "%0", "echo hello", "Enter"})
	if err != nil {
		t.Fatalf("send-keys: unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("send-keys exit code = %d, want 0", exitCode)
	}

	// Verify received data: "echo hello" + Enter (\r)
	mu.Lock()
	got := received
	mu.Unlock()
	want := "echo hello\r"
	if got != want {
		t.Errorf("received = %q, want %q", got, want)
	}
}

// --- Scenario 3: error paths ---

func TestIntegration_ErrorPaths(t *testing.T) {
	t.Run("invalid command arguments", func(t *testing.T) {
		// Parse with missing -t value for split-window
		_, err := tmux.Parse([]string{"split-window", "-t"})
		if err != nil {
			// Parse itself does not validate command args, it just puts them in Args.
			t.Fatalf("unexpected Parse error: %v", err)
		}
		// The error should come from ParseSplitWindow
		ctx, _ := newIntegrationContext(t, "test")
		result := &tmux.ParseResult{
			Command: "split-window",
			Args:    []string{"-t"},
		}
		exitCode, err := tmux.Execute(ctx, result)
		if err == nil {
			t.Fatal("expected error for missing -t value")
		}
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("send-keys to non-existent socket", func(t *testing.T) {
		ctx, _ := newIntegrationContext(t, "nonexistent-session-xxx")
		ctx.PaneManager.Register("")

		exitCode, err := parseAndExecute(t, ctx, []string{"send-keys", "-t", "%0", "hello"})
		if err == nil {
			t.Fatal("expected error for non-existent socket")
		}
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
	})

	t.Run("split-window controller error", func(t *testing.T) {
		ctx, mock := newIntegrationContext(t, "test")
		mock.SplitErr = fmt.Errorf("controller error")

		exitCode, err := parseAndExecute(t, ctx, []string{"split-window"})
		if err == nil {
			t.Fatal("expected error from controller failure")
		}
		if exitCode != 1 {
			t.Errorf("exit code = %d, want 1", exitCode)
		}
		if !strings.Contains(err.Error(), "controller error") {
			t.Errorf("error = %q, want to contain 'controller error'", err.Error())
		}
	})

	t.Run("no command specified", func(t *testing.T) {
		_, err := tmux.Parse([]string{})
		if err == nil {
			t.Fatal("expected error for empty args")
		}
		if !strings.Contains(err.Error(), "no command specified") {
			t.Errorf("error = %q, want 'no command specified'", err.Error())
		}
	})
}

// --- Scenario 4: Wrapper Setup + Env ---

func TestIntegration_WrapperSetupAndEnv(t *testing.T) {
	mock := &ghosttytest.MockController{}
	stateDir := t.TempDir()
	mgr := pane.NewWithDir(stateDir)

	// Use a known shimux path (test binary itself works for symlink target)
	shimuxPath := "/bin/echo" // Use a known binary

	w, err := wrapper.New(wrapper.Config{
		Controller:  mock,
		PaneManager: mgr,
		Session:     "test-session",
		ShimuxPath:    shimuxPath,
	})
	if err != nil {
		t.Fatalf("wrapper.New: %v", err)
	}

	// Setup creates temp directory with tmux symlink
	if err := w.Setup(); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	// Verify Env contains expected variables
	env := w.Env()
	envMap := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// SHIMUX=1
	if envMap["SHIMUX"] != "1" {
		t.Errorf("SHIMUX = %q, want '1'", envMap["SHIMUX"])
	}

	// SHIMUX_SESSION should be sanitized session name
	if envMap["SHIMUX_SESSION"] != "test-session" {
		t.Errorf("SHIMUX_SESSION = %q, want 'test-session'", envMap["SHIMUX_SESSION"])
	}

	// SHIMUX_PANE_ID=0
	if envMap["SHIMUX_PANE_ID"] != "0" {
		t.Errorf("SHIMUX_PANE_ID = %q, want '0'", envMap["SHIMUX_PANE_ID"])
	}

	// TMUX_PANE=%0
	if envMap["TMUX_PANE"] != "%0" {
		t.Errorf("TMUX_PANE = %q, want '%%0'", envMap["TMUX_PANE"])
	}

	// PATH should start with the tmpdir
	pathVal := envMap["PATH"]
	if pathVal == "" {
		t.Fatal("PATH not set in env")
	}
	tmpDir := strings.SplitN(pathVal, ":", 2)[0]

	// Verify tmux symlink exists in tmpDir
	symlinkPath := filepath.Join(tmpDir, "tmux")
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("tmux symlink not found: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("tmux is not a symlink")
	}

	// Verify symlink target
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if target != shimuxPath {
		t.Errorf("symlink target = %q, want %q", target, shimuxPath)
	}

	// TMUX variable should contain the tmpdir path
	tmuxVal := envMap["TMUX"]
	if !strings.HasPrefix(tmuxVal, tmpDir+"/shimux") {
		t.Errorf("TMUX = %q, want prefix %q", tmuxVal, tmpDir+"/shimux")
	}

	// Cleanup removes the temp directory
	if err := w.Cleanup(); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	// Verify temp directory is removed
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Errorf("temp directory still exists after Cleanup")
	}
}

// --- Scenario 5: Session name sanitize + socket path consistency ---

func TestIntegration_SessionNameAndSocketPath(t *testing.T) {
	// 1. GenerateSessionName sanitizes special characters
	sanitized := wrapper.GenerateSessionName("feature/new-cmd")
	if sanitized != "feature_new-cmd" {
		t.Errorf("GenerateSessionName('feature/new-cmd') = %q, want 'feature_new-cmd'", sanitized)
	}

	// 2. SafeSocketPath generates a valid path
	socketPath := agent.SafeSocketPath("feature_new-cmd", "0")
	if socketPath == "" {
		t.Fatal("SafeSocketPath returned empty string")
	}

	// 3. Path length must be <= 104 bytes (macOS sun_path limit)
	if len(socketPath) > 104 {
		t.Errorf("socket path length = %d, want <= 104", len(socketPath))
	}

	// 4. Long session name gets hashed
	longSession := strings.Repeat("a", 200)
	longPath := agent.SafeSocketPath(longSession, "0")
	if len(longPath) > 104 {
		t.Errorf("long session socket path length = %d, want <= 104", len(longPath))
	}

	// The hashed path should be different from the plain path
	plainPath := agent.SocketPath(longSession, "0")
	if len(plainPath) <= 104 {
		t.Skip("plain path is already short enough, no hashing needed")
	}
	if longPath == plainPath {
		t.Error("long session path was not hashed despite exceeding limit")
	}

	// 5. Verify path consistency: same input -> same output
	path1 := agent.SafeSocketPath("feature_new-cmd", "0")
	path2 := agent.SafeSocketPath("feature_new-cmd", "0")
	if path1 != path2 {
		t.Errorf("SafeSocketPath not deterministic: %q != %q", path1, path2)
	}
}

// --- Scenario 6: Multiple pane split and list consistency ---

func TestIntegration_MultiPaneSplitAndList(t *testing.T) {
	ctx, _ := newIntegrationContext(t, "multi-pane-test")

	// 1. Register initial pane (%0)
	_, err := ctx.PaneManager.Register("")
	if err != nil {
		t.Fatalf("register initial pane: %v", err)
	}

	// 2. split-window -h x3 (horizontal split 3 times) -> creates %1, %2, %3
	for i := 0; i < 3; i++ {
		resetStdout(ctx)
		exitCode, err := parseAndExecute(t, ctx, []string{"split-window", "-h", "-P", "-F", "#{pane_id}"})
		if err != nil {
			t.Fatalf("split-window -h #%d: unexpected error: %v", i+1, err)
		}
		if exitCode != 0 {
			t.Errorf("split-window -h #%d exit code = %d, want 0", i+1, exitCode)
		}
		out := strings.TrimSpace(stdout(ctx))
		wantID := fmt.Sprintf("%%%d", i+1)
		if out != wantID {
			t.Errorf("split-window -h #%d output = %q, want %q", i+1, out, wantID)
		}
	}

	// 3. split-window -v x1 (vertical split) -> creates %4
	resetStdout(ctx)
	exitCode, err := parseAndExecute(t, ctx, []string{"split-window", "-v", "-P", "-F", "#{pane_id}"})
	if err != nil {
		t.Fatalf("split-window -v: unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("split-window -v exit code = %d, want 0", exitCode)
	}
	out := strings.TrimSpace(stdout(ctx))
	if out != "%4" {
		t.Errorf("split-window -v output = %q, want %%4", out)
	}

	// 4. list-panes -> 5 panes (%0~%4)
	resetStdout(ctx)
	exitCode, err = parseAndExecute(t, ctx, []string{"list-panes", "-F", "#{pane_id}"})
	if err != nil {
		t.Fatalf("list-panes (5 panes): unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("list-panes exit code = %d, want 0", exitCode)
	}
	lines := strings.Split(strings.TrimSpace(stdout(ctx)), "\n")
	if len(lines) != 5 {
		t.Fatalf("list-panes lines = %d, want 5, output=%q", len(lines), stdout(ctx))
	}
	for i := 0; i < 5; i++ {
		want := fmt.Sprintf("%%%d", i)
		if lines[i] != want {
			t.Errorf("list-panes line %d = %q, want %q", i, lines[i], want)
		}
	}

	// 5. kill-pane -t %2
	exitCode, err = parseAndExecute(t, ctx, []string{"kill-pane", "-t", "%2"})
	if err != nil {
		t.Fatalf("kill-pane -t %%2: unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("kill-pane exit code = %d, want 0", exitCode)
	}

	// 6. list-panes -> 4 panes (%0, %1, %3, %4)
	resetStdout(ctx)
	exitCode, err = parseAndExecute(t, ctx, []string{"list-panes", "-F", "#{pane_id}"})
	if err != nil {
		t.Fatalf("list-panes (4 panes): unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("list-panes exit code = %d, want 0", exitCode)
	}
	lines = strings.Split(strings.TrimSpace(stdout(ctx)), "\n")
	if len(lines) != 4 {
		t.Fatalf("list-panes lines = %d, want 4, output=%q", len(lines), stdout(ctx))
	}
	wantPanes := []string{"%0", "%1", "%3", "%4"}
	for i, want := range wantPanes {
		if lines[i] != want {
			t.Errorf("list-panes line %d = %q, want %q", i, lines[i], want)
		}
	}
}
