package tmux

import (
	"strconv"
	"strings"
)

// FormatContext holds the values used to expand tmux format variables (#{...}).
type FormatContext struct {
	SessionName string
	SessionID   string
	WindowIndex string
	WindowID    string
	WindowName  string
	PaneID      string
	PaneIndex   string
	PanePID     int
	PaneTTY     string
	PanePath    string
	PaneWidth   int
	PaneHeight  int
	PaneActive  bool
}

// ExpandFormat expands tmux-style format variables in template using the
// provided context. Unknown variables are left as-is.
func ExpandFormat(template string, ctx *FormatContext) string {
	var b strings.Builder
	for {
		idx := strings.Index(template, "#{")
		if idx < 0 {
			b.WriteString(template)
			break
		}
		b.WriteString(template[:idx])
		rest := template[idx+2:]
		end := strings.IndexByte(rest, '}')
		if end < 0 {
			b.WriteString(template[idx:])
			break
		}
		name := rest[:end]
		b.WriteString(lookupVar(name, ctx))
		template = rest[end+1:]
	}
	return b.String()
}

// lookupVar resolves a tmux format variable name to its value.
// Unknown variables are returned in their original #{...} form.
func lookupVar(name string, ctx *FormatContext) string {
	switch name {
	case "session_name":
		return ctx.SessionName
	case "session_id":
		return ctx.SessionID
	case "window_index":
		return ctx.WindowIndex
	case "window_id":
		return ctx.WindowID
	case "window_name":
		return ctx.WindowName
	case "pane_id":
		return ctx.PaneID
	case "pane_index":
		return ctx.PaneIndex
	case "pane_pid":
		return strconv.Itoa(ctx.PanePID)
	case "pane_tty":
		return ctx.PaneTTY
	case "pane_current_path":
		return ctx.PanePath
	case "pane_width":
		return strconv.Itoa(ctx.PaneWidth)
	case "pane_height":
		return strconv.Itoa(ctx.PaneHeight)
	case "pane_active":
		if ctx.PaneActive {
			return "1"
		}
		return "0"
	default:
		return "#{" + name + "}"
	}
}
