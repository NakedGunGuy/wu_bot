package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"wu_bot_go/internal/config"
	"wu_bot_go/internal/game"
	wunet "wu_bot_go/internal/net"
)

// BotEngine manages the lifecycle of a single bot instance.
type BotEngine struct {
	Account  *config.AccountConfig
	Settings *Settings
	State    *State
	Scene    *game.Scene
	User     *game.User
	Stats    *game.Stats

	net      *wunet.NetClient
	sendCh   chan<- game.OutboundPacket
	packetCh chan game.InboundPacket
	logCh    chan LogEntry

	status   BotStatus
	statusMu sync.RWMutex

	cancel   context.CancelFunc
	wg       sync.WaitGroup

	// Module references (set during startModules)
	navigation      *Navigation
	configSwitcher  *ConfigSwitcher
	recover         *RecoverBehavior
	escape          *EscapeBehavior
	adminEscape     *AdminEscapeBehavior
	healthDetector  *HealthDetector
	enemyDetector   *EnemyDetector
	adminDetector   *AdminDetector
	autobuyDetector *AutoBuyDetector
	enrichDetector  *EnrichmentDetector
	breakDetector   *BreakDetector
	killController  *KillController
	collectController *CollectController
	killCollectCtrl *KillCollectController
	followController *FollowController
	equipManager   *EquipManager
	questManager   *QuestManager

	modulesEnabled bool
	discordWebhook string
}

// NewBotEngine creates a new bot engine for a given account config.
func NewBotEngine(acc *config.AccountConfig, jarPath, discordWebhook string) *BotEngine {
	packetCh := make(chan game.InboundPacket, 1024)
	logCh := make(chan LogEntry, 512)

	settings := NewSettings(acc)

	logFunc := func(msg string) {
		select {
		case logCh <- LogEntry{Time: time.Now(), Level: "info", Message: msg}:
		default:
		}
	}

	scene := game.NewScene(logFunc)
	// Set kill/collect target configs on scene
	for _, kt := range settings.KillTargets {
		scene.KillTargets = append(scene.KillTargets, game.KillTargetConfig{
			Name:           kt.Name,
			Priority:       kt.Priority,
			Ammo:           kt.Ammo,
			Rockets:        kt.Rockets,
			FarmNearPortal: kt.FarmNearPortal,
		})
	}
	for _, bt := range settings.CollectBoxTypes {
		scene.CollectBoxTypes = append(scene.CollectBoxTypes, game.CollectBoxConfig{
			Type:     bt.Type,
			Priority: bt.Priority,
		})
	}

	nc := wunet.NewNetClient(acc.Username, acc.Password, acc.Server, jarPath, packetCh, logFunc)

	return &BotEngine{
		Account:        acc,
		Settings:       settings,
		State:          NewState(),
		Scene:          scene,
		User:           game.NewUser(),
		Stats:          game.NewStats(),
		net:            nc,
		sendCh:         nc.SendCh(),
		packetCh:       packetCh,
		logCh:          logCh,
		status:         StatusStopped,
		discordWebhook: discordWebhook,
	}
}

// LogCh returns the log channel for TUI consumption.
func (e *BotEngine) LogCh() <-chan LogEntry {
	return e.logCh
}

// GetStatus returns the current bot status.
func (e *BotEngine) GetStatus() BotStatus {
	e.statusMu.RLock()
	defer e.statusMu.RUnlock()
	return e.status
}

func (e *BotEngine) setStatus(s BotStatus) {
	e.statusMu.Lock()
	defer e.statusMu.Unlock()
	e.status = s
}

// GetStatsSnapshot returns a stats snapshot for TUI display.
func (e *BotEngine) GetStatsSnapshot() game.StatsSnapshot {
	return e.Stats.GetSnapshot(e.Scene, e.User)
}

// Start begins the bot lifecycle: connect, authenticate, run modules.
func (e *BotEngine) Start() error {
	if e.GetStatus() == StatusRunning || e.GetStatus() == StatusConnecting {
		return fmt.Errorf("bot already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	e.setStatus(StatusConnecting)
	e.log("Starting bot for %s on %s", e.Account.Username, e.Account.Server)

	// Fetch meta-info and login
	if err := e.net.FetchMetaInfo(); err != nil {
		e.setStatus(StatusError)
		cancel()
		return fmt.Errorf("meta-info: %w", err)
	}

	if err := e.net.Login(); err != nil {
		e.setStatus(StatusError)
		cancel()
		return fmt.Errorf("login: %w", err)
	}

	// Start TCP bridge
	if err := e.net.Start(ctx); err != nil {
		e.setStatus(StatusError)
		cancel()
		return fmt.Errorf("start net: %w", err)
	}

	// Start packet dispatcher and interpolation
	e.wg.Add(2)
	go func() {
		defer e.wg.Done()
		e.packetDispatcher(ctx)
	}()
	go func() {
		defer e.wg.Done()
		e.Scene.RunInterpolation(ctx)
	}()

	// Wait for map load, then start modules
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		if err := e.Scene.WaitForMapLoad(ctx); err != nil {
			return
		}
		mapName, _, _ := e.Scene.GetMapInfo()
		e.log("Map loaded: %s", mapName)

		// Initialize stats with starting resources
		e.waitForUserData(ctx)
		snap := e.User.GetSnapshot()
		e.Stats.InitializeResources(snap.Credits, snap.PLT, snap.Honor, snap.Experience)

		e.setStatus(StatusRunning)
		e.startModules(ctx)
		e.runMainLoop(ctx)
	}()

	return nil
}

// Stop gracefully shuts down the bot.
func (e *BotEngine) Stop() {
	e.log("Stopping bot for %s", e.Account.Username)
	if e.cancel != nil {
		e.cancel()
	}
	e.net.Close()
	e.wg.Wait()
	e.setStatus(StatusStopped)
	e.log("Bot stopped")
}

func (e *BotEngine) waitForUserData(ctx context.Context) {
	for !e.User.GetIsLoaded() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (e *BotEngine) packetDispatcher(ctx context.Context) {
	for {
		select {
		case pkt := <-e.packetCh:
			e.handlePacket(&pkt)
		case <-ctx.Done():
			return
		}
	}
}

func (e *BotEngine) handlePacket(pkt *game.InboundPacket) {
	switch pkt.Type {
	case game.PacketJAREvent:
		if pkt.Event == "connected" {
			e.log("JAR connected to game server")
			e.net.SendAuth()
		}

	case game.PacketAuthAnswer:
		var payload game.AuthAnswerPayload
		json.Unmarshal(pkt.Payload, &payload)
		if payload.Success {
			e.log("Authentication successful")
		} else {
			e.log("Authentication FAILED")
			e.setStatus(StatusError)
		}

	case game.PacketGameStateResponse:
		var payload game.GameStateResponsePayload
		json.Unmarshal(pkt.Payload, &payload)
		e.Scene.HandleGameState(&payload)
		if payload.Confi != nil {
			e.State.SetConfigNum(payload.Confi)
		}

	case game.PacketUserInfoResponse:
		var payload game.UserInfoResponsePayload
		json.Unmarshal(pkt.Payload, &payload)
		e.User.HandleUserInfo(&payload)

	case game.PacketApiNotification:
		var payload game.ApiNotificationPayload
		json.Unmarshal(pkt.Payload, &payload)
		e.handleNotification(&payload)

	case game.PacketApiResponse:
		var payload game.ApiResponsePayload
		json.Unmarshal(pkt.Payload, &payload)
		e.handleApiResponse(&payload)

	case game.PacketGameEvent:
		var payload game.GameEventPayload
		json.Unmarshal(pkt.Payload, &payload)
		e.handleGameEvent(&payload)

	case game.PacketResourcesInfo:
		var payload game.ResourcesInfoPayload
		json.Unmarshal(pkt.Payload, &payload)
		if e.enrichDetector != nil {
			e.enrichDetector.HandleResourcesInfo(&payload)
		}

	case game.PacketEquipResponse:
		var payload game.EquipResponsePayload
		json.Unmarshal(pkt.Payload, &payload)
		if e.autobuyDetector != nil {
			e.autobuyDetector.HandleEquipResponse(&payload)
		}

	case game.PacketEquipMoveResponse:
		var payload game.EquipMoveResponsePayload
		json.Unmarshal(pkt.Payload, &payload)
		if e.equipManager != nil {
			e.equipManager.HandleEquipMoveResponse(&payload)
		}

	case game.PacketQuestsActionResponse:
		var payload game.QuestsActionResponsePayload
		json.Unmarshal(pkt.Payload, &payload)
		if e.questManager != nil {
			e.questManager.HandleQuestsResponse(&payload)
		}

	case game.PacketMissionsActionResponse:
		// Missions are separate from quests (resource miner system)
		var payload game.MissionsActionResponsePayload
		json.Unmarshal(pkt.Payload, &payload)
		_ = payload

	default:
		// Log ALL unhandled packet types for discovery
		payloadStr := string(pkt.Payload)
		if len(payloadStr) > 500 {
			payloadStr = payloadStr[:500] + "..."
		}
		e.log(">>> UNHANDLED PACKET: type=%s payload=%s", pkt.Type, payloadStr)
	}
}

func (e *BotEngine) handleNotification(payload *game.ApiNotificationPayload) {
	switch payload.Key {
	case "map-info":
		var info game.MapInfoNotification
		if err := json.Unmarshal([]byte(payload.NotificationJsonString), &info); err == nil {
			e.Scene.HandleMapInfo(&info)
			// Log map objects for quest station discovery
			if len(info.MapObjects) > 0 {
				e.log("Map objects: %s", string(info.MapObjects))
			}
		}
	case "logged-in-from-another-device":
		e.log("WARN: Logged in from another device, disconnecting")
		e.Stop()
	}

}

func (e *BotEngine) handleApiResponse(payload *game.ApiResponsePayload) {
	if e.autobuyDetector != nil {
		e.autobuyDetector.HandleApiResponse(payload)
	}
}

func (e *BotEngine) handleGameEvent(payload *game.GameEventPayload) {
	switch payload.ID {
	case game.EventShipDestroyed:
		e.log("Ship destroyed")
		e.Scene.HandleShipDestroyed()
	case game.EventShipRevived:
		e.log("Ship revived")
		e.Scene.HandleShipRevived()
	}
}

func (e *BotEngine) startModules(ctx context.Context) {
	e.navigation = NewNavigation(e.Scene, e.State, e.sendCh, e.logFunc())
	e.configSwitcher = NewConfigSwitcher(e.State, e.Settings, e.Stats, e.sendCh, e.logFunc())
	e.recover = NewRecoverBehavior(e.Scene, e.State, e.navigation, e.configSwitcher, e.Stats, e.sendCh, e.logFunc())
	e.escape = NewEscapeBehavior(e.Scene, e.State, e.Settings, e.navigation, e.configSwitcher, e.Stats, e.sendCh, e.logFunc())
	e.adminEscape = NewAdminEscapeBehavior(e.Scene, e.State, e.Settings, e.navigation, e.configSwitcher, e.Stats, e.sendCh, e.logFunc())

	e.healthDetector = NewHealthDetector(e.Scene, e.State, e.Settings)
	e.enemyDetector = NewEnemyDetector(e.Scene, e.State, e.Settings)
	e.adminDetector = NewAdminDetector(e.Scene, e.State, e.Settings, e.discordWebhook, e.Account.Server, e.logFunc())
	e.autobuyDetector = NewAutoBuyDetector(e.State, e.Settings, e.User, e.sendCh, e.logFunc())
	e.equipManager = NewEquipManager(e.Settings, e.sendCh, e.logFunc())
	e.autobuyDetector.SetEquipManager(e.equipManager)
	e.enrichDetector = NewEnrichmentDetector(e.State, e.Settings, e.sendCh, e.logFunc())
	e.breakDetector = NewBreakDetector(e.State, e.Settings, e.Stats, e.recover, e.logFunc())
	e.questManager = NewQuestManager(e.Scene, e.State, e.sendCh, e.logFunc())

	e.killController = NewKillController(e.Scene, e.State, e.Settings, e.navigation, e.configSwitcher, e.Stats, e.sendCh, e.logFunc())
	e.collectController = NewCollectController(e.Scene, e.State, e.Settings, e.navigation, e.configSwitcher, e.User, e.Stats, e.sendCh, e.logFunc())
	e.killCollectCtrl = NewKillCollectController(e.Scene, e.State, e.Settings, e.navigation, e.killController, e.collectController, e.configSwitcher, e.Stats, e.sendCh, e.logFunc())
	e.followController = NewFollowController(e.Scene, e.State, e.navigation, e.killController, e.configSwitcher, e.logFunc())

	// Start detectors
	e.wg.Add(7)
	go func() { defer e.wg.Done(); e.healthDetector.Run(ctx) }()
	go func() { defer e.wg.Done(); e.enemyDetector.Run(ctx) }()
	go func() { defer e.wg.Done(); e.adminDetector.Run(ctx) }()
	go func() { defer e.wg.Done(); e.autobuyDetector.Run(ctx) }()
	go func() { defer e.wg.Done(); e.enrichDetector.Run(ctx) }()
	go func() { defer e.wg.Done(); e.breakDetector.Run(ctx) }()
	go func() { defer e.wg.Done(); e.questManager.Run(ctx) }()

	// Start config loop
	e.wg.Add(1)
	go func() { defer e.wg.Done(); e.configSwitcher.Run(ctx) }()
}

func (e *BotEngine) runMainLoop(ctx context.Context) {
	// Initial delay before starting
	select {
	case <-time.After(3 * time.Second):
	case <-ctx.Done():
		return
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.mainLoopTick(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (e *BotEngine) mainLoopTick(ctx context.Context) {
	e.checkDeath(ctx)

	if !e.Scene.PlayerShipExists() {
		return
	}

	e.checkHealth()
	e.checkHealthAdviced()
	e.checkEnemy(ctx)
	e.checkAdmin(ctx)
	e.checkMap(ctx)
	e.checkState(ctx)
}

func (e *BotEngine) checkDeath(ctx context.Context) {
	if e.Scene.GetIsDead() {
		if !e.State.TransitionEnabled("death") {
			return
		}
		e.log("Player died")
		e.Stats.SetMessage("Player died")
		e.Stats.IncrementDeaths()
		e.recover.Stop()
		e.escape.Stop()
		// Send revive
		e.sendCh <- game.BuildRepairPacket()
		// Wait for revive
		for e.Scene.GetIsDead() {
			select {
			case <-time.After(100 * time.Millisecond):
			case <-ctx.Done():
				return
			}
		}
	} else {
		if !e.State.TransitionDisabled("death") {
			return
		}
		e.log("Player respawned")
		e.State.SetBoolTrigger("enemy", false)
		e.State.SetBoolTrigger("lowhealth", true)
		e.State.SetRecoverEnabled(true)
		e.Stats.SetMessage("Player respawned")
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return
		}
		e.State.SetRecoverEnabled(false)
		e.recover.Start(ctx)
	}
}

func (e *BotEngine) checkHealth() {
	if !e.State.Detectors.Health.Enabled {
		return
	}

	if e.State.Detectors.Health.LowHealthDetected {
		if !e.State.TransitionEnabled("lowhealth") {
			return
		}
		e.log("Low health detected, retreating")
		e.Stats.SetMessage("Low health detected, retreating")
		if e.State.GetKillAttacking() || e.State.GetKillInProgress() {
			e.killController.ResetState()
		}
		e.recover.Start(context.Background())
	} else {
		if e.State.GetRecoverEnabled() {
			return
		}
		if !e.State.TransitionDisabled("lowhealth") {
			return
		}
		e.log("Full health reached")
		e.Stats.SetMessage("Full health reached")
	}
}

func (e *BotEngine) checkHealthAdviced() {
	if e.State.Detectors.Health.HealthAdviced {
		if !e.State.TransitionEnabled("healthAdviced") {
			return
		}
		e.log("Advice health detected, retreating")
		e.Stats.SetMessage("Advice health detected, retreating")
		if e.State.GetKillAttacking() || e.State.GetKillInProgress() {
			e.killController.ResetState()
		}
		e.recover.Start(context.Background())
	} else {
		if e.State.GetRecoverEnabled() {
			return
		}
		if !e.State.TransitionDisabled("healthAdviced") {
			return
		}
		e.log("Advice health restored")
		e.Stats.SetMessage("Advice health restored")
	}
}

func (e *BotEngine) checkEnemy(ctx context.Context) {
	if !e.State.Detectors.Enemy.Enabled {
		return
	}

	if e.State.Detectors.Enemy.EnemyDetected {
		if !e.State.TransitionEnabled("enemy") {
			return
		}
		e.log("Enemy detected, escaping")
		e.Stats.SetMessage("Enemy detected, escaping...")
		e.recover.Stop()
		e.escape.Start(ctx, func() {
			e.State.TransitionDisabled("enemy")
		})
	}
}

func (e *BotEngine) checkAdmin(ctx context.Context) {
	if !e.State.Detectors.Admin.Enabled {
		return
	}
	if !e.Settings.Admin.Enabled {
		return
	}

	if e.State.Detectors.Admin.AdminDetected {
		if !e.State.TransitionEnabled("admin") {
			return
		}
		e.log("Admin detected, escaping")
		e.Stats.SetMessage("Admin detected, escaping...")
		e.recover.Stop()
		e.adminEscape.Start(ctx, func() {
			e.State.TransitionDisabled("admin")
		})
	}
}

func (e *BotEngine) checkMap(ctx context.Context) {
	mapName, _, _ := e.Scene.GetMapInfo()
	if mapName != e.Settings.WorkMap {
		if e.Scene.GetIsDead() || e.State.GetBoolTrigger("enemy") || e.State.GetBoolTrigger("lowhealth") {
			return
		}
		if !e.State.TransitionEnabled("wrongmap") {
			return
		}
		e.log("Wrong map %s, navigating to %s", mapName, e.Settings.WorkMap)
		e.Stats.SetMessage("Wrong map detected, navigating")
		go e.navigation.GoToMap(ctx, e.Settings.WorkMap)
	} else {
		if !e.State.TransitionDisabled("wrongmap") {
			return
		}
		e.log("Correct map reached")
		e.Stats.SetMessage("Correct map reached")
	}
}

func (e *BotEngine) checkState(ctx context.Context) {
	// Check preventing conditions
	preventing := false
	if e.Scene.GetIsDead() {
		preventing = true
	}
	if e.State.GetBoolTrigger("enemy") || e.State.GetBoolTrigger("admin") ||
		e.State.GetBoolTrigger("lowhealth") || e.State.GetBoolTrigger("break") ||
		e.State.GetBoolTrigger("questnav") {
		preventing = true
	}
	mapName, _, _ := e.Scene.GetMapInfo()
	if mapName != e.Settings.WorkMap {
		preventing = true
	}
	if e.State.GetRecoverEnabled() {
		preventing = true
	}

	if preventing {
		e.stopAllModules()
	} else {
		e.restartAllModules(ctx)
	}
}

func (e *BotEngine) stopAllModules() {
	if !e.modulesEnabled {
		return
	}
	e.modulesEnabled = false
	if e.killController != nil {
		e.killController.Stop()
	}
	if e.collectController != nil {
		e.collectController.Stop()
	}
	if e.killCollectCtrl != nil {
		e.killCollectCtrl.Stop()
	}
	if e.followController != nil {
		e.followController.Stop()
	}
}

func (e *BotEngine) restartAllModules(ctx context.Context) {
	if e.modulesEnabled {
		return
	}
	e.modulesEnabled = true
	e.log("Starting modules (mode=%s)", e.Settings.Mode)

	switch e.Settings.Mode {
	case "kill":
		go e.killController.Run(ctx)
	case "collect":
		go e.collectController.Run(ctx)
	case "killcollect":
		go e.killCollectCtrl.Run(ctx)
	case "follow":
		go e.followController.Run(ctx)
	}
}

func (e *BotEngine) logFunc() func(string) {
	return func(msg string) {
		select {
		case e.logCh <- LogEntry{Time: time.Now(), Level: "info", Message: msg}:
		default:
		}
	}
}

func (e *BotEngine) log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	select {
	case e.logCh <- LogEntry{Time: time.Now(), Level: "info", Message: msg}:
	default:
	}
}
