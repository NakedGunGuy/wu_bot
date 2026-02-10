package bot

import (
	"context"
	"fmt"
	"time"

	"wu_bot_go/internal/game"
)

// RecoverBehavior navigates to a safe zone and waits for full health/shield.
type RecoverBehavior struct {
	scene  *game.Scene
	state  *State
	nav    *Navigation
	config *ConfigSwitcher
	stats  *game.Stats
	sendCh chan<- game.OutboundPacket
	log    func(string)

	lastHealthMsg time.Time
}

func NewRecoverBehavior(scene *game.Scene, state *State, nav *Navigation, config *ConfigSwitcher, stats *game.Stats, sendCh chan<- game.OutboundPacket, log func(string)) *RecoverBehavior {
	return &RecoverBehavior{scene: scene, state: state, nav: nav, config: config, stats: stats, sendCh: sendCh, log: log}
}

// Start begins the recover loop.
func (r *RecoverBehavior) Start(ctx context.Context) {
	if r.state.GetRecoverEnabled() {
		return
	}
	r.state.SetRecoverEnabled(true)
	r.state.mu.Lock()
	r.state.Recover.FullHealth = false
	r.state.Recover.ConfigSwitched = false
	r.state.mu.Unlock()
	r.log("Recovery started")

	go r.run(ctx)
}

// Stop stops the recover loop.
func (r *RecoverBehavior) Stop() {
	r.state.SetRecoverEnabled(false)
	r.state.mu.Lock()
	r.state.Recover.ConfigSwitched = false
	r.state.mu.Unlock()
}

func (r *RecoverBehavior) run(ctx context.Context) {
	for r.state.GetRecoverEnabled() {
		r.update(ctx)
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			return
		}
	}
}

func (r *RecoverBehavior) update(ctx context.Context) {
	ps := r.scene.GetPlayerShip()
	if ps == nil {
		return
	}

	mapName, _, _ := r.scene.GetMapInfo()
	if _, ok := game.MapRegions[mapName]; !ok {
		return
	}

	if r.scene.GetSafeZone() {
		r.recoverHealth(ctx, ps)
		return
	}

	portal := game.FindClosestSafePortal(mapName, ps.X, ps.Y)
	if portal == nil {
		return
	}

	px, py := r.scene.GetPosition()
	if game.Distance(px, py, portal.X, portal.Y) < 100 {
		r.recoverHealth(ctx, ps)
	} else {
		r.stats.SetMessage("Recovering - Moving to portal")
		r.config.SwitchFleeMode(ctx)
		r.scene.MoveAndWait(ctx, r.sendCh, portal.X, portal.Y)
	}
}

func (r *RecoverBehavior) recoverHealth(ctx context.Context, ps *game.Ship) {
	healthPct := 0
	shieldPct := 100
	if ps.MaxHealth > 0 {
		healthPct = ps.Health * 100 / ps.MaxHealth
	}
	if ps.MaxShield > 0 {
		shieldPct = ps.Shield * 100 / ps.MaxShield
	}

	if time.Since(r.lastHealthMsg) >= 10*time.Second {
		r.log(fmt.Sprintf("Recovering: HP=%d%% Shield=%d%%", healthPct, shieldPct))
		r.lastHealthMsg = time.Now()
	}

	r.stats.SetMessage(fmt.Sprintf("Recovering - HP: %d%% | Shield: %d%%", healthPct, shieldPct))

	if healthPct < 100 || shieldPct < 100 {
		return
	}

	r.state.mu.RLock()
	switched := r.state.Recover.ConfigSwitched
	r.state.mu.RUnlock()

	if switched {
		r.log("Full health, recovery complete")
		r.stats.SetMessage("Recovery complete")
		r.state.mu.Lock()
		r.state.Recover.FullHealth = true
		r.state.mu.Unlock()
		r.Stop()
		return
	}

	r.state.mu.Lock()
	r.state.Recover.ConfigSwitched = true
	r.state.mu.Unlock()
	r.log("Switching config for recovery")
	r.stats.SetMessage("Recovering - Switching config")
	r.config.SendConfigPacket()
}
