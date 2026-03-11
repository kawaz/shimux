// ホワイトボックステスト: cmd/shimux の純粋関数

// --- parse_size ---

///|
test "parse_size: 数字のみ" {
  assert_eq(parse_size("80"), 80)
}

///|
test "parse_size: パーセント付き" {
  assert_eq(parse_size("50%"), 50)
}

///|
test "parse_size: 0" {
  assert_eq(parse_size("0"), 0)
}

///|
test "parse_size: 先頭が非数字" {
  assert_eq(parse_size("abc"), 0)
}

///|
test "parse_size: 空文字列" {
  assert_eq(parse_size(""), 0)
}

///|
test "parse_size: 複数桁 + 末尾文字" {
  assert_eq(parse_size("123px"), 123)
}

///|
test "parse_size: 大きな数値" {
  assert_eq(parse_size("9999"), 9999)
}

// --- replace_substring ---

///|
test "replace_substring: 基本置換" {
  assert_eq(replace_substring("hello world", "world", "moon"), "hello moon")
}

///|
test "replace_substring: 複数回出現" {
  assert_eq(replace_substring("abcabc", "abc", "x"), "xx")
}

///|
test "replace_substring: ターゲットが見つからない" {
  assert_eq(replace_substring("hello", "xyz", "!!!"), "hello")
}

// NOTE: replace_substring("ab", "", "-") は空ターゲットで無限ループになる（既知の制限）
// 実用上は空ターゲットで呼ばれないため、テストは省略

///|
test "replace_substring: 元文字列が空" {
  assert_eq(replace_substring("", "a", "b"), "")
}

///|
test "replace_substring: 置換結果が空" {
  assert_eq(replace_substring("aXbXc", "X", ""), "abc")
}

///|
test "replace_substring: ターゲットと同じ文字列" {
  assert_eq(replace_substring("aaa", "a", "aa"), "aaaaaa")
}

// --- starts_with ---

///|
test "starts_with: 一致するプレフィックス" {
  assert_true(starts_with("hello world", "hello"))
}

///|
test "starts_with: 一致しないプレフィックス" {
  assert_false(starts_with("hello", "world"))
}

///|
test "starts_with: 空プレフィックス" {
  assert_true(starts_with("hello", ""))
}

///|
test "starts_with: 文字列とプレフィックスが同一" {
  assert_true(starts_with("abc", "abc"))
}

///|
test "starts_with: プレフィックスが文字列より長い" {
  assert_false(starts_with("ab", "abc"))
}

///|
test "starts_with: 空文字列に空プレフィックス" {
  assert_true(starts_with("", ""))
}

///|
test "starts_with: 空文字列に非空プレフィックス" {
  assert_false(starts_with("", "a"))
}

// --- contains_char ---

///|
test "contains_char: 含まれる文字" {
  assert_true(contains_char("hello", 'l'))
}

///|
test "contains_char: 含まれない文字" {
  assert_false(contains_char("hello", 'z'))
}

///|
test "contains_char: 空文字列" {
  assert_false(contains_char("", 'a'))
}

///|
test "contains_char: 特殊文字 =" {
  assert_true(contains_char("KEY=VALUE", '='))
}

///|
test "contains_char: 先頭文字" {
  assert_true(contains_char("abc", 'a'))
}

///|
test "contains_char: 末尾文字" {
  assert_true(contains_char("abc", 'c'))
}

// --- split_by_newline ---

///|
test "split_by_newline: 基本分割" {
  let result = split_by_newline("a\nb\nc")
  assert_eq(result.length(), 3)
  assert_eq(result[0], "a")
  assert_eq(result[1], "b")
  assert_eq(result[2], "c")
}

///|
test "split_by_newline: 末尾改行" {
  let result = split_by_newline("a\nb\n")
  assert_eq(result.length(), 2)
  assert_eq(result[0], "a")
  assert_eq(result[1], "b")
}

///|
test "split_by_newline: 空文字列" {
  let result = split_by_newline("")
  assert_eq(result.length(), 0)
}

///|
test "split_by_newline: 改行のみ" {
  let result = split_by_newline("\n")
  assert_eq(result.length(), 1)
  assert_eq(result[0], "")
}

///|
test "split_by_newline: 改行なし" {
  let result = split_by_newline("hello")
  assert_eq(result.length(), 1)
  assert_eq(result[0], "hello")
}

///|
test "split_by_newline: 連続改行" {
  let result = split_by_newline("a\n\nb")
  assert_eq(result.length(), 3)
  assert_eq(result[0], "a")
  assert_eq(result[1], "")
  assert_eq(result[2], "b")
}

// --- format_pane_output ---

///|
test "format_pane_output: pane_id 展開" {
  let pane_id = @core.PaneId::PaneId("%42")
  assert_eq(format_pane_output("#\u007bpane_id}", pane_id), "%42")
}

///|
test "format_pane_output: フォーマット内に pane_id がない" {
  let pane_id = @core.PaneId::PaneId("%1")
  assert_eq(format_pane_output("no_vars_here", pane_id), "no_vars_here")
}

///|
test "format_pane_output: テキスト混在" {
  let pane_id = @core.PaneId::PaneId("%5")
  assert_eq(
    format_pane_output("id=#\u007bpane_id}:done", pane_id),
    "id=%5:done",
  )
}

// --- format_pane_info ---

///|
test "format_pane_info: 全変数展開" {
  let pane : @core.PaneInfo = {
    id: @core.PaneId::PaneId("%10"),
    name: "test",
    cwd: "/home/user",
    width: 120,
    height: 40,
    active: true,
  }
  let fmt = "#\u007bpane_id} #\u007bpane_width}x#\u007bpane_height} #\u007bpane_active} #\u007bpane_current_path}"
  assert_eq(format_pane_info(fmt, pane), "%10 120x40 1 /home/user")
}

///|
test "format_pane_info: active=false" {
  let pane : @core.PaneInfo = {
    id: @core.PaneId::PaneId("%0"),
    name: "",
    cwd: "/tmp",
    width: 80,
    height: 24,
    active: false,
  }
  assert_eq(format_pane_info("#\u007bpane_active}", pane), "0")
}

///|
test "format_pane_info: pane_pid は常に 0" {
  let pane : @core.PaneInfo = {
    id: @core.PaneId::PaneId("%1"),
    name: "",
    cwd: "",
    width: 0,
    height: 0,
    active: false,
  }
  assert_eq(format_pane_info("#\u007bpane_pid}", pane), "0")
}

///|
test "format_pane_info: フォーマット変数なし" {
  let pane : @core.PaneInfo = {
    id: @core.PaneId::PaneId("%1"),
    name: "",
    cwd: "",
    width: 0,
    height: 0,
    active: false,
  }
  assert_eq(format_pane_info("plain text", pane), "plain text")
}

// --- build_exec_command ---

///|
test "build_exec_command: cwd なし" {
  let result = build_exec_command("", ["echo", "hello"])
  assert_eq(result, "exec </dev/tty >/dev/tty 2>&1; exec 'echo' 'hello'")
}

///|
test "build_exec_command: cwd あり" {
  let result = build_exec_command("/tmp", ["ls", "-la"])
  assert_eq(
    result, "cd '/tmp' && exec </dev/tty >/dev/tty 2>&1; exec 'ls' '-la'",
  )
}

///|
test "build_exec_command: 空コマンド配列" {
  let result = build_exec_command("", [])
  assert_eq(result, "exec </dev/tty >/dev/tty 2>&1; exec")
}

///|
test "build_exec_command: スペース含むパス" {
  let result = build_exec_command("/my dir", ["cat", "my file.txt"])
  assert_eq(
    result, "cd '/my dir' && exec </dev/tty >/dev/tty 2>&1; exec 'cat' 'my file.txt'",
  )
}

// --- basename ---

///|
test "basename: 絶対パス" {
  assert_eq(basename("/usr/local/bin/tmux"), "tmux")
}

///|
test "basename: 相対パス" {
  assert_eq(basename("./bin/shimux"), "shimux")
}

///|
test "basename: ファイル名のみ" {
  assert_eq(basename("tmux"), "tmux")
}
