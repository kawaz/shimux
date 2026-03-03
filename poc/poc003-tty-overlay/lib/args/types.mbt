///|
pub(all) enum ContentSource {
  Static(String)
  Command(String)
} derive(Show, Eq)

///|
pub(all) enum VerticalPos {
  Top(Int)
  Bottom(Int)
} derive(Show, Eq)

///|
pub(all) enum HorizontalPos {
  Left(Int)
  Right(Int)
} derive(Show, Eq)

///|
pub(all) enum PositionMode {
  Simple
  Decstbm
  Cell
} derive(Show, Eq)

///|
pub(all) struct Config {
  content_source : ContentSource
  interval_ms : Int // default 5000
  width : Int? // None = auto
  height : Int? // None = auto
  vertical : VerticalPos // default Top(1)
  horizontal : HorizontalPos // default Right(1)
  fg_color : String? // None = auto
  bg_color : String? // None = auto
  position_mode : PositionMode // default Simple
  command : Array[String] // args after --
} derive(Show, Eq)
