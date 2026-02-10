package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"wu_bot_go/internal/game"
)

var adminNames = map[string]bool{
	"Curunir": true,
	"Vicc":    true,
	"Dayn":    true,
}

// AdminDetector monitors for admin ships and sends Discord webhook alerts.
type AdminDetector struct {
	scene          *game.Scene
	state          *State
	settings       *Settings
	webhookURL     string
	serverID       string
	log            func(string)

	reportedAdmins map[string]*reportedMapData
	reportMu       sync.Mutex
	resetInterval  time.Duration
}

type reportedMapData struct {
	ships     map[int]bool
	timestamp time.Time
}

func NewAdminDetector(scene *game.Scene, state *State, settings *Settings, webhookURL, serverID string, log func(string)) *AdminDetector {
	return &AdminDetector{
		scene:          scene,
		state:          state,
		settings:       settings,
		webhookURL:     webhookURL,
		serverID:       serverID,
		log:            log,
		reportedAdmins: make(map[string]*reportedMapData),
		resetInterval:  20 * time.Minute,
	}
}

// Run starts the admin detection loop (100ms interval).
func (a *AdminDetector) Run(ctx context.Context) {
	a.state.mu.Lock()
	a.state.Detectors.Admin.Enabled = true
	a.state.mu.Unlock()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.update()
		case <-ctx.Done():
			return
		}
	}
}

func (a *AdminDetector) update() {
	ships := a.scene.GetShipsSnapshot()
	mapName, _, _ := a.scene.GetMapInfo()

	a.reportMu.Lock()
	defer a.reportMu.Unlock()

	// Reset old map data
	if data, ok := a.reportedAdmins[mapName]; ok {
		if time.Since(data.timestamp) >= a.resetInterval {
			delete(a.reportedAdmins, mapName)
		}
	}

	var adminShips []game.Ship
	for _, ship := range ships {
		isAdmin := false
		if len(ship.DroneArray) > 8 {
			isAdmin = true
		}
		if adminNames[ship.Name] {
			isAdmin = true
		}
		if isAdmin {
			adminShips = append(adminShips, ship)
		}
	}

	if len(adminShips) > 0 {
		newAdmins := a.processNewAdmins(adminShips, mapName)
		if len(newAdmins) > 0 {
			go a.sendWebhook(newAdmins, mapName)
		}
		a.state.mu.Lock()
		a.state.Detectors.Admin.AdminDetected = true
		a.state.mu.Unlock()
	} else {
		a.state.mu.Lock()
		a.state.Detectors.Admin.AdminDetected = false
		a.state.mu.Unlock()
	}
}

func (a *AdminDetector) processNewAdmins(admins []game.Ship, mapName string) []game.Ship {
	if _, ok := a.reportedAdmins[mapName]; !ok {
		a.reportedAdmins[mapName] = &reportedMapData{
			ships:     make(map[int]bool),
			timestamp: time.Now(),
		}
	}

	data := a.reportedAdmins[mapName]
	var newAdmins []game.Ship
	for _, ship := range admins {
		if !data.ships[ship.ID] {
			data.ships[ship.ID] = true
			newAdmins = append(newAdmins, ship)
		}
	}
	return newAdmins
}

func (a *AdminDetector) sendWebhook(admins []game.Ship, mapName string) {
	if a.webhookURL == "" {
		a.log(fmt.Sprintf("Admin Alert: %d admin(s) on %s", len(admins), mapName))
		return
	}

	now := time.Now().Unix()
	desc := ""
	for _, ship := range admins {
		desc += fmt.Sprintf("Ship Position (X, Y): %d, %d\n", ship.X/100, ship.Y/100)
	}

	body := map[string]interface{}{
		"content": fmt.Sprintf("Admin Alert: %d admin(s) on %s server %s\nDetected <t:%d:R> at <t:%d:f>",
			len(admins), mapName, a.serverID, now, now),
		"embeds": []map[string]interface{}{
			{
				"description": desc,
				"color":       0xff0000,
			},
		},
	}

	jsonBody, _ := json.Marshal(body)
	resp, err := http.Post(a.webhookURL, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		a.log(fmt.Sprintf("Webhook error: %v", err))
		return
	}
	resp.Body.Close()
}
