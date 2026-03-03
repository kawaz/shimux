///|
extern "C" fn shimux_add(a : Int, b : Int) -> Int = "shimux_add"

///|
#borrow(data)
extern "C" fn shimux_to_upper(data : Bytes) -> Bytes = "shimux_to_upper"

///|
#borrow(data)
extern "C" fn shimux_bytes_len(data : Bytes) -> Int = "shimux_bytes_len"

// Step 2: termios operations

///|
extern "C" fn shimux_isatty(fd : Int) -> Int = "shimux_isatty"

///|
extern "C" fn shimux_get_rows(fd : Int) -> Int = "shimux_get_rows"

///|
extern "C" fn shimux_get_cols(fd : Int) -> Int = "shimux_get_cols"

///|
extern "C" fn shimux_enable_raw_mode(fd : Int) -> Int = "shimux_enable_raw_mode"

///|
extern "C" fn shimux_disable_raw_mode(fd : Int) -> Int = "shimux_disable_raw_mode"

///|
extern "C" fn shimux_read_byte(fd : Int) -> Int = "shimux_read_byte"

///|
#borrow(data)
extern "C" fn shimux_write(fd : Int, data : Bytes) -> Int = "shimux_write"

// Step 3: PTY creation + child process

///|
#borrow(cmd)
extern "C" fn shimux_forkpty_exec(cmd : Bytes) -> Int = "shimux_forkpty_exec"

// Step 4: I/O proxy

///|
#borrow(log_path)
extern "C" fn shimux_proxy_io(
  master_fd : Int,
  log_path : Bytes
) -> Int = "shimux_proxy_io"

// Process exit (wrapped in Rust to avoid C declaration conflict with stdlib)
///|
extern "C" fn shimux_exit(status : Int) -> Unit = "shimux_exit"

// Test helper: forkpty + exec + wait (no I/O proxy)
///|
#borrow(cmd)
extern "C" fn shimux_forkpty_exec_wait(cmd : Bytes) -> Int = "shimux_forkpty_exec_wait"

/// Convert a MoonBit String to UTF-8 encoded Bytes.
/// Design rationale: String.to_bytes() produces UTF-16 LE, but the Rust FFI
/// functions expect UTF-8. Using @utf8.encode() produces correct UTF-8 Bytes.
fn to_utf8(s : String) -> Bytes {
  @utf8.encode(s)
}

///|
fn run_demo() -> Unit {
  // Step 1: Int passthrough
  let result = shimux_add(3, 7)
  println("3 + 7 = \{result}")

  // Step 1: Bytes passthrough
  let input : Bytes = b"hello, moonbit"
  let len = shimux_bytes_len(input)
  println("bytes len = \{len}")
  let upper = shimux_to_upper(input)
  println("upper = \{upper}")

  // Step 2: isatty check
  let is_tty = shimux_isatty(0)
  println("stdin is tty: \{is_tty}")

  // Step 2: terminal size
  let rows = shimux_get_rows(0)
  let cols = shimux_get_cols(0)
  println("terminal size: \{rows} rows x \{cols} cols")

  // Step 2: raw mode test (only if stdin is a tty)
  if is_tty == 1 {
    let rc = shimux_enable_raw_mode(0)
    if rc == 0 {
      let _ = shimux_write(1, b"Press any key... ")
      let byte = shimux_read_byte(0)
      let _ = shimux_write(1, b"\r\n")
      let _ = shimux_write(1, to_utf8("You pressed byte: \{byte}\r\n"))
      let _ = shimux_disable_raw_mode(0)
    } else {
      println("Failed to enable raw mode")
    }
  } else {
    println("Not a tty, skipping raw mode test")
  }
}

///|
fn run_proxy(cmd : String, log_path : String) -> Unit {
  let master_fd = shimux_forkpty_exec(to_utf8(cmd))
  if master_fd < 0 {
    println("Failed to forkpty")
    shimux_exit(1)
  }
  let status = shimux_proxy_io(master_fd, to_utf8(log_path))
  shimux_exit(status)
}

///|
fn main {
  let args = @env.args()
  if args.length() > 1 {
    // PTY proxy mode: first arg is command, optional second arg is log path
    let cmd = args[1]
    let log_path = if args.length() > 2 { args[2] } else { "" }
    run_proxy(cmd, log_path)
  } else {
    // No arguments: run Step 1-2 demo
    run_demo()
  }
}

///|
test "shimux_add: basic addition" {
  inspect(shimux_add(3, 7), content="10")
  inspect(shimux_add(0, 0), content="0")
  inspect(shimux_add(-1, 1), content="0")
  inspect(shimux_add(100, 200), content="300")
}

///|
test "shimux_bytes_len: byte length" {
  inspect(shimux_bytes_len(b"hello"), content="5")
  inspect(shimux_bytes_len(b""), content="0")
  inspect(shimux_bytes_len(b"hello, moonbit"), content="14")
}

///|
test "shimux_to_upper: ASCII uppercase conversion" {
  inspect(shimux_to_upper(b"hello"), content="b\"HELLO\"")
  inspect(shimux_to_upper(b"Hello, World"), content="b\"HELLO, WORLD\"")
  inspect(shimux_to_upper(b"ABC"), content="b\"ABC\"")
  inspect(shimux_to_upper(b"123"), content="b\"123\"")
}

// Step 2 tests

///|
test "shimux_isatty: returns valid boolean" {
  // In test environment, stdin may or may not be a tty.
  // The result must be 0 or 1.
  let result = shimux_isatty(0)
  inspect(result >= 0 && result <= 1, content="true")
}

///|
test "shimux_isatty: invalid fd returns 0" {
  // fd -1 is never a valid tty
  inspect(shimux_isatty(-1), content="0")
}

///|
test "shimux_get_rows: returns positive or error" {
  // If stdin is a tty, rows should be positive.
  // If not a tty, returns -1.
  let rows = shimux_get_rows(0)
  inspect(rows == -1 || rows > 0, content="true")
}

///|
test "shimux_get_cols: returns positive or error" {
  let cols = shimux_get_cols(0)
  inspect(cols == -1 || cols > 0, content="true")
}

///|
test "shimux_get_rows: invalid fd returns -1" {
  inspect(shimux_get_rows(-1), content="-1")
}

///|
test "shimux_get_cols: invalid fd returns -1" {
  inspect(shimux_get_cols(-1), content="-1")
}

///|
test "shimux_disable_raw_mode: without enable returns -1" {
  // Calling disable without enable should fail
  inspect(shimux_disable_raw_mode(0), content="-1")
}

///|
test "shimux_write: write to stdout" {
  // Writing empty bytes should succeed with 0 bytes written
  inspect(shimux_write(1, b""), content="0")
}

///|
test "shimux_read_byte: invalid fd returns -1" {
  inspect(shimux_read_byte(-1), content="-1")
}

// Step 3-4 tests

///|
test "to_utf8: converts String to UTF-8 Bytes" {
  inspect(to_utf8("hello"), content="b\"hello\"")
  inspect(shimux_bytes_len(to_utf8("hello")), content="5")
  inspect(to_utf8(""), content="b\"\"")
}

///|
test "shimux_forkpty_exec: empty command returns -1" {
  inspect(shimux_forkpty_exec(b""), content="-1")
}

///|
test "shimux_forkpty_exec_wait: echo returns 0" {
  inspect(shimux_forkpty_exec_wait(b"echo hello"), content="0")
}

///|
test "shimux_forkpty_exec_wait: false returns 1" {
  inspect(shimux_forkpty_exec_wait(b"false"), content="1")
}
