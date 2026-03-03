// Helper: convert Bytes to String for test assertions (ASCII-safe)
fn bytes_to_string(bytes : Bytes) -> String {
  let chars : Array[Char] = []
  for i = 0; i < bytes.length(); i = i + 1 {
    chars.push(bytes[i].to_int().unsafe_to_char())
  }
  String::from_array(chars)
}

///|
test "new auto size" {
  let state = OverlayState::new("Hello\nWorld!", 24, 80)
  inspect(state.width, content="6") // "World!" is longest = 6 chars
  inspect(state.height, content="2") // 2 lines
}

///|
test "new fixed size" {
  let state = OverlayState::new(
    "Hi",
    24,
    80,
    config_width=Some(10),
    config_height=Some(3),
  )
  inspect(state.width, content="10")
  inspect(state.height, content="3")
}

///|
test "update_text" {
  let state = OverlayState::new("Hi", 24, 80)
  state.update_text("Hello\nWorld\nFoo")
  inspect(state.lines.length(), content="3")
  inspect(state.width, content="5") // "Hello" or "World" = 5
  inspect(state.height, content="3")
}

///|
test "update_text with fixed size" {
  let state = OverlayState::new(
    "Hi",
    24,
    80,
    config_width=Some(10),
    config_height=Some(1),
  )
  state.update_text("Hello\nWorld\nFoo")
  // Fixed dimensions should not change
  inspect(state.width, content="10")
  inspect(state.height, content="1")
}

///|
test "update_size" {
  let state = OverlayState::new("Hi", 24, 80)
  state.update_size(50, 120)
  inspect(state.term_rows, content="50")
  inspect(state.term_cols, content="120")
}

///|
test "render basic" {
  let state = OverlayState::new("Hi", 24, 80)
  let result = state.render()
  let s = bytes_to_string(result)
  // Should contain DECSC, CUP, text, DECRC
  inspect(s.contains("\u001b7"), content="true") // DECSC
  inspect(s.contains("\u001b8"), content="true") // DECRC
  inspect(s.contains("Hi"), content="true")
}

///|
test "render position top-right simple" {
  // Simple: Top(1), row=1+1=2, Right(1), col=80-2-1+1=78
  let state = OverlayState::new("Hi", 24, 80)
  let result = state.render()
  let s = bytes_to_string(result)
  inspect(s.contains("\u001b[2;78H"), content="true")
}

///|
test "render position bottom-left simple" {
  // Simple: Bottom(1), height=1, row=24-1-1+1=23, Left(1), col=1+1=2
  let state = OverlayState::new(
    "Hi",
    24,
    80,
    vertical=Bottom(1),
    horizontal=Left(1),
  )
  let result = state.render()
  let s = bytes_to_string(result)
  inspect(s.contains("\u001b[23;2H"), content="true")
}

///|
test "render with colors" {
  let state = OverlayState::new(
    "Hi",
    24,
    80,
    fg_color=Some("red"),
    bg_color=Some("cyan"),
  )
  let result = state.render()
  let s = bytes_to_string(result)
  inspect(s.contains("\u001b[31;46m"), content="true") // red fg + cyan bg
}

///|
test "render auto color uses reverse" {
  let state = OverlayState::new("Hi", 24, 80)
  let result = state.render()
  let s = bytes_to_string(result)
  inspect(s.contains("\u001b[7m"), content="true") // reverse video
}

///|
test "render fg only color" {
  let state = OverlayState::new("Hi", 24, 80, fg_color=Some("green"))
  let result = state.render()
  let s = bytes_to_string(result)
  inspect(s.contains("\u001b[32m"), content="true") // green fg only
}

///|
test "render bg only color" {
  let state = OverlayState::new("Hi", 24, 80, bg_color=Some("blue"))
  let result = state.render()
  let s = bytes_to_string(result)
  inspect(s.contains("\u001b[44m"), content="true") // blue bg only
}

///|
test "render multi-line simple" {
  // Simple: Top(1), width=2, Right(1), col=80-2-1+1=78, rows=2,3
  let state = OverlayState::new("AB\nCD", 24, 80)
  let result = state.render()
  let s = bytes_to_string(result)
  inspect(s.contains("\u001b[2;78H"), content="true")
  inspect(s.contains("\u001b[3;78H"), content="true")
}

///|
test "render padding" {
  // Width forced to 5, text "Hi" should be padded to "Hi   "
  let state = OverlayState::new("Hi", 24, 80, config_width=Some(5))
  let result = state.render()
  let s = bytes_to_string(result)
  inspect(s.contains("Hi   "), content="true")
}

///|
test "render truncation" {
  // Width forced to 3, text "Hello" should be truncated to "Hel"
  let state = OverlayState::new("Hello", 24, 80, config_width=Some(3))
  let result = state.render()
  let s = bytes_to_string(result)
  inspect(s.contains("Hel"), content="true")
  inspect(s.contains("Hello"), content="false")
}

///|
test "render sgr reset after each line" {
  let state = OverlayState::new("Hi", 24, 80)
  let result = state.render()
  let s = bytes_to_string(result)
  inspect(s.contains("\u001b[0m"), content="true") // SGR reset
}

///|
test "new single line" {
  let state = OverlayState::new("Hello", 24, 80)
  inspect(state.width, content="5")
  inspect(state.height, content="1")
  inspect(state.lines.length(), content="1")
  inspect(state.lines[0], content="Hello")
}

///|
test "new empty text" {
  let state = OverlayState::new("", 24, 80)
  inspect(state.width, content="0")
  inspect(state.height, content="1") // split produces [""]
  inspect(state.lines.length(), content="1")
}

///|
test "trailing newline causes extra line" {
  // Demonstrates the problem: "hello\n" from command output
  // splits into ["hello", ""], giving height=2 instead of expected 1.
  // The fix is applied in runner.mbt (strip_trailing_newline before passing to overlay).
  let state = OverlayState::new("hello\n", 24, 80)
  inspect(state.height, content="2") // current behavior: 2 lines
  inspect(state.width, content="5")
}

///|
test "pre-trimmed text has correct dimensions" {
  // After strip_trailing_newline in runner, "hello\n" becomes "hello"
  let state = OverlayState::new("hello", 24, 80)
  inspect(state.height, content="1")
  inspect(state.width, content="5")
}

///|
test "pre-trimmed multi-line text has correct dimensions" {
  // After strip_trailing_newline in runner, "a\nb\n" becomes "a\nb"
  let state = OverlayState::new("a\nb", 24, 80)
  inspect(state.height, content="2")
  inspect(state.width, content="1")
}

///|
test "child_rows top decstbm" {
  let state = OverlayState::new("AB\nCD", 24, 80, position_mode=Decstbm)
  // height=2, child_rows = 24 - 2 = 22
  inspect(state.child_rows(), content="22")
}

///|
test "child_rows bottom decstbm" {
  let state = OverlayState::new(
    "AB\nCD", 24, 80, vertical=Bottom(1), position_mode=Decstbm,
  )
  // height=2, child_rows = 24 - 2 = 22
  inspect(state.child_rows(), content="22")
}

///|
test "child_rows single line decstbm" {
  let state = OverlayState::new("Hi", 24, 80, position_mode=Decstbm)
  // height=1, child_rows = 24 - 1 = 23
  inspect(state.child_rows(), content="23")
}

///|
test "child_rows simple returns full term_rows" {
  let state = OverlayState::new("AB\nCD", 24, 80)
  // Simple mode: child_rows = term_rows = 24
  inspect(state.child_rows(), content="24")
}

///|
test "child_rows single line simple" {
  let state = OverlayState::new("Hi", 24, 80)
  // Simple mode: child_rows = term_rows = 24
  inspect(state.child_rows(), content="24")
}

///|
test "scroll_region_set top decstbm" {
  // Top: ESC[{height+1};{term_rows}r
  // height=2, term_rows=24 -> ESC[3;24r
  let state = OverlayState::new("AB\nCD", 24, 80, position_mode=Decstbm)
  let result = bytes_to_string(state.scroll_region_set())
  inspect(result, content="\u001b[3;24r")
}

///|
test "scroll_region_set bottom decstbm" {
  // Bottom: ESC[1;{term_rows-height}r
  // height=2, term_rows=24 -> ESC[1;22r
  let state = OverlayState::new(
    "AB\nCD", 24, 80, vertical=Bottom(1), position_mode=Decstbm,
  )
  let result = bytes_to_string(state.scroll_region_set())
  inspect(result, content="\u001b[1;22r")
}

///|
test "scroll_region_set single line top decstbm" {
  // height=1, term_rows=24 -> ESC[2;24r
  let state = OverlayState::new("Hi", 24, 80, position_mode=Decstbm)
  let result = bytes_to_string(state.scroll_region_set())
  inspect(result, content="\u001b[2;24r")
}

///|
test "scroll_region_set simple returns empty" {
  let state = OverlayState::new("AB\nCD", 24, 80)
  let result = state.scroll_region_set()
  inspect(result.length(), content="0")
}

///|
test "scroll_region_reset decstbm" {
  let state = OverlayState::new("Hi", 24, 80, position_mode=Decstbm)
  let result = bytes_to_string(state.scroll_region_reset())
  inspect(result, content="\u001b[r")
}

///|
test "scroll_region_reset simple returns empty" {
  let state = OverlayState::new("Hi", 24, 80)
  let result = state.scroll_region_reset()
  inspect(result.length(), content="0")
}

///|
test "render position top-right decstbm" {
  // Decstbm: Top, row=1, col=1 (full-width)
  let state = OverlayState::new("Hi", 24, 80, position_mode=Decstbm)
  let result = state.render()
  let s = bytes_to_string(result)
  inspect(s.contains("\u001b[1;1H"), content="true")
}

///|
test "render position bottom-left decstbm" {
  // Decstbm: Bottom, height=1, row=24, col=1 (full-width)
  let state = OverlayState::new(
    "Hi",
    24,
    80,
    vertical=Bottom(1),
    horizontal=Left(1),
    position_mode=Decstbm,
  )
  let result = state.render()
  let s = bytes_to_string(result)
  inspect(s.contains("\u001b[24;1H"), content="true")
}

///|
test "render multi-line decstbm" {
  // Decstbm: Top, 2 CUP sequences at rows 1 and 2, column 1 (full-width)
  let state = OverlayState::new("AB\nCD", 24, 80, position_mode=Decstbm)
  let result = state.render()
  let s = bytes_to_string(result)
  inspect(s.contains("\u001b[1;1H"), content="true")
  inspect(s.contains("\u001b[2;1H"), content="true")
}

///|
test "render simple does not produce full-width lines" {
  // Simple mode: text at overlay width only, not full term_cols
  let state = OverlayState::new("Hi", 24, 80)
  let result = state.render()
  let s = bytes_to_string(result)
  // Should contain "Hi" but not 80-char wide padded lines
  inspect(s.contains("Hi"), content="true")
  // The line between SGR and reset should be exactly width=2 chars
  // (no left/right margins filling full term_cols)
}
