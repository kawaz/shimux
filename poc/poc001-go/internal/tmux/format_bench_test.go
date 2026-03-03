package tmux

import "testing"

// --- ExpandFormat benchmarks ---

func BenchmarkExpandFormat_SingleVar(b *testing.B) {
	ctx := &FormatContext{
		SessionName: "my-session",
		PaneID:      "%0",
	}
	for b.Loop() {
		ExpandFormat("#{session_name}", ctx)
	}
}

func BenchmarkExpandFormat_MultipleVars(b *testing.B) {
	ctx := &FormatContext{
		SessionName: "my-session",
		SessionID:   "$0",
		WindowIndex: "0",
		WindowID:    "@0",
		PaneID:      "%0",
		PaneActive:  true,
	}
	template := "#{session_name}:#{window_index}.#{pane_id}:#{pane_active}"
	for b.Loop() {
		ExpandFormat(template, ctx)
	}
}

func BenchmarkExpandFormat_AllVars(b *testing.B) {
	ctx := &FormatContext{
		SessionName: "test-session",
		SessionID:   "$0",
		WindowIndex: "0",
		WindowID:    "@0",
		WindowName:  "main",
		PaneID:      "%5",
		PanePID:     12345,
		PanePath:    "/home/user",
		PaneWidth:   120,
		PaneHeight:  40,
		PaneActive:  true,
	}
	template := "#{session_name} #{session_id} #{window_index} #{window_id} #{window_name} #{pane_id} #{pane_pid} #{pane_current_path} #{pane_width} #{pane_height} #{pane_active}"
	for b.Loop() {
		ExpandFormat(template, ctx)
	}
}

func BenchmarkExpandFormat_NoVars(b *testing.B) {
	ctx := &FormatContext{}
	for b.Loop() {
		ExpandFormat("plain text without any variables", ctx)
	}
}

func BenchmarkExpandFormat_UnknownVars(b *testing.B) {
	ctx := &FormatContext{}
	template := "#{unknown_var1} #{unknown_var2} #{unknown_var3}"
	for b.Loop() {
		ExpandFormat(template, ctx)
	}
}

func BenchmarkExpandFormat_MixedTextAndVars(b *testing.B) {
	ctx := &FormatContext{
		SessionName: "session",
		PaneID:      "%0",
		PaneWidth:   120,
		PaneHeight:  40,
	}
	template := "Session: #{session_name} | Pane: #{pane_id} (#{pane_width}x#{pane_height})"
	for b.Loop() {
		ExpandFormat(template, ctx)
	}
}

func BenchmarkExpandFormat_LongTemplate(b *testing.B) {
	ctx := &FormatContext{
		SessionName: "test-session",
		PaneID:      "%0",
		PaneActive:  true,
	}
	// Build a long template with many repeated variables
	template := ""
	for i := 0; i < 100; i++ {
		template += "#{session_name}:#{pane_id} "
	}
	b.ResetTimer()
	for b.Loop() {
		ExpandFormat(template, ctx)
	}
}

func BenchmarkExpandFormat_UnclosedBrace(b *testing.B) {
	ctx := &FormatContext{}
	// Malformed template: unclosed #{
	template := "hello #{session_name} world #{unclosed"
	for b.Loop() {
		ExpandFormat(template, ctx)
	}
}
