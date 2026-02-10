package bot

import (
	"context"
	"time"

	"wu_bot_go/internal/game"
)

// KillCollectController combines kill and collect behavior.
type KillCollectController struct {
	scene    *game.Scene
	state    *State
	settings *Settings
	nav      *Navigation
	kill     *KillController
	collect  *CollectController
	config   *ConfigSwitcher
	stats    *game.Stats
	sendCh   chan<- game.OutboundPacket
	log      func(string)
}

func NewKillCollectController(scene *game.Scene, state *State, settings *Settings, nav *Navigation, kill *KillController, collect *CollectController, config *ConfigSwitcher, stats *game.Stats, sendCh chan<- game.OutboundPacket, log func(string)) *KillCollectController {
	return &KillCollectController{
		scene: scene, state: state, settings: settings, nav: nav,
		kill: kill, collect: collect, config: config, stats: stats,
		sendCh: sendCh, log: log,
	}
}

// Run starts the kill+collect controller loop (20ms interval).
func (kc *KillCollectController) Run(ctx context.Context) {
	kc.state.SetKillEnabled(true)
	kc.state.SetCollectEnabled(true)
	kc.state.mu.Lock()
	kc.state.KillAndCollect.Enabled = true
	kc.state.mu.Unlock()
	kc.log("KillCollect controller started")

	for {
		kc.state.mu.RLock()
		enabled := kc.state.KillAndCollect.Enabled
		kc.state.mu.RUnlock()
		if !enabled {
			return
		}

		if kc.state.IsRecoverOrEscape() {
			select {
			case <-time.After(100 * time.Millisecond):
			case <-ctx.Done():
				return
			}
			continue
		}

		kc.update(ctx)

		select {
		case <-time.After(20 * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}
}

// Stop stops the kill+collect controller.
func (kc *KillCollectController) Stop() {
	kc.state.mu.Lock()
	kc.state.KillAndCollect.Enabled = false
	kc.state.mu.Unlock()
	kc.kill.Stop()
	kc.collect.Stop()
}

func (kc *KillCollectController) update(ctx context.Context) {
	if kc.state.GetKillInProgress() {
		// While killing, opportunistically collect nearby boxes
		kc.state.mu.RLock()
		attacking := kc.state.Kill.Attacking
		collecting := kc.state.Collect.Collecting
		kc.state.mu.RUnlock()

		if attacking && !collecting && !kc.scene.GetIsMoving() {
			box := kc.collect.FindBox()
			if box != nil {
				boxData := kc.scene.GetCollectableByID(*box)
				px, py := kc.scene.GetPosition()
				if boxData != nil {
					dist := game.Distance(px, py, boxData.X, boxData.Y)
					if dist < 800 || boxData.Type == 3 {
						kc.collect.CollectWait(ctx, *box)
					}
				}
			}
		}
	} else {
		kc.state.mu.RLock()
		collecting := kc.state.Collect.Collecting
		kc.state.mu.RUnlock()
		if collecting {
			return
		}
		if kc.state.Detectors.Break.BreakAdviced || kc.state.Detectors.Health.HealthAdviced {
			return
		}

		box := kc.collect.FindBox()
		enemy := kc.kill.FindEnemy()

		if box != nil && enemy != nil {
			boxData := kc.scene.GetCollectableByID(*box)
			enemyShip := kc.scene.GetShip(*enemy)
			if boxData != nil && enemyShip != nil {
				if boxData.Priority > enemyShip.Priority {
					kc.collect.CollectWait(ctx, *box)
					return
				}
			}
		}

		if enemy != nil {
			kc.state.mu.Lock()
			kc.state.Kill.TargetedID = enemy
			kc.state.Kill.KillInProgress = false
			kc.state.mu.Unlock()
			kc.kill.kill(ctx, *enemy)
			return
		}

		if box != nil {
			kc.collect.CollectWait(ctx, *box)
			return
		}

		if !kc.scene.GetIsMoving() {
			kc.config.SwitchFlyMode(ctx)
			kc.nav.MoveToRandomPoint()
		}
	}
}
