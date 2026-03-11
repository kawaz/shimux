/// Wrapper モードのセットアップロジック
///
/// `shimux -- command args...` で起動したとき、子プロセスが `tmux` を呼ぶと
/// shimux 自身が PATH 経由で起動するよう環境を整える。

///|
/// Wrapper モードの設定
pub(all) struct WrapperConfig {
  tmpdir : String
  self_exe : String
  session_name : String
  pid : Int
  command : Array[String]
}

///|
/// コマンドパスからベースネーム（ファイル名部分）を抽出する
pub fn extract_command_name(command : String) -> String {
  // 末尾の / を除去
  let mut s = command
  while s.length() > 1 && s[s.length() - 1] == '/' {
    s = s[:s.length() - 1].to_string()
  }
  // 最後の / 以降を取得
  let mut last_slash = -1
  for i in 0..<s.length() {
    if s[i] == '/' {
      last_slash = i
    }
  }
  if last_slash >= 0 {
    s[last_slash + 1:].to_string()
  } else {
    s
  }
}

///|
/// セッション名を生成する
///
/// `shimux-{command_basename}-{pid}` の形式。
pub fn generate_session_name(command : String, pid : Int) -> String {
  let basename = extract_command_name(command)
  "shimux-" + basename + "-" + pid.to_string()
}

///|
/// TMUX 環境変数の値を組み立てる
///
/// 形式: `{tmpdir}/shimux,{pid},0`
pub fn build_tmux_env(tmpdir : String, pid : Int) -> String {
  tmpdir + "/shimux," + pid.to_string() + ",0"
}

///|
/// symlink のパスを構築する
pub fn build_symlink_path(tmpdir : String, name : String) -> String {
  tmpdir + "/" + name
}

///|
/// PATH 環境変数を構築する（tmpdir を先頭に追加）
pub fn build_path_env(tmpdir : String, existing_path : String) -> String {
  if existing_path.length() == 0 {
    tmpdir
  } else {
    tmpdir + ":" + existing_path
  }
}

///|
/// 設定するべき環境変数一覧を生成する
pub fn build_env_vars(
  config : WrapperConfig,
  existing_path : String,
) -> Array[(String, String)] {
  [
    ("TMUX", build_tmux_env(config.tmpdir, config.pid)),
    ("TMUX_PANE", "%0"),
    ("SHIMUX_SESSION", config.session_name),
    ("PATH", build_path_env(config.tmpdir, existing_path)),
  ]
}

///|
/// シェル用にシングルクオートでエスケープする
///
/// 引数をシングルクオートで囲み、引数内のシングルクオートを
/// `'\''` に置換する。
pub fn shell_quote(s : String) -> String {
  let buf = StringBuilder::new()
  buf.write_char('\'')
  for ch in s {
    if ch == '\'' {
      buf.write_string("'\\''")
    } else {
      buf.write_char(ch)
    }
  }
  buf.write_char('\'')
  buf.to_string()
}

///|
/// コマンド配列をシェルコマンド文字列に結合する
pub fn build_shell_command(command : Array[String]) -> String {
  let parts = Array::new()
  for arg in command {
    parts.push(shell_quote(arg))
  }
  // exec で shimux プロセスを子プロセスに置き換える
  let buf = StringBuilder::new()
  buf.write_string("exec")
  for part in parts {
    buf.write_char(' ')
    buf.write_string(part)
  }
  buf.to_string()
}

///|
/// Wrapper モードのエントリポイント
///
/// 1. 一時ディレクトリを作成
/// 2. tmux → shimux への symlink を作成
/// 3. 環境変数を設定
/// 4. 子プロセスを起動し、その終了コードで終了
pub fn run(command : Array[String]) -> Unit {
  if command.length() == 0 {
    eprintln("shimux: wrapper mode requires a command")
    exit(1)
  }
  // 自分自身の実行ファイルパスを取得
  let exe = self_exe()
  if exe.length() == 0 {
    eprintln("shimux: failed to determine self executable path")
    exit(1)
  }
  // PID 取得
  let pid = getpid()
  // 一時ディレクトリ作成
  let tmpdir = mkdtemp("/tmp/shimux")
  if tmpdir.length() == 0 {
    eprintln("shimux: failed to create temporary directory")
    exit(1)
  }
  // tmux → shimux への symlink 作成
  let link = build_symlink_path(tmpdir, "tmux")
  if !symlink(exe, link) {
    eprintln("shimux: failed to create symlink: " + link)
    exit(1)
  }
  // セッション名生成
  let session_name = generate_session_name(command[0], pid)
  // WrapperConfig 構築
  let config : WrapperConfig = {
    tmpdir,
    self_exe: exe,
    session_name,
    pid,
    command,
  }
  // 環境変数設定
  let existing_path = getenv("PATH")
  let env_vars = build_env_vars(config, existing_path)
  for pair in env_vars {
    if !setenv(pair.0, pair.1) {
      eprintln("shimux: failed to set environment variable: " + pair.0)
      exit(1)
    }
  }
  // 子プロセスを起動
  let shell_cmd = build_shell_command(command)
  let exit_code = run_command(shell_cmd)
  exit(exit_code)
}
