package tmux

import "testing"

// --- Parse (3-stage parser) benchmarks ---

func BenchmarkParse_SimpleCommand(b *testing.B) {
	args := []string{"has-session", "-t", "my-session"}
	for b.Loop() {
		Parse(args)
	}
}

func BenchmarkParse_GlobalOptionsAndCommand(b *testing.B) {
	args := []string{"-L", "mysock", "-S", "/tmp/sock", "split-window", "-h", "-t", "%0"}
	for b.Loop() {
		Parse(args)
	}
}

func BenchmarkParse_VersionFlag(b *testing.B) {
	args := []string{"-V"}
	for b.Loop() {
		Parse(args)
	}
}

// --- ParseSplitWindow benchmarks ---

func BenchmarkParseSplitWindow_NoArgs(b *testing.B) {
	var args []string
	for b.Loop() {
		ParseSplitWindow(args)
	}
}

func BenchmarkParseSplitWindow_AllFlags(b *testing.B) {
	args := []string{"-h", "-v", "--down", "-t", "%0", "-l", "50%", "-P", "-F", "#{pane_id}"}
	for b.Loop() {
		ParseSplitWindow(args)
	}
}

func BenchmarkParseSplitWindow_DirectionLastWins(b *testing.B) {
	// Simulate conflicting direction flags (last-wins rule)
	args := []string{"-h", "-v", "--left", "--right", "--up", "--down"}
	for b.Loop() {
		ParseSplitWindow(args)
	}
}

// --- ParseSendKeys benchmarks ---

func BenchmarkParseSendKeys_SimpleText(b *testing.B) {
	args := []string{"-t", "%0", "hello", "world"}
	for b.Loop() {
		ParseSendKeys(args)
	}
}

func BenchmarkParseSendKeys_DoubleDash(b *testing.B) {
	args := []string{"-t", "%0", "-l", "--", "-h", "not-a-flag", "Enter"}
	for b.Loop() {
		ParseSendKeys(args)
	}
}

func BenchmarkParseSendKeys_ManyKeys(b *testing.B) {
	args := make([]string, 0, 52)
	args = append(args, "-t", "%0")
	for i := 0; i < 50; i++ {
		args = append(args, "x")
	}
	b.ResetTimer()
	for b.Loop() {
		ParseSendKeys(args)
	}
}

// --- ParseListPanes benchmarks ---

func BenchmarkParseListPanes_WithFormat(b *testing.B) {
	args := []string{"-F", "#{pane_id}:#{pane_active}:#{session_name}", "-t", "session:0"}
	for b.Loop() {
		ParseListPanes(args)
	}
}

// --- ParseDisplayMessage benchmarks ---

func BenchmarkParseDisplayMessage_PrintWithFormat(b *testing.B) {
	args := []string{"-p", "-t", "%0", "#{session_name}:#{pane_id}"}
	for b.Loop() {
		ParseDisplayMessage(args)
	}
}

// --- ParseHasSession benchmarks ---

func BenchmarkParseHasSession(b *testing.B) {
	args := []string{"-t", "my-session-name"}
	for b.Loop() {
		ParseHasSession(args)
	}
}

// --- ParseSelectPane benchmarks ---

func BenchmarkParseSelectPane_AllFlags(b *testing.B) {
	args := []string{"-t", "%0", "-P", "bg=red", "-T", "my-title"}
	for b.Loop() {
		ParseSelectPane(args)
	}
}

// --- ParseKillPane benchmarks ---

func BenchmarkParseKillPane(b *testing.B) {
	args := []string{"-t", "%42"}
	for b.Loop() {
		ParseKillPane(args)
	}
}

// --- ParseNewWindow benchmarks ---

func BenchmarkParseNewWindow_AllFlags(b *testing.B) {
	args := []string{"-t", "session:0", "-n", "my-window", "-P", "-F", "#{window_id}"}
	for b.Loop() {
		ParseNewWindow(args)
	}
}

// --- ParseShowOptions benchmarks ---

func BenchmarkParseShowOptions(b *testing.B) {
	args := []string{"-g", "prefix"}
	for b.Loop() {
		ParseShowOptions(args)
	}
}
