package tmux

import (
	"reflect"
	"testing"
)

// TC-P001: グローバルオプション -L <socket> のパース
func TestParseGlobal(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantSocket  string
		wantPath    string
		wantCommand string
		wantArgs    []string
		wantVersion bool
		wantErr     bool
	}{
		// TC-P001a
		{"L with split-window", []string{"-L", "claude-swarm", "split-window", "-h"},
			"claude-swarm", "", "split-window", []string{"-h"}, false, false},
		// TC-P001b
		{"L with send-keys", []string{"-L", "my-socket", "send-keys", "-t", "%0", "ls", "Enter"},
			"my-socket", "", "send-keys", []string{"-t", "%0", "ls", "Enter"}, false, false},
		// TC-P001c
		{"no L", []string{"split-window", "-h"},
			"", "", "split-window", []string{"-h"}, false, false},
		// TC-P001d
		{"L no value", []string{"-L"},
			"", "", "", nil, false, true},
		// TC-P001e
		{"version flag", []string{"-V"},
			"", "", "", nil, true, false},
		// TC-P001f
		{"split-window -v", []string{"split-window", "-v"},
			"", "", "split-window", []string{"-v"}, false, false},
		// TC-P001g
		{"version before command", []string{"-V", "split-window"},
			"", "", "", nil, true, false},
		// TC-P001h
		{"S option", []string{"-S", "/tmp/my.sock", "split-window"},
			"", "/tmp/my.sock", "split-window", nil, false, false},
		// TC-P001i
		{"S no value", []string{"-S"},
			"", "", "", nil, false, true},
		// Additional: -f option
		{"f option", []string{"-f", "/etc/tmux.conf", "split-window", "-h"},
			"", "", "split-window", []string{"-h"}, false, false},
		// Additional: -f no value
		{"f no value", []string{"-f"},
			"", "", "", nil, false, true},
		// Additional: no args at all
		{"no args", []string{},
			"", "", "", nil, false, true},
		// Additional: combined global options
		{"L and f", []string{"-L", "mysock", "-f", "/etc/tmux.conf", "send-keys", "hello"},
			"mysock", "", "send-keys", []string{"hello"}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantVersion {
				if !result.Global.Version {
					t.Error("expected version flag")
				}
				return
			}
			if result.Global.SocketName != tt.wantSocket {
				t.Errorf("socket name = %q, want %q", result.Global.SocketName, tt.wantSocket)
			}
			if result.Global.SocketPath != tt.wantPath {
				t.Errorf("socket path = %q, want %q", result.Global.SocketPath, tt.wantPath)
			}
			if result.Command != tt.wantCommand {
				t.Errorf("command = %q, want %q", result.Command, tt.wantCommand)
			}
			if tt.wantArgs != nil {
				if !reflect.DeepEqual(result.Args, tt.wantArgs) {
					t.Errorf("args = %v, want %v", result.Args, tt.wantArgs)
				}
			}
		})
	}
}

// TC-P002: ターゲット指定 -t のパース（コマンド固有パーサ経由）
func TestParseSplitWindowTarget(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantTarget string
		wantErr    bool
	}{
		// TC-P002a
		{"pane id %0", []string{"split-window", "-t", "%0"},
			"%0", false},
		// TC-P002b
		{"pane id %123", []string{"split-window", "-t", "%123"},
			"%123", false},
		// TC-P002c
		{"session name", []string{"split-window", "-t", "session_name"},
			"session_name", false},
		// TC-P002d
		{"complex session name", []string{"split-window", "-t", "shimux_worktree-feature_new"},
			"shimux_worktree-feature_new", false},
		// TC-P002e
		{"t no value", []string{"split-window", "-t"},
			"", true},
		// TC-P002f
		{"session:window.pane", []string{"split-window", "-t", "shimux:0.1"},
			"shimux:0.1", false},
		// TC-P002g
		{"session:window", []string{"split-window", "-t", "shimux:0"},
			"shimux:0", false},
		// TC-P002h
		{"no session, window.pane", []string{"split-window", "-t", ":0.1"},
			":0.1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.args)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			sw, err := ParseSplitWindow(result.Args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error from ParseSplitWindow, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSplitWindow error: %v", err)
			}
			if sw.Target != tt.wantTarget {
				t.Errorf("target = %q, want %q", sw.Target, tt.wantTarget)
			}
		})
	}
}

// TC-P003: -P -F 出力フラグのパース
func TestParseSplitWindowPrintFormat(t *testing.T) {
	tests := []struct {
		name           string
		cmdArgs        []string
		wantPrintAfter bool
		wantFormat     string
	}{
		// TC-P003a
		{"P and F", []string{"-P", "-F", "#{pane_id}"},
			true, "#{pane_id}"},
		// TC-P003b
		{"P and F compound", []string{"-P", "-F", "#{pane_id}:#{pane_width}x#{pane_height}"},
			true, "#{pane_id}:#{pane_width}x#{pane_height}"},
		// TC-P003c
		{"P without F", []string{"-P"},
			true, ""},
		// TC-P003d
		{"F without P", []string{"-F", "#{pane_id}"},
			false, "#{pane_id}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw, err := ParseSplitWindow(tt.cmdArgs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sw.PrintAfter != tt.wantPrintAfter {
				t.Errorf("printAfter = %v, want %v", sw.PrintAfter, tt.wantPrintAfter)
			}
			if sw.Format != tt.wantFormat {
				t.Errorf("format = %q, want %q", sw.Format, tt.wantFormat)
			}
		})
	}
}

// TC-P006b: 相反フラグの後勝ち（split-window -h/-v）
func TestParseSplitWindowLastWins(t *testing.T) {
	tests := []struct {
		name           string
		cmdArgs        []string
		wantHorizontal bool
	}{
		// TC-P006b: -h -v → -v wins (vertical split = down)
		{"h then v", []string{"-h", "-v"}, false},
		// -v -h → -h wins (horizontal split = right)
		{"v then h", []string{"-v", "-h"}, true},
		// single -h
		{"h only", []string{"-h"}, true},
		// single -v
		{"v only", []string{"-v"}, false},
		// no flag (default is vertical = down, like tmux)
		{"no flag", []string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw, err := ParseSplitWindow(tt.cmdArgs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sw.Horizontal != tt.wantHorizontal {
				t.Errorf("horizontal = %v, want %v", sw.Horizontal, tt.wantHorizontal)
			}
		})
	}
}

// TC-P006c,d,e,f: send-keys パーサ
func TestParseSendKeys(t *testing.T) {
	tests := []struct {
		name        string
		cmdArgs     []string
		wantTarget  string
		wantLiteral bool
		wantKeys    []string
		wantErr     bool
	}{
		// TC-P006c: quotes are passed through
		{"quoted text", []string{"echo", "'hello world'"},
			"", false, []string{"echo", "'hello world'"}, false},
		// TC-P006d: double quotes
		{"double quoted", []string{"echo", "\"nested\""},
			"", false, []string{"echo", "\"nested\""}, false},
		// TC-P006e: -t with send-keys
		{"t with target", []string{"-t", "%0", "ls -la", "Enter"},
			"%0", false, []string{"ls -la", "Enter"}, false},
		// TC-P006f: empty text
		{"empty text", []string{""},
			"", false, []string{""}, false},
		// -l literal mode
		{"literal flag", []string{"-l", "Enter"},
			"", true, []string{"Enter"}, false},
		// -l and -t combined
		{"literal and target", []string{"-l", "-t", "%0", "Enter"},
			"%0", true, []string{"Enter"}, false},
		// -t and -l combined (order reversed)
		{"target and literal", []string{"-t", "%0", "-l", "Enter"},
			"%0", true, []string{"Enter"}, false},
		// -t missing value
		{"t no value", []string{"-t"},
			"", false, nil, true},
		// no args at all (send-keys with no keys)
		{"no args", []string{},
			"", false, nil, false},
		// -- separator
		{"double dash separator", []string{"-t", "%0", "--", "-l", "not a flag"},
			"%0", false, []string{"-l", "not a flag"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sk, err := ParseSendKeys(tt.cmdArgs)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sk.Target != tt.wantTarget {
				t.Errorf("target = %q, want %q", sk.Target, tt.wantTarget)
			}
			if sk.Literal != tt.wantLiteral {
				t.Errorf("literal = %v, want %v", sk.Literal, tt.wantLiteral)
			}
			if tt.wantKeys != nil {
				if !reflect.DeepEqual(sk.Keys, tt.wantKeys) {
					t.Errorf("keys = %v, want %v", sk.Keys, tt.wantKeys)
				}
			}
		})
	}
}

// TC-P006g: select-pane パーサ
func TestParseSelectPane(t *testing.T) {
	tests := []struct {
		name       string
		cmdArgs    []string
		wantTarget string
		wantStyle  string
		wantTitle  string
	}{
		{"target only", []string{"-t", "%0"},
			"%0", "", ""},
		// TC-P006g: complex select-pane with -P and -T
		{"target style title", []string{"-t", "%0", "-P", "bg=#282828", "-T", "agent-1"},
			"%0", "bg=#282828", "agent-1"},
		{"no args", []string{},
			"", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sp, err := ParseSelectPane(tt.cmdArgs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sp.Target != tt.wantTarget {
				t.Errorf("target = %q, want %q", sp.Target, tt.wantTarget)
			}
			if sp.Style != tt.wantStyle {
				t.Errorf("style = %q, want %q", sp.Style, tt.wantStyle)
			}
			if sp.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", sp.Title, tt.wantTitle)
			}
		})
	}
}

// TC-P007: new-window の -n オプションパース
func TestParseNewWindow(t *testing.T) {
	tests := []struct {
		name           string
		cmdArgs        []string
		wantTarget     string
		wantName       string
		wantPrintAfter bool
		wantFormat     string
		wantErr        bool
	}{
		// TC-P007a
		{"n with name", []string{"-n", "work"},
			"", "work", false, "", false},
		// TC-P007b
		{"n with name and P F", []string{"-n", "teammate-agent1", "-P", "-F", "#{pane_id}"},
			"", "teammate-agent1", true, "#{pane_id}", false},
		// TC-P007c
		{"n no value", []string{"-n"},
			"", "", false, "", true},
		// with target
		{"t and n", []string{"-t", "mysession", "-n", "work"},
			"mysession", "work", false, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nw, err := ParseNewWindow(tt.cmdArgs)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if nw.Target != tt.wantTarget {
				t.Errorf("target = %q, want %q", nw.Target, tt.wantTarget)
			}
			if nw.Name != tt.wantName {
				t.Errorf("name = %q, want %q", nw.Name, tt.wantName)
			}
			if nw.PrintAfter != tt.wantPrintAfter {
				t.Errorf("printAfter = %v, want %v", nw.PrintAfter, tt.wantPrintAfter)
			}
			if nw.Format != tt.wantFormat {
				t.Errorf("format = %q, want %q", nw.Format, tt.wantFormat)
			}
		})
	}
}

// TC-P006a: 空コマンド名のエラー
func TestParseEmptyCommand(t *testing.T) {
	_, err := Parse([]string{""})
	if err == nil {
		t.Fatal("expected error for empty command name, got nil")
	}
}

// TC-P007 related: -- separator stops option parsing
func TestParseSendKeysDoubleDash(t *testing.T) {
	sk, err := ParseSendKeys([]string{"-t", "%1", "--", "-l", "Enter"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sk.Target != "%1" {
		t.Errorf("target = %q, want %%1", sk.Target)
	}
	// After --, "-l" should be treated as a key, not a flag
	if sk.Literal {
		t.Error("literal should be false; -l after -- is a key argument")
	}
	want := []string{"-l", "Enter"}
	if !reflect.DeepEqual(sk.Keys, want) {
		t.Errorf("keys = %v, want %v", sk.Keys, want)
	}
}

// split-window with -l (size) option - recognized but value captured
func TestParseSplitWindowSize(t *testing.T) {
	sw, err := ParseSplitWindow([]string{"-h", "-t", "%0", "-l", "70%"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sw.Horizontal {
		t.Error("expected horizontal = true")
	}
	if sw.Target != "%0" {
		t.Errorf("target = %q, want %%0", sw.Target)
	}
	if sw.Size != "70%" {
		t.Errorf("size = %q, want 70%%", sw.Size)
	}
}

// shimux-specific direction flags
func TestParseSplitWindowDirectionFlags(t *testing.T) {
	tests := []struct {
		name          string
		cmdArgs       []string
		wantDirection string
	}{
		{"--left", []string{"--left"}, "left"},
		{"--right", []string{"--right"}, "right"},
		{"--up", []string{"--up"}, "up"},
		{"--down", []string{"--down"}, "down"},
		{"-h (right)", []string{"-h"}, "right"},
		{"-v (down)", []string{"-v"}, "down"},
		{"default (down)", []string{}, "down"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sw, err := ParseSplitWindow(tt.cmdArgs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sw.Direction != tt.wantDirection {
				t.Errorf("direction = %q, want %q", sw.Direction, tt.wantDirection)
			}
		})
	}
}

// Full parse integration: global options + command-specific parse
func TestParseFullIntegration(t *testing.T) {
	// Split-window with global -L and command-specific -h -t -P -F
	result, err := Parse([]string{"-L", "claude-swarm", "split-window", "-h", "-t", "%0", "-P", "-F", "#{pane_id}"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Global.SocketName != "claude-swarm" {
		t.Errorf("socket = %q, want claude-swarm", result.Global.SocketName)
	}
	if result.Command != "split-window" {
		t.Errorf("command = %q, want split-window", result.Command)
	}
	sw, err := ParseSplitWindow(result.Args)
	if err != nil {
		t.Fatalf("ParseSplitWindow error: %v", err)
	}
	if !sw.Horizontal {
		t.Error("expected horizontal = true")
	}
	if sw.Target != "%0" {
		t.Errorf("target = %q, want %%0", sw.Target)
	}
	if !sw.PrintAfter {
		t.Error("expected printAfter = true")
	}
	if sw.Format != "#{pane_id}" {
		t.Errorf("format = %q, want #{pane_id}", sw.Format)
	}
}
