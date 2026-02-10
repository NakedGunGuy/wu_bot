package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DiscordWebhookURL string          `yaml:"discord_webhook_url"`
	JarPath           string          `yaml:"jar_path"`
	LogLevel          string          `yaml:"log_level"`
	Accounts          []AccountConfig `yaml:"accounts"`
}

type AccountConfig struct {
	Username  string          `yaml:"username"`
	Password  string          `yaml:"password"`
	Server    string          `yaml:"server"`
	AutoStart bool            `yaml:"auto_start"`
	Mode      string          `yaml:"mode"`
	Settings  AccountSettings `yaml:"settings"`
}

type AccountSettings struct {
	WorkMap         string            `yaml:"work_map"`
	KillTargets     []KillTarget      `yaml:"kill_targets"`
	CollectBoxTypes []CollectBoxType  `yaml:"collect_box_types"`
	Health          HealthSettings    `yaml:"health"`
	Escape          EscapeSettings    `yaml:"escape"`
	Admin           AdminSettings     `yaml:"admin"`
	Config          ConfigSettings    `yaml:"config"`
	Autobuy         AutobuySettings   `yaml:"autobuy"`
	Enrichment      EnrichmentSettings `yaml:"enrichment"`
	Break           BreakSettings     `yaml:"break"`
	Kill            KillSettings      `yaml:"kill"`
}

type KillTarget struct {
	Name           string `yaml:"name"`
	Priority       int    `yaml:"priority"`
	Ammo           int    `yaml:"ammo"`
	Rockets        int    `yaml:"rockets"`
	FarmNearPortal bool   `yaml:"farm_near_portal"`
}

type CollectBoxType struct {
	Type     int `yaml:"type"`
	Priority int `yaml:"priority"`
}

type HealthSettings struct {
	MinHP    int `yaml:"min_hp"`
	AdviceHP int `yaml:"advice_hp"`
}

type EscapeSettings struct {
	Enabled bool `yaml:"enabled"`
	DelayMs int  `yaml:"delay_ms"`
}

type AdminSettings struct {
	Enabled      bool `yaml:"enabled"`
	DelayMinutes int  `yaml:"delay_minutes"`
}

type ConfigSettings struct {
	Attacking            int  `yaml:"attacking"`
	Fleeing              int  `yaml:"fleeing"`
	Flying               int  `yaml:"flying"`
	SwitchOnShieldsDown  bool `yaml:"switch_on_shields_down"`
}

type AutobuySettings struct {
	Laser     map[string]bool        `yaml:"laser"`
	Rockets   map[string]bool        `yaml:"rockets"`
	Key       KeySettings            `yaml:"key"`
	Equipment EquipmentAutobuySettings `yaml:"equipment"`
}

type EquipmentAutobuySettings struct {
	Enabled       bool   `yaml:"enabled"`
	LaserTitle    string `yaml:"laser_title"`
	ShieldGenTitle string `yaml:"shield_gen_title"`
	SpeedGenTitle string `yaml:"speed_gen_title"`
	LaserCount    int    `yaml:"laser_count"`
	GenCount      int    `yaml:"gen_count"`
}

type KeySettings struct {
	Enabled bool `yaml:"enabled"`
	SavePLT int  `yaml:"save_plt"`
}

type EnrichmentSettings struct {
	Lasers  EnrichmentModule `yaml:"lasers"`
	Rockets EnrichmentModule `yaml:"rockets"`
	Shields EnrichmentModule `yaml:"shields"`
	Speed   EnrichmentModule `yaml:"speed"`
}

type EnrichmentModule struct {
	Enabled      bool `yaml:"enabled"`
	MaterialType int  `yaml:"material_type"`
	Amount       int  `yaml:"amount"`
	MinAmount    int  `yaml:"min_amount"`
}

type BreakSettings struct {
	IntervalMinutes int `yaml:"interval_minutes"`
	DurationMinutes int `yaml:"duration_minutes"`
}

type KillSettings struct {
	TargetEngagedNPC bool `yaml:"target_engaged_npc"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Defaults
	if cfg.JarPath == "" {
		cfg.JarPath = "./wupacket.jar"
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}

	for i := range cfg.Accounts {
		a := &cfg.Accounts[i]
		if a.Server == "" {
			a.Server = "eu1"
		}
		if a.Mode == "" {
			a.Mode = "kill"
		}
		if a.Settings.Health.MinHP == 0 {
			a.Settings.Health.MinHP = 30
		}
		if a.Settings.Health.AdviceHP == 0 {
			a.Settings.Health.AdviceHP = 70
		}
		if a.Settings.Config.Attacking == 0 {
			a.Settings.Config.Attacking = 1
		}
		if a.Settings.Config.Fleeing == 0 {
			a.Settings.Config.Fleeing = 2
		}
		if a.Settings.Config.Flying == 0 {
			a.Settings.Config.Flying = 2
		}
		if a.Settings.Escape.DelayMs == 0 {
			a.Settings.Escape.DelayMs = 20000
		}
		if a.Settings.Admin.DelayMinutes == 0 {
			a.Settings.Admin.DelayMinutes = 5
		}
		if a.Settings.Break.IntervalMinutes == 0 {
			a.Settings.Break.IntervalMinutes = 60
		}
		if a.Settings.Break.DurationMinutes == 0 {
			a.Settings.Break.DurationMinutes = 5
		}
		if a.Settings.Autobuy.Key.SavePLT == 0 {
			a.Settings.Autobuy.Key.SavePLT = 50000
		}
		if a.Settings.Autobuy.Equipment.LaserCount == 0 {
			a.Settings.Autobuy.Equipment.LaserCount = 15
		}
		if a.Settings.Autobuy.Equipment.GenCount == 0 {
			a.Settings.Autobuy.Equipment.GenCount = 15
		}
	}

	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
