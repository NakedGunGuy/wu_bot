package game

import (
	"context"
	"encoding/json"
	"math"
	"sync"
	"time"
)

// Ship represents a tracked ship in the game scene.
type Ship struct {
	ID               int
	Name             string
	ClanTag          string
	Corporation      string
	X                int
	Y                int
	TargetX          int
	TargetY          int
	IsMoving         bool
	Selected         int
	IsAttacking      bool
	InAttackRange    bool
	ShipType         int
	Health           int
	MaxHealth        int
	Shield           int
	MaxShield        int
	Cargo            int
	MaxCargo         int
	Speed            int
	DroneArray       []int
	Destroyed        bool
	LastUpdateTime   time.Time
	LastCoordUpdate  time.Time

	// Resolved from kill target config
	Priority         int
	ConfiguredAmmo   int
	ConfiguredRockets int
	FarmNearPortal   bool
}

// Collectable represents a collectible box in the scene.
type Collectable struct {
	ID         int
	Type       int
	X          int
	Y          int
	ExistOnMap bool
	Priority   int
}

// Scene tracks all game world state: ships, collectibles, player position.
type Scene struct {
	mu sync.RWMutex

	PlayerID    int
	X           int
	Y           int
	TargetX     int
	TargetY     int
	IsMoving    bool
	IsDead      bool
	SafeZone    bool

	CurrentMap       string
	CurrentMapWidth  int
	CurrentMapHeight int

	Ships        map[int]*Ship
	Collectibles []*Collectable

	MapLoaded    bool
	mapLoadedCh  chan struct{}
	mapLoadOnce  sync.Once

	// Movement completion signal
	stopMoveCh chan struct{}

	// Kill target config for resolving NPC priorities
	KillTargets []KillTargetConfig

	// Collect box config for resolving priorities
	CollectBoxTypes []CollectBoxConfig

	logFunc func(string)
}

type KillTargetConfig struct {
	Name           string
	Priority       int
	Ammo           int
	Rockets        int
	FarmNearPortal bool
}

type CollectBoxConfig struct {
	Type     int
	Priority int
}

// NewScene creates a new scene tracker.
func NewScene(logFunc func(string)) *Scene {
	return &Scene{
		Ships:       make(map[int]*Ship),
		mapLoadedCh: make(chan struct{}),
		stopMoveCh:  make(chan struct{}, 1),
		logFunc:     logFunc,
	}
}

// WaitForMapLoad blocks until the map is loaded.
func (s *Scene) WaitForMapLoad(ctx context.Context) error {
	select {
	case <-s.mapLoadedCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// HandleGameState processes a GameStateResponsePacket payload.
func (s *Scene) HandleGameState(payload *GameStateResponsePayload) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if payload.PlayerID != 0 {
		s.PlayerID = payload.PlayerID
	}
	s.SafeZone = payload.SafeZone

	for _, cu := range payload.Collectables {
		s.updateCollectable(cu)
	}

	for _, su := range payload.Ships {
		s.analyzeShip(su)
	}

	if payload.Confi != nil {
		// Config number update - handled by engine
	}
}

// HandleMapInfo processes a map-info notification.
func (s *Scene) HandleMapInfo(info *MapInfoNotification) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.CurrentMap = info.Name
	s.CurrentMapWidth = info.Width
	s.CurrentMapHeight = info.Height

	if !s.MapLoaded {
		s.MapLoaded = true
		s.mapLoadOnce.Do(func() {
			close(s.mapLoadedCh)
		})
	}
}

// HandleShipDestroyed marks the player as dead.
func (s *Scene) HandleShipDestroyed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsDead = true
}

// HandleShipRevived marks the player as alive.
func (s *Scene) HandleShipRevived() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsDead = false
}

// RunInterpolation runs the position interpolation loop at 100ms intervals.
func (s *Scene) RunInterpolation(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updatePositions()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scene) updatePositions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for id, ship := range s.Ships {
		heartbeatElapsed := now.Sub(ship.LastUpdateTime).Milliseconds()
		if heartbeatElapsed >= 300 {
			delete(s.Ships, id)
			continue
		}

		if ship.Speed <= 0 {
			continue
		}

		if ship.X != ship.TargetX || ship.Y != ship.TargetY {
			elapsed := now.Sub(ship.LastCoordUpdate).Seconds()
			dist := Distance(ship.X, ship.Y, ship.TargetX, ship.TargetY)
			covered := float64(ship.Speed) * elapsed

			if covered >= dist {
				ship.X = ship.TargetX
				ship.Y = ship.TargetY
				ship.IsMoving = false
			} else {
				ratio := covered / dist
				ship.X += int(float64(ship.TargetX-ship.X) * ratio)
				ship.Y += int(float64(ship.TargetY-ship.Y) * ratio)
				ship.IsMoving = true
			}
			ship.LastCoordUpdate = now
		} else {
			ship.IsMoving = false
		}

		// Sync player position
		if ship.ID == s.PlayerID {
			s.X = ship.X
			s.Y = ship.Y
			s.TargetX = ship.TargetX
			s.TargetY = ship.TargetY
			wasMoving := s.IsMoving
			s.IsMoving = ship.IsMoving
			if wasMoving && !s.IsMoving {
				select {
				case s.stopMoveCh <- struct{}{}:
				default:
				}
			}
		}
	}
}

func (s *Scene) analyzeShip(su ShipUpdate) {
	ship, ok := s.Ships[su.ID]
	if !ok {
		ship = &Ship{
			ID:             su.ID,
			LastUpdateTime: time.Now(),
		}
		s.Ships[su.ID] = ship
	}
	ship.Destroyed = su.Destroyed

	now := time.Now()

	for _, ch := range su.Changes {
		switch ch.ID {
		case ChangeHeartbeatID:
			ship.LastUpdateTime = now

		case ChangeNameID:
			var name string
			json.Unmarshal(ch.Data, &name)
			ship.Name = name
			// Resolve kill target config
			if ship.Priority == 0 {
				for _, kt := range s.KillTargets {
					if kt.Name == name {
						ship.Priority = kt.Priority
						ship.ConfiguredAmmo = kt.Ammo
						ship.ConfiguredRockets = kt.Rockets
						ship.FarmNearPortal = kt.FarmNearPortal
						break
					}
				}
			}

		case ChangeClanTagID:
			var tag string
			json.Unmarshal(ch.Data, &tag)
			ship.ClanTag = tag

		case ChangeCorporationID:
			var corp string
			json.Unmarshal(ch.Data, &corp)
			ship.Corporation = corp

		case ChangePositionID:
			var coords []int
			json.Unmarshal(ch.Data, &coords)
			if len(coords) >= 2 {
				ship.X = coords[0]
				ship.Y = coords[1]
				ship.LastCoordUpdate = now
				if ship.TargetX == ship.X && ship.TargetY == ship.Y {
					ship.IsMoving = false
				}
			}

		case ChangeTargetPosID:
			var coords []int
			json.Unmarshal(ch.Data, &coords)
			if len(coords) >= 2 {
				ship.TargetX = coords[0]
				ship.TargetY = coords[1]
				if ship.TargetX != ship.X || ship.TargetY != ship.Y {
					ship.IsMoving = true
				}
			}

		case ChangeSelectedID:
			var sel int
			json.Unmarshal(ch.Data, &sel)
			ship.Selected = sel

		case ChangeIsAttackingID:
			var attacking bool
			json.Unmarshal(ch.Data, &attacking)
			ship.IsAttacking = attacking

		case ChangeInRangeID:
			var inRange bool
			json.Unmarshal(ch.Data, &inRange)
			ship.InAttackRange = inRange

		case ChangeShipTypeID:
			var st int
			json.Unmarshal(ch.Data, &st)
			ship.ShipType = st

		case ChangeHealthID:
			var hp int
			json.Unmarshal(ch.Data, &hp)
			ship.Health = hp

		case ChangeMaxHealthID:
			var maxHP int
			json.Unmarshal(ch.Data, &maxHP)
			ship.MaxHealth = maxHP

		case ChangeShieldID:
			var sh int
			json.Unmarshal(ch.Data, &sh)
			ship.Shield = sh

		case ChangeMaxShieldID:
			var maxSh int
			json.Unmarshal(ch.Data, &maxSh)
			ship.MaxShield = maxSh

		case ChangeCargoID:
			var cargo int
			json.Unmarshal(ch.Data, &cargo)
			ship.Cargo = cargo

		case ChangeMaxCargoID:
			var maxCargo int
			json.Unmarshal(ch.Data, &maxCargo)
			ship.MaxCargo = maxCargo

		case ChangeSpeedID:
			var speed int
			json.Unmarshal(ch.Data, &speed)
			ship.Speed = speed

		case ChangeDroneArrayID:
			var drones []int
			json.Unmarshal(ch.Data, &drones)
			ship.DroneArray = drones
		}
	}
}

func (s *Scene) updateCollectable(cu CollectableUpdate) {
	priority := 0
	for _, bt := range s.CollectBoxTypes {
		if bt.Type == cu.Type {
			priority = bt.Priority
			break
		}
	}

	// Find existing
	for i, c := range s.Collectibles {
		if c.ID == cu.ID {
			if cu.ExistOnMap {
				s.Collectibles[i] = &Collectable{
					ID: cu.ID, Type: cu.Type, X: cu.X, Y: cu.Y,
					ExistOnMap: true, Priority: priority,
				}
			} else {
				// Remove
				s.Collectibles = append(s.Collectibles[:i], s.Collectibles[i+1:]...)
			}
			return
		}
	}

	if cu.ExistOnMap {
		s.Collectibles = append(s.Collectibles, &Collectable{
			ID: cu.ID, Type: cu.Type, X: cu.X, Y: cu.Y,
			ExistOnMap: true, Priority: priority,
		})
	}
}

// --- Thread-safe accessors ---

// GetPlayerShip returns a copy of the player's ship data.
func (s *Scene) GetPlayerShip() *Ship {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ship, ok := s.Ships[s.PlayerID]
	if !ok {
		return nil
	}
	cp := *ship
	return &cp
}

// PlayerShipExists checks if the player ship exists.
func (s *Scene) PlayerShipExists() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.Ships[s.PlayerID]
	return ok
}

// ShipExists checks if a ship exists.
func (s *Scene) ShipExists(id int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.Ships[id]
	return ok
}

// GetShip returns a copy of a ship by ID.
func (s *Scene) GetShip(id int) *Ship {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ship, ok := s.Ships[id]
	if !ok {
		return nil
	}
	cp := *ship
	return &cp
}

// GetPosition returns the player's current position.
func (s *Scene) GetPosition() (int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.X, s.Y
}

// GetMapInfo returns current map name, width, height.
func (s *Scene) GetMapInfo() (string, int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CurrentMap, s.CurrentMapWidth, s.CurrentMapHeight
}

// GetSafeZone returns whether player is in a safe zone.
func (s *Scene) GetSafeZone() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SafeZone
}

// GetIsDead returns whether the player is dead.
func (s *Scene) GetIsDead() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.IsDead
}

// GetIsMoving returns whether the player is moving.
func (s *Scene) GetIsMoving() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.IsMoving
}

// WaitForStop blocks until the player stops moving.
func (s *Scene) WaitForStop(ctx context.Context) {
	if !s.GetIsMoving() {
		return
	}
	select {
	case <-s.stopMoveCh:
	case <-ctx.Done():
	}
}

// SendMove queues a move action.
func (s *Scene) SendMove(sendCh chan<- OutboundPacket, x, y int) {
	if x == 0 && y == 0 {
		return
	}
	s.mu.Lock()
	if ship, ok := s.Ships[s.PlayerID]; ok {
		ship.TargetX = x
		ship.TargetY = y
		ship.IsMoving = true
	}
	s.IsMoving = true
	s.TargetX = x
	s.TargetY = y
	s.mu.Unlock()

	sendCh <- BuildMoveAction(x, y)
}

// SendMoveCommand sends a non-blocking move (doesn't track arrival).
func (s *Scene) SendMoveCommand(sendCh chan<- OutboundPacket, x, y int) {
	if x == 0 && y == 0 {
		return
	}
	sendCh <- BuildMoveAction(x, y)
}

// MoveAndWait sends a move and waits for arrival.
func (s *Scene) MoveAndWait(ctx context.Context, sendCh chan<- OutboundPacket, x, y int) {
	s.SendMove(sendCh, x, y)
	s.WaitForStop(ctx)
}

// FindClosestEnemy finds the closest NPC with priority scoring.
func (s *Scene) FindClosestEnemy(targetEngaged bool) *Ship {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var bestTarget *Ship
	bestScore := math.Inf(-1)

	for _, ship := range s.Ships {
		if ship.Priority == 0 {
			continue
		}

		// Skip if being attacked by another player
		beingAttacked := false
		for _, other := range s.Ships {
			if other.ID != ship.ID && other.Selected == ship.ID && other.IsAttacking {
				beingAttacked = true
				break
			}
		}
		if beingAttacked {
			continue
		}

		dist := Distance(s.X, s.Y, ship.X, ship.Y)
		distScore := 1.0 - math.Min(dist/2000.0, 1.0)

		score := float64(ship.Priority)*1000.0 + distScore*100.0

		if targetEngaged {
			if ship.Selected == s.PlayerID && ship.IsAttacking {
				score += 10000
			}
		}

		if score > bestScore {
			bestScore = score
			cp := *ship
			bestTarget = &cp
		}
	}

	return bestTarget
}

// FindClosestBox finds the closest collectible box with priority scoring.
func (s *Scene) FindClosestBox(bootyKeys int) *Collectable {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var playerShip *Ship
	if ps, ok := s.Ships[s.PlayerID]; ok {
		playerShip = ps
	}

	var best *Collectable
	bestScore := math.Inf(-1)

	isCargoFull := playerShip != nil && playerShip.Cargo >= playerShip.MaxCargo-5

	for _, c := range s.Collectibles {
		if !c.ExistOnMap || c.Priority == 0 {
			continue
		}
		if c.Type == 3 && bootyKeys <= 0 {
			continue
		}
		if c.Type == 1 && isCargoFull {
			continue
		}

		dist := Distance(s.X, s.Y, c.X, c.Y)
		distScore := 1.0 - math.Min(dist/2000.0, 1.0)
		score := float64(c.Priority)*1000.0 + distScore*100.0

		if score > bestScore {
			bestScore = score
			cp := *c
			best = &cp
		}
	}

	return best
}

// GetCollectableByID returns a copy of a collectable by ID, or nil if not found.
func (s *Scene) GetCollectableByID(id int) *Collectable {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.Collectibles {
		if c.ID == id {
			cp := *c
			return &cp
		}
	}
	return nil
}

// RemoveCollectable removes a collectable by ID.
func (s *Scene) RemoveCollectable(id int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.Collectibles {
		if c.ID == id {
			s.Collectibles = append(s.Collectibles[:i], s.Collectibles[i+1:]...)
			return
		}
	}
}

// GetShipsSnapshot returns a copy of all ships (for TUI/stats).
func (s *Scene) GetShipsSnapshot() map[int]Ship {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[int]Ship, len(s.Ships))
	for id, ship := range s.Ships {
		result[id] = *ship
	}
	return result
}

// GetPlayerID returns the player's ship ID (thread-safe).
func (s *Scene) GetPlayerID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.PlayerID
}
