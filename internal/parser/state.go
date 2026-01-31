package parser

import (
	"encoding/json"
	"io/fs"
	"os"
	"time"
)

// stateFileMode is the file permission for state.json.
const stateFileMode fs.FileMode = 0644

// State は前回処理状態を表す
type State struct {
	LastProcessedAt time.Time `json:"lastProcessedAt"`
	LastCommentID   string    `json:"lastCommentId"`
}

// StateManager は状態の読み書きを管理する
type StateManager struct {
	path string
}

// NewStateManager は新しいStateManagerを生成する
func NewStateManager(path string) *StateManager {
	return &StateManager{path: path}
}

// LoadState はstate.jsonから状態を読み込む
// ファイルが存在しない場合は1ヶ月前をデフォルトとする初期状態を返す
func (sm *StateManager) LoadState() (*State, error) {
	data, err := os.ReadFile(sm.path)
	if os.IsNotExist(err) {
		// 初回実行時のデフォルト状態
		return &State{
			LastProcessedAt: time.Now().AddDate(0, -1, 0),
			LastCommentID:   "",
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// SaveState は状態をstate.jsonに保存する
func (sm *StateManager) SaveState(state *State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sm.path, data, stateFileMode)
}

// UpdateState は状態を更新して保存する
func (sm *StateManager) UpdateState(processedAt time.Time, commentID string) error {
	state := &State{
		LastProcessedAt: processedAt,
		LastCommentID:   commentID,
	}
	return sm.SaveState(state)
}
