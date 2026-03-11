// tmux CLI パーサのテスト

// === Version ===

///|
test "parse -V returns Version" {
  let cmd = parse(["-V"])
  assert_eq(cmd, TmuxCommand::Version)
}

// === HasSession ===

///|
test "parse has-session -t mysession" {
  let cmd = parse(["has-session", "-t", "mysession"])
  assert_eq(cmd, TmuxCommand::HasSession(target="mysession"))
}

// === SwitchClient ===

///|
test "parse switch-client -t mysession" {
  let cmd = parse(["switch-client", "-t", "mysession"])
  assert_eq(cmd, TmuxCommand::SwitchClient(target="mysession"))
}

// === NewSession ===

///|
test "parse new-session -A -s test -c /tmp -- vim foo.txt" {
  let cmd = parse([
    "new-session", "-A", "-s", "test", "-c", "/tmp", "--", "vim", "foo.txt",
  ])
  assert_eq(
    cmd,
    TmuxCommand::NewSession(
      attach_if_exists=true,
      detached=false,
      session="test",
      cwd="/tmp",
      command=["vim", "foo.txt"],
    ),
  )
}

///|
test "parse new-session -s test (minimal)" {
  let cmd = parse(["new-session", "-s", "test"])
  assert_eq(
    cmd,
    TmuxCommand::NewSession(
      attach_if_exists=false,
      detached=false,
      session="test",
      cwd="",
      command=[],
    ),
  )
}

///|
test "parse new-session -d -s hidden" {
  let cmd = parse(["new-session", "-d", "-s", "claude-hidden"])
  assert_eq(
    cmd,
    TmuxCommand::NewSession(
      attach_if_exists=false,
      detached=true,
      session="claude-hidden",
      cwd="",
      command=[],
    ),
  )
}

// === ShowOptions ===

///|
test "parse show-options -g prefix" {
  let cmd = parse(["show-options", "-g", "prefix"])
  assert_eq(
    cmd,
    TmuxCommand::ShowOptions(
      global=true,
      pane=false,
      value_only=false,
      target="",
      name="prefix",
    ),
  )
}

///|
test "parse show-options -gv prefix" {
  let cmd = parse(["show-options", "-gv", "prefix"])
  assert_eq(
    cmd,
    TmuxCommand::ShowOptions(
      global=true,
      pane=false,
      value_only=true,
      target="",
      name="prefix",
    ),
  )
}

///|
test "parse show-options -p -v window-active-style" {
  let cmd = parse(["show-options", "-p", "-v", "window-active-style"])
  assert_eq(
    cmd,
    TmuxCommand::ShowOptions(
      global=false,
      pane=true,
      value_only=true,
      target="",
      name="window-active-style",
    ),
  )
}

///|
test "parse set -p window-active-style bg=#404040" {
  let cmd = parse(["set", "-p", "window-active-style", "bg=#404040"])
  assert_eq(
    cmd,
    TmuxCommand::SetOption(
      pane=true,
      window=false,
      unset=false,
      target="",
      name="window-active-style",
      value="bg=#404040",
    ),
  )
}

///|
test "parse set -p -u window-active-style" {
  let cmd = parse(["set", "-p", "-u", "window-active-style"])
  assert_eq(
    cmd,
    TmuxCommand::SetOption(
      pane=true,
      window=false,
      unset=true,
      target="",
      name="window-active-style",
      value="",
    ),
  )
}

// === SplitWindow ===

///|
test "parse split-window -h -t %0 -P -F format" {
  let cmd = parse(["split-window", "-h", "-t", "%0", "-P", "-F", "#{pane_id}"])
  assert_eq(
    cmd,
    TmuxCommand::SplitWindow(
      horizontal=true,
      target="%0",
      size="",
      print_info=true,
      format="#{pane_id}",
    ),
  )
}

///|
test "parse split-window -h -l 30% -t %1" {
  let cmd = parse(["split-window", "-h", "-l", "30%", "-t", "%1"])
  assert_eq(
    cmd,
    TmuxCommand::SplitWindow(
      horizontal=true,
      target="%1",
      size="30%",
      print_info=false,
      format="",
    ),
  )
}

// === NewWindow ===

///|
test "parse new-window -t sess -n mywin -P -F format" {
  let cmd = parse([
    "new-window", "-t", "sess", "-n", "mywin", "-P", "-F", "#{window_id}",
  ])
  assert_eq(
    cmd,
    TmuxCommand::NewWindow(
      session="sess",
      name="mywin",
      print_info=true,
      format="#{window_id}",
    ),
  )
}

// === SendKeys ===

///|
test "parse send-keys -t %0 ls Enter" {
  let cmd = parse(["send-keys", "-t", "%0", "ls", "Enter"])
  assert_eq(cmd, TmuxCommand::SendKeys(target="%0", keys=["ls", "Enter"]))
}

///|
test "parse send-keys -t %1 C-c" {
  let cmd = parse(["send-keys", "-t", "%1", "C-c"])
  assert_eq(cmd, TmuxCommand::SendKeys(target="%1", keys=["C-c"]))
}

// === SelectPane ===

///|
test "parse select-pane -t %0 -P bg=#333 -T title" {
  let cmd = parse(["select-pane", "-t", "%0", "-P", "bg=#333", "-T", "title"])
  assert_eq(
    cmd,
    TmuxCommand::SelectPane(target="%0", style="bg=#333", title="title"),
  )
}

// === KillPane ===

///|
test "parse kill-pane -t %0" {
  let cmd = parse(["kill-pane", "-t", "%0"])
  assert_eq(cmd, TmuxCommand::KillPane(target="%0"))
}

// === ListPanes ===

///|
test "parse list-panes -F format" {
  let cmd = parse(["list-panes", "-F", "#{pane_id}:#{pane_title}"])
  assert_eq(cmd, TmuxCommand::ListPanes(format="#{pane_id}:#{pane_title}"))
}

// === DisplayMessage ===

///|
test "parse display-message -p format" {
  let cmd = parse(["display-message", "-p", "#{session_name}"])
  assert_eq(
    cmd,
    TmuxCommand::DisplayMessage(print=true, format="#{session_name}"),
  )
}

// === Unknown ===

///|
test "parse unknown command" {
  let cmd = parse(["some-unknown-command", "-x", "y"])
  assert_eq(cmd, TmuxCommand::Unknown("some-unknown-command"))
}

// === グローバルオプション ===

///|
test "parse global option -L name split-window -h" {
  let cmd = parse(["-L", "claude-swarm", "split-window", "-h"])
  assert_eq(
    cmd,
    TmuxCommand::SplitWindow(
      horizontal=true,
      target="",
      size="",
      print_info=false,
      format="",
    ),
  )
}

///|
test "parse global option -S path has-session -t foo" {
  let cmd = parse(["-S", "/tmp/tmux.sock", "has-session", "-t", "foo"])
  assert_eq(cmd, TmuxCommand::HasSession(target="foo"))
}

///|
test "parse global option -f file send-keys -t %0 ls" {
  let cmd = parse(["-f", "/etc/tmux.conf", "send-keys", "-t", "%0", "ls"])
  assert_eq(cmd, TmuxCommand::SendKeys(target="%0", keys=["ls"]))
}

// === エラーケース ===

///|
test "parse empty input returns error" {
  let result : Result[_, _] = try? parse([])
  inspect(result is Err(_), content="true")
}

///|
test "parse has-session missing -t value returns error" {
  let result : Result[_, _] = try? parse(["has-session", "-t"])
  inspect(result is Err(_), content="true")
}

///|
test "parse split-window missing format value" {
  let result : Result[_, _] = try? parse(["split-window", "-F"])
  inspect(result is Err(_), content="true")
}

///|
test "parse new-session missing session name" {
  let result : Result[_, _] = try? parse(["new-session", "-s"])
  inspect(result is Err(_), content="true")
}

///|
test "parse new-window missing target value" {
  let result : Result[_, _] = try? parse(["new-window", "-t"])
  inspect(result is Err(_), content="true")
}

///|
test "parse select-pane missing style value" {
  let result : Result[_, _] = try? parse(["select-pane", "-P"])
  inspect(result is Err(_), content="true")
}

///|
test "parse set-option missing target value" {
  let result : Result[_, _] = try? parse(["set-option", "-t"])
  inspect(result is Err(_), content="true")
}

///|
test "parse show-options missing target value" {
  let result : Result[_, _] = try? parse(["show-options", "-t"])
  inspect(result is Err(_), content="true")
}

// === split_direction ===

///|
test "split_direction horizontal" {
  inspect(
    split_direction(true) is @core.SplitDirection::Horizontal,
    content="true",
  )
}

///|
test "split_direction vertical" {
  inspect(
    split_direction(false) is @core.SplitDirection::Vertical,
    content="true",
  )
}
