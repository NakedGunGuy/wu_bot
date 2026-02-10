package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"wu_bot_go/internal/game"
)

type itemConfig struct {
	Amount    int
	MinAmount int
	ItemID    int
	Price     int
	Currency  string
}

// AutoBuyDetector monitors ammo levels and auto-purchases from the shop.
type AutoBuyDetector struct {
	state    *State
	settings *Settings
	user     *game.User
	sendCh   chan<- game.OutboundPacket
	log      func(string)

	laserConfigs  map[string]*itemConfig
	rocketConfigs map[string]*itemConfig
	keyConfig     *itemConfig
	items         []game.ShopItem
	requestID     int
	buyInProgress bool
	catalogLogged bool

	// Equipment autobuy state
	equipmentItems []game.EquipmentItem // current equipment from EquipResponsePacket
	equipSlots     game.EquipSlots      // laser/gen/ext slot counts
	equipLoaded    bool
	equipRequested bool
	equipManager   *EquipManager
}

func NewAutoBuyDetector(state *State, settings *Settings, user *game.User, sendCh chan<- game.OutboundPacket, log func(string)) *AutoBuyDetector {
	return &AutoBuyDetector{
		state:    state,
		settings: settings,
		user:     user,
		sendCh:   sendCh,
		log:      log,
		requestID: 50,
		laserConfigs: map[string]*itemConfig{
			"RLX_1":    {Amount: 10000, MinAmount: 1000},
			"GLX_2":    {Amount: 10000, MinAmount: 1000},
			"BLX_3":    {Amount: 10000, MinAmount: 1000},
			"GLX_2_AS": {Amount: 10000, MinAmount: 1000},
			"MRS_6X":   {Amount: 1000, MinAmount: 1000},
		},
		rocketConfigs: map[string]*itemConfig{
			"KEP_410": {Amount: 1000, MinAmount: 100},
			"NC_30":   {Amount: 1000, MinAmount: 100},
			"TNC_130": {Amount: 1000, MinAmount: 100},
		},
		keyConfig: &itemConfig{Amount: 1, MinAmount: 1},
	}
}

// SetEquipManager sets the equip manager reference for triggering auto-equip after purchases.
func (a *AutoBuyDetector) SetEquipManager(em *EquipManager) {
	a.equipManager = em
}

// HandleEquipResponse processes EquipResponsePacket to track current equipment.
func (a *AutoBuyDetector) HandleEquipResponse(payload *game.EquipResponsePayload) {
	a.equipSlots = game.EquipSlots{
		LaserSlots: payload.LaserSlots,
		GenSlots:   payload.GenSlots,
		ExtSlots:   payload.ExtSlots,
	}

	// Combine onShip and equip for full inventory view
	allItems := make([]game.EquipmentItem, 0, len(payload.OnShip)+len(payload.Equip))
	allItems = append(allItems, payload.OnShip...)
	allItems = append(allItems, payload.Equip...)
	a.equipmentItems = allItems
	a.equipLoaded = true

	if !a.equipRequested {
		return
	}
	a.equipRequested = false

	// Log equipment state
	laserCount := 0
	shieldCount := 0
	speedCount := 0
	for _, item := range payload.OnShip {
		switch item.Type {
		case game.EquipTypeLaser:
			laserCount++
		case game.EquipTypeShieldGen:
			shieldCount++
		case game.EquipTypeSpeedGen:
			speedCount++
		}
	}
	a.log(fmt.Sprintf("Equipment on ship: %d lasers, %d shields, %d speed (slots: %d laser, %d gen)",
		laserCount, shieldCount, speedCount, payload.LaserSlots, payload.GenSlots))

	// Forward to equip manager for mounting
	if a.equipManager != nil {
		a.equipManager.HandleEquipResponse(payload)
	}
}

// HandleApiResponse processes shop API responses.
func (a *AutoBuyDetector) HandleApiResponse(payload *game.ApiResponsePayload) {
	if payload.URI != "shop/items/v2" {
		return
	}

	var resp game.ShopItemsResponse
	if err := json.Unmarshal([]byte(payload.ResponseDataJson), &resp); err != nil {
		a.log(fmt.Sprintf("Shop parse error: %v (data: %.100s)", err, payload.ResponseDataJson))
		return
	}

	a.items = resp.ItemsDataList

	// One-time catalog dump for discovery
	if !a.catalogLogged {
		a.catalogLogged = true
		a.log("=== SHOP CATALOG DUMP ===")
		for _, item := range a.items {
			a.log(fmt.Sprintf("  itemId=%d kind=%s title=%q price=%.0f currency=%s qty=%.0f",
				item.ItemID, item.ItemKindID, item.Title, item.Price, item.CurrencyKindID, item.Quantity))
		}
		a.log(fmt.Sprintf("=== END CATALOG (%d items) ===", len(a.items)))
	}

	a.updateItemConfigs()
}

func (a *AutoBuyDetector) updateItemConfigs() {
	for typeName, cfg := range a.laserConfigs {
		title := strings.ReplaceAll(typeName, "_", "-")
		for _, item := range a.items {
			if item.Title == title {
				cfg.ItemID = item.ItemID
				if item.Quantity > 0 {
					cfg.Price = int(float64(cfg.Amount) / item.Quantity * item.Price)
				}
				if item.CurrencyKindID == "currency_2" {
					cfg.Currency = "plt"
				} else {
					cfg.Currency = "credits"
				}
				break
			}
		}
	}

	for typeName, cfg := range a.rocketConfigs {
		title := strings.ReplaceAll(typeName, "_", "-")
		for _, item := range a.items {
			if item.Title == title {
				cfg.ItemID = item.ItemID
				if item.Quantity > 0 {
					cfg.Price = int(float64(cfg.Amount) / item.Quantity * item.Price)
				}
				if item.CurrencyKindID == "currency_2" {
					cfg.Currency = "plt"
				} else {
					cfg.Currency = "credits"
				}
				break
			}
		}
	}

	for _, item := range a.items {
		if item.ItemKindID == "key_1" {
			a.keyConfig.ItemID = item.ItemID
			a.keyConfig.Price = int(float64(a.keyConfig.Amount) * item.Price)
			if item.CurrencyKindID == "currency_2" {
				a.keyConfig.Currency = "plt"
			} else {
				a.keyConfig.Currency = "credits"
			}
			break
		}
	}
}

// Run starts the autobuy check loop (5s interval).
func (a *AutoBuyDetector) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.checkResources(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (a *AutoBuyDetector) checkResources(ctx context.Context) {
	if a.buyInProgress || !a.user.GetIsLoaded() {
		return
	}

	if a.items == nil {
		a.sendCh <- game.BuildApiRequestPacket(a.requestID, "shop/items/v2", map[string]interface{}{})
		a.requestID++
		return
	}

	a.buyInProgress = true
	defer func() { a.buyInProgress = false }()

	snap := a.user.GetSnapshot()

	// Check lasers
	for typeName, enabled := range a.settings.Autobuy.Laser {
		if !enabled {
			continue
		}
		cfg, ok := a.laserConfigs[typeName]
		if !ok || cfg.ItemID == 0 {
			continue
		}
		current := snap.Lasers[typeName]
		if current < cfg.MinAmount {
			if a.canAfford(snap, cfg) {
				a.log(fmt.Sprintf("Buying %d %s for %d %s", cfg.Amount, typeName, cfg.Price, cfg.Currency))
				a.buyItem(cfg.ItemID, cfg.Amount, cfg.Price)
				time.Sleep(1 * time.Second)
			}
		}
	}

	// Check rockets
	for typeName, enabled := range a.settings.Autobuy.Rockets {
		if !enabled {
			continue
		}
		cfg, ok := a.rocketConfigs[typeName]
		if !ok || cfg.ItemID == 0 {
			continue
		}
		current := snap.Rockets[typeName]
		if current < cfg.MinAmount {
			if a.canAfford(snap, cfg) {
				a.log(fmt.Sprintf("Buying %d %s for %d %s", cfg.Amount, typeName, cfg.Price, cfg.Currency))
				a.buyItem(cfg.ItemID, cfg.Amount, cfg.Price)
				time.Sleep(1 * time.Second)
			}
		}
	}

	// Check booty keys
	if a.settings.Autobuy.Key.Enabled && a.keyConfig.ItemID != 0 {
		if snap.BootyKeys < a.keyConfig.MinAmount && snap.PLT >= a.settings.Autobuy.Key.SavePLT {
			if snap.PLT >= a.keyConfig.Price {
				a.log(fmt.Sprintf("Buying booty key for %d PLT", a.keyConfig.Price))
				a.buyItem(a.keyConfig.ItemID, a.keyConfig.Amount, a.keyConfig.Price)
			}
		}
	}

	// Check equipment
	if a.settings.Autobuy.Equipment.Enabled {
		a.checkEquipment(snap)
	}
}

func (a *AutoBuyDetector) checkEquipment(snap game.UserSnapshot) {
	// First, request equipment info if we haven't yet
	if !a.equipLoaded {
		if !a.equipRequested {
			a.equipRequested = true
			a.sendCh <- game.BuildEquipRequestPacket(1)
		}
		return
	}

	cfg := a.settings.Autobuy.Equipment

	// Count what we have (both on ship and in inventory)
	totalLasers := 0
	totalShieldGens := 0
	totalSpeedGens := 0
	for _, item := range a.equipmentItems {
		switch item.Type {
		case game.EquipTypeLaser:
			totalLasers++
		case game.EquipTypeShieldGen:
			totalShieldGens++
		case game.EquipTypeSpeedGen:
			totalSpeedGens++
		}
	}

	bought := false

	// Buy lasers if needed
	if cfg.LaserTitle != "" && totalLasers < cfg.LaserCount {
		needed := cfg.LaserCount - totalLasers
		for _, item := range a.items {
			if item.Title == cfg.LaserTitle {
				price := int(item.Price)
				for i := 0; i < needed; i++ {
					if snap.Credits >= price {
						a.log(fmt.Sprintf("Buying equipment: %s (%d/%d) for %d credits",
							cfg.LaserTitle, totalLasers+i+1, cfg.LaserCount, price))
						a.buyItem(item.ItemID, 1, price)
						time.Sleep(500 * time.Millisecond)
						bought = true
					}
				}
				break
			}
		}
	}

	// Buy shield gens if needed
	if cfg.ShieldGenTitle != "" && totalShieldGens < cfg.GenCount {
		needed := cfg.GenCount - totalShieldGens
		for _, item := range a.items {
			if item.Title == cfg.ShieldGenTitle {
				price := int(item.Price)
				for i := 0; i < needed; i++ {
					if snap.Credits >= price {
						a.log(fmt.Sprintf("Buying equipment: %s (%d/%d) for %d credits",
							cfg.ShieldGenTitle, totalShieldGens+i+1, cfg.GenCount, price))
						a.buyItem(item.ItemID, 1, price)
						time.Sleep(500 * time.Millisecond)
						bought = true
					}
				}
				break
			}
		}
	}

	// Buy speed gens if needed
	if cfg.SpeedGenTitle != "" && totalSpeedGens < cfg.GenCount {
		needed := cfg.GenCount - totalSpeedGens
		for _, item := range a.items {
			if item.Title == cfg.SpeedGenTitle {
				price := int(item.Price)
				for i := 0; i < needed; i++ {
					if snap.Credits >= price {
						a.log(fmt.Sprintf("Buying equipment: %s (%d/%d) for %d credits",
							cfg.SpeedGenTitle, totalSpeedGens+i+1, cfg.GenCount, price))
						a.buyItem(item.ItemID, 1, price)
						time.Sleep(500 * time.Millisecond)
						bought = true
					}
				}
				break
			}
		}
	}

	// After buying, refresh equipment list and trigger equip
	if bought {
		a.equipLoaded = false
		a.equipRequested = true
		time.Sleep(1 * time.Second)
		a.sendCh <- game.BuildEquipRequestPacket(1)
	}
}

func (a *AutoBuyDetector) canAfford(snap game.UserSnapshot, cfg *itemConfig) bool {
	if cfg.Currency == "plt" {
		return snap.PLT >= cfg.Price
	}
	return snap.Credits >= cfg.Price
}

func (a *AutoBuyDetector) buyItem(itemID int, quantity, price int) {
	a.sendCh <- game.BuildApiRequestPacket(a.requestID, "shop/buy", map[string]interface{}{
		"quantity": quantity,
		"itemId":   itemID,
		"price":    price,
	})
	a.requestID++
}
