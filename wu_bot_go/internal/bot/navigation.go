package bot

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"wu_bot_go/internal/game"
)

// Navigation handles movement: follow, orbit, roam, BFS map traverse.
type Navigation struct {
	scene  *game.Scene
	state  *State
	sendCh chan<- game.OutboundPacket
	log    func(string)

	orbitAngle float64
}

func NewNavigation(scene *game.Scene, state *State, sendCh chan<- game.OutboundPacket, log func(string)) *Navigation {
	return &Navigation{scene: scene, state: state, sendCh: sendCh, log: log}
}

// StartFollowing follows a ship by ID with prediction and randomization.
func (n *Navigation) StartFollowing(ctx context.Context, shipID int) {
	n.state.mu.Lock()
	if n.state.Navigation.Following {
		n.state.mu.Unlock()
		return
	}
	n.state.Navigation.Following = true
	n.state.mu.Unlock()

	threshold := 500.0

	for n.state.Navigation.Following && n.scene.ShipExists(shipID) {
		ship := n.scene.GetShip(shipID)
		if ship == nil {
			break
		}

		px, py := n.scene.GetPosition()
		dist := game.Distance(px, py, ship.X, ship.Y)

		if dist > threshold {
			targetX, targetY := ship.X, ship.Y

			if ship.IsMoving && ship.TargetX != 0 && ship.TargetY != 0 {
				moveVX := float64(ship.TargetX - ship.X)
				moveVY := float64(ship.TargetY - ship.Y)
				predFactor := math.Min(1.0, (dist-threshold)/2000.0)
				targetX = ship.X + int(moveVX*predFactor)
				targetY = ship.Y + int(moveVY*predFactor)
			}

			randOffset := func() int {
				return int((rand.Float64() - 0.5) * math.Min(150, dist*0.15))
			}
			targetX += randOffset()
			targetY += randOffset()

			_, w, h := n.scene.GetMapInfo()
			targetX = clamp(targetX, 0, w)
			targetY = clamp(targetY, 0, h)

			moveDist := game.Distance(px, py, targetX, targetY)
			if moveDist > 100 {
				n.scene.SendMove(n.sendCh, targetX, targetY)
			}
		}

		select {
		case <-time.After(time.Duration(200+rand.Intn(300)) * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}

	n.state.mu.Lock()
	n.state.Navigation.Following = false
	n.state.mu.Unlock()
}

// StopFollowing stops the following loop.
func (n *Navigation) StopFollowing() {
	n.state.mu.Lock()
	defer n.state.mu.Unlock()
	n.state.Navigation.Following = false
}

// StartOrbiting orbits around a ship at a given radius.
func (n *Navigation) StartOrbiting(ctx context.Context, shipID int, radius float64) {
	n.state.mu.Lock()
	if n.state.Navigation.Orbiting {
		n.state.mu.Unlock()
		return
	}
	n.state.Navigation.Orbiting = true
	n.state.mu.Unlock()

	for n.state.Navigation.Orbiting && n.scene.ShipExists(shipID) {
		ship := n.scene.GetShip(shipID)
		if ship == nil {
			break
		}

		n.orbitAngle += 0.4 + rand.Float64()*0.3
		if n.orbitAngle > math.Pi*2 {
			n.orbitAngle -= math.Pi * 2
		}

		currentRadius := radius - 50 + rand.Float64()*100

		targetX := int(float64(ship.X) + math.Cos(n.orbitAngle)*currentRadius)
		targetY := int(float64(ship.Y) + math.Sin(n.orbitAngle)*currentRadius)

		_, w, h := n.scene.GetMapInfo()
		targetX = clamp(targetX, 0, w)
		targetY = clamp(targetY, 0, h)

		n.scene.SendMoveCommand(n.sendCh, targetX, targetY)

		select {
		case <-time.After(time.Duration(400+rand.Intn(400)) * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}

	n.state.mu.Lock()
	n.state.Navigation.Orbiting = false
	n.state.mu.Unlock()
}

// StopOrbiting stops the orbit loop.
func (n *Navigation) StopOrbiting() {
	n.state.mu.Lock()
	defer n.state.mu.Unlock()
	n.state.Navigation.Orbiting = false
}

// GetDistanceToID returns the distance from the player to a ship.
func (n *Navigation) GetDistanceToID(shipID int) float64 {
	ship := n.scene.GetShip(shipID)
	if ship == nil {
		return -1
	}
	px, py := n.scene.GetPosition()
	return game.Distance(px, py, ship.X, ship.Y)
}

// MoveToRandomPoint sends a random move command.
func (n *Navigation) MoveToRandomPoint() {
	_, w, h := n.scene.GetMapInfo()
	if w == 0 && h == 0 {
		return
	}
	x := rand.Intn(w)
	y := rand.Intn(h)
	n.scene.SendMove(n.sendCh, x, y)
}

// GoToMap navigates to a destination map using BFS pathfinding.
func (n *Navigation) GoToMap(ctx context.Context, dest string) {
	n.state.mu.Lock()
	if n.state.Navigation.InNavigation {
		n.state.mu.Unlock()
		return
	}
	n.state.Navigation.InNavigation = true
	n.state.mu.Unlock()

	defer func() {
		n.state.mu.Lock()
		n.state.Navigation.InNavigation = false
		n.state.mu.Unlock()
	}()

	for {
		currentMap, _, _ := n.scene.GetMapInfo()
		if currentMap == dest {
			n.log(fmt.Sprintf("Arrived at %s", dest))
			return
		}

		path := game.FindPath(currentMap, dest)
		if path == nil {
			n.log(fmt.Sprintf("No path from %s to %s", currentMap, dest))
			return
		}

		portal := path[0].Portal

		// Check if recover/escape is active
		if n.state.GetRecoverEnabled() || n.state.GetEscapeEnabled() {
			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				return
			}
			continue
		}

		n.scene.MoveAndWait(ctx, n.sendCh, portal.X, portal.Y)

		px, py := n.scene.GetPosition()
		if game.Distance(px, py, portal.X, portal.Y) < 100 {
			n.log("At portal, jumping...")
			n.sendCh <- game.BuildJumpPortalAction()
			select {
			case <-time.After(6 * time.Second):
			case <-ctx.Done():
				return
			}
		}

		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return
		}
	}
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
