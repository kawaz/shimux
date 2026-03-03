///|
test "parse minimal --content" {
  let config = @args.parse(["--content", "Hello", "--", "bash"])
  inspect(config.content_source, content="Static(\"Hello\")")
  inspect(config.command, content="[\"bash\"]")
  inspect(config.interval_ms, content="5000")
  inspect(config.vertical, content="Top(1)")
  inspect(config.horizontal, content="Right(1)")
  inspect(config.position_mode, content="Simple")
}

///|
test "parse --content-command with interval" {
  let config = @args.parse(["--content-command", "date", "--interval", "2s", "--", "bash"])
  inspect(config.content_source, content="Command(\"date\")")
  inspect(config.interval_ms, content="2000")
}

///|
test "parse interval formats" {
  let c1 = @args.parse(["--content", "x", "--interval", "500ms", "--", "bash"])
  inspect(c1.interval_ms, content="500")
  let c2 = @args.parse(["--content", "x", "--interval", "1m", "--", "bash"])
  inspect(c2.interval_ms, content="60000")
  let c3 = @args.parse(["--content", "x", "--interval", "3", "--", "bash"])
  inspect(c3.interval_ms, content="3000")
}

///|
test "parse position options" {
  let c1 = @args.parse(["--content", "x", "--bottom", "2", "--left", "3", "--", "bash"])
  inspect(c1.vertical, content="Bottom(2)")
  inspect(c1.horizontal, content="Left(3)")
}

///|
test "parse size options" {
  let config = @args.parse(["--content", "x", "--width", "20", "--height", "5", "--", "bash"])
  inspect(config.width, content="Some(20)")
  inspect(config.height, content="Some(5)")
}

///|
test "parse size auto" {
  let config = @args.parse(["--content", "x", "--width", "auto", "--", "bash"])
  inspect(config.width, content="None")
}

///|
test "parse color options" {
  let config = @args.parse(["--content", "x", "--fg-color", "red", "--bg-color", "cyan", "--", "bash"])
  inspect(config.fg_color, content="Some(\"red\")")
  inspect(config.bg_color, content="Some(\"cyan\")")
}

///|
test "parse error: no content" {
  let msg = try {
    let _ = @args.parse(["--", "bash"])
    ""
  } catch {
    Failure::Failure(m) => m
    _ => ""
  }
  inspect(msg.contains("content"), content="true")
}

///|
test "parse error: no command" {
  let msg = try {
    let _ = @args.parse(["--content", "x"])
    ""
  } catch {
    Failure::Failure(m) => m
    _ => ""
  }
  inspect(msg.contains("command"), content="true")
}

///|
test "parse error: conflicting vertical" {
  let failed = try {
    let _ = @args.parse(["--content", "x", "--top", "1", "--bottom", "2", "--", "bash"])
    false
  } catch {
    _ => true
  }
  inspect(failed, content="true")
}

///|
test "parse multi-word command" {
  let config = @args.parse(["--content", "x", "--", "ls", "-la", "/tmp"])
  inspect(config.command, content="[\"ls\", \"-la\", \"/tmp\"]")
}

///|
test "parse --help raises __HELP__" {
  let msg = try {
    let _ = @args.parse(["--help"])
    "no_error"
  } catch {
    Failure::Failure(m) => m
    _ => "other_error"
  }
  inspect(msg.contains("__HELP__"), content="true")
}

///|
test "parse unknown option" {
  let msg = try {
    let _ = @args.parse(["--unknown", "--", "bash"])
    ""
  } catch {
    Failure::Failure(m) => m
    _ => ""
  }
  inspect(msg.contains("unknown"), content="true")
}

///|
test "parse both content and content-command" {
  let failed = try {
    let _ = @args.parse(
      ["--content", "x", "--content-command", "y", "--", "bash"],
    )
    false
  } catch {
    _ => true
  }
  inspect(failed, content="true")
}

///|
test "parse error: conflicting horizontal" {
  let failed = try {
    let _ = @args.parse(
      ["--content", "x", "--left", "1", "--right", "2", "--", "bash"],
    )
    false
  } catch {
    _ => true
  }
  inspect(failed, content="true")
}

///|
test "parse error: invalid interval format" {
  let msg = try {
    let _ = @args.parse(["--content", "x", "--interval", "abc", "--", "bash"])
    ""
  } catch {
    Failure::Failure(m) => m
    _ => ""
  }
  inspect(msg.contains("duration"), content="true")
}

///|
test "help_text contains usage line" {
  let text = @args.help_text()
  inspect(text.contains("Usage: tty-overlay"), content="true")
}

///|
test "help_text contains all options" {
  let text = @args.help_text()
  inspect(text.contains("--content TEXT"), content="true")
  inspect(text.contains("--content-command CMD"), content="true")
  inspect(text.contains("--interval DURATION"), content="true")
  inspect(text.contains("--help"), content="true")
  inspect(text.contains("--fg-color"), content="true")
  inspect(text.contains("--bg-color"), content="true")
  inspect(text.contains("--position-mode"), content="true")
}

///|
test "parse --position-mode simple (default)" {
  let config = @args.parse(["--content", "x", "--", "bash"])
  inspect(config.position_mode, content="Simple")
}

///|
test "parse --position-mode decstbm" {
  let config = @args.parse(
    ["--content", "x", "--position-mode", "decstbm", "--", "bash"],
  )
  inspect(config.position_mode, content="Decstbm")
}

///|
test "parse --position-mode cell raises error" {
  let msg = try {
    let _ = @args.parse(
      ["--content", "x", "--position-mode", "cell", "--", "bash"],
    )
    ""
  } catch {
    Failure::Failure(m) => m
    _ => ""
  }
  inspect(msg.contains("cell mode is not yet implemented"), content="true")
}

///|
test "parse --position-mode invalid raises error" {
  let msg = try {
    let _ = @args.parse(
      ["--content", "x", "--position-mode", "invalid", "--", "bash"],
    )
    ""
  } catch {
    Failure::Failure(m) => m
    _ => ""
  }
  inspect(msg.contains("invalid position mode"), content="true")
}
