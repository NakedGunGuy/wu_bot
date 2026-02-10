package bot

import (
	"context"
	"fmt"
	"time"

	"wu_bot_go/internal/config"
	"wu_bot_go/internal/game"
)

// EnrichmentDetector handles periodic equipment enrichment.
type EnrichmentDetector struct {
	state    *State
	settings *Settings
	sendCh   chan<- game.OutboundPacket
	log      func(string)

	resources [9]int // cerium(0), mercury(1), erbium(2), piritid(3), darkonit(4), uranit(5), azurit(6), dungid(7), xureon(8)
}

func NewEnrichmentDetector(state *State, settings *Settings, sendCh chan<- game.OutboundPacket, log func(string)) *EnrichmentDetector {
	return &EnrichmentDetector{state: state, settings: settings, sendCh: sendCh, log: log}
}

// HandleResourcesInfo updates resource amounts from ResourcesInfoResponsePacket.
func (e *EnrichmentDetector) HandleResourcesInfo(payload *game.ResourcesInfoPayload) {
	for i, r := range payload.Resources {
		if i < len(e.resources) {
			e.resources[i] = r.Amount
		}
	}
}

// Run starts the enrichment check loop (10min interval).
func (e *EnrichmentDetector) Run(ctx context.Context) {
	for {
		// Request current resources
		e.sendCh <- game.BuildResourcesRequestPacket()

		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			return
		}

		e.attemptUpgrade()

		select {
		case <-time.After(10*time.Minute - 2*time.Second):
		case <-ctx.Done():
			return
		}
	}
}

func (e *EnrichmentDetector) attemptUpgrade() {
	// Shields and speed
	e.upgradeModule(game.EnrichShields, e.settings.Enrichment.Shields)
	e.upgradeModule(game.EnrichSpeed, e.settings.Enrichment.Speed)

	// Special handling for lasers/rockets sharing material
	lasers := e.settings.Enrichment.Lasers
	rockets := e.settings.Enrichment.Rockets

	if lasers.Enabled && rockets.Enabled && lasers.MaterialType == rockets.MaterialType {
		available := e.getResourceAmount(lasers.MaterialType)
		half := available / 2
		if half > 0 {
			e.upgrade(game.EnrichLasers, lasers.MaterialType, half)
			e.upgrade(game.EnrichRockets, rockets.MaterialType, half)
		}
	} else {
		if lasers.Enabled {
			amount := e.getResourceAmount(lasers.MaterialType)
			if amount > 0 {
				e.upgrade(game.EnrichLasers, lasers.MaterialType, amount)
			}
		}
		if rockets.Enabled {
			amount := e.getResourceAmount(rockets.MaterialType)
			if amount > 0 {
				e.upgrade(game.EnrichRockets, rockets.MaterialType, amount)
			}
		}
	}
}

func (e *EnrichmentDetector) upgradeModule(moduleIdx int, cfg config.EnrichmentModule) {
	if !cfg.Enabled || cfg.MaterialType == 0 {
		return
	}
	available := e.getResourceAmount(cfg.MaterialType)
	if available >= cfg.MinAmount {
		e.upgrade(moduleIdx, cfg.MaterialType, cfg.Amount)
	} else if available > 0 {
		e.upgrade(moduleIdx, cfg.MaterialType, available)
	}
}

func (e *EnrichmentDetector) upgrade(moduleIdx, materialType, amount int) {
	if amount == 0 || materialType == 0 {
		return
	}
	e.log(fmt.Sprintf("Enriching module %d with material %d x%d", moduleIdx, materialType, amount))
	e.sendCh <- game.BuildResourcesActionPacket(2, []int{moduleIdx, materialType, amount})
}

func (e *EnrichmentDetector) getResourceAmount(materialType int) int {
	if materialType >= 0 && materialType < len(e.resources) {
		return e.resources[materialType]
	}
	return 0
}
