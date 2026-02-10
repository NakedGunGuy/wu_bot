package bot

import (
	"context"
	"time"

	"wu_bot_go/internal/game"
)

// EnemyDetector monitors for enemy player ships.
type EnemyDetector struct {
	scene    *game.Scene
	state    *State
	settings *Settings

	companyID string
}

func NewEnemyDetector(scene *game.Scene, state *State, settings *Settings) *EnemyDetector {
	return &EnemyDetector{scene: scene, state: state, settings: settings}
}

// Run starts the enemy detection loop (100ms interval).
func (e *EnemyDetector) Run(ctx context.Context) {
	if !e.settings.Escape.Enabled {
		return
	}
	e.state.mu.Lock()
	e.state.Detectors.Enemy.Enabled = true
	e.state.mu.Unlock()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.update()
		case <-ctx.Done():
			return
		}
	}
}

func (e *EnemyDetector) update() {
	if e.companyID == "" {
		ps := e.scene.GetPlayerShip()
		if ps == nil {
			return
		}
		e.companyID = ps.Corporation
		return
	}

	ships := e.scene.GetShipsSnapshot()
	playerID := e.scene.GetPlayerID()
	detected := false

	for _, ship := range ships {
		if ship.Corporation != "" && ship.Corporation != e.companyID && ship.ID != playerID {
			detected = true
			break
		}
	}

	e.state.mu.Lock()
	e.state.Detectors.Enemy.EnemyDetected = detected
	e.state.mu.Unlock()
}
