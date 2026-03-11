/// tmux CLI パーサ
///
/// tmux のコマンドライン引数を TmuxCommand に変換する。
/// Claude Code Agent Teams が発行するコマンドのみを対象とする。

///|
/// 引数配列から次の値を取得するヘルパー
fn next_arg(
  args : Array[String],
  i : Ref[Int],
  opt : String,
) -> String raise ParseError {
  let next = i.val + 1
  if next >= args.length() {
    raise MissingArgument(opt)
  }
  i.val = next
  args[next]
}

///|
/// tmux コマンドライン引数をパースする
pub fn parse(args : Array[String]) -> TmuxCommand raise ParseError {
  if args.length() == 0 {
    raise EmptyInput
  }
  let i : Ref[Int] = Ref::new(0)
  // Phase 1: グローバルオプションをスキップ
  while i.val < args.length() {
    let arg = args[i.val]
    if arg == "-V" {
      return Version
    } else if arg == "-L" || arg == "-S" || arg == "-f" {
      // 次の引数（値）もスキップ
      i.val = i.val + 2
    } else {
      break
    }
  }
  if i.val >= args.length() {
    raise EmptyInput
  }
  // Phase 2: コマンド名を取得
  let command = args[i.val]
  i.val = i.val + 1
  // Phase 3: コマンド固有オプションをパース
  match command {
    "has-session" => parse_has_session(args, i)
    "switch-client" => parse_switch_client(args, i)
    "new-session" => parse_new_session(args, i)
    "show-options" => parse_show_options(args, i)
    "set-option" | "set" => parse_set_option(args, i)
    "split-window" => parse_split_window(args, i)
    "new-window" => parse_new_window(args, i)
    "send-keys" => parse_send_keys(args, i)
    "select-pane" => parse_select_pane(args, i)
    "kill-pane" => parse_kill_pane(args, i)
    "list-panes" => parse_list_panes(args, i)
    "display-message" => parse_display_message(args, i)
    _ => Unknown(command)
  }
}

///|
/// has-session: -t target
fn parse_has_session(
  args : Array[String],
  i : Ref[Int],
) -> TmuxCommand raise ParseError {
  let mut target = ""
  while i.val < args.length() {
    match args[i.val] {
      "-t" => target = next_arg(args, i, "-t")
      _ => ()
    }
    i.val = i.val + 1
  }
  HasSession(target~)
}

///|
/// switch-client: -t target
fn parse_switch_client(
  args : Array[String],
  i : Ref[Int],
) -> TmuxCommand raise ParseError {
  let mut target = ""
  while i.val < args.length() {
    match args[i.val] {
      "-t" => target = next_arg(args, i, "-t")
      _ => ()
    }
    i.val = i.val + 1
  }
  SwitchClient(target~)
}

///|
/// new-session: -A -s session -c cwd -- command...
fn parse_new_session(
  args : Array[String],
  i : Ref[Int],
) -> TmuxCommand raise ParseError {
  let mut attach_if_exists = false
  let mut detached = false
  let mut session = ""
  let mut cwd = ""
  let command : Array[String] = []
  while i.val < args.length() {
    match args[i.val] {
      "-A" => attach_if_exists = true
      "-d" => detached = true
      "-s" => session = next_arg(args, i, "-s")
      "-c" => cwd = next_arg(args, i, "-c")
      "-n" | "-P" | "-F" => {
        // -n window-name, -P print, -F format: 値を消費してスキップ
        let _ = next_arg(args, i, args[i.val])
      }
      "--" => {
        i.val = i.val + 1
        while i.val < args.length() {
          command.push(args[i.val])
          i.val = i.val + 1
        }
        break
      }
      _ => ()
    }
    i.val = i.val + 1
  }
  NewSession(attach_if_exists~, detached~, session~, cwd~, command~)
}

///|
/// show-options: -g -p -v -t target name
fn parse_show_options(
  args : Array[String],
  i : Ref[Int],
) -> TmuxCommand raise ParseError {
  let mut global = false
  let mut pane = false
  let mut value_only = false
  let mut target = ""
  let mut name = ""
  while i.val < args.length() {
    match args[i.val] {
      "-g" => global = true
      "-p" => pane = true
      "-v" => value_only = true
      "-gv" | "-vg" => {
        global = true
        value_only = true
      }
      "-pv" | "-vp" => {
        pane = true
        value_only = true
      }
      "-t" => target = next_arg(args, i, "-t")
      _ => name = args[i.val]
    }
    i.val = i.val + 1
  }
  ShowOptions(global~, pane~, value_only~, target~, name~)
}

///|
/// set-option / set: -p -w -u -t target name value
fn parse_set_option(
  args : Array[String],
  i : Ref[Int],
) -> TmuxCommand raise ParseError {
  let mut pane = false
  let mut window = false
  let mut unset = false
  let mut target = ""
  let mut name = ""
  let mut value = ""
  while i.val < args.length() {
    match args[i.val] {
      "-p" => pane = true
      "-w" => window = true
      "-u" => unset = true
      "-t" => target = next_arg(args, i, "-t")
      _ => if name == "" { name = args[i.val] } else { value = args[i.val] }
    }
    i.val = i.val + 1
  }
  SetOption(pane~, window~, unset~, target~, name~, value~)
}

///|
/// split-window: -h -t target -l size -P -F format
fn parse_split_window(
  args : Array[String],
  i : Ref[Int],
) -> TmuxCommand raise ParseError {
  let mut horizontal = false
  let mut target = ""
  let mut size = ""
  let mut print_info = false
  let mut format = ""
  while i.val < args.length() {
    match args[i.val] {
      "-h" => horizontal = true
      "-t" => target = next_arg(args, i, "-t")
      "-l" => size = next_arg(args, i, "-l")
      "-P" => print_info = true
      "-F" => format = next_arg(args, i, "-F")
      _ => ()
    }
    i.val = i.val + 1
  }
  SplitWindow(horizontal~, target~, size~, print_info~, format~)
}

///|
/// new-window: -t session -n name -P -F format
fn parse_new_window(
  args : Array[String],
  i : Ref[Int],
) -> TmuxCommand raise ParseError {
  let mut session = ""
  let mut name = ""
  let mut print_info = false
  let mut format = ""
  while i.val < args.length() {
    match args[i.val] {
      "-t" => session = next_arg(args, i, "-t")
      "-n" => name = next_arg(args, i, "-n")
      "-P" => print_info = true
      "-F" => format = next_arg(args, i, "-F")
      _ => ()
    }
    i.val = i.val + 1
  }
  NewWindow(session~, name~, print_info~, format~)
}

///|
/// send-keys: -t target keys...
fn parse_send_keys(
  args : Array[String],
  i : Ref[Int],
) -> TmuxCommand raise ParseError {
  let mut target = ""
  let keys : Array[String] = []
  while i.val < args.length() {
    match args[i.val] {
      "-t" => target = next_arg(args, i, "-t")
      _ => keys.push(args[i.val])
    }
    i.val = i.val + 1
  }
  SendKeys(target~, keys~)
}

///|
/// select-pane: -t target -P style -T title
fn parse_select_pane(
  args : Array[String],
  i : Ref[Int],
) -> TmuxCommand raise ParseError {
  let mut target = ""
  let mut style = ""
  let mut title = ""
  while i.val < args.length() {
    match args[i.val] {
      "-t" => target = next_arg(args, i, "-t")
      "-P" => style = next_arg(args, i, "-P")
      "-T" => title = next_arg(args, i, "-T")
      _ => ()
    }
    i.val = i.val + 1
  }
  SelectPane(target~, style~, title~)
}

///|
/// kill-pane: -t target
fn parse_kill_pane(
  args : Array[String],
  i : Ref[Int],
) -> TmuxCommand raise ParseError {
  let mut target = ""
  while i.val < args.length() {
    match args[i.val] {
      "-t" => target = next_arg(args, i, "-t")
      _ => ()
    }
    i.val = i.val + 1
  }
  KillPane(target~)
}

///|
/// list-panes: -F format
fn parse_list_panes(
  args : Array[String],
  i : Ref[Int],
) -> TmuxCommand raise ParseError {
  let mut format = ""
  while i.val < args.length() {
    match args[i.val] {
      "-F" => format = next_arg(args, i, "-F")
      _ => ()
    }
    i.val = i.val + 1
  }
  ListPanes(format~)
}

///|
/// display-message: -p format
fn parse_display_message(args : Array[String], i : Ref[Int]) -> TmuxCommand {
  let mut print = false
  let mut format = ""
  while i.val < args.length() {
    match args[i.val] {
      "-p" => print = true
      _ => format = args[i.val]
    }
    i.val = i.val + 1
  }
  DisplayMessage(print~, format~)
}
