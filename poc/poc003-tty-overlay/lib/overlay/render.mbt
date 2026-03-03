/// Construct a new OverlayState.
/// text is split by "\n" to compute lines.
/// width/height are auto-computed from lines unless config overrides are provided.
pub fn OverlayState::new(
  text : String,
  term_rows : Int,
  term_cols : Int,
  config_width~ : Int? = None,
  config_height~ : Int? = None,
  vertical~ : VerticalPos = Top(1),
  horizontal~ : HorizontalPos = Right(1),
  fg_color~ : String? = None,
  bg_color~ : String? = None,
  position_mode~ : PositionMode = Simple
) -> OverlayState {
  let lines = split_lines(text)
  let width = match config_width {
    Some(w) => w
    None => max_line_length(lines)
  }
  let height = match config_height {
    Some(h) => h
    None => lines.length()
  }
  {
    text,
    lines,
    width,
    height,
    term_rows,
    term_cols,
    config_width,
    config_height,
    vertical,
    horizontal,
    fg_color,
    bg_color,
    position_mode,
  }
}

/// Update the overlay text and recompute lines/width/height (auto only).
pub fn OverlayState::update_text(self : OverlayState, text : String) -> Unit {
  self.text = text
  self.lines = split_lines(text)
  match self.config_width {
    Some(_) => ()
    None => self.width = max_line_length(self.lines)
  }
  match self.config_height {
    Some(_) => ()
    None => self.height = self.lines.length()
  }
}

/// Update terminal dimensions.
pub fn OverlayState::update_size(
  self : OverlayState,
  term_rows : Int,
  term_cols : Int
) -> Unit {
  self.term_rows = term_rows
  self.term_cols = term_cols
}

/// Render the overlay as ANSI escape sequences in UTF-8 bytes.
///
/// Rendering differs by position_mode:
/// - Simple: DECSC+CUP at computed (row,col), each line rendered at width only.
/// - Decstbm: Full-width lines anchored to terminal edges via DECSTBM.
/// - Cell: unreachable (rejected at parse time).
///
/// Sequence structure:
/// 1. DECSC (save cursor)
/// 2. For each line: CUP (move cursor) + SGR (color) + text + SGR reset
/// 3. DECRC (restore cursor)
pub fn OverlayState::render(self : OverlayState) -> Bytes {
  match self.position_mode {
    Simple => self.render_simple()
    Decstbm => self.render_decstbm()
    Cell => abort("unreachable: cell mode should be rejected at parse time")
  }
}

/// Render overlay using DECSC+CUP method (Simple mode).
/// Position is computed from vertical/horizontal offsets. Each line is
/// rendered at overlay width only (not full terminal width).
fn OverlayState::render_simple(self : OverlayState) -> Bytes {
  let buf : Array[Byte] = []
  // 1. DECSC: save cursor position
  write_ascii(buf, "\u001b7")
  // Compute starting row based on vertical position
  let start_row = match self.vertical {
    Top(n) => n + 1
    Bottom(n) => self.term_rows - self.height - n + 1
  }
  // Compute starting column based on horizontal position
  let start_col = match self.horizontal {
    Left(n) => n + 1
    Right(n) => self.term_cols - self.width - n + 1
  }
  // Build SGR sequence
  let sgr = build_sgr(self.fg_color, self.bg_color)
  // 2. Render each line at overlay width
  for i = 0; i < self.height; i = i + 1 {
    let row = start_row + i
    let col = start_col
    // CUP: move cursor to (row, col)
    write_ascii(buf, "\u001b[")
    write_ascii(buf, row.to_string())
    write_ascii(buf, ";")
    write_ascii(buf, col.to_string())
    write_ascii(buf, "H")
    // SGR: color setting
    write_ascii(buf, sgr)
    // Render line padded/truncated to overlay width
    let line = if i < self.lines.length() {
      self.lines[i]
    } else {
      ""
    }
    write_ascii(buf, pad_or_truncate(line, self.width))
    // SGR reset
    write_ascii(buf, "\u001b[0m")
  }
  // 3. DECRC: restore cursor position
  write_ascii(buf, "\u001b8")
  Bytes::from_array(buf[:])
}

/// Render overlay using DECSTBM method.
/// Each line is rendered full-width (term_cols) with text positioned
/// according to the horizontal setting.
fn OverlayState::render_decstbm(self : OverlayState) -> Bytes {
  let buf : Array[Byte] = []
  // 1. DECSC: save cursor position
  write_ascii(buf, "\u001b7")
  // Compute starting row based on vertical position
  // Design rationale: In DECSTBM mode, Top means rows 1..height (always
  // at the top edge), Bottom means rows (term_rows-height+1)..term_rows
  // (always at the bottom edge). The Int parameter is ignored since the
  // overlay is anchored to the edge of the scroll region.
  let start_row = match self.vertical {
    Top(_) => 1
    Bottom(_) => self.term_rows - self.height + 1
  }
  // Build SGR sequence
  let sgr = build_sgr(self.fg_color, self.bg_color)
  // 2. Render each line as full-width (term_cols)
  for i = 0; i < self.height; i = i + 1 {
    let row = start_row + i
    // CUP: move cursor to (row, 1) -- always column 1 for full-width
    write_ascii(buf, "\u001b[")
    write_ascii(buf, row.to_string())
    write_ascii(buf, ";1H")
    // SGR: color setting
    write_ascii(buf, sgr)
    // Build full-width line with text positioned by horizontal setting
    let line = if i < self.lines.length() {
      self.lines[i]
    } else {
      ""
    }
    let text_width = self.width
    let full_line = build_full_width_line(
      line, text_width, self.term_cols, self.horizontal,
    )
    write_ascii(buf, full_line)
    // SGR reset
    write_ascii(buf, "\u001b[0m")
  }
  // 3. DECRC: restore cursor position
  write_ascii(buf, "\u001b8")
  Bytes::from_array(buf[:])
}

/// Returns the DECSTBM sequence to set scroll region.
/// Only effective in Decstbm mode; returns empty Bytes in Simple mode.
/// Top: scroll region starts after overlay rows.
/// Bottom: scroll region ends before overlay rows.
pub fn OverlayState::scroll_region_set(self : OverlayState) -> Bytes {
  match self.position_mode {
    Simple => Bytes::from_array([])
    Decstbm => {
      let buf : Array[Byte] = []
      match self.vertical {
        Top(_) => {
          // ESC[{height+1};{term_rows}r
          write_ascii(buf, "\u001b[")
          write_ascii(buf, (self.height + 1).to_string())
          write_ascii(buf, ";")
          write_ascii(buf, self.term_rows.to_string())
          write_ascii(buf, "r")
        }
        Bottom(_) => {
          // ESC[1;{term_rows-height}r
          write_ascii(buf, "\u001b[1;")
          write_ascii(buf, (self.term_rows - self.height).to_string())
          write_ascii(buf, "r")
        }
      }
      Bytes::from_array(buf[:])
    }
    Cell => abort("unreachable: cell mode should be rejected at parse time")
  }
}

/// Returns the DECSTBM reset sequence (full screen scroll).
/// Only effective in Decstbm mode; returns empty Bytes in Simple mode.
pub fn OverlayState::scroll_region_reset(self : OverlayState) -> Bytes {
  match self.position_mode {
    Simple => Bytes::from_array([])
    Decstbm => {
      let buf : Array[Byte] = []
      write_ascii(buf, "\u001b[r")
      Bytes::from_array(buf[:])
    }
    Cell => abort("unreachable: cell mode should be rejected at parse time")
  }
}

/// Returns the number of rows available for the child process.
/// In Simple mode, the full terminal height is available.
/// In Decstbm mode, overlay height is subtracted.
pub fn OverlayState::child_rows(self : OverlayState) -> Int {
  match self.position_mode {
    Simple => self.term_rows
    Decstbm => self.term_rows - self.height
    Cell => abort("unreachable: cell mode should be rejected at parse time")
  }
}

// --- Internal helpers ---

/// Split text by "\n" into lines.
fn split_lines(text : String) -> Array[String] {
  text.split("\n").map(fn(sv) { sv.to_string() }).collect()
}

/// Compute the maximum length among lines.
fn max_line_length(lines : Array[String]) -> Int {
  let mut max = 0
  for i = 0; i < lines.length(); i = i + 1 {
    let len = lines[i].length()
    if len > max {
      max = len
    }
  }
  max
}

/// Write an ASCII string as bytes into the buffer.
/// Design rationale: ANSI escape sequences are purely ASCII, so each char
/// maps directly to one byte. This avoids UTF-16 to UTF-8 conversion overhead.
fn write_ascii(buf : Array[Byte], s : String) -> Unit {
  for c in s {
    buf.push(c.to_int().to_byte())
  }
}

/// Build SGR (Select Graphic Rendition) escape sequence.
fn build_sgr(fg : String?, bg : String?) -> String {
  match (fg, bg) {
    (None, None) => "\u001b[7m" // auto: reverse video
    (Some(fg_name), Some(bg_name)) => {
      let fg_code = color_name_to_fg(fg_name)
      let bg_code = color_name_to_bg(bg_name)
      "\u001b[" + fg_code.to_string() + ";" + bg_code.to_string() + "m"
    }
    (Some(fg_name), None) => {
      let fg_code = color_name_to_fg(fg_name)
      "\u001b[" + fg_code.to_string() + "m"
    }
    (None, Some(bg_name)) => {
      let bg_code = color_name_to_bg(bg_name)
      "\u001b[" + bg_code.to_string() + "m"
    }
  }
}

/// Map color name to SGR foreground code.
fn color_name_to_fg(name : String) -> Int {
  match name {
    "black" => 30
    "red" => 31
    "green" => 32
    "yellow" => 33
    "blue" => 34
    "magenta" => 35
    "cyan" => 36
    "white" => 37
    _ => 7 // fallback: reverse video
  }
}

/// Map color name to SGR background code.
fn color_name_to_bg(name : String) -> Int {
  match name {
    "black" => 40
    "red" => 41
    "green" => 42
    "yellow" => 43
    "blue" => 44
    "magenta" => 45
    "cyan" => 46
    "white" => 47
    _ => 49 // fallback: default background
  }
}

/// Build a full-width line (term_cols chars) with text positioned by horizontal setting.
/// The text is first padded/truncated to text_width, then placed within term_cols
/// according to horizontal alignment.
fn build_full_width_line(
  line : String,
  text_width : Int,
  term_cols : Int,
  horizontal : HorizontalPos
) -> String {
  let padded_text = pad_or_truncate(line, text_width)
  let left_margin = match horizontal {
    Left(n) => clamp_min(n, 0)
    Right(n) => clamp_min(term_cols - text_width - n, 0)
  }
  let right_margin = clamp_min(term_cols - left_margin - text_width, 0)
  let buf = StringBuilder::new(size_hint=term_cols)
  for i = 0; i < left_margin; i = i + 1 {
    buf.write_char(' ')
  }
  buf.write_string(padded_text)
  for i = 0; i < right_margin; i = i + 1 {
    buf.write_char(' ')
  }
  buf.to_string()
}

/// Clamp a value to a minimum.
fn clamp_min(value : Int, min : Int) -> Int {
  if value < min { min } else { value }
}

/// Pad or truncate a line to exactly the given width.
/// Shorter lines are right-padded with spaces; longer lines are truncated.
fn pad_or_truncate(line : String, width : Int) -> String {
  let len = line.length()
  if len == width {
    line
  } else if len > width {
    // Truncate: build substring manually to avoid failable StringView ops
    let buf = StringBuilder::new(size_hint=width)
    let mut count = 0
    for c in line {
      if count >= width {
        break
      }
      buf.write_char(c)
      count = count + 1
    }
    buf.to_string()
  } else {
    // Pad with spaces
    let buf = StringBuilder::new(size_hint=width)
    buf.write_string(line)
    for i = len; i < width; i = i + 1 {
      buf.write_char(' ')
    }
    buf.to_string()
  }
}
