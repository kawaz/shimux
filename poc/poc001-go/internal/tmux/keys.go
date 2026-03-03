package tmux

import (
	"fmt"
	"strings"
)

// specialKeys maps tmux special key names to their escape sequences.
// Includes tmux-compatible aliases (e.g. Insert=IC, Delete=DC, PageDown=NPage, etc.).
var specialKeys = map[string]string{
	// Basic control characters
	"Enter":  "\r",
	"Escape": "\x1b",
	"Tab":    "\t",
	"Space":  " ",
	"BSpace": "\x7f",
	"BTab":   "\x1b[Z",

	// Arrow keys
	"Up":    "\x1b[A",
	"Down":  "\x1b[B",
	"Right": "\x1b[C",
	"Left":  "\x1b[D",

	// Navigation keys
	"DC":   "\x1b[3~",
	"End":  "\x1b[F",
	"Home": "\x1b[H",
	"IC":   "\x1b[2~",

	// Page keys
	"NPage": "\x1b[6~",
	"PPage": "\x1b[5~",

	// tmux aliases for navigation/page keys
	"Insert":   "\x1b[2~",
	"Delete":   "\x1b[3~",
	"PageDown": "\x1b[6~",
	"PgDn":     "\x1b[6~",
	"PageUp":   "\x1b[5~",
	"PgUp":     "\x1b[5~",

	// Function keys (F1-F12)
	"F1":  "\x1bOP",
	"F2":  "\x1bOQ",
	"F3":  "\x1bOR",
	"F4":  "\x1bOS",
	"F5":  "\x1b[15~",
	"F6":  "\x1b[17~",
	"F7":  "\x1b[18~",
	"F8":  "\x1b[19~",
	"F9":  "\x1b[20~",
	"F10": "\x1b[21~",
	"F11": "\x1b[23~",
	"F12": "\x1b[24~",

	// C0 control characters (tmux [XXX] notation)
	"[NUL]": "\x00",
	"[SOH]": "\x01",
	"[STX]": "\x02",
	"[ETX]": "\x03",
	"[EOT]": "\x04",
	"[ENQ]": "\x05",
	"[ASC]": "\x06", // ACK in standard, tmux uses ASC
	"[BEL]": "\x07",
	"[BS]":  "\x08",
	"[LF]":  "\x0a",
	"[VT]":  "\x0b",
	"[FF]":  "\x0c",
	"[SO]":  "\x0e",
	"[SI]":  "\x0f",
	"[DLE]": "\x10",
	"[DC1]": "\x11",
	"[DC2]": "\x12",
	"[DC3]": "\x13",
	"[DC4]": "\x14",
	"[NAK]": "\x15",
	"[SYN]": "\x16",
	"[ETB]": "\x17",
	"[CAN]": "\x18",
	"[EM]":  "\x19",
	"[SUB]": "\x1a",
	"[FS]":  "\x1c",
	"[GS]":  "\x1d",
	"[RS]":  "\x1e",
	"[US]":  "\x1f",

	// Numeric keypad
	"KP/":     "/",
	"KP*":     "*",
	"KP-":     "-",
	"KP+":     "+",
	"KP.":     ".",
	"KPEnter": "\r",
	"KP0":     "0",
	"KP1":     "1",
	"KP2":     "2",
	"KP3":     "3",
	"KP4":     "4",
	"KP5":     "5",
	"KP6":     "6",
	"KP7":     "7",
	"KP8":     "8",
	"KP9":     "9",
}

// modifiableKeys defines keys that support xterm-style modifier parameters.
// Each entry maps a key name to its base CSI sequence components:
//   - prefix: the part before the modifier parameter
//   - suffix: the final character(s)
//   - paramStyle: "csi_tilde" for CSI n ; mod ~ format, "csi_letter" for CSI 1 ; mod X format,
//     "ss3_to_csi" for SS3 X -> CSI 1 ; mod X conversion
type modifiableKeyInfo struct {
	// For csi_tilde: ESC [ <num> ; <mod> ~
	// For csi_letter: ESC [ 1 ; <mod> <letter>
	// For ss3_to_csi: ESC [ 1 ; <mod> <letter> (converted from SS3)
	paramStyle string
	num        string // numeric parameter for tilde-style
	letter     byte   // final letter for letter-style and ss3-style
}

var modifiableKeys = map[string]modifiableKeyInfo{
	// Arrow keys: ESC [ 1 ; mod X
	"Up":    {paramStyle: "csi_letter", letter: 'A'},
	"Down":  {paramStyle: "csi_letter", letter: 'B'},
	"Right": {paramStyle: "csi_letter", letter: 'C'},
	"Left":  {paramStyle: "csi_letter", letter: 'D'},

	// Home/End: ESC [ 1 ; mod H/F
	"Home": {paramStyle: "csi_letter", letter: 'H'},
	"End":  {paramStyle: "csi_letter", letter: 'F'},

	// Navigation keys: ESC [ n ; mod ~
	"IC":       {paramStyle: "csi_tilde", num: "2"},
	"Insert":   {paramStyle: "csi_tilde", num: "2"},
	"DC":       {paramStyle: "csi_tilde", num: "3"},
	"Delete":   {paramStyle: "csi_tilde", num: "3"},
	"PPage":    {paramStyle: "csi_tilde", num: "5"},
	"PageUp":   {paramStyle: "csi_tilde", num: "5"},
	"PgUp":     {paramStyle: "csi_tilde", num: "5"},
	"NPage":    {paramStyle: "csi_tilde", num: "6"},
	"PageDown": {paramStyle: "csi_tilde", num: "6"},
	"PgDn":     {paramStyle: "csi_tilde", num: "6"},

	// Function keys F1-F4: SS3 -> CSI 1 ; mod X
	"F1": {paramStyle: "ss3_to_csi", letter: 'P'},
	"F2": {paramStyle: "ss3_to_csi", letter: 'Q'},
	"F3": {paramStyle: "ss3_to_csi", letter: 'R'},
	"F4": {paramStyle: "ss3_to_csi", letter: 'S'},

	// Function keys F5-F12: ESC [ n ; mod ~
	"F5":  {paramStyle: "csi_tilde", num: "15"},
	"F6":  {paramStyle: "csi_tilde", num: "17"},
	"F7":  {paramStyle: "csi_tilde", num: "18"},
	"F8":  {paramStyle: "csi_tilde", num: "19"},
	"F9":  {paramStyle: "csi_tilde", num: "20"},
	"F10": {paramStyle: "csi_tilde", num: "21"},
	"F11": {paramStyle: "csi_tilde", num: "23"},
	"F12": {paramStyle: "csi_tilde", num: "24"},
}

// parseModifiers extracts modifier prefixes (C-, M-, S-) from a key name.
// Returns the modifier flags (ctrl, meta, shift) and the remaining base key name.
// Supports chained modifiers like C-M-x, S-C-x, etc.
// tmux accepts both upper and lower case modifier letters (C-/c-, M-/m-, S-/s-).
func parseModifiers(key string) (ctrl, meta, shift bool, baseKey string) {
	s := key
	for len(s) >= 2 && s[1] == '-' {
		switch s[0] {
		case 'C', 'c':
			ctrl = true
		case 'M', 'm':
			meta = true
		case 'S', 's':
			shift = true
		default:
			return ctrl, meta, shift, s
		}
		s = s[2:]
	}
	return ctrl, meta, shift, s
}

// xtermModParam computes the xterm modifier parameter value.
// xterm encoding: parameter = 1 + sum of active modifiers
//
//	Shift=1, Meta/Alt=2, Ctrl=4
//
// So: Shift=2, Meta=3, Shift+Meta=4, Ctrl=5, Shift+Ctrl=6, Meta+Ctrl=7, Shift+Meta+Ctrl=8
func xtermModParam(ctrl, meta, shift bool) int {
	mod := 1
	if shift {
		mod += 1
	}
	if meta {
		mod += 2
	}
	if ctrl {
		mod += 4
	}
	return mod
}

// buildModifiedSequence generates an xterm-style modified key escape sequence.
func buildModifiedSequence(info modifiableKeyInfo, modParam int) string {
	switch info.paramStyle {
	case "csi_tilde":
		// ESC [ num ; mod ~
		return fmt.Sprintf("\x1b[%s;%d~", info.num, modParam)
	case "csi_letter", "ss3_to_csi":
		// ESC [ 1 ; mod letter
		return fmt.Sprintf("\x1b[1;%d%c", modParam, info.letter)
	}
	return ""
}

// ExpandSpecialKey converts a tmux special key name to its escape sequence.
//
// Processing order:
//  1. Direct lookup in specialKeys map (exact match, no modifiers).
//  2. Parse modifier prefixes (C-, M-, S- and combinations).
//  3. For modified special keys (e.g. C-Up, S-F1), generate xterm-style sequences.
//  4. For C-<letter> (a-z, A-Z), generate the corresponding control character (0x01-0x1A).
//  5. For C-@ generate NUL (0x00).
//  6. For M-<key>, prepend ESC (0x1B) to the expanded base key.
//  7. For S-<letter>, output the uppercase letter.
//  8. If nothing matches, return the input unchanged.
func ExpandSpecialKey(key string) string {
	// 1. Direct lookup (handles unmodified keys and aliases)
	if seq, ok := specialKeys[key]; ok {
		return seq
	}

	// 2. Parse modifier prefixes
	ctrl, meta, shift, baseKey := parseModifiers(key)
	hasModifier := ctrl || meta || shift

	if !hasModifier {
		// No modifiers and not in specialKeys: return as-is
		return key
	}

	// 3. Modified special key with xterm-style parameter
	if info, ok := modifiableKeys[baseKey]; ok {
		modParam := xtermModParam(ctrl, meta, shift)
		return buildModifiedSequence(info, modParam)
	}

	// 4-7. Modified single character or simple key
	expanded := expandBaseKey(baseKey, ctrl, meta, shift)
	return expanded
}

// expandBaseKey handles modifier application to a base key (single char or simple name).
func expandBaseKey(baseKey string, ctrl, meta, shift bool) string {
	var result string

	// Resolve baseKey: check specialKeys first, then treat as literal
	if seq, ok := specialKeys[baseKey]; ok {
		result = seq
	} else if len(baseKey) == 1 {
		ch := baseKey[0]

		// Apply Shift to single letter
		if shift && ch >= 'a' && ch <= 'z' {
			ch = ch - 'a' + 'A'
			shift = false // consumed
		}

		// Apply Ctrl to letter
		if ctrl {
			upper := ch
			if upper >= 'a' && upper <= 'z' {
				upper = upper - 'a' + 'A'
			}
			if upper >= 'A' && upper <= 'Z' {
				result = string(rune(upper - 'A' + 1))
				ctrl = false // consumed
			} else if upper == '@' {
				result = "\x00"
				ctrl = false
			} else {
				// C- with non-letter: return as-is for the char part
				result = string(ch)
			}
		} else {
			result = string(ch)
		}
	} else {
		// Multi-character base key not in specialKeys: cannot apply char-level modifiers
		// Return the original constructed key name
		return rebuildKeyName(ctrl, meta, shift, baseKey)
	}

	// Apply remaining Ctrl (if not consumed by letter handling above)
	// This shouldn't normally happen since Ctrl only applies to letters and @
	// but we handle it for robustness.

	// Apply Meta: prepend ESC
	if meta {
		result = "\x1b" + result
	}

	return result
}

// rebuildKeyName reconstructs a key name with modifier prefixes.
// Used as fallback when the key cannot be resolved to an escape sequence.
func rebuildKeyName(ctrl, meta, shift bool, baseKey string) string {
	var b strings.Builder
	if ctrl {
		b.WriteString("C-")
	}
	if meta {
		b.WriteString("M-")
	}
	if shift {
		b.WriteString("S-")
	}
	b.WriteString(baseKey)
	return b.String()
}

// BuildSendKeysData concatenates the given arguments into a single string
// suitable for writing to a PTY. When literal is false, each argument is
// expanded through ExpandSpecialKey. When literal is true, arguments are
// concatenated as-is without expansion.
func BuildSendKeysData(args []string, literal bool) string {
	var b strings.Builder
	for _, arg := range args {
		if literal {
			b.WriteString(arg)
		} else {
			b.WriteString(ExpandSpecialKey(arg))
		}
	}
	return b.String()
}
