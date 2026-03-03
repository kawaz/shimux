// Entry point for tty-overlay.
// Parses CLI arguments, creates a Runner, and runs the event loop.

/// Strip MoonBit source location prefix from fail() messages.
/// e.g. "lib/args.mbt:10:5@pkg FAILED: foo" -> "foo"
/// Design rationale: MoonBit's fail() prepends source location and
/// "FAILED: " to the message. We strip it for clean CLI output.
fn strip_fail_prefix(msg : String) -> String {
  let marker = "FAILED: "
  let marker_chars : Array[Char] = []
  for c in marker {
    marker_chars.push(c)
  }
  let msg_chars : Array[Char] = []
  for c in msg {
    msg_chars.push(c)
  }
  let marker_len = marker_chars.length()
  let msg_len = msg_chars.length()
  if msg_len < marker_len {
    return msg
  }
  // Search for "FAILED: " substring
  let mut found = -1
  for i = 0; i <= msg_len - marker_len; i = i + 1 {
    let mut matched = true
    for j = 0; j < marker_len; j = j + 1 {
      if msg_chars[i + j] != marker_chars[j] {
        matched = false
        break
      }
    }
    if matched {
      found = i
      break
    }
  }
  if found < 0 {
    msg
  } else {
    let start = found + marker_len
    let buf = StringBuilder::new()
    for i = start; i < msg_len; i = i + 1 {
      buf.write_char(msg_chars[i])
    }
    buf.to_string()
  }
}

fn main {
  // Collect CLI arguments (skip argv[0] = program name)
  let argc = @ffi.argc()
  let args : Array[String] = []
  for i = 1; i < argc; i = i + 1 {
    match @ffi.argv(i) {
      Some(arg) => args.push(arg)
      None => ()
    }
  }
  // No arguments: show help and exit
  if args.length() == 0 {
    let help_bytes = @ffi.to_utf8(@args.help_text())
    let _ = @ffi.io_write(1, help_bytes)
    @ffi.proc_exit(0)
    panic() // unreachable
  }
  // Parse arguments
  let config = try {
    @args.parse(args)
  } catch {
    Failure::Failure(msg) => {
      if msg.contains("__HELP__") {
        let help_bytes = @ffi.to_utf8(@args.help_text())
        let _ = @ffi.io_write(1, help_bytes)
        @ffi.proc_exit(0)
        panic() // unreachable
      }
      let err_bytes = @ffi.to_utf8(
        "Error: " + strip_fail_prefix(msg) + "\n",
      )
      let _ = @ffi.io_write(2, err_bytes)
      @ffi.proc_exit(1)
      panic() // unreachable
    }
    _ => {
      let err_bytes = @ffi.to_utf8("Error: unknown parse error\n")
      let _ = @ffi.io_write(2, err_bytes)
      @ffi.proc_exit(1)
      panic() // unreachable
    }
  }
  // Create and run the Runner
  let runner = try {
    @runner.Runner::new(config)
  } catch {
    Failure::Failure(msg) => {
      let err_bytes = @ffi.to_utf8(
        "Error: " + strip_fail_prefix(msg) + "\n",
      )
      let _ = @ffi.io_write(2, err_bytes)
      @ffi.proc_exit(1)
      panic() // unreachable
    }
    _ => {
      let err_bytes = @ffi.to_utf8("Error: runner initialization failed\n")
      let _ = @ffi.io_write(2, err_bytes)
      @ffi.proc_exit(1)
      panic() // unreachable
    }
  }
  let status = runner.run()
  @ffi.proc_exit(status)
}
