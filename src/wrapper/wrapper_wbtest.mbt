// wrapper_wbtest.mbt - Wrapper モードのロジックテスト
//
// FFI を使わない純粋関数のテスト。
// 実際の子プロセス起動は統合テストで扱う。

// --- セッション名の生成 ---

///|
test "generate_session_name with simple command" {
  let name = generate_session_name("claude", 12345)
  assert_eq(name, "shimux-claude-12345")
}

///|
test "generate_session_name with path command extracts basename" {
  let name = generate_session_name("/usr/bin/claude", 99)
  assert_eq(name, "shimux-claude-99")
}

///|
test "generate_session_name with trailing slash" {
  // エッジケース: パスが / で終わる場合
  let name = generate_session_name("/usr/bin/", 1)
  assert_eq(name, "shimux-bin-1")
}

// --- TMUX 環境変数の組み立て ---

///|
test "build_tmux_env with tmpdir and pid" {
  let env = build_tmux_env("/tmp/shimux.abc123", 42)
  assert_eq(env, "/tmp/shimux.abc123/shimux,42,0")
}

// --- symlink パスの構築 ---

///|
test "build_symlink_path" {
  let path = build_symlink_path("/tmp/shimux.abc123", "tmux")
  assert_eq(path, "/tmp/shimux.abc123/tmux")
}

// --- PATH の組み立て ---

///|
test "build_path_env prepends tmpdir to existing PATH" {
  let path = build_path_env("/tmp/shimux.abc123", "/usr/bin:/bin")
  assert_eq(path, "/tmp/shimux.abc123:/usr/bin:/bin")
}

///|
test "build_path_env with empty existing PATH" {
  let path = build_path_env("/tmp/shimux.abc123", "")
  assert_eq(path, "/tmp/shimux.abc123")
}

// --- コマンド名の抽出 ---

///|
test "extract_command_name from simple name" {
  let name = extract_command_name("claude")
  assert_eq(name, "claude")
}

///|
test "extract_command_name from absolute path" {
  let name = extract_command_name("/usr/local/bin/claude")
  assert_eq(name, "claude")
}

///|
test "extract_command_name from relative path" {
  let name = extract_command_name("./bin/claude")
  assert_eq(name, "claude")
}

// --- WrapperConfig の構築 ---

///|
test "WrapperConfig construction" {
  let config : WrapperConfig = {
    tmpdir: "/tmp/shimux.abc123",
    self_exe: "/usr/local/bin/shimux",
    session_name: "shimux-claude-42",
    pid: 42,
    command: ["claude", "--worktree", "--tmux"],
  }
  assert_eq(config.tmpdir, "/tmp/shimux.abc123")
  assert_eq(config.self_exe, "/usr/local/bin/shimux")
  assert_eq(config.session_name, "shimux-claude-42")
  assert_eq(config.pid, 42)
  assert_eq(config.command.length(), 3)
}

// --- 環境変数一覧の生成 ---

///|
test "build_env_vars returns all required variables" {
  let config : WrapperConfig = {
    tmpdir: "/tmp/shimux.abc123",
    self_exe: "/usr/local/bin/shimux",
    session_name: "shimux-claude-42",
    pid: 42,
    command: ["claude"],
  }
  let vars = build_env_vars(config, "/usr/bin:/bin")
  // vars は (name, value) のペア配列
  let tmux = find_env_var(vars, "TMUX")
  assert_eq(tmux, Some("/tmp/shimux.abc123/shimux,42,0"))
  let tmux_pane = find_env_var(vars, "TMUX_PANE")
  assert_eq(tmux_pane, Some("%0"))
  let session = find_env_var(vars, "SHIMUX_SESSION")
  assert_eq(session, Some("shimux-claude-42"))
  let path = find_env_var(vars, "PATH")
  assert_eq(path, Some("/tmp/shimux.abc123:/usr/bin:/bin"))
}

// --- シェルクオート ---

///|
test "shell_quote simple string" {
  let q = shell_quote("hello")
  assert_eq(q, "'hello'")
}

///|
test "shell_quote with single quote" {
  let q = shell_quote("it's")
  assert_eq(q, "'it'\\''s'")
}

///|
test "shell_quote with spaces" {
  let q = shell_quote("hello world")
  assert_eq(q, "'hello world'")
}

///|
test "shell_quote empty string" {
  let q = shell_quote("")
  assert_eq(q, "''")
}

// --- シェルコマンドの構築 ---

///|
test "build_shell_command with simple command" {
  let cmd = build_shell_command(["echo", "hello"])
  assert_eq(cmd, "exec 'echo' 'hello'")
}

///|
test "build_shell_command with complex args" {
  let cmd = build_shell_command(["claude", "--worktree", "--tmux"])
  assert_eq(cmd, "exec 'claude' '--worktree' '--tmux'")
}

///|
/// テスト用ヘルパー: 環境変数リストから名前で検索
fn find_env_var(vars : Array[(String, String)], name : String) -> String? {
  for pair in vars {
    if pair.0 == name {
      return Some(pair.1)
    }
  }
  None
}
