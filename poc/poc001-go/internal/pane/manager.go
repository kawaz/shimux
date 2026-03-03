package pane

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// Pane はペインの状態を表す。
type Pane struct {
	ID          string `json:"id"`
	TTY         string `json:"tty"`
	Active      bool   `json:"active"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	PID         int    `json:"pid,omitempty"`
	CurrentPath string `json:"current_path,omitempty"`
}

// managerState はSave/Loadで永続化される状態。
type managerState struct {
	Panes  []*Pane `json:"panes"`
	NextID int     `json:"next_id"`
}

// Manager はペインの状態を管理する。
type Manager struct {
	mu         sync.RWMutex
	panes      map[string]*Pane
	nextPaneID int
	stateFile  string
	order      []string
}

// DefaultStateDir はデフォルトのペイン状態ディレクトリパスを返す。
// パス: ~/.local/state/shimux/
func DefaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".local", "state", "shimux")
}

// NewWithDir はステートディレクトリを指定してManagerを作成する。
// ステートファイルは <dir>/panes.json に配置される。
func NewWithDir(stateDir string) *Manager {
	return NewManager(filepath.Join(stateDir, "panes.json"))
}

// NewManager は新しいManagerを作成する。
func NewManager(stateFile string) *Manager {
	return &Manager{
		panes:      make(map[string]*Pane),
		nextPaneID: 0,
		stateFile:  stateFile,
		order:      nil,
	}
}

// Register は新しいペインを登録し、自動採番されたIDを割り当てる。
func (m *Manager) Register(tty string) (*Pane, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.nextID()
	p := &Pane{
		ID:     id,
		TTY:    tty,
		Active: len(m.panes) == 0, // 最初のペインをアクティブにする
	}
	m.panes[id] = p
	m.order = append(m.order, id)
	return p, nil
}

// Unregister はペインを登録解除する。
func (m *Manager) Unregister(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.panes[id]; !ok {
		return fmt.Errorf("pane %q not found", id)
	}
	delete(m.panes, id)
	// order からも削除
	for i, oid := range m.order {
		if oid == id {
			m.order = append(m.order[:i], m.order[i+1:]...)
			break
		}
	}
	return nil
}

// Get はIDでペインを取得する。
func (m *Manager) Get(id string) (*Pane, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.panes[id]
	if !ok {
		return nil, fmt.Errorf("pane %q not found", id)
	}
	return p, nil
}

// List は登録されている全ペインをスライスで返す。
func (m *Manager) List() []*Pane {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Pane, 0, len(m.order))
	for _, id := range m.order {
		if p, ok := m.panes[id]; ok {
			result = append(result, p)
		}
	}
	return result
}

// IndexOf はペインIDの登録順インデックスを返す。見つからない場合は -1。
func (m *Manager) IndexOf(id string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for i, oid := range m.order {
		if oid == id {
			return i
		}
	}
	return -1
}

// OrderedList はペインIDを登録順で返す。
func (m *Manager) OrderedList() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]string, len(m.order))
	copy(result, m.order)
	return result
}

// GetActive はアクティブなペインを返す。なければnilを返す。
func (m *Manager) GetActive() *Pane {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.panes {
		if p.Active {
			return p
		}
	}
	return nil
}

// SetActive は指定したIDのペインをアクティブにし、他を非アクティブにする。
func (m *Manager) SetActive(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.panes[id]; !ok {
		return fmt.Errorf("pane %q not found", id)
	}
	for _, p := range m.panes {
		p.Active = p.ID == id
	}
	return nil
}

// NextID は次のペインIDを文字列で返し、カウンタをインクリメントする。
func (m *Manager) NextID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.nextID()
}

// nextID はロックなしでIDを採番する内部メソッド。
// 呼び出し元が既にロックを保持していること。
func (m *Manager) nextID() string {
	id := strconv.Itoa(m.nextPaneID)
	m.nextPaneID++
	return id
}

// Save はペイン状態をファイルにアトミックに書き込む。
func (m *Manager) Save() error {
	m.mu.RLock()
	panes := make([]*Pane, 0, len(m.order))
	for _, id := range m.order {
		if p, ok := m.panes[id]; ok {
			panes = append(panes, p)
		}
	}
	state := managerState{
		Panes:  panes,
		NextID: m.nextPaneID,
	}
	m.mu.RUnlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	dir := filepath.Dir(m.stateFile)
	tmp, err := os.CreateTemp(dir, "panes-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, m.stateFile); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

// Load はファイルからペイン状態を読み込む。
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.stateFile)
	if err != nil {
		return fmt.Errorf("read state file: %w", err)
	}

	var state managerState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("unmarshal state: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.panes = make(map[string]*Pane)
	m.order = nil
	for _, p := range state.Panes {
		m.panes[p.ID] = p
		m.order = append(m.order, p.ID)
	}
	m.nextPaneID = state.NextID
	return nil
}
