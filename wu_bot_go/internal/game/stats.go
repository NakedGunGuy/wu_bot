package game

import (
	"fmt"
	"sync"
	"time"
)

// Stats tracks bot performance metrics.
type Stats struct {
	mu sync.RWMutex

	Kills  int
	Deaths int

	CargoBoxes    int
	ResourceBoxes int
	GreenBoxes    int

	StartCredits int
	StartPLT     int
	StartHonor   int
	StartExp     int

	StartTime    time.Time
	MessageState string
	ConfigNum    int
}

// NewStats creates a new stats tracker.
func NewStats() *Stats {
	return &Stats{
		StartTime: time.Now(),
	}
}

// InitializeResources sets starting resource values for per-hour calculations.
func (s *Stats) InitializeResources(credits, plt, honor, exp int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.StartCredits = credits
	s.StartPLT = plt
	s.StartHonor = honor
	s.StartExp = exp
	s.StartTime = time.Now()
}

func (s *Stats) IncrementKills() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Kills++
}

func (s *Stats) IncrementDeaths() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Deaths++
}

func (s *Stats) IncrementCargoBoxes() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CargoBoxes++
}

func (s *Stats) IncrementResourceBoxes() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ResourceBoxes++
}

func (s *Stats) IncrementGreenBoxes() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.GreenBoxes++
}

func (s *Stats) SetMessage(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MessageState = msg
}

func (s *Stats) SetConfig(config int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ConfigNum = config
}

// StatsSnapshot is a read-only snapshot for display.
type StatsSnapshot struct {
	MessageState string
	Kills        int
	Deaths       int
	KDRatio      string
	CreditsPerHr int
	PLTPerHr     int
	HonorPerHr   int
	ExpPerHr     int
	CargoBoxes   int
	ResourceBoxes int
	GreenBoxes   int
	CargoPercent int
	HealthPercent int
	ShieldPercent int
	ConfigNum    int
	RunTime      string
	Credits      int
	PLT          int
	Map          string
	PosX         int
	PosY         int
	Level        int
	Experience   int
}

// GetSnapshot returns a complete stats snapshot for display.
func (s *Stats) GetSnapshot(scene *Scene, user *User) StatsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hoursSinceStart := time.Since(s.StartTime).Hours()

	snap := StatsSnapshot{
		MessageState: s.MessageState,
		Kills:        s.Kills,
		Deaths:       s.Deaths,
		CargoBoxes:   s.CargoBoxes,
		ResourceBoxes: s.ResourceBoxes,
		GreenBoxes:   s.GreenBoxes,
		ConfigNum:    s.ConfigNum,
		RunTime:      formatRunTime(s.StartTime),
	}

	// KD ratio
	if s.Deaths == 0 {
		snap.KDRatio = fmt.Sprintf("%d", s.Kills)
	} else {
		snap.KDRatio = fmt.Sprintf("%.2f", float64(s.Kills)/float64(s.Deaths))
	}

	// Per-hour rates
	userSnap := user.GetSnapshot()
	if hoursSinceStart > 0 {
		snap.CreditsPerHr = int(float64(userSnap.Credits-s.StartCredits) / hoursSinceStart)
		snap.PLTPerHr = int(float64(userSnap.PLT-s.StartPLT) / hoursSinceStart)
		snap.HonorPerHr = int(float64(userSnap.Honor-s.StartHonor) / hoursSinceStart)
		snap.ExpPerHr = int(float64(userSnap.Experience-s.StartExp) / hoursSinceStart)
	}

	snap.Credits = userSnap.Credits
	snap.PLT = userSnap.PLT
	snap.Level = userSnap.Level
	snap.Experience = userSnap.Experience

	// Ship info
	ps := scene.GetPlayerShip()
	if ps != nil {
		if ps.MaxHealth > 0 {
			snap.HealthPercent = ps.Health * 100 / ps.MaxHealth
		}
		if ps.MaxShield > 0 {
			snap.ShieldPercent = ps.Shield * 100 / ps.MaxShield
		}
		if ps.MaxCargo > 0 {
			snap.CargoPercent = ps.Cargo * 100 / ps.MaxCargo
		}
	}

	mapName, _, _ := scene.GetMapInfo()
	snap.Map = mapName
	x, y := scene.GetPosition()
	snap.PosX = x / 100
	snap.PosY = y / 100

	return snap
}

func formatRunTime(start time.Time) string {
	d := time.Since(start)
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}
