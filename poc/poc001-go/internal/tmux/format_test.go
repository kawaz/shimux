package tmux

import "testing"

func TestExpandFormat(t *testing.T) {
	ctx := &FormatContext{
		SessionName: "shimux",
		WindowIndex: "0",
		WindowID:    "@0",
		WindowName:  "ghostty",
		SessionID:   "$0",
		PaneID:      "%0",
		PaneIndex:   "0",
		PanePID:     12345,
		PaneTTY:     "/dev/ttys005",
		PanePath:    "/home/user",
		PaneWidth:   80,
		PaneHeight:  24,
		PaneActive:  true,
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		// TC-P004a
		{"pane_id", "#{pane_id}", "%0"},
		// TC-P004b
		{"pane_width", "#{pane_width}", "80"},
		// TC-P004c
		{"pane_height", "#{pane_height}", "24"},
		// TC-P004d
		{"pane_active", "#{pane_active}", "1"},
		// TC-P004e
		{"pane_current_path", "#{pane_current_path}", "/home/user"},
		// TC-P004f
		{"pane_pid", "#{pane_pid}", "12345"},
		// TC-P004g
		{"session_name", "#{session_name}", "shimux"},
		// TC-P004h
		{"window_index", "#{window_index}", "0"},
		// TC-P004i
		{"window_id", "#{window_id}", "@0"},
		// TC-P004j
		{"session_id", "#{session_id}", "$0"},
		// TC-P004k
		{"window_name", "#{window_name}", "ghostty"},
		// TC-P004q
		{"pane_tty", "#{pane_tty}", "/dev/ttys005"},
		// TC-P004r
		{"pane_index", "#{pane_index}", "0"},
		// TC-P004l composite
		{"composite", "#{pane_id}:#{pane_width}x#{pane_height}:#{pane_active}", "%0:80x24:1"},
		// TC-P004m
		{"prefix text", "session=#{session_name}", "session=shimux"},
		// TC-P004n
		{"plain text", "プレーンテキスト", "プレーンテキスト"},
		// TC-P004o
		{"unknown var", "#{unknown_variable}", "#{unknown_variable}"},
		// TC-P004p
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandFormat(tt.template, ctx)
			if got != tt.expected {
				t.Errorf("ExpandFormat(%q) = %q, want %q", tt.template, got, tt.expected)
			}
		})
	}
}

func TestExpandFormatInactive(t *testing.T) {
	ctx := &FormatContext{PaneActive: false}
	got := ExpandFormat("#{pane_active}", ctx)
	if got != "0" {
		t.Errorf("inactive pane_active = %q, want %q", got, "0")
	}
}
