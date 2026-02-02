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
	LastProcessedAt  time.Time      `json:"lastProcessedAt"`
	ProposalStatuses map[int]Status `json:"proposalStatuses,omitempty"`
	LastCommentID    string         `json:"lastCommentId"`
	IsFresh          bool           `json:"-"`
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
// ファイルが存在しない場合はIsFresh=trueの初期状態を返す
func (sm *StateManager) LoadState() (*State, error) {
	data, err := os.ReadFile(sm.path)
	if os.IsNotExist(err) {
		// 初回実行時: IsFresh=trueで最新コメントのみ処理するようマーク
		return &State{
			LastProcessedAt:  time.Time{}, // zero time
			LastCommentID:    "",
			ProposalStatuses: make(map[int]Status),
			IsFresh:          true,
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	// Initialize map if nil (for backward compatibility with old state files)
	if state.ProposalStatuses == nil {
		state.ProposalStatuses = make(map[int]Status)
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
// 既存の状態がある場合はProposalStatusesを保持し、processedAtとcommentIDのみ更新する
func (sm *StateManager) UpdateState(processedAt time.Time, commentID string) error {
	// Load existing state to preserve ProposalStatuses
	existing, err := sm.LoadState()
	if err != nil {
		existing = &State{
			ProposalStatuses: make(map[int]Status),
		}
	}

	existing.LastProcessedAt = processedAt
	existing.LastCommentID = commentID
	return sm.SaveState(existing)
}

// UpdateProposalStatus は指定されたproposalのステータスを更新する
func (sm *StateManager) UpdateProposalStatus(issueNumber int, status Status) error {
	state, err := sm.LoadState()
	if err != nil {
		return err
	}

	if state.ProposalStatuses == nil {
		state.ProposalStatuses = make(map[int]Status)
	}
	state.ProposalStatuses[issueNumber] = status

	return sm.SaveState(state)
}

// GetProposalStatus は指定されたproposalの最新ステータスを取得する
// 存在しない場合は空文字列とfalseを返す
func (sm *StateManager) GetProposalStatus(issueNumber int) (Status, bool) {
	state, err := sm.LoadState()
	if err != nil {
		return "", false
	}

	status, ok := state.ProposalStatuses[issueNumber]
	return status, ok
}
