package bot

import (
	"context"
	"time"

	"wu_bot_go/internal/game"
)

// CollectController handles box collection with priority scoring.
type CollectController struct {
	scene    *game.Scene
	state    *State
	settings *Settings
	nav      *Navigation
	config   *ConfigSwitcher
	user     *game.User
	stats    *game.Stats
	sendCh   chan<- game.OutboundPacket
	log      func(string)
}

func NewCollectController(scene *game.Scene, state *State, settings *Settings, nav *Navigation, config *ConfigSwitcher, user *game.User, stats *game.Stats, sendCh chan<- game.OutboundPacket, log func(string)) *CollectController {
	return &CollectController{
		scene: scene, state: state, settings: settings, nav: nav,
		config: config, user: user, stats: stats, sendCh: sendCh, log: log,
	}
}

// Run starts the collect controller loop (20ms interval).
func (c *CollectController) Run(ctx context.Context) {
	c.state.SetCollectEnabled(true)
	c.log("Collect controller started")

	for c.state.GetCollectEnabled() {
		c.update(ctx)
		select {
		case <-time.After(20 * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}
}

// Stop stops the collect controller.
func (c *CollectController) Stop() {
	c.state.SetCollectEnabled(false)
	c.resetState()
}

func (c *CollectController) update(ctx context.Context) {
	c.state.mu.RLock()
	targetBox := c.state.Collect.TargetBox
	collecting := c.state.Collect.Collecting
	c.state.mu.RUnlock()

	if targetBox == nil {
		if c.state.Detectors.Break.BreakAdviced || c.state.Detectors.Health.HealthAdviced {
			return
		}
		found := c.findBoxWhileMoving(ctx)
		if found != nil {
			c.state.mu.Lock()
			c.state.Collect.TargetBox = found
			c.state.mu.Unlock()
		}
	}

	c.state.mu.RLock()
	targetBox = c.state.Collect.TargetBox
	c.state.mu.RUnlock()

	if !collecting && targetBox != nil {
		c.config.SwitchFlyMode(ctx)
		c.CollectWait(ctx, *targetBox)
	}
}

// CollectWait moves to a box, collects it, and waits.
func (c *CollectController) CollectWait(ctx context.Context, boxID int) {
	c.state.mu.Lock()
	c.state.Collect.Collecting = true
	c.state.mu.Unlock()

	// Find the box data
	box := c.scene.GetCollectableByID(boxID)
	if box == nil {
		c.resetState()
		return
	}

	c.scene.MoveAndWait(ctx, c.sendCh, box.X, box.Y+95)

	if !c.state.GetCollectEnabled() {
		return
	}

	c.sendCh <- game.BuildCollectPacket(box.ID)

	// Remove from scene
	c.scene.RemoveCollectable(box.ID)

	switch box.Type {
	case 0:
		c.stats.IncrementCargoBoxes()
		c.stats.SetMessage("Collected cargo box")
	case 1:
		c.stats.IncrementResourceBoxes()
		c.stats.SetMessage("Collected resource box")
	case 3:
		select {
		case <-time.After(6 * time.Second):
		case <-ctx.Done():
		}
		c.stats.IncrementGreenBoxes()
		c.stats.SetMessage("Collected green box")
	}

	c.resetState()
}

// FindBox returns the ID of the closest collectible box, or nil.
func (c *CollectController) FindBox() *int {
	box := c.scene.FindClosestBox(c.user.GetBootyKeys())
	if box == nil {
		return nil
	}
	return &box.ID
}

func (c *CollectController) findBoxWhileMoving(ctx context.Context) *int {
	for c.state.GetCollectEnabled() {
		found := c.FindBox()
		if found != nil {
			return found
		}

		if !c.scene.GetIsMoving() {
			c.stats.SetMessage("Collecting - Roaming")
			c.nav.MoveToRandomPoint()
		}

		select {
		case <-time.After(20 * time.Millisecond):
		case <-ctx.Done():
			return nil
		}
	}
	return nil
}

func (c *CollectController) resetState() {
	c.state.mu.Lock()
	c.state.Collect.TargetBox = nil
	c.state.Collect.Collecting = false
	c.state.mu.Unlock()
}
