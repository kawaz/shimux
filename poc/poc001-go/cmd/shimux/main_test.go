package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// --- TC-MAIN-001: no arguments ---

func TestRunNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runMain(nil, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if !strings.Contains(stderr.String(), "no arguments") {
		t.Errorf("stderr = %q, want message containing 'no arguments'", stderr.String())
	}
}

// --- TC-MAIN-002: -V flag in tmux mode ---

func TestRunVersionTmuxMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "-V"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	out := stdout.String()
	if !strings.Contains(out, "shimux") {
		t.Errorf("stdout = %q, want message containing 'shimux'", out)
	}
	if !strings.Contains(out, "tmux-compatible") {
		t.Errorf("stdout = %q, want message containing 'tmux-compatible'", out)
	}
}

// --- TC-MAIN-003: -V flag in shimux mode ---

func TestRunVersionShimuxMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"shimux", "-V"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	out := stdout.String()
	if !strings.Contains(out, "shimux") {
		t.Errorf("stdout = %q, want message containing 'shimux'", out)
	}
	// shimux mode should NOT contain "tmux-compatible"
	if strings.Contains(out, "tmux-compatible") {
		t.Errorf("stdout = %q, should NOT contain 'tmux-compatible' in shimux mode", out)
	}
}

// --- TC-MAIN-004: tmux mode detection via argv[0] ---

func TestRunTmuxModeDetection(t *testing.T) {
	// tmux has-session -t <session> should succeed when SHIMUX_SESSION matches
	t.Setenv("SHIMUX_SESSION", "test-session")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "has-session", "-t", "test-session"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0 (has-session should match)", exitCode)
	}
}

// --- TC-MAIN-005: shimux mode detection via argv[0] ---

func TestRunShimuxModeDetection(t *testing.T) {
	// shimux without -- should show usage
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"shimux"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (no -- separator)", exitCode)
	}
	if !strings.Contains(stderr.String(), "usage") {
		t.Errorf("stderr = %q, want message containing 'usage'", stderr.String())
	}
}

// --- TC-MAIN-006: SHIMUX=1 env triggers tmux mode ---

func TestRunSHIMUXEnvTmuxMode(t *testing.T) {
	t.Setenv("SHIMUX", "1")
	t.Setenv("SHIMUX_SESSION", "env-session")
	var stdout, stderr bytes.Buffer
	// argv[0] is neither "tmux" nor "shimux", but SHIMUX=1 so tmux mode
	exitCode := runMain([]string{"/usr/local/bin/sometool", "has-session", "-t", "env-session"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0 (SHIMUX=1 should trigger tmux mode)", exitCode)
	}
}

// --- TC-MAIN-007: shimux -- without command ---

func TestRunShimuxModeNoCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"shimux", "--"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (no command after --)", exitCode)
	}
	if !strings.Contains(stderr.String(), "usage") {
		t.Errorf("stderr = %q, want message containing 'usage'", stderr.String())
	}
}

// --- TC-MAIN-008: tmux mode has-session with matching session ---

func TestRunTmuxHasSessionMatch(t *testing.T) {
	t.Setenv("SHIMUX_SESSION", "my-session")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "has-session", "-t", "my-session"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
}

// --- TC-MAIN-009: tmux mode has-session with non-matching session ---

func TestRunTmuxHasSessionNoMatch(t *testing.T) {
	t.Setenv("SHIMUX_SESSION", "my-session")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "has-session", "-t", "other-session"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (session should not match)", exitCode)
	}
}

// --- TC-MAIN-010: tmux mode parse error ---

func TestRunTmuxModeParseError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// -L without value should be a parse error
	exitCode := runMain([]string{"tmux", "-L"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (parse error)", exitCode)
	}
	if !strings.Contains(stderr.String(), "shimux:") {
		t.Errorf("stderr = %q, want message prefixed with 'shimux:'", stderr.String())
	}
}

// --- TC-MAIN-011: shimux -V before -- ---

func TestRunShimuxVersionBeforeSeparator(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"shimux", "-V", "--", "echo", "hello"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0 (-V should be handled before --)", exitCode)
	}
	out := stdout.String()
	if !strings.Contains(out, "shimux") {
		t.Errorf("stdout = %q, want message containing 'shimux'", out)
	}
}

// --- TC-MAIN-012: shimux -- requires ghostty environment ---

func TestRunShimuxModeRequiresGhostty(t *testing.T) {
	// Ensure we are NOT in ghostty environment
	t.Setenv("TERM_PROGRAM", "xterm")
	os.Unsetenv("GHOSTTY_RESOURCES_DIR")

	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"shimux", "--", "echo", "hello"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (not in ghostty)", exitCode)
	}
	if !strings.Contains(stderr.String(), "ghostty") {
		t.Errorf("stderr = %q, want message containing 'ghostty'", stderr.String())
	}
}

// --- TC-MAIN-013: tmux mode no command specified ---

func TestRunTmuxModeNoCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (no command)", exitCode)
	}
}

// --- TC-MAIN-014: tmux mode display-message ---

func TestRunTmuxDisplayMessage(t *testing.T) {
	t.Setenv("SHIMUX_SESSION", "test-display")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "display-message", "-p", "#{session_name}"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	out := strings.TrimSpace(stdout.String())
	if out != "test-display" {
		t.Errorf("stdout = %q, want 'test-display'", out)
	}
}

// --- TC-MAIN-015: tmux mode list-panes on empty manager ---

func TestRunTmuxListPanesEmpty(t *testing.T) {
	t.Setenv("SHIMUX_SESSION", "test-list")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "list-panes"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if stdout.String() != "" {
		t.Errorf("stdout = %q, want empty (no panes)", stdout.String())
	}
}

// --- TC-MAIN-016: default mode (unknown argv[0], no SHIMUX env) falls to shimux mode ---

func TestRunDefaultModeIsShimux(t *testing.T) {
	t.Setenv("SHIMUX", "")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"/some/unknown/binary"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (shimux mode, no -- separator)", exitCode)
	}
	if !strings.Contains(stderr.String(), "usage") {
		t.Errorf("stderr = %q, want message containing 'usage'", stderr.String())
	}
}

// --- TC-MAIN-017: -V after -- should NOT show version (shimux mode) ---

func TestRunShimuxVersionAfterSeparator_NotVersion(t *testing.T) {
	// -V after -- is part of the command, not a version flag
	// Without ghostty environment, should fail at ghostty check (not show version)
	t.Setenv("TERM_PROGRAM", "xterm")
	t.Setenv("GHOSTTY_RESOURCES_DIR", "")

	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"shimux", "--", "-V"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (not in ghostty, -V after -- is a command)", exitCode)
	}
	// Should NOT contain version info
	if strings.Contains(stdout.String(), "shimux") {
		t.Errorf("stdout = %q, -V after -- should NOT show version", stdout.String())
	}
	// Should show ghostty error
	if !strings.Contains(stderr.String(), "ghostty") {
		t.Errorf("stderr = %q, should mention ghostty requirement", stderr.String())
	}
}

// --- TC-MAIN-018: tmux mode switch-client is no-op ---

func TestRunTmuxSwitchClient(t *testing.T) {
	t.Setenv("SHIMUX_SESSION", "test-switch")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "switch-client"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0 (switch-client is no-op)", exitCode)
	}
}

// --- TC-MAIN-019: tmux mode new-session is no-op ---

func TestRunTmuxNewSession(t *testing.T) {
	t.Setenv("SHIMUX_SESSION", "test-new")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "new-session"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0 (new-session is no-op)", exitCode)
	}
}

// --- TC-MAIN-020: tmux mode show-options -g prefix ---

func TestRunTmuxShowOptionsPrefix(t *testing.T) {
	t.Setenv("SHIMUX_SESSION", "test-show")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "show-options", "-g", "prefix"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	out := strings.TrimSpace(stdout.String())
	if out != "prefix C-b" {
		t.Errorf("stdout = %q, want 'prefix C-b'", out)
	}
}

// --- TC-MAIN-021: tmux mode show-options without -g ---

func TestRunTmuxShowOptionsNoGlobal(t *testing.T) {
	t.Setenv("SHIMUX_SESSION", "test-show")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "show-options", "prefix"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	// Without -g, show-options should produce no output
	if stdout.String() != "" {
		t.Errorf("stdout = %q, want empty (no -g flag)", stdout.String())
	}
}

// --- TC-MAIN-022: tmux mode unsupported command ---

func TestRunTmuxUnsupportedCommand(t *testing.T) {
	t.Setenv("SHIMUX_SESSION", "test-unsupported")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "rename-window", "-t", "0", "newname"}, &stdout, &stderr)
	// unsupported command should return 0 (with stderr warning)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0 (unsupported command is warn-only)", exitCode)
	}
	if !strings.Contains(stderr.String(), "unsupported") {
		t.Errorf("stderr = %q, want message containing 'unsupported'", stderr.String())
	}
}

// --- TC-MAIN-023: tmux mode display-message without -p ---

func TestRunTmuxDisplayMessageNoPrint(t *testing.T) {
	t.Setenv("SHIMUX_SESSION", "test-display-nop")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "display-message", "#{session_name}"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	// Without -p, display-message is no-op (no status line)
	if stdout.String() != "" {
		t.Errorf("stdout = %q, want empty (no -p flag)", stdout.String())
	}
}

// --- TC-MAIN-024: tmux mode -V with global option ---

func TestRunTmuxVersionWithGlobalOption(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "-L", "mysocket", "-V"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout.String(), "shimux") {
		t.Errorf("stdout = %q, want message containing 'shimux'", stdout.String())
	}
}

// --- TC-MAIN-025: tmux mode -S parse error ---

func TestRunTmuxParseSError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "-S"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (parse error)", exitCode)
	}
}

// --- TC-MAIN-026: tmux mode -f parse error ---

func TestRunTmuxParseFError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "-f"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (parse error)", exitCode)
	}
}

// --- TC-MAIN-027: tmux mode select-pane is no-op ---

func TestRunTmuxSelectPane(t *testing.T) {
	t.Setenv("SHIMUX_SESSION", "test-select")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "select-pane", "-t", "%0"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0 (select-pane is no-op)", exitCode)
	}
}

// --- TC-MAIN-028: shimux mode with multiple -- separators ---

func TestRunShimuxModeMultipleSeparators(t *testing.T) {
	// The first -- should be the separator, subsequent -- are part of the command
	t.Setenv("TERM_PROGRAM", "xterm")
	t.Setenv("GHOSTTY_RESOURCES_DIR", "")

	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"shimux", "--", "echo", "--", "hello"}, &stdout, &stderr)
	// Should fail because not in ghostty, but should parse args correctly
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (not in ghostty)", exitCode)
	}
	if !strings.Contains(stderr.String(), "ghostty") {
		t.Errorf("stderr = %q, want ghostty error", stderr.String())
	}
}

// --- TC-MAIN-029: tmux mode has-session with -t missing value ---

func TestRunTmuxHasSessionMissingTarget(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "has-session", "-t"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (missing -t value)", exitCode)
	}
}

// --- TC-MAIN-030: tmux mode display-message with format variables ---

func TestRunTmuxDisplayMessageFormatVars(t *testing.T) {
	t.Setenv("SHIMUX_SESSION", "format-test")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "display-message", "-p", "session=#{session_name}"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	out := strings.TrimSpace(stdout.String())
	if out != "session=format-test" {
		t.Errorf("stdout = %q, want 'session=format-test'", out)
	}
}

// --- TC-MAIN-030b: display-message with unknown format variable ---

func TestRunTmuxDisplayMessageUnknownVar(t *testing.T) {
	t.Setenv("SHIMUX_SESSION", "unknown-var-test")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", "display-message", "-p", "#{unknown_var}"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	out := strings.TrimSpace(stdout.String())
	// Unknown variables should be left as-is
	if out != "#{unknown_var}" {
		t.Errorf("stdout = %q, want '#{unknown_var}' (unknown var preserved)", out)
	}
}

// --- TC-MAIN-031: SHIMUX env set to non-1 value ---

func TestRunSHIMUXEnvNonOne_FallsToShimuxMode(t *testing.T) {
	t.Setenv("SHIMUX", "0")
	var stdout, stderr bytes.Buffer
	// SHIMUX=0 should NOT trigger tmux mode; should fall to shimux mode
	exitCode := runMain([]string{"/usr/local/bin/sometool"}, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (shimux mode, no -- separator)", exitCode)
	}
	if !strings.Contains(stderr.String(), "usage") {
		t.Errorf("stderr = %q, want message containing 'usage'", stderr.String())
	}
}

// --- TC-MAIN-032: tmux mode with argv[0] containing path ---

func TestRunTmuxModeViaFullPath(t *testing.T) {
	// argv[0] = /usr/bin/tmux should trigger tmux mode (Base extracts "tmux")
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"/usr/bin/tmux", "-V"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	if !strings.Contains(stdout.String(), "tmux-compatible") {
		t.Errorf("stdout = %q, want 'tmux-compatible'", stdout.String())
	}
}

// --- TC-MAIN-033: shimux mode with argv[0] containing path ---

func TestRunShimuxModeViaFullPath(t *testing.T) {
	// argv[0] = /usr/local/bin/shimux should trigger shimux mode
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"/usr/local/bin/shimux", "-V"}, &stdout, &stderr)
	if exitCode != 0 {
		t.Errorf("exit code = %d, want 0", exitCode)
	}
	out := stdout.String()
	if !strings.Contains(out, "shimux") {
		t.Errorf("stdout = %q, want 'shimux'", out)
	}
	if strings.Contains(out, "tmux-compatible") {
		t.Errorf("stdout = %q, should NOT contain 'tmux-compatible' for shimux mode", out)
	}
}

// --- TC-MAIN-034: tmux mode empty command name error ---

func TestRunTmuxModeEmptyCommandName(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"tmux", ""}, &stdout, &stderr)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (empty command name)", exitCode)
	}
}

// --- TC-MAIN-035: shimux mode only flags before -- (no -V) ---

func TestRunShimuxModeUnknownFlagBeforeSeparator(t *testing.T) {
	// Flags other than -V before -- should be ignored, then -- cmd
	t.Setenv("TERM_PROGRAM", "xterm")
	t.Setenv("GHOSTTY_RESOURCES_DIR", "")

	var stdout, stderr bytes.Buffer
	exitCode := runMain([]string{"shimux", "-x", "--", "echo", "hello"}, &stdout, &stderr)
	// Should fail at ghostty check (not crash)
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1 (not in ghostty)", exitCode)
	}
	if !strings.Contains(stderr.String(), "ghostty") {
		t.Errorf("stderr = %q, want ghostty error", stderr.String())
	}
}
