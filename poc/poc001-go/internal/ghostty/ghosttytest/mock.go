// Package ghosttytest provides test utilities for the ghostty package.
package ghosttytest

import "github.com/kawaz/shimux/internal/ghostty"

// MockCall はモックの呼び出し記録。
type MockCall struct {
	Method string
	Args   []interface{}
}

// MockController はテスト用のController実装。
type MockController struct {
	Calls    []MockCall
	SplitErr error
	GotoErr  error
	CloseErr error
}

func (m *MockController) NewWindow() error {
	m.Calls = append(m.Calls, MockCall{Method: "NewWindow"})
	return nil
}

func (m *MockController) NewTab() error {
	m.Calls = append(m.Calls, MockCall{Method: "NewTab"})
	return nil
}

func (m *MockController) NewSplit(d ghostty.SplitDirection) error {
	m.Calls = append(m.Calls, MockCall{Method: "NewSplit", Args: []interface{}{d}})
	return m.SplitErr
}

func (m *MockController) GotoSplit(d ghostty.GotoDirection) error {
	m.Calls = append(m.Calls, MockCall{Method: "GotoSplit", Args: []interface{}{d}})
	return m.GotoErr
}

func (m *MockController) CloseSurface() error {
	m.Calls = append(m.Calls, MockCall{Method: "CloseSurface"})
	return m.CloseErr
}
