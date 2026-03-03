// Re-export position types from args package to avoid duplication.
///|
type VerticalPos = @args.VerticalPos

///|
type HorizontalPos = @args.HorizontalPos

///|
type PositionMode = @args.PositionMode

///|
pub struct OverlayState {
  mut text : String
  mut lines : Array[String]
  mut width : Int
  mut height : Int
  mut term_rows : Int
  mut term_cols : Int
  config_width : Int? // None = auto
  config_height : Int? // None = auto
  vertical : VerticalPos
  horizontal : HorizontalPos
  fg_color : String? // None = auto (reverse video)
  bg_color : String? // None = auto
  position_mode : PositionMode // default Simple
}
