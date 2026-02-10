package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wu_bot_go/internal/bot"
	"wu_bot_go/internal/config"
	"wu_bot_go/internal/manager"
	"wu_bot_go/internal/tui"
)

func main() {
	headless := flag.Bool("headless", false, "Run in headless mode (no TUI)")
	cfgPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	mgr := manager.NewBotManager(cfg, *cfgPath)

	if *headless {
		runHeadless(mgr)
	} else {
		runTUI(mgr, *cfgPath)
	}
}

func runHeadless(mgr *manager.BotManager) {
	fmt.Println("WU Bot - Headless Mode")
	fmt.Printf("Loaded %d accounts\n", len(mgr.GetConfig().Accounts))

	mgr.StartAutoStartWithLogDrainer(func(username string, engine *bot.BotEngine) {
		go drainLogs(username, engine)
	})

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Periodic stats printer
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Println("\nShutting down...")
			mgr.StopAll()
			fmt.Println("Goodbye!")
			return
		case <-ticker.C:
			printStats(mgr)
		}
	}
}

func drainLogs(username string, engine *bot.BotEngine) {
	for entry := range engine.LogCh() {
		fmt.Printf("[%s] %s: %s\n", entry.Time.Format("15:04:05"), username, entry.Message)
	}
}

func printStats(mgr *manager.BotManager) {
	bots := mgr.ListBots()
	fmt.Println("---")
	for _, b := range bots {
		fmt.Printf("[%s] %s | %s | Map: %s | HP: %d%% | Kills: %d | Credits/hr: %d | %s | %s\n",
			b.Status, b.Username, b.Server, b.Map, b.HealthPercent,
			b.Kills, b.CreditsPerHr, b.RunTime, b.Message)
	}
}

func runTUI(mgr *manager.BotManager, cfgPath string) {
	if err := tui.Run(mgr, cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
