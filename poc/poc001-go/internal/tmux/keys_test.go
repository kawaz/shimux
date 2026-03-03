package tmux

import "testing"

func TestExpandSpecialKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// TC-P005a~TC-P005am: Original tests (must not break)
		{"TC-P005a Enter", "Enter", "\r"},
		{"TC-P005b Escape", "Escape", "\x1b"},
		{"TC-P005c Tab", "Tab", "\t"},
		{"TC-P005d Space", "Space", " "},
		{"TC-P005e BSpace", "BSpace", "\x7f"},
		{"TC-P005f C-c", "C-c", "\x03"},
		{"TC-P005g C-d", "C-d", "\x04"},
		{"TC-P005h C-z", "C-z", "\x1a"},
		{"TC-P005i C-l", "C-l", "\x0c"},
		{"TC-P005j C-a", "C-a", "\x01"},
		{"TC-P005k C-e", "C-e", "\x05"},
		{"TC-P005l C-k", "C-k", "\x0b"},
		{"TC-P005m C-u", "C-u", "\x15"},
		{"TC-P005n C-w", "C-w", "\x17"},
		{"TC-P005o Up", "Up", "\x1b[A"},
		{"TC-P005p Down", "Down", "\x1b[B"},
		{"TC-P005q Right", "Right", "\x1b[C"},
		{"TC-P005r Left", "Left", "\x1b[D"},
		{"TC-P005s plain text", "hello", "hello"},
		{"TC-P005t BTab", "BTab", "\x1b[Z"},
		{"TC-P005u DC", "DC", "\x1b[3~"},
		{"TC-P005v End", "End", "\x1b[F"},
		{"TC-P005w Home", "Home", "\x1b[H"},
		{"TC-P005x IC", "IC", "\x1b[2~"},
		{"TC-P005y NPage", "NPage", "\x1b[6~"},
		{"TC-P005z PPage", "PPage", "\x1b[5~"},
		{"TC-P005aa F1", "F1", "\x1bOP"},
		{"TC-P005ab F2", "F2", "\x1bOQ"},
		{"TC-P005ac F3", "F3", "\x1bOR"},
		{"TC-P005ad F4", "F4", "\x1bOS"},
		{"TC-P005ae F5", "F5", "\x1b[15~"},
		{"TC-P005af F6", "F6", "\x1b[17~"},
		{"TC-P005ag F7", "F7", "\x1b[18~"},
		{"TC-P005ah F8", "F8", "\x1b[19~"},
		{"TC-P005ai F9", "F9", "\x1b[20~"},
		{"TC-P005aj F10", "F10", "\x1b[21~"},
		{"TC-P005ak F11", "F11", "\x1b[23~"},
		{"TC-P005al F12", "F12", "\x1b[24~"},
		{"TC-P005am empty", "", ""},

		// --- New tests: Aliases ---
		{"alias Insert", "Insert", "\x1b[2~"},
		{"alias Delete", "Delete", "\x1b[3~"},
		{"alias PageDown", "PageDown", "\x1b[6~"},
		{"alias PgDn", "PgDn", "\x1b[6~"},
		{"alias PageUp", "PageUp", "\x1b[5~"},
		{"alias PgUp", "PgUp", "\x1b[5~"},

		// --- New tests: C0 control characters ---
		{"C0 [NUL]", "[NUL]", "\x00"},
		{"C0 [SOH]", "[SOH]", "\x01"},
		{"C0 [STX]", "[STX]", "\x02"},
		{"C0 [ETX]", "[ETX]", "\x03"},
		{"C0 [EOT]", "[EOT]", "\x04"},
		{"C0 [ENQ]", "[ENQ]", "\x05"},
		{"C0 [ASC]", "[ASC]", "\x06"},
		{"C0 [BEL]", "[BEL]", "\x07"},
		{"C0 [BS]", "[BS]", "\x08"},
		{"C0 [LF]", "[LF]", "\x0a"},
		{"C0 [VT]", "[VT]", "\x0b"},
		{"C0 [FF]", "[FF]", "\x0c"},
		{"C0 [SO]", "[SO]", "\x0e"},
		{"C0 [SI]", "[SI]", "\x0f"},
		{"C0 [DLE]", "[DLE]", "\x10"},
		{"C0 [DC1]", "[DC1]", "\x11"},
		{"C0 [DC2]", "[DC2]", "\x12"},
		{"C0 [DC3]", "[DC3]", "\x13"},
		{"C0 [DC4]", "[DC4]", "\x14"},
		{"C0 [NAK]", "[NAK]", "\x15"},
		{"C0 [SYN]", "[SYN]", "\x16"},
		{"C0 [ETB]", "[ETB]", "\x17"},
		{"C0 [CAN]", "[CAN]", "\x18"},
		{"C0 [EM]", "[EM]", "\x19"},
		{"C0 [SUB]", "[SUB]", "\x1a"},
		{"C0 [FS]", "[FS]", "\x1c"},
		{"C0 [GS]", "[GS]", "\x1d"},
		{"C0 [RS]", "[RS]", "\x1e"},
		{"C0 [US]", "[US]", "\x1f"},

		// --- New tests: Numeric keypad ---
		{"KP0", "KP0", "0"},
		{"KP1", "KP1", "1"},
		{"KP2", "KP2", "2"},
		{"KP3", "KP3", "3"},
		{"KP4", "KP4", "4"},
		{"KP5", "KP5", "5"},
		{"KP6", "KP6", "6"},
		{"KP7", "KP7", "7"},
		{"KP8", "KP8", "8"},
		{"KP9", "KP9", "9"},
		{"KP/", "KP/", "/"},
		{"KP*", "KP*", "*"},
		{"KP-", "KP-", "-"},
		{"KP+", "KP+", "+"},
		{"KP.", "KP.", "."},
		{"KPEnter", "KPEnter", "\r"},

		// --- New tests: Extended C-x patterns ---
		{"C-A (uppercase)", "C-A", "\x01"},
		{"C-Z (uppercase)", "C-Z", "\x1a"},
		{"C-@ (NUL)", "C-@", "\x00"},

		// --- New tests: Meta (M-x) modifier ---
		{"M-a", "M-a", "\x1ba"},
		{"M-z", "M-z", "\x1bz"},
		{"M-A", "M-A", "\x1bA"},
		{"M-1", "M-1", "\x1b1"},

		// --- New tests: Shift (S-x) modifier ---
		{"S-a becomes A", "S-a", "A"},
		{"S-z becomes Z", "S-z", "Z"},

		// --- New tests: Combined modifiers on single chars ---
		{"C-M-a", "C-M-a", "\x1b\x01"},
		{"C-M-c", "C-M-c", "\x1b\x03"},
		{"M-C-a", "M-C-a", "\x1b\x01"}, // order-independent

		// --- New tests: Modified special keys (xterm-style) ---
		// Ctrl + arrow: modifier param = 5
		{"C-Up", "C-Up", "\x1b[1;5A"},
		{"C-Down", "C-Down", "\x1b[1;5B"},
		{"C-Right", "C-Right", "\x1b[1;5C"},
		{"C-Left", "C-Left", "\x1b[1;5D"},

		// Shift + arrow: modifier param = 2
		{"S-Up", "S-Up", "\x1b[1;2A"},
		{"S-Down", "S-Down", "\x1b[1;2B"},
		{"S-Right", "S-Right", "\x1b[1;2C"},
		{"S-Left", "S-Left", "\x1b[1;2D"},

		// Meta + arrow: modifier param = 3
		{"M-Up", "M-Up", "\x1b[1;3A"},
		{"M-Down", "M-Down", "\x1b[1;3B"},
		{"M-Right", "M-Right", "\x1b[1;3C"},
		{"M-Left", "M-Left", "\x1b[1;3D"},

		// Ctrl+Shift + arrow: modifier param = 6
		{"C-S-Up", "C-S-Up", "\x1b[1;6A"},
		{"C-S-Right", "C-S-Right", "\x1b[1;6C"},

		// Ctrl+Meta + arrow: modifier param = 7
		{"C-M-Up", "C-M-Up", "\x1b[1;7A"},
		{"C-M-Left", "C-M-Left", "\x1b[1;7D"},

		// Meta+Shift: modifier param = 4
		{"M-S-Up", "M-S-Up", "\x1b[1;4A"},

		// All three: modifier param = 8
		{"C-M-S-Up", "C-M-S-Up", "\x1b[1;8A"},

		// Modified Home/End
		{"C-Home", "C-Home", "\x1b[1;5H"},
		{"C-End", "C-End", "\x1b[1;5F"},
		{"S-Home", "S-Home", "\x1b[1;2H"},
		{"S-End", "S-End", "\x1b[1;2F"},

		// Modified page keys (tilde-style)
		{"C-PPage", "C-PPage", "\x1b[5;5~"},
		{"C-NPage", "C-NPage", "\x1b[6;5~"},
		{"S-PPage", "S-PPage", "\x1b[5;2~"},
		{"S-NPage", "S-NPage", "\x1b[6;2~"},
		{"C-PageUp", "C-PageUp", "\x1b[5;5~"},
		{"C-PageDown", "C-PageDown", "\x1b[6;5~"},

		// Modified IC/DC
		{"C-IC", "C-IC", "\x1b[2;5~"},
		{"C-DC", "C-DC", "\x1b[3;5~"},
		{"S-Insert", "S-Insert", "\x1b[2;2~"},
		{"S-Delete", "S-Delete", "\x1b[3;2~"},

		// Modified function keys (F1-F4: SS3 -> CSI conversion)
		{"C-F1", "C-F1", "\x1b[1;5P"},
		{"C-F2", "C-F2", "\x1b[1;5Q"},
		{"S-F1", "S-F1", "\x1b[1;2P"},
		{"M-F1", "M-F1", "\x1b[1;3P"},
		{"S-F4", "S-F4", "\x1b[1;2S"},

		// Modified function keys (F5-F12: tilde-style)
		{"C-F5", "C-F5", "\x1b[15;5~"},
		{"S-F5", "S-F5", "\x1b[15;2~"},
		{"M-F5", "M-F5", "\x1b[15;3~"},
		{"C-F12", "C-F12", "\x1b[24;5~"},
		{"S-F12", "S-F12", "\x1b[24;2~"},

		// --- Case-insensitive modifier prefixes ---
		{"lowercase c- prefix", "c-a", "\x01"},
		{"lowercase m- prefix", "m-a", "\x1ba"},
		{"lowercase s- prefix", "s-a", "A"},

		// --- Edge cases ---
		{"M-Enter", "M-Enter", "\x1b\r"},
		{"M-Escape", "M-Escape", "\x1b\x1b"},
		{"M-Space", "M-Space", "\x1b "},
		{"M-Tab", "M-Tab", "\x1b\t"},
		{"unknown key passthrough", "XYZ", "XYZ"},
		{"C-unknown multi-char", "C-XYZ", "C-XYZ"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandSpecialKey(tt.input)
			if got != tt.expected {
				t.Errorf("ExpandSpecialKey(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildSendKeysData(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		literal  bool
		expected string
	}{
		// TC-C002a
		{"single text", []string{"echo hello"}, false, "echo hello"},
		// TC-C002b
		{"text + Enter", []string{"echo hello", "Enter"}, false, "echo hello\r"},
		// TC-C002d
		{"multiple args concat", []string{"echo", " ", "hello"}, false, "echo hello"},
		// TC-C002e
		{"multiple args + Enter", []string{"echo", " ", "hello", " ", "world", "Enter"}, false, "echo hello world\r"},
		// TC-C002f
		{"C-c only", []string{"C-c"}, false, "\x03"},
		// TC-C002g
		{"empty args", []string{}, false, ""},
		// TC-C002k literal mode
		{"literal Enter", []string{"Enter"}, true, "Enter"},
		// TC-C002m literal + text
		{"literal mixed", []string{"Enter", "Space"}, true, "EnterSpace"},

		// New: Modified keys in BuildSendKeysData
		{"C-Up in sequence", []string{"C-Up"}, false, "\x1b[1;5A"},
		{"M-a in sequence", []string{"M-a"}, false, "\x1ba"},
		{"mixed special and modified", []string{"echo", " ", "hello", "Enter", "C-c"}, false, "echo hello\r\x03"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSendKeysData(tt.args, tt.literal)
			if got != tt.expected {
				t.Errorf("BuildSendKeysData(%v, %v) = %q, want %q", tt.args, tt.literal, got, tt.expected)
			}
		})
	}
}

func TestParseModifiers(t *testing.T) {
	tests := []struct {
		name                          string
		input                         string
		wantCtrl, wantMeta, wantShift bool
		wantBase                      string
	}{
		{"no modifiers", "Up", false, false, false, "Up"},
		{"C- prefix", "C-a", true, false, false, "a"},
		{"M- prefix", "M-a", false, true, false, "a"},
		{"S- prefix", "S-a", false, false, true, "a"},
		{"C-M- prefix", "C-M-a", true, true, false, "a"},
		{"C-S- prefix", "C-S-a", true, false, true, "a"},
		{"M-S- prefix", "M-S-a", false, true, true, "a"},
		{"C-M-S- prefix", "C-M-S-a", true, true, true, "a"},
		{"M-C- order", "M-C-a", true, true, false, "a"},
		{"lowercase c-", "c-a", true, false, false, "a"},
		{"lowercase m-", "m-a", false, true, false, "a"},
		{"lowercase s-", "s-a", false, false, true, "a"},
		{"C-Up", "C-Up", true, false, false, "Up"},
		{"M-F1", "M-F1", false, true, false, "F1"},
		{"S-Left", "S-Left", false, false, true, "Left"},
		{"no dash after letter", "Hello", false, false, false, "Hello"},
		{"single char", "a", false, false, false, "a"},
		{"empty string", "", false, false, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl, meta, shift, base := parseModifiers(tt.input)
			if ctrl != tt.wantCtrl || meta != tt.wantMeta || shift != tt.wantShift || base != tt.wantBase {
				t.Errorf("parseModifiers(%q) = (%v, %v, %v, %q), want (%v, %v, %v, %q)",
					tt.input, ctrl, meta, shift, base,
					tt.wantCtrl, tt.wantMeta, tt.wantShift, tt.wantBase)
			}
		})
	}
}

func TestXtermModParam(t *testing.T) {
	tests := []struct {
		name              string
		ctrl, meta, shift bool
		want              int
	}{
		{"no modifiers", false, false, false, 1},
		{"shift only", false, false, true, 2},
		{"meta only", false, true, false, 3},
		{"shift+meta", false, true, true, 4},
		{"ctrl only", true, false, false, 5},
		{"shift+ctrl", true, false, true, 6},
		{"meta+ctrl", true, true, false, 7},
		{"shift+meta+ctrl", true, true, true, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xtermModParam(tt.ctrl, tt.meta, tt.shift)
			if got != tt.want {
				t.Errorf("xtermModParam(%v, %v, %v) = %d, want %d",
					tt.ctrl, tt.meta, tt.shift, got, tt.want)
			}
		})
	}
}
