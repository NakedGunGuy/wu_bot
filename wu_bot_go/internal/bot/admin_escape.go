package bot

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"wu_bot_go/internal/game"
)

// AdminEscapeBehavior handles fleeing from admin players.
type AdminEscapeBehavior struct {
	scene    *game.Scene
	state    *State
	settings *Settings
	nav      *Navigation
	config   *ConfigSwitcher
	stats    *game.Stats
	sendCh   chan<- game.OutboundPacket
	log      func(string)

	callback func()
}

func NewAdminEscapeBehavior(scene *game.Scene, state *State, settings *Settings, nav *Navigation, config *ConfigSwitcher, stats *game.Stats, sendCh chan<- game.OutboundPacket, log func(string)) *AdminEscapeBehavior {
	return &AdminEscapeBehavior{scene: scene, state: state, settings: settings, nav: nav, config: config, stats: stats, sendCh: sendCh, log: log}
}

// Start begins the admin escape loop.
func (a *AdminEscapeBehavior) Start(ctx context.Context, callback func()) {
	if a.state.GetAdminEnabled() {
		return
	}
	a.state.SetAdminEnabled(true)
	a.state.mu.Lock()
	a.state.Admin.WaitedTime = 0
	a.state.mu.Unlock()
	a.callback = callback

	go a.run(ctx)
}

// Stop stops the admin escape loop.
func (a *AdminEscapeBehavior) Stop() {
	a.state.SetAdminEnabled(false)
	a.callback = nil
	a.state.mu.Lock()
	a.state.Admin.WaitedTime = 0
	a.state.mu.Unlock()
}

func (a *AdminEscapeBehavior) run(ctx context.Context) {
	for a.state.GetAdminEnabled() {
		a.update(ctx)
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}
}

func (a *AdminEscapeBehavior) update(ctx context.Context) {
	ps := a.scene.GetPlayerShip()
	if ps == nil {
		return
	}

	mapName, _, _ := a.scene.GetMapInfo()
	portal := game.FindClosestSafePortal(mapName, ps.X, ps.Y)
	if portal == nil {
		return
	}

	isBeingAttacked := a.checkBeingAttacked()
	delayMs := a.settings.Admin.DelayMinutes * 60 * 1000

	if a.scene.GetSafeZone() {
		for !a.state.Detectors.Admin.AdminDetected {
			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				return
			}
			a.state.mu.Lock()
			a.state.Admin.WaitedTime += 1000
			waited := a.state.Admin.WaitedTime
			a.state.mu.Unlock()

			remaining := int(math.Ceil(float64(delayMs-waited) / 60000.0))
			a.stats.SetMessage(fmt.Sprintf("AdminEscape - Waiting %d min", remaining))

			if waited >= delayMs {
				if a.callback != nil {
					a.callback()
				}
				a.Stop()
				return
			}
		}
		a.state.mu.Lock()
		a.state.Admin.WaitedTime = 0
		a.state.mu.Unlock()
		a.stats.SetMessage("AdminEscape - Safe zone, admin still present")

	} else if game.Distance(ps.X, ps.Y, portal.X, portal.Y) < 100 {
		a.stats.SetMessage("AdminEscape - At portal, not safe")
		if isBeingAttacked {
			a.sendCh <- game.BuildJumpPortalAction()
			select {
			case <-time.After(6 * time.Second):
			case <-ctx.Done():
				return
			}
		}
	} else {
		a.config.SwitchFleeMode(ctx)
		a.scene.MoveAndWait(ctx, a.sendCh, portal.X, portal.Y)
	}
}

func (a *AdminEscapeBehavior) checkBeingAttacked() bool {
	ships := a.scene.GetShipsSnapshot()
	playerID := a.scene.GetPlayerID()
	for _, ship := range ships {
		if ship.Selected == playerID && ship.IsAttacking && !strings.Contains(ship.Name, "-=(") {
			return true
		}
	}
	return false
}
