// Package tmux provides a tmux-compatible argument parser for shimux.
//
// The parser uses a 3-stage approach:
//
//	Stage 1: Consume global options (-L, -S, -f, -V)
//	Stage 2: Identify the command name (first non-option argument)
//	Stage 3: Pass remaining arguments to command-specific parsers
package tmux

import "fmt"

// GlobalOptions holds tmux global options parsed from before the command name.
type GlobalOptions struct {
	SocketName string // -L <name>
	SocketPath string // -S <path>
	ConfigFile string // -f <file>
	Version    bool   // -V
}

// ParseResult holds the result of the 3-stage parse.
type ParseResult struct {
	Global  GlobalOptions
	Command string
	Args    []string
}

// Parse performs the 3-stage parse of tmux-compatible arguments.
//
// Stage 1: Consume global options before the command name.
// Stage 2: Identify the command name (first non-option argument).
// Stage 3: Remaining arguments become Args for command-specific parsing.
func Parse(args []string) (*ParseResult, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no command specified")
	}

	result := &ParseResult{}
	i := 0

	// Stage 1: global options
	for i < len(args) {
		arg := args[i]

		switch arg {
		case "-V":
			result.Global.Version = true
			return result, nil
		case "-L":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-L requires a value")
			}
			result.Global.SocketName = args[i]
		case "-S":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-S requires a value")
			}
			result.Global.SocketPath = args[i]
		case "-f":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-f requires a value")
			}
			result.Global.ConfigFile = args[i]
		default:
			// Not a global option; this should be the command name
			goto stage2
		}
		i++
	}

	// If we consumed everything without finding a command, check version
	if result.Global.Version {
		return result, nil
	}
	return nil, fmt.Errorf("no command specified")

stage2:
	// Stage 2: command name
	cmd := args[i]
	if cmd == "" {
		return nil, fmt.Errorf("empty command name")
	}
	result.Command = cmd
	i++

	// Stage 3: remaining args
	if i < len(args) {
		result.Args = args[i:]
	}

	return result, nil
}

// SplitWindowOptions holds parsed options for the split-window command.
type SplitWindowOptions struct {
	Horizontal bool   // -h flag (horizontal split = right direction)
	Direction  string // "left", "right", "up", "down" (default: "down")
	Target     string // -t <target>
	Size       string // -l <size> (recognized but may be ignored in Phase 1)
	PrintAfter bool   // -P flag
	Format     string // -F <format>
}

// ParseSplitWindow parses split-window command-specific arguments.
//
// Supports: -h, -v, -t, -l, -P, -F, --left, --right, --up, --down
// Conflicting flags (-h/-v) use last-wins rule per design spec.
func ParseSplitWindow(args []string) (*SplitWindowOptions, error) {
	opts := &SplitWindowOptions{
		Direction: "down", // tmux default: vertical split (down)
	}
	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-h":
			opts.Horizontal = true
			opts.Direction = "right"
		case "-v":
			opts.Horizontal = false
			opts.Direction = "down"
		case "--left":
			opts.Horizontal = false
			opts.Direction = "left"
		case "--right":
			opts.Horizontal = true
			opts.Direction = "right"
		case "--up":
			opts.Horizontal = false
			opts.Direction = "up"
		case "--down":
			opts.Horizontal = false
			opts.Direction = "down"
		case "-t":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-t requires a value")
			}
			opts.Target = args[i]
		case "-l":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-l requires a value")
			}
			opts.Size = args[i]
		case "-P":
			opts.PrintAfter = true
		case "-F":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-F requires a value")
			}
			opts.Format = args[i]
		}
		i++
	}
	return opts, nil
}

// SendKeysOptions holds parsed options for the send-keys command.
type SendKeysOptions struct {
	Target  string   // -t <target>
	Literal bool     // -l flag
	Keys    []string // remaining key arguments
}

// ParseSendKeys parses send-keys command-specific arguments.
//
// Supports: -t, -l, --
// After --, all remaining arguments are treated as key arguments.
func ParseSendKeys(args []string) (*SendKeysOptions, error) {
	opts := &SendKeysOptions{}
	i := 0
	for i < len(args) {
		arg := args[i]

		if arg == "--" {
			// Everything after -- is key arguments
			i++
			opts.Keys = append(opts.Keys, args[i:]...)
			return opts, nil
		}

		switch arg {
		case "-t":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-t requires a value")
			}
			opts.Target = args[i]
		case "-l":
			opts.Literal = true
		default:
			// Non-option argument: this and everything after are keys
			opts.Keys = append(opts.Keys, args[i:]...)
			return opts, nil
		}
		i++
	}
	return opts, nil
}

// SelectPaneOptions holds parsed options for the select-pane command.
type SelectPaneOptions struct {
	Target string // -t <target>
	Style  string // -P <style> (recognized, ignored in Phase 1)
	Title  string // -T <title> (recognized, ignored in Phase 1)
}

// ParseSelectPane parses select-pane command-specific arguments.
//
// Supports: -t, -P (style), -T (title)
// -P and -T are recognized but ignored in Phase 1 per design spec.
func ParseSelectPane(args []string) (*SelectPaneOptions, error) {
	opts := &SelectPaneOptions{}
	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-t":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-t requires a value")
			}
			opts.Target = args[i]
		case "-P":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-P requires a value")
			}
			opts.Style = args[i]
		case "-T":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-T requires a value")
			}
			opts.Title = args[i]
		}
		i++
	}
	return opts, nil
}

// NewWindowOptions holds parsed options for the new-window command.
type NewWindowOptions struct {
	Target     string // -t <target>
	Name       string // -n <name>
	PrintAfter bool   // -P flag
	Format     string // -F <format>
}

// ParseNewWindow parses new-window command-specific arguments.
//
// Supports: -t, -n, -P, -F
func ParseNewWindow(args []string) (*NewWindowOptions, error) {
	opts := &NewWindowOptions{}
	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-t":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-t requires a value")
			}
			opts.Target = args[i]
		case "-n":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-n requires a value")
			}
			opts.Name = args[i]
		case "-P":
			opts.PrintAfter = true
		case "-F":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-F requires a value")
			}
			opts.Format = args[i]
		}
		i++
	}
	return opts, nil
}

// HasSessionOptions holds parsed options for the has-session command.
type HasSessionOptions struct {
	Target string // -t <target>
}

// ParseHasSession parses has-session command-specific arguments.
//
// Supports: -t
func ParseHasSession(args []string) (*HasSessionOptions, error) {
	opts := &HasSessionOptions{}
	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-t":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-t requires a value")
			}
			opts.Target = args[i]
		}
		i++
	}
	return opts, nil
}

// KillPaneOptions holds parsed options for the kill-pane command.
type KillPaneOptions struct {
	Target string // -t <target>
}

// ParseKillPane parses kill-pane command-specific arguments.
//
// Supports: -t
func ParseKillPane(args []string) (*KillPaneOptions, error) {
	opts := &KillPaneOptions{}
	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-t":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-t requires a value")
			}
			opts.Target = args[i]
		}
		i++
	}
	return opts, nil
}

// ListPanesOptions holds parsed options for the list-panes command.
type ListPanesOptions struct {
	Format string // -F <format>
	Target string // -t <target>
}

// ParseListPanes parses list-panes command-specific arguments.
//
// Supports: -F, -t
func ParseListPanes(args []string) (*ListPanesOptions, error) {
	opts := &ListPanesOptions{}
	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-F":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-F requires a value")
			}
			opts.Format = args[i]
		case "-t":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-t requires a value")
			}
			opts.Target = args[i]
		}
		i++
	}
	return opts, nil
}

// DisplayMessageOptions holds parsed options for the display-message command.
type DisplayMessageOptions struct {
	Print  bool   // -p flag (print to stdout)
	Format string // format string (positional argument or -F value)
	Target string // -t <target>
}

// ParseDisplayMessage parses display-message command-specific arguments.
//
// Supports: -p, -t, -F, positional format string
// The last non-option positional argument is used as the format string.
func ParseDisplayMessage(args []string) (*DisplayMessageOptions, error) {
	opts := &DisplayMessageOptions{}
	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-p":
			opts.Print = true
		case "-t":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-t requires a value")
			}
			opts.Target = args[i]
		case "-F":
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("-F requires a value")
			}
			opts.Format = args[i]
		default:
			// Positional argument is the format string
			opts.Format = arg
		}
		i++
	}
	return opts, nil
}

// ShowOptionsOptions holds parsed options for the show-options command.
type ShowOptionsOptions struct {
	Global bool   // -g flag
	Option string // option name (positional argument)
}

// ParseShowOptions parses show-options command-specific arguments.
//
// Supports: -g, positional option name
func ParseShowOptions(args []string) (*ShowOptionsOptions, error) {
	opts := &ShowOptionsOptions{}
	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "-g":
			opts.Global = true
		default:
			opts.Option = arg
		}
		i++
	}
	return opts, nil
}
