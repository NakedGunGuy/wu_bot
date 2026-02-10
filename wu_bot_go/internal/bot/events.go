package bot

import "time"

// LogEntry represents a log message from a bot.
type LogEntry struct {
	Time    time.Time
	Level   string // "info", "warn", "error"
	Message string
}

// BotStatus represents the current status of a bot.
type BotStatus int

const (
	StatusStopped BotStatus = iota
	StatusConnecting
	StatusRunning
	StatusError
	StatusBreak
)

func (s BotStatus) String() string {
	switch s {
	case StatusStopped:
		return "Stopped"
	case StatusConnecting:
		return "Connecting"
	case StatusRunning:
		return "Running"
	case StatusError:
		return "Error"
	case StatusBreak:
		return "Break"
	default:
		return "Unknown"
	}
}
