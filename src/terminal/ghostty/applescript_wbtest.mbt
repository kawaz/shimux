// AppleScript 組み立てロジックのテスト

// === escape_applescript ===

///|
test "escape_applescript: plain text unchanged" {
  assert_eq(escape_applescript("hello"), "hello")
}

///|
test "escape_applescript: backslash escaped" {
  assert_eq(escape_applescript("a\\b"), "a\\\\b")
}

///|
test "escape_applescript: double quote escaped" {
  assert_eq(escape_applescript("say \"hi\""), "say \\\"hi\\\"")
}

///|
test "escape_applescript: mixed escaping" {
  assert_eq(escape_applescript("a\\\"b"), "a\\\\\\\"b")
}

///|
test "escape_applescript: empty string" {
  assert_eq(escape_applescript(""), "")
}

// === build_split_script ===

///|
test "build_split_script: horizontal split" {
  let script = build_split_script("right")
  inspect(script.contains("split"), content="true")
  inspect(script.contains("direction right"), content="true")
}

///|
test "build_split_script: vertical split" {
  let script = build_split_script("down")
  inspect(script.contains("direction down"), content="true")
}

// === build_split_with_config_script ===

///|
test "build_split_with_config_script: with command and cwd" {
  let script = build_split_with_config_script(
    "right",
    "/bin/sh -c 'exec claude'",
    "/home/user",
    [],
    true,
  )
  inspect(script.contains("surface configuration"), content="true")
  inspect(
    script.contains("command:\"/bin/sh -c 'exec claude'\""),
    content="true",
  )
  inspect(
    script.contains("initial working directory:\"/home/user\""),
    content="true",
  )
  inspect(script.contains("wait after command:true"), content="true")
  inspect(script.contains("direction right"), content="true")
  inspect(script.contains("id of newTerm"), content="true")
}

///|
test "build_split_with_config_script: with env vars" {
  let script = build_split_with_config_script(
    "down",
    "/bin/sh -c 'exec test'",
    "",
    ["FOO=bar", "BAZ=qux"],
    true,
  )
  inspect(
    script.contains("environment variables:{\"FOO=bar\", \"BAZ=qux\"}"),
    content="true",
  )
  inspect(script.contains("direction down"), content="true")
  // cwd が空なので initial working directory は含まれない
  inspect(script.contains("initial working directory"), content="false")
}

///|
test "build_split_with_config_script: escapes special chars in command" {
  let script = build_split_with_config_script(
    "right",
    "/bin/sh -c 'echo \"hello\"'",
    "",
    [],
    false,
  )
  // ダブルクオートがエスケープされている
  inspect(script.contains("\\\"hello\\\""), content="true")
  // wait_after_command が false なので含まれない
  inspect(script.contains("wait after command"), content="false")
}

///|
test "build_split_with_config_script: no command (empty)" {
  let script = build_split_with_config_script("right", "", "", [], false)
  // command が空なので command: は含まれない
  inspect(script.contains("command:"), content="false")
  // 空の record
  inspect(script.contains("set cfgRecord to {}"), content="true")
}

// === build_input_text_script ===

///|
test "build_input_text_script: sends text to terminal" {
  let script = build_input_text_script("42", "ls -la\\n")
  inspect(script.contains("input text"), content="true")
  inspect(script.contains("terminal id \"42\""), content="true")
}

///|
test "build_input_text_script: appends return when text ends with newline" {
  let script = build_input_text_script("42", "echo hello\n")
  // return を結合する形式になる
  inspect(script.contains("(\"echo hello\" & return)"), content="true")
  // 末尾 \n は取り除かれている
  inspect(script.contains("\\n"), content="false")
}

///|
test "build_input_text_script: escapes quotes in text" {
  let script = build_input_text_script("1", "echo \"hello\"")
  // The script should contain escaped quotes
  inspect(script.contains("\\\""), content="true")
}

// === build_send_key_script ===

///|
test "build_send_key_script: simple key" {
  let script = build_send_key_script("42", "c", "control")
  inspect(script.contains("send key"), content="true")
  inspect(script.contains("terminal id \"42\""), content="true")
  inspect(script.contains("modifiers \"control\""), content="true")
}

///|
test "build_send_key_script: Enter key (no modifiers)" {
  let script = build_send_key_script("42", "return", "")
  inspect(script.contains("send key \"return\""), content="true")
  // No modifiers clause when empty
  inspect(script.contains("modifiers"), content="false")
}

// === build_focus_script ===

///|
test "build_focus_script: focuses terminal by id" {
  let script = build_focus_script("7")
  inspect(script.contains("focus terminal id \"7\""), content="true")
}

// === build_close_script ===

///|
test "build_close_script: closes terminal by id" {
  let script = build_close_script("5")
  inspect(script.contains("close terminal id \"5\""), content="true")
}

// === build_list_panes_script ===

///|
test "build_list_panes_script: queries all terminals" {
  let script = build_list_panes_script()
  inspect(script.contains("every terminal"), content="true")
  inspect(script.contains("id of t"), content="true")
}

// === build_resize_script ===

///|
test "build_resize_script: resize right" {
  let script = build_resize_script("42", "right", 50)
  inspect(script.contains("resize_split"), content="true")
  inspect(script.contains("terminal id \"42\""), content="true")
}

// === tmux_keys_to_ghostty ===

///|
test "tmux_keys_to_ghostty: Enter key" {
  let (key, modifiers) = tmux_keys_to_ghostty("Enter")
  assert_eq(key, "return")
  assert_eq(modifiers, "")
}

///|
test "tmux_keys_to_ghostty: C-c" {
  let (key, modifiers) = tmux_keys_to_ghostty("C-c")
  assert_eq(key, "c")
  assert_eq(modifiers, "control")
}

///|
test "tmux_keys_to_ghostty: C-d" {
  let (key, modifiers) = tmux_keys_to_ghostty("C-d")
  assert_eq(key, "d")
  assert_eq(modifiers, "control")
}

///|
test "tmux_keys_to_ghostty: Escape" {
  let (key, modifiers) = tmux_keys_to_ghostty("Escape")
  assert_eq(key, "escape")
  assert_eq(modifiers, "")
}

///|
test "tmux_keys_to_ghostty: Tab" {
  let (key, modifiers) = tmux_keys_to_ghostty("Tab")
  assert_eq(key, "tab")
  assert_eq(modifiers, "")
}

///|
test "tmux_keys_to_ghostty: Space" {
  let (key, modifiers) = tmux_keys_to_ghostty("Space")
  assert_eq(key, "space")
  assert_eq(modifiers, "")
}

///|
test "tmux_keys_to_ghostty: BSpace" {
  let (key, modifiers) = tmux_keys_to_ghostty("BSpace")
  assert_eq(key, "delete")
  assert_eq(modifiers, "")
}

///|
test "tmux_keys_to_ghostty: plain text as literal" {
  let (key, modifiers) = tmux_keys_to_ghostty("ls")
  assert_eq(key, "ls")
  assert_eq(modifiers, "")
}

// === build_osascript_cmd ===

///|
test "build_osascript_cmd: wraps script in osascript -e" {
  let cmd = build_osascript_cmd("tell application \"Ghostty\" to beep")
  inspect(cmd.contains("osascript"), content="true")
  inspect(cmd.contains("-e"), content="true")
}

// === parse_list_panes_output ===

///|
test "parse_list_panes_output: single line" {
  let panes = parse_list_panes_output("42,myterm,/home/user")
  assert_eq(panes.length(), 1)
  assert_eq(panes[0].id, @core.PaneId("%42"))
  assert_eq(panes[0].name, "myterm")
  assert_eq(panes[0].cwd, "/home/user")
}

///|
test "parse_list_panes_output: multiple lines" {
  let panes = parse_list_panes_output("1,term1,/home\n2,term2,/tmp")
  assert_eq(panes.length(), 2)
  assert_eq(panes[0].id, @core.PaneId("%1"))
  assert_eq(panes[1].id, @core.PaneId("%2"))
}

///|
test "parse_list_panes_output: empty string" {
  let panes = parse_list_panes_output("")
  assert_eq(panes.length(), 0)
}

///|
test "parse_list_panes_output: trailing newline ignored" {
  let panes = parse_list_panes_output("42,myterm,/home\n")
  assert_eq(panes.length(), 1)
}

// === format_var_expand ===

///|
test "format_var_expand: session name" {
  let state : @core.SessionState = {
    name: "my-session",
    panes: [],
    next_pane_id: 1,
  }
  let result = format_var_expand("#{session_name}", state, None)
  assert_eq(result, "my-session")
}

///|
test "format_var_expand: pane_id with active pane" {
  let pane : @core.PaneInfo = {
    id: @core.PaneId("%42"),
    name: "main",
    cwd: "/home",
    width: 80,
    height: 24,
    active: true,
  }
  let state : @core.SessionState = {
    name: "sess",
    panes: [pane],
    next_pane_id: 2,
  }
  let result = format_var_expand("#{pane_id}", state, Some(pane))
  assert_eq(result, "%42")
}

///|
test "format_var_expand: multiple variables" {
  let pane : @core.PaneInfo = {
    id: @core.PaneId("%1"),
    name: "editor",
    cwd: "/home/user",
    width: 120,
    height: 40,
    active: true,
  }
  let state : @core.SessionState = {
    name: "work",
    panes: [pane],
    next_pane_id: 2,
  }
  let result = format_var_expand(
    "#{session_name}:#{pane_id}:#{pane_width}x#{pane_height}",
    state,
    Some(pane),
  )
  assert_eq(result, "work:%1:120x40")
}

///|
test "format_var_expand: pane_current_path" {
  let pane : @core.PaneInfo = {
    id: @core.PaneId("%5"),
    name: "shell",
    cwd: "/tmp/work",
    width: 80,
    height: 24,
    active: true,
  }
  let state : @core.SessionState = { name: "s", panes: [pane], next_pane_id: 6 }
  let result = format_var_expand("#{pane_current_path}", state, Some(pane))
  assert_eq(result, "/tmp/work")
}

///|
test "format_var_expand: no variable" {
  let state : @core.SessionState = { name: "s", panes: [], next_pane_id: 1 }
  let result = format_var_expand("plain text", state, None)
  assert_eq(result, "plain text")
}

///|
test "format_var_expand: window_index always 0" {
  let state : @core.SessionState = { name: "s", panes: [], next_pane_id: 1 }
  let result = format_var_expand("#{window_index}", state, None)
  assert_eq(result, "0")
}

///|
test "format_var_expand: pane_active" {
  let pane_active : @core.PaneInfo = {
    id: @core.PaneId("%1"),
    name: "a",
    cwd: "/",
    width: 80,
    height: 24,
    active: true,
  }
  let pane_inactive : @core.PaneInfo = {
    id: @core.PaneId("%2"),
    name: "b",
    cwd: "/",
    width: 80,
    height: 24,
    active: false,
  }
  let state : @core.SessionState = {
    name: "s",
    panes: [pane_active, pane_inactive],
    next_pane_id: 3,
  }
  assert_eq(format_var_expand("#{pane_active}", state, Some(pane_active)), "1")
  assert_eq(
    format_var_expand("#{pane_active}", state, Some(pane_inactive)),
    "0",
  )
}

///|
test "format_var_expand: unclosed variable marker outputs literal" {
  let state : @core.SessionState = { name: "s", panes: [], next_pane_id: 1 }
  let result = format_var_expand("#{broken", state, None)
  assert_eq(result, "#{broken")
}

///|
test "format_var_expand: empty variable name" {
  let state : @core.SessionState = { name: "s", panes: [], next_pane_id: 1 }
  let result = format_var_expand("#{}", state, None)
  assert_eq(result, "")
}

///|
test "format_var_expand: unknown variable name" {
  let state : @core.SessionState = { name: "s", panes: [], next_pane_id: 1 }
  let result = format_var_expand("#{unknown_var}", state, None)
  assert_eq(result, "")
}

///|
test "format_var_expand: hash not followed by brace" {
  let state : @core.SessionState = { name: "s", panes: [], next_pane_id: 1 }
  let result = format_var_expand("#abc", state, None)
  assert_eq(result, "#abc")
}

///|
test "format_var_expand: plain text without variables" {
  let state : @core.SessionState = { name: "s", panes: [], next_pane_id: 1 }
  let result = format_var_expand("plain text", state, None)
  assert_eq(result, "plain text")
}

// === tmux_keys_to_ghostty: Arrow keys ===

///|
test "tmux_keys_to_ghostty: Up arrow" {
  let (key, modifiers) = tmux_keys_to_ghostty("Up")
  assert_eq(key, "up")
  assert_eq(modifiers, "")
}

///|
test "tmux_keys_to_ghostty: Down arrow" {
  let (key, modifiers) = tmux_keys_to_ghostty("Down")
  assert_eq(key, "down")
  assert_eq(modifiers, "")
}

///|
test "tmux_keys_to_ghostty: Left arrow" {
  let (key, modifiers) = tmux_keys_to_ghostty("Left")
  assert_eq(key, "left")
  assert_eq(modifiers, "")
}

///|
test "tmux_keys_to_ghostty: Right arrow" {
  let (key, modifiers) = tmux_keys_to_ghostty("Right")
  assert_eq(key, "right")
  assert_eq(modifiers, "")
}
