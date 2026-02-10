package bot

import (
	"fmt"
	"time"

	"wu_bot_go/internal/game"
)

// EquipManager handles auto-mounting equipment to the correct ship configs.
// Shield gens -> config 1, speed gens -> config 2, lasers -> config 1 (then config 2).
type EquipManager struct {
	settings *Settings
	sendCh   chan<- game.OutboundPacket
	log      func(string)

	pendingMount bool
}

func NewEquipManager(settings *Settings, sendCh chan<- game.OutboundPacket, log func(string)) *EquipManager {
	return &EquipManager{
		settings: settings,
		sendCh:   sendCh,
		log:      log,
	}
}

// HandleEquipResponse processes equipment data and mounts unequipped items.
func (em *EquipManager) HandleEquipResponse(payload *game.EquipResponsePayload) {
	if em.pendingMount {
		return
	}
	em.pendingMount = true
	defer func() { em.pendingMount = false }()

	// Find items in equip (inventory) that are not on ship - these need mounting
	unequipped := payload.Equip

	if len(unequipped) == 0 {
		return
	}

	// Count what's already on ship per type
	onShipLasers := 0
	onShipShieldGens := 0
	onShipSpeedGens := 0
	for _, item := range payload.OnShip {
		switch item.Type {
		case game.EquipTypeLaser:
			onShipLasers++
		case game.EquipTypeShieldGen:
			onShipShieldGens++
		case game.EquipTypeSpeedGen:
			onShipSpeedGens++
		}
	}

	em.log(fmt.Sprintf("Equip manager: %d unequipped items, on ship: %d/%d lasers, %d/%d gens (shield=%d, speed=%d)",
		len(unequipped), onShipLasers, payload.LaserSlots,
		onShipShieldGens+onShipSpeedGens, payload.GenSlots,
		onShipShieldGens, onShipSpeedGens))

	mounted := 0

	// Mount shield gens to config 1
	for _, item := range unequipped {
		if item.Type != game.EquipTypeShieldGen {
			continue
		}
		if onShipShieldGens+onShipSpeedGens >= payload.GenSlots {
			break
		}
		em.log(fmt.Sprintf("Mounting shield gen (id=%d) to config 1", item.ID))
		em.sendCh <- game.BuildEquipMovePacket(item, 1, false)
		onShipShieldGens++
		mounted++
		time.Sleep(500 * time.Millisecond)
	}

	// Mount speed gens to config 2
	for _, item := range unequipped {
		if item.Type != game.EquipTypeSpeedGen {
			continue
		}
		if onShipShieldGens+onShipSpeedGens >= payload.GenSlots {
			break
		}
		em.log(fmt.Sprintf("Mounting speed gen (id=%d) to config 2", item.ID))
		em.sendCh <- game.BuildEquipMovePacket(item, 2, false)
		onShipSpeedGens++
		mounted++
		time.Sleep(500 * time.Millisecond)
	}

	// Mount lasers to config 1
	for _, item := range unequipped {
		if item.Type != game.EquipTypeLaser {
			continue
		}
		if onShipLasers >= payload.LaserSlots {
			break
		}
		em.log(fmt.Sprintf("Mounting laser (id=%d) to config 1", item.ID))
		em.sendCh <- game.BuildEquipMovePacket(item, 1, false)
		onShipLasers++
		mounted++
		time.Sleep(500 * time.Millisecond)
	}

	if mounted > 0 {
		em.log(fmt.Sprintf("Equip manager: mounted %d items", mounted))
	}
}

// HandleEquipMoveResponse logs the result of an equipment move operation.
func (em *EquipManager) HandleEquipMoveResponse(payload *game.EquipMoveResponsePayload) {
	if payload.Status == 0 {
		em.log("Equipment move: success")
	} else {
		em.log(fmt.Sprintf("Equipment move: status %d", payload.Status))
	}
}
