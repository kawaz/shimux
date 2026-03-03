package ghostty

// SplitDirection はペイン分割の方向を表す。
type SplitDirection int

const (
	SplitDown SplitDirection = iota
	SplitRight
	SplitLeft
	SplitUp
)

// GotoDirection はペインフォーカス移動の方向を表す。
type GotoDirection int

const (
	GotoNext GotoDirection = iota
	GotoPrevious
)

// Controller はGhosttyターミナル操作の抽象インターフェース。
type Controller interface {
	NewWindow() error
	NewTab() error
	NewSplit(direction SplitDirection) error
	GotoSplit(direction GotoDirection) error
	CloseSurface() error
}
