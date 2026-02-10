package bot

import (
	"context"
	"fmt"
	"math"
	"time"

	"wu_bot_go/internal/game"
)

// BreakDetector handles scheduled anti-ban breaks.
type BreakDetector struct {
	state    *State
	settings *Settings
	stats    *game.Stats
	recover  *RecoverBehavior
	log      func(string)
}

func NewBreakDetector(state *State, settings *Settings, stats *game.Stats, recover *RecoverBehavior, log func(string)) *BreakDetector {
	return &BreakDetector{state: state, settings: settings, stats: stats, recover: recover, log: log}
}

// Run starts the break check loop (1s interval).
func (b *BreakDetector) Run(ctx context.Context) {
	if b.settings.Break.IntervalMinutes == 0 || b.settings.Break.DurationMinutes == 0 {
		return
	}

	b.state.mu.Lock()
	b.state.Detectors.Break.Enabled = true
	b.state.Detectors.Break.LastBreakTime = time.Now().UnixMilli()
	b.state.mu.Unlock()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.update(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (b *BreakDetector) update(ctx context.Context) {
	b.state.mu.RLock()
	lastBreak := b.state.Detectors.Break.LastBreakTime
	breakDetected := b.state.Detectors.Break.BreakDetected
	b.state.mu.RUnlock()

	now := time.Now().UnixMilli()
	intervalMs := int64(b.settings.Break.IntervalMinutes) * 60 * 1000

	if now-lastBreak < intervalMs {
		return
	}

	if breakDetected {
		return
	}

	// Advise break - pause new actions
	b.state.mu.Lock()
	b.state.Detectors.Break.BreakAdviced = true
	b.state.mu.Unlock()

	// Check if ship is busy
	if b.isShipBusy() {
		return
	}

	b.log("Break time started")
	b.stats.SetMessage("Break time started")
	b.state.mu.Lock()
	b.state.Detectors.Break.BreakDetected = true
	b.state.mu.Unlock()
	b.state.SetBoolTrigger("break", true)

	// Navigate to safe zone
	b.recover.Start(ctx)

	// Wait for break duration
	durationMs := int64(b.settings.Break.DurationMinutes) * 60 * 1000
	endTime := time.Now().UnixMilli() + durationMs

	for time.Now().UnixMilli() < endTime {
		remaining := int(math.Ceil(float64(endTime-time.Now().UnixMilli()) / 60000.0))
		b.stats.SetMessage(fmt.Sprintf("Break remaining: %d min", remaining))
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			return
		}
	}

	b.log("Break finished")
	b.stats.SetMessage("Break finished")
	b.state.mu.Lock()
	b.state.Detectors.Break.LastBreakTime = time.Now().UnixMilli()
	b.state.Detectors.Break.BreakDetected = false
	b.state.Detectors.Break.BreakAdviced = false
	b.state.mu.Unlock()
	b.state.SetBoolTrigger("break", false)
	b.recover.Stop()
}

func (b *BreakDetector) isShipBusy() bool {
	if b.state.GetKillAttacking() || b.state.GetKillInProgress() {
		return true
	}
	if b.state.GetRecoverEnabled() || b.state.GetEscapeEnabled() {
		return true
	}
	b.state.mu.RLock()
	collecting := b.state.Collect.Collecting
	b.state.mu.RUnlock()
	return collecting
}
