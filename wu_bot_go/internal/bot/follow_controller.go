package bot

import (
	"context"
	"math/rand"
	"time"

	"wu_bot_go/internal/game"
)

// FollowController follows another player and assists in combat.
type FollowController struct {
	scene    *game.Scene
	state    *State
	nav      *Navigation
	kill     *KillController
	config   *ConfigSwitcher
	log      func(string)

	masterID int
	enabled  bool
}

func NewFollowController(scene *game.Scene, state *State, nav *Navigation, kill *KillController, config *ConfigSwitcher, log func(string)) *FollowController {
	return &FollowController{
		scene:  scene,
		state:  state,
		nav:    nav,
		kill:   kill,
		config: config,
		log:    log,
	}
}

// SetMasterID sets the player ID to follow.
func (f *FollowController) SetMasterID(id int) {
	f.masterID = id
}

// Run starts the follow controller loop (400ms interval).
func (f *FollowController) Run(ctx context.Context) {
	if f.masterID == 0 {
		f.log("Follow: no master ID set")
		return
	}
	f.enabled = true
	f.state.SetKillEnabled(true)
	f.log("Follow controller started")

	threshold := 500.0
	catchup := 1000.0

	for f.enabled {
		select {
		case <-time.After(400 * time.Millisecond):
		case <-ctx.Done():
			return
		}

		master := f.scene.GetShip(f.masterID)
		if master == nil {
			continue
		}

		// If master is attacking, help kill
		if master.Selected != 0 && master.IsAttacking {
			f.log("Master attacking, assisting")
			f.kill.kill(ctx, master.Selected)
			continue
		}

		// Regular following
		f.config.SwitchFlyMode(ctx)

		if master.X == 0 && master.Y == 0 {
			continue
		}

		px, py := f.scene.GetPosition()
		dist := game.Distance(px, py, master.X, master.Y)

		if dist == 0 || dist > catchup {
			f.scene.SendMove(f.nav.sendCh, master.X, master.Y)
		} else if dist > threshold {
			devX := rand.Intn(500) - 250
			devY := rand.Intn(500) - 250
			f.scene.SendMove(f.nav.sendCh, master.X+devX, master.Y+devY)
		}
	}
}

// Stop stops the follow controller.
func (f *FollowController) Stop() {
	f.enabled = false
}
