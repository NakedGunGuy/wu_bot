package bot

import (
	"sync"
	"time"
)

// State tracks all runtime state for a bot, protected by a mutex.
type State struct {
	mu sync.RWMutex

	Enabled bool

	Config  ConfigState
	Recover RecoverState
	Escape  EscapeState
	Admin   AdminState

	Detectors DetectorStates

	BoolTriggers map[string]bool

	Navigation NavigationState
	Kill       KillState
	Collect    CollectState
	Enrichment EnabledState
	KillAndCollect EnabledState
}

type ConfigState struct {
	Enabled bool
	Config  *int   // Current config number
	Mode    string // "attacking", "fleeing", "flying", "attackingChange", "attackingFinal"
}

type RecoverState struct {
	Enabled        bool
	FullHealth     bool
	ConfigSwitched bool
}

type EscapeState struct {
	Enabled    bool
	WaitedTime int // ms
}

type AdminState struct {
	Enabled    bool
	WaitedTime int // ms
}

type DetectorStates struct {
	Health HealthDetectorState
	Enemy  EnemyDetectorState
	Admin  AdminDetectorState
	Break  BreakDetectorState
}

type HealthDetectorState struct {
	Enabled          bool
	LowHealthDetected bool
	HealthAdviced    bool
	ShieldsDown      bool
}

type EnemyDetectorState struct {
	Enabled       bool
	EnemyDetected bool
}

type AdminDetectorState struct {
	Enabled       bool
	AdminDetected bool
}

type BreakDetectorState struct {
	Enabled       bool
	BreakDetected bool
	BreakAdviced  bool
	LastBreakTime int64 // unix ms
}

type NavigationState struct {
	InNavigation bool
	Following    bool
	Orbiting     bool
}

type KillState struct {
	Enabled        bool
	KillInProgress bool
	TargetedID     *int
	Selected       bool
	Attacking      bool
	CurrentAmmo    *int
	CurrentRocket  *int
}

type CollectState struct {
	Enabled    bool
	Collecting bool
	TargetBox  *int
}

type EnabledState struct {
	Enabled bool
}

// NewState creates an initialized State.
func NewState() *State {
	return &State{
		BoolTriggers: map[string]bool{
			"lowhealth":    false,
			"healthAdviced": false,
			"admin":        false,
			"death":        false,
			"enemy":        false,
			"wrongmap":     false,
			"break":        false,
		},
		Config: ConfigState{Mode: "attacking"},
		Detectors: DetectorStates{
			Break: BreakDetectorState{
				LastBreakTime: currentTimeMs(),
			},
		},
	}
}

// --- BoolManager pattern: edge-detection for state transitions ---

// TransitionEnabled returns true ONLY on first call when the trigger was false.
// Subsequent calls return false (already triggered). This prevents duplicate actions.
func (s *State) TransitionEnabled(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.BoolTriggers[key] {
		s.BoolTriggers[key] = true
		return true // First trigger
	}
	return false // Already triggered
}

// TransitionDisabled returns true ONLY on first call when the trigger was true.
// Resets the trigger to false.
func (s *State) TransitionDisabled(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.BoolTriggers[key] {
		s.BoolTriggers[key] = false
		return true // First reset
	}
	return false // Already reset
}

// SetBoolTrigger sets a bool trigger directly.
func (s *State) SetBoolTrigger(key string, val bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.BoolTriggers[key] = val
}

// GetBoolTrigger reads a bool trigger.
func (s *State) GetBoolTrigger(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.BoolTriggers[key]
}

// --- Thread-safe getters/setters for nested state ---

func (s *State) SetRecoverEnabled(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Recover.Enabled = v
}

func (s *State) GetRecoverEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Recover.Enabled
}

func (s *State) SetEscapeEnabled(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Escape.Enabled = v
}

func (s *State) GetEscapeEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Escape.Enabled
}

func (s *State) SetAdminEnabled(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Admin.Enabled = v
}

func (s *State) GetAdminEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Admin.Enabled
}

func (s *State) SetKillEnabled(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Kill.Enabled = v
}

func (s *State) GetKillEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Kill.Enabled
}

func (s *State) SetCollectEnabled(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Collect.Enabled = v
}

func (s *State) GetCollectEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Collect.Enabled
}

func (s *State) GetKillAttacking() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Kill.Attacking
}

func (s *State) GetKillInProgress() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Kill.KillInProgress
}

func (s *State) IsRecoverOrEscape() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Recover.Enabled || s.Escape.Enabled || s.Admin.Enabled
}

func (s *State) GetConfigMode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Config.Mode
}

func (s *State) SetConfigMode(mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config.Mode = mode
}

func (s *State) GetConfigNum() *int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Config.Config
}

func (s *State) SetConfigNum(v *int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Config.Config = v
}

func currentTimeMs() int64 {
	return time.Now().UnixMilli()
}
