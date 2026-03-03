// ===== helpers =====

///|
test "to_utf8: converts String to UTF-8 Bytes" {
  inspect(to_utf8("hello"), content="b\"hello\"")
  inspect(to_utf8(""), content="b\"\"")
  // マルチバイト: "あ" = 0xE3 0x81 0x82 (3 bytes)
  let a = to_utf8("あ")
  inspect(a.length(), content="3")
}

// ===== pty =====

///|
test "pty_open: returns valid handle" {
  let handle = pty_open(80, 24)
  inspect(not(handle is None), content="true")
  // cleanup
  let _ = pty_close(handle.unwrap())
}

///|
test "pty_master_fd: returns valid fd" {
  let handle = pty_open(80, 24).unwrap()
  let fd = pty_master_fd(handle)
  inspect(fd >= 0, content="true")
  let _ = pty_close(handle)
}

///|
test "pty_close: succeeds" {
  let handle = pty_open(80, 24).unwrap()
  inspect(pty_close(handle), content="true")
}

///|
test "pty_spawn: empty command returns None" {
  let handle = pty_open(80, 24).unwrap()
  let pid = pty_spawn(handle, b"")
  inspect(pid, content="None")
  let _ = pty_close(handle)
}

///|
test "pty_resize: succeeds on valid handle" {
  let handle = pty_open(80, 24).unwrap()
  inspect(pty_resize(handle, 120, 40), content="true")
  let _ = pty_close(handle)
}

// W4: pty_spawn normal case
///|
test "pty_spawn: echo hello returns valid pid and waitpid succeeds" {
  let handle = pty_open(80, 24).unwrap()
  let pid = pty_spawn(handle, to_utf8("echo hello"))
  inspect(not(pid is None), content="true")
  let child_pid = pid.unwrap()
  inspect(child_pid > 0, content="true")
  // Wait for child to finish (blocking)
  let result = proc_waitpid(child_pid, false)
  inspect(not(result is None), content="true")
  // Verify exit status is 0
  match result {
    Some((_, status)) => inspect(status, content="0")
    None => inspect(false, content="true") // waitpid failed unexpectedly
  }
  let _ = pty_close(handle)
}

// ===== tty =====

// W6: Fix tautological test - use meaningful assertions
///|
test "tty_is_tty: invalid fd returns false" {
  inspect(tty_is_tty(-1), content="false")
}

///|
test "tty_is_tty: does not crash on stdin" {
  // CI/test environments may not have a tty on stdin, so just verify no crash
  let _ = tty_is_tty(0)
}

// W5: tty_set_raw + tty_restore on pty fd
///|
test "tty_set_raw and tty_restore: works on pty master fd" {
  let handle = pty_open(80, 24).unwrap()
  let master_fd = pty_master_fd(handle)
  // master_fd is a pty device, so tty_save/set_raw/restore should work
  let tty_handle = tty_save(master_fd)
  inspect(not(tty_handle is None), content="true")
  let saved = tty_handle.unwrap()
  inspect(tty_set_raw(master_fd), content="true")
  inspect(tty_restore(saved), content="true")
  let _ = pty_close(handle)
}

///|
test "tty_size: returns None or valid size" {
  // テスト環境では tty でない場合 None を返す
  match tty_size(0) {
    None => inspect(true, content="true")
    Some((rows, cols)) => {
      inspect(rows > 0, content="true")
      inspect(cols > 0, content="true")
    }
  }
}

///|
test "tty_size: invalid fd returns None" {
  inspect(tty_size(-1), content="None")
}

///|
test "tty_save: invalid fd returns None" {
  // save していない無効なハンドルで restore → Rust 側で -1 を返すはず
  // TtyHandle を直接作れないので tty_save を無効 fd で呼ぶ
  let handle = tty_save(-1)
  inspect(handle is None, content="true")
}

// ===== io =====

///|
test "io_write: empty bytes to stdout returns 0" {
  inspect(io_write(1, b""), content="0")
}

///|
test "io_read: invalid fd returns -1" {
  let buf = Bytes::make(64, 0)
  inspect(io_read(-1, buf, 64), content="-1")
}

///|
test "io_close: invalid fd returns false" {
  inspect(io_close(-1), content="false")
}

// W5: io_set_nonblocking on pty master_fd
///|
test "io_set_nonblocking: works on pty master fd" {
  let handle = pty_open(80, 24).unwrap()
  let master_fd = pty_master_fd(handle)
  inspect(io_set_nonblocking(master_fd), content="true")
  let _ = pty_close(handle)
}

// W5: io_poll with pty master_fd
///|
test "io_poll: timeout 0 on pty master_fd returns without error" {
  let handle = pty_open(80, 24).unwrap()
  let master_fd = pty_master_fd(handle)
  let fds = pollfd_pack([(master_fd, POLLIN)])
  // timeout=0 means immediate return; result >= 0 means no error
  let result = io_poll(fds, 1, 0)
  inspect(result >= 0, content="true")
  let _ = pty_close(handle)
}

///|
test "pollfd_pack: creates correct byte layout" {
  let buf = pollfd_pack([(5, POLLIN)])
  // 8 bytes per entry
  inspect(buf.length(), content="8")
  // fd = 5 at offset 0 (LE i32)
  inspect(buf[0].to_int(), content="5")
  inspect(buf[1].to_int(), content="0")
  inspect(buf[2].to_int(), content="0")
  inspect(buf[3].to_int(), content="0")
  // events = POLLIN (0x0001) at offset 4 (LE i16)
  inspect(buf[4].to_int(), content="1")
  inspect(buf[5].to_int(), content="0")
  // revents = 0 at offset 6 (LE i16)
  inspect(buf[6].to_int(), content="0")
  inspect(buf[7].to_int(), content="0")
}

///|
test "pollfd_revents: reads revents from buffer" {
  let buf = pollfd_pack([(5, POLLIN)])
  // revents is initially 0
  inspect(pollfd_revents(buf, 0), content="0")
}

///|
test "pollfd_pack: multiple entries" {
  let buf = pollfd_pack([(3, POLLIN), (7, POLLHUP)])
  inspect(buf.length(), content="16")
  // entry 0: fd=3
  inspect(buf[0].to_int(), content="3")
  // entry 1: fd=7
  inspect(buf[8].to_int(), content="7")
  // entry 1: events = POLLHUP (0x0010)
  inspect(buf[12].to_int(), content="16")
  inspect(buf[13].to_int(), content="0")
}

// ===== test helpers =====

///|
#borrow(path)
extern "C" fn shimux_test_mkdir(path : Bytes) -> Int = "shimux_test_mkdir"

///|
#borrow(path)
extern "C" fn shimux_test_rmdir(path : Bytes) -> Int = "shimux_test_rmdir"

// ===== sig =====

///|
test "sig_ignore: SIGPIPE succeeds" {
  inspect(sig_ignore(SIGPIPE), content="true")
}

// W5: sig_setup_winch on pty master_fd
///|
test "sig_setup_winch: succeeds on pty master fd" {
  let handle = pty_open(80, 24).unwrap()
  let master_fd = pty_master_fd(handle)
  inspect(sig_setup_winch(master_fd), content="true")
  let _ = pty_close(handle)
}

// ===== proc =====

///|
test "proc_waitpid: pid -1 with nohang returns None" {
  // pid -1 + WNOHANG: 子プロセスがないので None
  inspect(proc_waitpid(-1, true), content="None")
}

///|
test "proc_kill: pid 0 with signal 0 checks process group" {
  // signal 0 は kill のエラーチェック用。pid 0 は自プロセスグループ → 成功するはず
  inspect(proc_kill(0, 0), content="true")
}

// ===== helpers: getenv, getpid, getuid, ensure_dir =====

///|
test "getenv returns non-empty for PATH" {
  let path = getenv("PATH")
  assert_true(path.length() > 0)
}

///|
test "getenv returns empty for nonexistent var" {
  let val = getenv("SHIMUX_NONEXISTENT_VAR_12345")
  assert_eq(val, "")
}

///|
test "getpid returns positive" {
  assert_true(getpid() > 0)
}

///|
test "getuid returns valid uid" {
  // uid is unsigned, any value is valid but should be consistent
  let uid = getuid()
  let uid2 = getuid()
  assert_eq(uid, uid2)
}

///|
test "ensure_dir creates directory" {
  let dir = "/tmp/shimux_test_ensure_dir_" + getpid().to_string()
  assert_true(ensure_dir(dir))
  // cleanup
  let _ = shimux_test_rmdir(to_utf8(dir))
}

// ===== argc / argv =====

///|
test "argc: returns positive" {
  // Test binary always has at least the program name
  assert_true(argc() >= 1)
}

///|
test "argv: index 0 returns Some" {
  let arg0 = argv(0)
  inspect(not(arg0 is None), content="true")
}

///|
test "argv: out of bounds returns None" {
  let arg = argv(99999)
  inspect(arg, content="None")
}

// ===== monotonic_ms =====

///|
test "monotonic_ms: returns positive" {
  let t = monotonic_ms()
  assert_true(t > 0L)
}

///|
test "monotonic_ms: monotonically increasing" {
  let t1 = monotonic_ms()
  let t2 = monotonic_ms()
  assert_true(t2 >= t1)
}

// ===== exec_capture =====

///|
test "exec_capture: echo hello" {
  let text = exec_capture("echo hello")
  inspect(text, content="hello\n")
}

///|
test "exec_capture: multi-line output" {
  let text = exec_capture("echo line1; echo line2")
  inspect(text, content="line1\nline2\n")
}

///|
test "exec_capture: empty output" {
  let text = exec_capture("true")
  inspect(text, content="")
}

// ===== pty_spawn_argv =====

///|
test "pty_spawn_argv: empty args returns None" {
  let handle = pty_open(80, 24).unwrap()
  let pid = pty_spawn_argv(handle, [])
  inspect(pid, content="None")
  let _ = pty_close(handle)
}

///|
test "pty_spawn_argv: echo hello returns valid pid" {
  let handle = pty_open(80, 24).unwrap()
  let pid = pty_spawn_argv(handle, ["/bin/echo", "hello"])
  inspect(not(pid is None), content="true")
  let child_pid = pid.unwrap()
  inspect(child_pid > 0, content="true")
  let result = proc_waitpid(child_pid, false)
  inspect(not(result is None), content="true")
  match result {
    Some((_, status)) => inspect(status, content="0")
    None => inspect(false, content="true")
  }
  let _ = pty_close(handle)
}
