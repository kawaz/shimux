// CLI argument parser for tty-overlay.
// Hand-written parser without external dependencies.

///|
/// Return help text describing all options.
pub fn help_text() -> String {
  let text =
    #|Usage: tty-overlay [options] -- command [args...]
    #|
    #|Overlay text on top of a child process's terminal output.
    #|
    #|Options:
    #|  --content TEXT         Static text to display
    #|  --content-command CMD  Shell command whose output is displayed
    #|  --interval DURATION    Update interval for --content-command (default: 5s)
    #|                         Formats: 500ms, 5s, 1m, or plain number (seconds)
    #|  --width N|auto         Overlay width (default: auto)
    #|  --height N|auto        Overlay height (default: auto)
    #|  --top N                Vertical position from top (default: 1)
    #|  --bottom N             Vertical position from bottom
    #|  --left N               Horizontal position from left
    #|  --right N              Horizontal position from right (default: 1)
    #|  --fg-color COLOR|auto  Foreground color (default: auto)
    #|  --bg-color COLOR|auto  Background color (default: auto)
    #|  --position-mode MODE   Position mode: simple, decstbm, cell (default: simple)
    #|  --help                 Show this help message
    #|
    #|Colors: black, red, green, yellow, blue, magenta, cyan, white, auto
    #|
    #|Examples:
    #|  tty-overlay --content "Hello World" -- bash
    #|  tty-overlay --content-command "date +%H:%M:%S" --interval 2s -- bash
    #|  tty-overlay --content "Status" --bottom 1 --left 1 -- bash
    #|  tty-overlay --content "Alert" --fg-color red --bg-color white -- bash
  text + "\n"
}

///|
/// Parse CLI arguments into Config.
/// `args` is the argument array excluding the program name (argv[1..]).
/// Returns Config on success, raises Failure on validation error.
/// Raises Failure("__HELP__") when --help is specified.
pub fn parse(args : Array[String]) -> Config raise {
  let mut content : ContentSource? = None
  let mut interval_ms = 5000
  let mut width : Int? = None
  let mut height : Int? = None
  let mut vertical : VerticalPos? = None
  let mut horizontal : HorizontalPos? = None
  let mut fg_color : String? = None
  let mut bg_color : String? = None
  let mut position_mode : PositionMode = Simple
  let command : Array[String] = []
  let mut i = 0
  let mut found_separator = false
  while i < args.length() {
    let arg = args[i]
    if arg == "--" {
      // Everything after -- is the command
      found_separator = true
      i = i + 1
      while i < args.length() {
        command.push(args[i])
        i = i + 1
      }
      break
    }
    if arg.has_prefix("--") {
      match arg {
        "--help" => fail("__HELP__")
        "--content" => {
          i = i + 1
          if i >= args.length() {
            fail("--content requires a value")
          }
          if content is Some(_) {
            fail("--content and --content-command are mutually exclusive")
          }
          content = Some(Static(args[i]))
        }
        "--content-command" => {
          i = i + 1
          if i >= args.length() {
            fail("--content-command requires a value")
          }
          if content is Some(_) {
            fail("--content and --content-command are mutually exclusive")
          }
          content = Some(Command(args[i]))
        }
        "--interval" => {
          i = i + 1
          if i >= args.length() {
            fail("--interval requires a value")
          }
          interval_ms = parse_duration(args[i])
        }
        "--width" => {
          i = i + 1
          if i >= args.length() {
            fail("--width requires a value")
          }
          width = parse_size(args[i])
        }
        "--height" => {
          i = i + 1
          if i >= args.length() {
            fail("--height requires a value")
          }
          height = parse_size(args[i])
        }
        "--top" => {
          i = i + 1
          if i >= args.length() {
            fail("--top requires a value")
          }
          if vertical is Some(_) {
            fail("--top and --bottom are mutually exclusive")
          }
          vertical = Some(Top(parse_int_arg("--top", args[i])))
        }
        "--bottom" => {
          i = i + 1
          if i >= args.length() {
            fail("--bottom requires a value")
          }
          if vertical is Some(_) {
            fail("--top and --bottom are mutually exclusive")
          }
          vertical = Some(Bottom(parse_int_arg("--bottom", args[i])))
        }
        "--left" => {
          i = i + 1
          if i >= args.length() {
            fail("--left requires a value")
          }
          if horizontal is Some(_) {
            fail("--left and --right are mutually exclusive")
          }
          horizontal = Some(Left(parse_int_arg("--left", args[i])))
        }
        "--right" => {
          i = i + 1
          if i >= args.length() {
            fail("--right requires a value")
          }
          if horizontal is Some(_) {
            fail("--left and --right are mutually exclusive")
          }
          horizontal = Some(Right(parse_int_arg("--right", args[i])))
        }
        "--fg-color" => {
          i = i + 1
          if i >= args.length() {
            fail("--fg-color requires a value")
          }
          let val = args[i]
          fg_color = if val == "auto" { None } else { Some(val) }
        }
        "--bg-color" => {
          i = i + 1
          if i >= args.length() {
            fail("--bg-color requires a value")
          }
          let val = args[i]
          bg_color = if val == "auto" { None } else { Some(val) }
        }
        "--position-mode" => {
          i = i + 1
          if i >= args.length() {
            fail("--position-mode requires a value")
          }
          match args[i] {
            "simple" => position_mode = Simple
            "decstbm" => position_mode = Decstbm
            "cell" => fail("cell mode is not yet implemented")
            _ =>
              fail(
                "invalid position mode: " + args[i] + " (expected: simple, decstbm, cell)",
              )
          }
        }
        _ => fail("unknown option: " + arg)
      }
    } else {
      fail("unexpected argument: " + arg)
    }
    i = i + 1
  }
  // Validate required fields
  let content_source = match content {
    Some(cs) => cs
    None => fail("either --content or --content-command is required")
  }
  if not(found_separator) || command.length() == 0 {
    fail("command is required after --")
  }
  {
    content_source,
    interval_ms,
    width,
    height,
    vertical: vertical.unwrap_or(Top(1)),
    horizontal: horizontal.unwrap_or(Right(1)),
    fg_color,
    bg_color,
    position_mode,
    command,
  }
}

///|
/// Parse a duration string into milliseconds.
/// Supports: "500ms", "5s", "1m", "3" (plain number = seconds).
fn parse_duration(s : String) -> Int raise {
  if s.has_suffix("ms") {
    let num_str = try { s[:s.length() - 2] } catch { _ => fail("invalid duration: " + s) }
    try {
      @strconv.parse_int(num_str)
    } catch {
      _ => fail("invalid duration number: " + s)
    }
  } else if s.has_suffix("m") {
    let num_str = try { s[:s.length() - 1] } catch { _ => fail("invalid duration: " + s) }
    let n : Int = try {
      @strconv.parse_int(num_str)
    } catch {
      _ => fail("invalid duration number: " + s)
    }
    n * 60000
  } else if s.has_suffix("s") {
    let num_str = try { s[:s.length() - 1] } catch { _ => fail("invalid duration: " + s) }
    let n : Int = try {
      @strconv.parse_int(num_str)
    } catch {
      _ => fail("invalid duration number: " + s)
    }
    n * 1000
  } else {
    // Plain number = seconds
    let n : Int = try {
      @strconv.parse_int(s[:])
    } catch {
      _ => fail("invalid duration: " + s)
    }
    n * 1000
  }
}

///|
/// Parse a size value: "auto" returns None, number returns Some(n).
fn parse_size(s : String) -> Int? raise {
  if s == "auto" {
    None
  } else {
    let n : Int = try {
      @strconv.parse_int(s[:])
    } catch {
      _ => fail("invalid size: " + s)
    }
    Some(n)
  }
}

///|
/// Parse a required integer argument.
fn parse_int_arg(opt_name : String, s : String) -> Int raise {
  try {
    @strconv.parse_int(s[:])
  } catch {
    _ => fail(opt_name + " requires an integer value, got: " + s)
  }
}
