package bot

import "wu_bot_go/internal/config"

// Settings holds read-only bot settings derived from config.
type Settings struct {
	WorkMap string
	Mode    string

	KillTargets     []config.KillTarget
	CollectBoxTypes []config.CollectBoxType

	Health HealthSettings
	Escape EscapeSettings
	Admin  AdminSettings
	Config ConfigSwitchSettings
	Kill   KillModuleSettings

	Autobuy    config.AutobuySettings
	Enrichment config.EnrichmentSettings
	Break      BreakModuleSettings
}

type HealthSettings struct {
	MinHP    int
	AdviceHP int
}

type EscapeSettings struct {
	Enabled bool
	DelayMs int
}

type AdminSettings struct {
	Enabled      bool
	DelayMinutes int
}

type ConfigSwitchSettings struct {
	Attacking           int
	Fleeing             int
	Flying              int
	SwitchOnShieldsDown bool
}

type KillModuleSettings struct {
	TargetEngagedNPC bool
}

type BreakModuleSettings struct {
	IntervalMinutes int
	DurationMinutes int
}

// NewSettings creates Settings from an AccountConfig.
func NewSettings(acc *config.AccountConfig) *Settings {
	return &Settings{
		WorkMap: acc.Settings.WorkMap,
		Mode:    acc.Mode,

		KillTargets:     acc.Settings.KillTargets,
		CollectBoxTypes: acc.Settings.CollectBoxTypes,

		Health: HealthSettings{
			MinHP:    acc.Settings.Health.MinHP,
			AdviceHP: acc.Settings.Health.AdviceHP,
		},
		Escape: EscapeSettings{
			Enabled: acc.Settings.Escape.Enabled,
			DelayMs: acc.Settings.Escape.DelayMs,
		},
		Admin: AdminSettings{
			Enabled:      acc.Settings.Admin.Enabled,
			DelayMinutes: acc.Settings.Admin.DelayMinutes,
		},
		Config: ConfigSwitchSettings{
			Attacking:           acc.Settings.Config.Attacking,
			Fleeing:             acc.Settings.Config.Fleeing,
			Flying:              acc.Settings.Config.Flying,
			SwitchOnShieldsDown: acc.Settings.Config.SwitchOnShieldsDown,
		},
		Kill: KillModuleSettings{
			TargetEngagedNPC: acc.Settings.Kill.TargetEngagedNPC,
		},
		Autobuy:    acc.Settings.Autobuy,
		Enrichment: acc.Settings.Enrichment,
		Break: BreakModuleSettings{
			IntervalMinutes: acc.Settings.Break.IntervalMinutes,
			DurationMinutes: acc.Settings.Break.DurationMinutes,
		},
	}
}
