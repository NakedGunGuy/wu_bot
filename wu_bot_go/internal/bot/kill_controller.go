package bot

import (
	"context"
	"fmt"
	"math"
	"time"

	"wu_bot_go/internal/game"
)

// KillController handles NPC hunting with priority scoring.
type KillController struct {
	scene    *game.Scene
	state    *State
	settings *Settings
	nav      *Navigation
	config   *ConfigSwitcher
	stats    *game.Stats
	sendCh   chan<- game.OutboundPacket
	log      func(string)

	portalFarmDist float64
}

func NewKillController(scene *game.Scene, state *State, settings *Settings, nav *Navigation, config *ConfigSwitcher, stats *game.Stats, sendCh chan<- game.OutboundPacket, log func(string)) *KillController {
	return &KillController{
		scene: scene, state: state, settings: settings, nav: nav,
		config: config, stats: stats, sendCh: sendCh, log: log,
		portalFarmDist: 3000,
	}
}

// Run starts the kill controller loop (100ms interval).
func (k *KillController) Run(ctx context.Context) {
	k.state.SetKillEnabled(true)
	k.log("Kill controller started")

	for k.state.GetKillEnabled() {
		if k.state.IsRecoverOrEscape() {
			select {
			case <-time.After(100 * time.Millisecond):
			case <-ctx.Done():
				return
			}
			continue
		}
		k.update(ctx)
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}
}

// Stop stops the kill controller.
func (k *KillController) Stop() {
	k.state.SetKillEnabled(false)
	k.ResetState()
}

func (k *KillController) update(ctx context.Context) {
	if k.state.IsRecoverOrEscape() {
		return
	}

	k.state.mu.RLock()
	targetedID := k.state.Kill.TargetedID
	k.state.mu.RUnlock()

	if targetedID == nil {
		if k.state.Detectors.Break.BreakAdviced || k.state.Detectors.Health.HealthAdviced {
			return
		}
		found := k.findEnemyWhileMoving(ctx)
		if found != nil {
			k.state.mu.Lock()
			k.state.Kill.TargetedID = found
			k.state.mu.Unlock()
		}
	} else {
		k.kill(ctx, *targetedID)
	}
}

func (k *KillController) kill(ctx context.Context, npcID int) {
	if k.state.GetKillInProgress() {
		return
	}

	k.stats.SetMessage("Killing NPC...")
	k.state.mu.Lock()
	k.state.Kill.KillInProgress = true
	k.state.mu.Unlock()

	dist := k.nav.GetDistanceToID(npcID)
	if dist < 0 {
		k.ResetState()
		return
	}

	if dist > 500 && !k.state.GetKillAttacking() {
		k.config.SwitchFlyMode(ctx)
		go k.nav.StartFollowing(ctx, npcID)
	}

	k.initiateAttack(ctx, npcID)

	ship := k.scene.GetShip(npcID)
	if ship != nil && ship.FarmNearPortal {
		k.handlePortalFarming(ctx, npcID)
	}

	k.awaitKill(ctx, npcID)
	k.stats.SetMessage("NPC killed")
}

func (k *KillController) awaitKill(ctx context.Context, npcID int) {
	for k.state.GetKillAttacking() && k.state.GetKillEnabled() {
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return
		}

		if k.state.IsRecoverOrEscape() {
			k.ResetState()
			return
		}

		ps := k.scene.GetPlayerShip()
		if ps != nil && ps.Selected != npcID {
			k.stats.IncrementKills()
			k.ResetState()
			return
		}

		dist := k.nav.GetDistanceToID(npcID)
		if dist < 0 || dist > 1200 {
			k.nav.StopOrbiting()
			k.ResetState()
			return
		}
	}
}

func (k *KillController) initiateAttack(ctx context.Context, npcID int) {
	for !k.state.GetKillAttacking() && k.state.GetKillEnabled() {
		if !k.scene.ShipExists(npcID) {
			k.ResetState()
			return
		}

		k.selectTarget(npcID)

		dist := k.nav.GetDistanceToID(npcID)
		if dist < 0 {
			k.ResetState()
			return
		}

		k.state.mu.RLock()
		selected := k.state.Kill.Selected
		k.state.mu.RUnlock()

		if dist < 600 && selected {
			k.config.SwitchAttackMode(ctx)
			k.sendCh <- game.BuildAttackAction()

			k.state.mu.Lock()
			k.state.Kill.Attacking = true
			k.state.mu.Unlock()

			k.nav.StopFollowing()
			go k.nav.StartOrbiting(ctx, npcID, 400)

			ship := k.scene.GetShip(npcID)
			if ship == nil {
				k.ResetState()
				return
			}

			k.state.mu.RLock()
			currentAmmo := k.state.Kill.CurrentAmmo
			currentRocket := k.state.Kill.CurrentRocket
			k.state.mu.RUnlock()

			if currentAmmo == nil || *currentAmmo != ship.ConfiguredAmmo {
				time.Sleep(100 * time.Millisecond)
				k.switchAmmo(ship.ConfiguredAmmo)
			}

			if currentRocket == nil || *currentRocket != ship.ConfiguredRockets {
				time.Sleep(100 * time.Millisecond)
				k.switchRocket(ship.ConfiguredRockets)
			}
		}

		select {
		case <-time.After(20 * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}
}

func (k *KillController) selectTarget(npcID int) {
	if !k.scene.ShipExists(npcID) {
		return
	}

	k.state.mu.RLock()
	selected := k.state.Kill.Selected
	k.state.mu.RUnlock()

	if !selected && k.nav.GetDistanceToID(npcID) < 700 {
		k.sendCh <- game.BuildSelectAction(npcID)
		k.state.mu.Lock()
		k.state.Kill.Selected = true
		k.state.mu.Unlock()
	}
}

func (k *KillController) findEnemyWhileMoving(ctx context.Context) *int {
	for k.state.GetKillEnabled() {
		enemy := k.FindEnemy()
		if enemy != nil {
			return enemy
		}

		if !k.scene.GetIsMoving() {
			k.stats.SetMessage("Killing - Roaming")
			k.config.SwitchFlyMode(ctx)
			k.nav.MoveToRandomPoint()
			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				return nil
			}
		}

		select {
		case <-time.After(20 * time.Millisecond):
		case <-ctx.Done():
			return nil
		}
	}
	return nil
}

// FindEnemy returns the ID of the closest targetable enemy, or nil.
func (k *KillController) FindEnemy() *int {
	enemy := k.scene.FindClosestEnemy(k.settings.Kill.TargetEngagedNPC)
	if enemy == nil {
		return nil
	}
	px, py := k.scene.GetPosition()
	dist := game.Distance(px, py, enemy.X, enemy.Y)
	if dist > 2000 {
		return nil
	}
	return &enemy.ID
}

func (k *KillController) switchAmmo(ammoType int) {
	if ammoType < 1 || ammoType > 6 {
		return
	}
	k.log(fmt.Sprintf("Switching to ammo x%d", ammoType))
	k.sendCh <- game.BuildSwitchAmmoAction(ammoType)
	k.state.mu.Lock()
	k.state.Kill.CurrentAmmo = &ammoType
	k.state.mu.Unlock()
}

func (k *KillController) switchRocket(rocketType int) {
	if rocketType < 1 || rocketType > 8 {
		return
	}
	k.log(fmt.Sprintf("Switching to rocket %d", rocketType))
	k.sendCh <- game.BuildRocketSwitchPacket(rocketType)
	k.state.mu.Lock()
	k.state.Kill.CurrentRocket = &rocketType
	k.state.mu.Unlock()
}

// ResetState clears kill state and stops navigation.
func (k *KillController) ResetState() {
	k.sendCh <- game.BuildDeselectAction()
	k.sendCh <- game.BuildSelectAction(0)

	k.state.mu.Lock()
	k.state.Kill.KillInProgress = false
	k.state.Kill.Attacking = false
	k.state.Kill.TargetedID = nil
	k.state.Kill.Selected = false
	k.state.mu.Unlock()

	k.nav.StopFollowing()
	k.nav.StopOrbiting()
}

func (k *KillController) handlePortalFarming(ctx context.Context, npcID int) {
	ship := k.scene.GetShip(npcID)
	if ship == nil || !k.state.GetKillAttacking() {
		return
	}

	mapName, _, _ := k.scene.GetMapInfo()
	px, py := k.scene.GetPosition()
	portal := game.FindClosestSafePortal(mapName, px, py)
	if portal == nil {
		return
	}

	portalDist := game.Distance(px, py, portal.X, portal.Y)

	if portalDist <= k.portalFarmDist {
		return
	}

	angleToPortal := math.Atan2(float64(portal.Y-py), float64(portal.X-px))

	for k.state.GetKillAttacking() && k.scene.ShipExists(npcID) {
		px, py = k.scene.GetPosition()
		currentDist := game.Distance(px, py, portal.X, portal.Y)
		if currentDist <= k.portalFarmDist {
			break
		}

		ps := k.scene.GetPlayerShip()
		if ps != nil && !ps.InAttackRange {
			time.Sleep(1 * time.Second)
			continue
		}

		moveX := px + int(math.Cos(angleToPortal)*600)
		moveY := py + int(math.Sin(angleToPortal)*600)

		if game.Distance(moveX, moveY, portal.X, portal.Y) < 1000 {
			break
		}

		k.scene.MoveAndWait(ctx, k.sendCh, moveX, moveY)
		time.Sleep(100 * time.Millisecond)
	}
}
