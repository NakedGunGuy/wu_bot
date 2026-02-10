package bot

import (
	"context"
	"fmt"
	"time"

	"wu_bot_go/internal/game"
)

// ConfigSwitcher handles game config switching (attack/flee/fly modes).
type ConfigSwitcher struct {
	state    *State
	settings *Settings
	stats    *game.Stats
	sendCh   chan<- game.OutboundPacket
	log      func(string)

	lastConfigChange time.Time
	changeInProgress bool
	advisedConfig    *int
}

func NewConfigSwitcher(state *State, settings *Settings, stats *game.Stats, sendCh chan<- game.OutboundPacket, log func(string)) *ConfigSwitcher {
	return &ConfigSwitcher{
		state:    state,
		settings: settings,
		stats:    stats,
		sendCh:   sendCh,
		log:      log,
	}
}

// Run monitors for shield-down config switching.
func (c *ConfigSwitcher) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.update(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (c *ConfigSwitcher) update(ctx context.Context) {
	if !c.settings.Config.SwitchOnShieldsDown {
		return
	}
	if !c.state.GetKillAttacking() || !c.state.Detectors.Health.ShieldsDown {
		return
	}

	mode := c.state.GetConfigMode()
	switch mode {
	case "attacking":
		config := 2
		if c.settings.Config.Attacking == 2 {
			config = 1
		}
		c.log("Shields down, switching config")
		c.state.SetConfigMode("attackingChange")
		c.SwitchConfig(ctx, config)
	case "attackingChange":
		c.log("Shields down on secondary, switching to primary")
		c.state.SetConfigMode("attackingFinal")
		c.SwitchConfig(ctx, c.settings.Config.Attacking)
	}
}

// SwitchAttackMode switches to attack config.
func (c *ConfigSwitcher) SwitchAttackMode(ctx context.Context) {
	mode := c.state.GetConfigMode()
	if mode == "attacking" || mode == "attackingChange" || mode == "attackingFinal" {
		return
	}
	c.state.SetConfigMode("attacking")
	c.SwitchConfig(ctx, c.settings.Config.Attacking)
}

// SwitchFleeMode switches to flee config.
func (c *ConfigSwitcher) SwitchFleeMode(ctx context.Context) {
	if c.state.GetConfigMode() == "fleeing" {
		return
	}
	c.state.SetConfigMode("fleeing")
	c.SwitchConfig(ctx, c.settings.Config.Fleeing)
}

// SwitchFlyMode switches to fly config.
func (c *ConfigSwitcher) SwitchFlyMode(ctx context.Context) {
	if c.state.GetConfigMode() == "flying" {
		return
	}
	c.state.SetConfigMode("flying")
	c.SwitchConfig(ctx, c.settings.Config.Flying)
}

// SwitchConfig switches to a specific config number, retrying until confirmed.
func (c *ConfigSwitcher) SwitchConfig(ctx context.Context, target int) {
	c.advisedConfig = &target
	current := c.state.GetConfigNum()
	if current != nil && *current == target {
		return
	}
	if c.changeInProgress {
		return
	}
	c.changeInProgress = true
	defer func() { c.changeInProgress = false }()

	// Rate limit: 5s between config changes
	elapsed := time.Since(c.lastConfigChange)
	if elapsed < 5*time.Second {
		select {
		case <-time.After(5*time.Second - elapsed + 100*time.Millisecond):
		case <-ctx.Done():
			return
		}
	}

	c.sendConfigPacket()
	waitCount := 0

	for {
		current = c.state.GetConfigNum()
		if c.advisedConfig != nil && current != nil && *current == *c.advisedConfig {
			break
		}

		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return
		}

		waitCount++
		if waitCount > 100 {
			c.sendConfigPacket()
			waitCount = 0
		}
	}

	c.log(fmt.Sprintf("Config switched to %d", target))
	c.stats.SetConfig(target)
}

// SendConfigPacket sends a single config switch packet.
func (c *ConfigSwitcher) SendConfigPacket() {
	c.sendConfigPacket()
}

func (c *ConfigSwitcher) sendConfigPacket() {
	c.lastConfigChange = time.Now()
	c.sendCh <- game.BuildSwitchConfigAction()
}
