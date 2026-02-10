package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"wu_bot_go/internal/game"
)

// EscapeBehavior handles fleeing from enemy players.
type EscapeBehavior struct {
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

func NewEscapeBehavior(scene *game.Scene, state *State, settings *Settings, nav *Navigation, config *ConfigSwitcher, stats *game.Stats, sendCh chan<- game.OutboundPacket, log func(string)) *EscapeBehavior {
	return &EscapeBehavior{scene: scene, state: state, settings: settings, nav: nav, config: config, stats: stats, sendCh: sendCh, log: log}
}

// Start begins the escape loop.
func (e *EscapeBehavior) Start(ctx context.Context, callback func()) {
	if e.state.GetEscapeEnabled() {
		return
	}
	e.state.SetEscapeEnabled(true)
	e.state.mu.Lock()
	e.state.Escape.WaitedTime = 0
	e.state.mu.Unlock()
	e.callback = callback

	go e.run(ctx)
}

// Stop stops the escape loop.
func (e *EscapeBehavior) Stop() {
	e.state.SetEscapeEnabled(false)
	e.callback = nil
	e.state.mu.Lock()
	e.state.Escape.WaitedTime = 0
	e.state.mu.Unlock()
}

func (e *EscapeBehavior) run(ctx context.Context) {
	for e.state.GetEscapeEnabled() {
		e.update(ctx)
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}
}

func (e *EscapeBehavior) update(ctx context.Context) {
	ps := e.scene.GetPlayerShip()
	if ps == nil {
		return
	}

	mapName, _, _ := e.scene.GetMapInfo()
	portal := game.FindClosestSafePortal(mapName, ps.X, ps.Y)
	if portal == nil {
		return
	}

	isBeingAttacked := e.checkBeingAttacked()

	if e.scene.GetSafeZone() {
		// In safe zone - wait for enemy to leave
		for !e.state.Detectors.Enemy.EnemyDetected {
			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				return
			}
			e.state.mu.Lock()
			e.state.Escape.WaitedTime += 1000
			waited := e.state.Escape.WaitedTime
			e.state.mu.Unlock()

			e.stats.SetMessage(fmt.Sprintf("Escaping - Enemy left, waiting %ds", waited/1000))

			if waited >= e.settings.Escape.DelayMs {
				if e.callback != nil {
					e.callback()
				}
				e.Stop()
				return
			}
		}
		e.state.mu.Lock()
		e.state.Escape.WaitedTime = 0
		e.state.mu.Unlock()
		e.stats.SetMessage("Escaping - In safe zone, enemy still present")

	} else if game.Distance(ps.X, ps.Y, portal.X, portal.Y) < 100 {
		e.stats.SetMessage("Escaping - At portal, not safe")
		if isBeingAttacked {
			e.stats.SetMessage("Escaping - Under attack, jumping portal")
			e.sendCh <- game.BuildJumpPortalAction()
			select {
			case <-time.After(6 * time.Second):
			case <-ctx.Done():
				return
			}
			e.sendCh <- game.BuildJumpPortalAction()
		}
	} else {
		e.config.SwitchFleeMode(ctx)
		e.scene.MoveAndWait(ctx, e.sendCh, portal.X, portal.Y)
	}
}

func (e *EscapeBehavior) checkBeingAttacked() bool {
	ships := e.scene.GetShipsSnapshot()
	playerID := e.scene.GetPlayerID()
	for _, ship := range ships {
		if ship.Selected == playerID && ship.IsAttacking && !strings.Contains(ship.Name, "-=(") {
			return true
		}
	}
	return false
}
