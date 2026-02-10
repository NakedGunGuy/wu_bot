package bot

import (
	"context"
	"time"

	"wu_bot_go/internal/game"
)

// HealthDetector monitors player HP and shield levels.
type HealthDetector struct {
	scene    *game.Scene
	state    *State
	settings *Settings
}

func NewHealthDetector(scene *game.Scene, state *State, settings *Settings) *HealthDetector {
	return &HealthDetector{scene: scene, state: state, settings: settings}
}

// Run starts the health detection loop (200ms interval).
func (h *HealthDetector) Run(ctx context.Context) {
	if h.settings.Health.MinHP == 0 {
		return
	}
	h.state.mu.Lock()
	h.state.Detectors.Health.Enabled = true
	h.state.mu.Unlock()

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.update()
		case <-ctx.Done():
			return
		}
	}
}

func (h *HealthDetector) update() {
	ps := h.scene.GetPlayerShip()
	if ps == nil {
		return
	}

	healthPct := 0
	shieldPct := 100
	if ps.MaxHealth > 0 {
		healthPct = ps.Health * 100 / ps.MaxHealth
	}
	if ps.MaxShield > 0 {
		shieldPct = ps.Shield * 100 / ps.MaxShield
	}

	h.state.mu.Lock()
	defer h.state.mu.Unlock()

	h.state.Detectors.Health.ShieldsDown = shieldPct <= 0

	if h.settings.Health.AdviceHP > 0 && healthPct < h.settings.Health.AdviceHP {
		h.state.Detectors.Health.HealthAdviced = true
	} else {
		h.state.Detectors.Health.HealthAdviced = false
	}

	if healthPct < h.settings.Health.MinHP {
		h.state.Detectors.Health.LowHealthDetected = true
	} else if healthPct >= 100 && shieldPct >= 100 {
		h.state.Detectors.Health.LowHealthDetected = false
	}
}
