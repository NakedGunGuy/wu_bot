package manager

import (
	"fmt"
	"sync"

	"wu_bot_go/internal/bot"
	"wu_bot_go/internal/config"
)

// BotManager manages multiple bot instances.
type BotManager struct {
	mu      sync.RWMutex
	bots    map[string]*bot.BotEngine // keyed by username
	cfg     *config.Config
	cfgPath string
}

// NewBotManager creates a new bot manager.
func NewBotManager(cfg *config.Config, cfgPath string) *BotManager {
	return &BotManager{
		bots:    make(map[string]*bot.BotEngine),
		cfg:     cfg,
		cfgPath: cfgPath,
	}
}

// GetConfig returns the current config.
func (m *BotManager) GetConfig() *config.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

// StartBot starts a bot for the given username.
func (m *BotManager) StartBot(username string) error {
	m.mu.Lock()

	if _, exists := m.bots[username]; exists {
		m.mu.Unlock()
		return fmt.Errorf("bot %s already running", username)
	}

	acc := m.findAccount(username)
	if acc == nil {
		m.mu.Unlock()
		return fmt.Errorf("account %s not found", username)
	}

	engine := bot.NewBotEngine(acc, m.cfg.JarPath, m.cfg.DiscordWebhookURL)
	// Register engine before starting so TUI can see "Connecting" status
	m.bots[username] = engine
	m.mu.Unlock()

	// Start outside the lock since it blocks waiting for JAR connection
	if err := engine.Start(); err != nil {
		m.mu.Lock()
		delete(m.bots, username)
		m.mu.Unlock()
		return fmt.Errorf("start bot %s: %w", username, err)
	}

	return nil
}

// StopBot stops a running bot.
func (m *BotManager) StopBot(username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	engine, exists := m.bots[username]
	if !exists {
		return fmt.Errorf("bot %s not running", username)
	}

	engine.Stop()
	delete(m.bots, username)
	return nil
}

// GetBot returns a bot engine by username.
func (m *BotManager) GetBot(username string) *bot.BotEngine {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.bots[username]
}

// ListBots returns info about all configured accounts and their status.
func (m *BotManager) ListBots() []BotInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var infos []BotInfo
	for _, acc := range m.cfg.Accounts {
		info := BotInfo{
			Username: acc.Username,
			Server:   acc.Server,
			Mode:     acc.Mode,
		}

		if engine, exists := m.bots[acc.Username]; exists {
			info.Status = engine.GetStatus()
			snap := engine.GetStatsSnapshot()
			info.Map = snap.Map
			info.HealthPercent = snap.HealthPercent
			info.Kills = snap.Kills
			info.CreditsPerHr = snap.CreditsPerHr
			info.RunTime = snap.RunTime
			info.Message = snap.MessageState
		} else {
			info.Status = bot.StatusStopped
		}

		infos = append(infos, info)
	}
	return infos
}

// StartAutoStart starts all bots with auto_start: true.
func (m *BotManager) StartAutoStart() {
	for _, acc := range m.cfg.Accounts {
		if acc.AutoStart {
			if err := m.StartBot(acc.Username); err != nil {
				fmt.Printf("Failed to auto-start %s: %v\n", acc.Username, err)
			}
		}
	}
}

// StartAutoStartWithLogDrainer starts all auto_start bots and sets up a log drainer
// for each one immediately after engine creation (before Start() completes).
func (m *BotManager) StartAutoStartWithLogDrainer(drainer func(string, *bot.BotEngine)) {
	for i := range m.cfg.Accounts {
		acc := &m.cfg.Accounts[i]
		if acc.AutoStart {
			engine := bot.NewBotEngine(acc, m.cfg.JarPath, m.cfg.DiscordWebhookURL)
			m.mu.Lock()
			m.bots[acc.Username] = engine
			m.mu.Unlock()

			// Start draining logs before engine.Start() so we don't lose messages
			drainer(acc.Username, engine)

			if err := engine.Start(); err != nil {
				fmt.Printf("Failed to auto-start %s: %v\n", acc.Username, err)
				m.mu.Lock()
				delete(m.bots, acc.Username)
				m.mu.Unlock()
			}
		}
	}
}

// StopAll stops all running bots.
func (m *BotManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for username, engine := range m.bots {
		engine.Stop()
		delete(m.bots, username)
	}
}

// AddAccount adds a new account to the config.
func (m *BotManager) AddAccount(acc config.AccountConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.cfg.Accounts {
		if existing.Username == acc.Username {
			return fmt.Errorf("account %s already exists", acc.Username)
		}
	}

	m.cfg.Accounts = append(m.cfg.Accounts, acc)
	return config.Save(m.cfgPath, m.cfg)
}

// RemoveAccount removes an account from the config and stops it if running.
func (m *BotManager) RemoveAccount(username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop if running
	if engine, exists := m.bots[username]; exists {
		engine.Stop()
		delete(m.bots, username)
	}

	// Remove from config
	for i, acc := range m.cfg.Accounts {
		if acc.Username == username {
			m.cfg.Accounts = append(m.cfg.Accounts[:i], m.cfg.Accounts[i+1:]...)
			return config.Save(m.cfgPath, m.cfg)
		}
	}

	return fmt.Errorf("account %s not found", username)
}

func (m *BotManager) findAccount(username string) *config.AccountConfig {
	for i := range m.cfg.Accounts {
		if m.cfg.Accounts[i].Username == username {
			return &m.cfg.Accounts[i]
		}
	}
	return nil
}

// BotInfo holds display info about a bot.
type BotInfo struct {
	Username      string
	Server        string
	Mode          string
	Status        bot.BotStatus
	Map           string
	HealthPercent int
	Kills         int
	CreditsPerHr  int
	RunTime       string
	Message       string
}
