# Wu Bot Progress Notes - 2026-02-10

## What Was Built

### Auto-Buy Equipment
- Extended `autobuy_detector.go` with equipment purchasing (lasers, shield gens, speed gens)
- Sends `EquipRequestPacket` to check current equipment, buys from shop if below target counts
- Config in `config.yaml` under `autobuy.equipment`
- **Tested working:** bought 15 LaserGun-1 + 15 ShieldGen-1 automatically

### Auto-Equip (EquipManager)
- New `equip_manager.go` - mounts unequipped items after purchase
- Shield gens -> config 1, speed gens -> config 2, lasers -> config 1
- Uses `EquipMoveRequestPacket` (Kryonet packet, not API)
- **Limitation:** Shuttle only has 1 laser + 1 gen slot, so extras sit in inventory

### Quest Discovery Module
- New `quest_detector.go` - probes actionIds and logs responses
- JAR bridge updated with `QuestsActionRequestPacket`, `MissionsActionRequestPacket`, `EquipMoveRequestPacket`

### Headless Log Draining
- `main.go` now drains bot log channels in headless mode so all log messages appear in `docker compose logs`
- Uses `StartAutoStartWithLogDrainer` to set up drainer before engine.Start()

### Bug Fixes Found During Testing
- `ShopItem.ItemID` was `string`, server sends `int` -> fixed to `int`
- `ShopItem.Price` / `Quantity` were `int`, server sends `float64` (e.g. `0.0`) -> fixed to `float64`
- Headless mode wasn't draining log channel -> added `StartAutoStartWithLogDrainer`

---

## Shop Catalog (Discovered)

What the shop actually has at our current level:

| itemId | kind | title | price | currency |
|--------|------|-------|-------|----------|
| 1 | ship_10 | Shuttle | 0 | credits |
| 2 | ship_20 | Zephyrus | 90,000 | credits |
| 3 | ship_30 | Thorus | 180,000 | credits |
| 4 | ship_41 | Veles-X | 400 | currency_3 |
| 5 | ship_40 | Veles | 270,000 | credits |
| 12 | ammo.laser_1 | RLX-1 | 100 | credits (qty 10) |
| 17 | ammo.rocket_1 | KEP-410 | 100 | credits (qty 1) |
| **24** | equipment.lasergun_1 | **LaserGun-1** | 40,000 | credits |
| **27** | equipment.shieldgen_1 | **ShieldGen-1** | 20,000 | credits |
| **30** | equipment.speedgen_1 | **Accelerator-1** | 80,000 | credits |
| 39 | equipment.extension_7 | A-CMPR | 15,000 | PLT |

**Problem:** LG-3, SG3N-B03 etc. are higher-tier items not available at our level. Config updated to use `LaserGun-1`, `ShieldGen-1`, `Accelerator-1`.

### Current Ship Stats
- **1 laser slot, 1 gen slot** (basic Shuttle)
- Already has 1 laser + 1 shield equipped
- Need to level up / buy a better ship to get more slots (Zephyrus = 90k credits)

---

## Map Objects (U-1)

```json
[
  {"type": "SPACE_STATION",   "x": 1000,  "y": 1000},
  {"type": "TRADE_STATION",   "x": 15000, "y": 9000},
  {"type": "QUEST_STATION",   "x": 3000,  "y": 1000},
  {"type": "NORMAL_TELEPORT", "x": 1000,  "y": 9000},
  {"type": "NORMAL_TELEPORT", "x": 15000, "y": 1000}
]
```

There IS a quest station at (3000, 1000). May need to be near it to accept quests.

---

## Quest Discovery Results (Exhaustive)

### What Was Tried

| Method | ActionIds/URIs | Result |
|--------|---------------|--------|
| QuestsActionRequestPacket | actionIds 1-10, with nil and [1] data | **NO RESPONSE** - packet type is dead |
| MissionsActionRequestPacket | actionIds 1-20, nil data | **Only actionId=3 responds: `[0,100,1]`** |
| MissionsAction(3) | data=nil, 1, 0-10 (ints) | Always returns same `[0,100,1]` |
| MissionsAction(3) | data=[1], [0], [0,100,1] (arrays) | **Kryo error** - ArrayList not registered |
| MissionsAction(3) | data={missionId:1}, {id:1} (maps) | No response |
| MissionsAction(1-10) | data=1 (int) | Only actionId=3 responds |
| API: quest/* | list, available, active, info, all | **No response** - URIs don't exist |
| API: quests/* | list, available | **No response** |
| API: missions/* | list, available, active, info | **No response** |
| API: quest/prioritize | questId 1-5 | **No response** |
| API: missions/miner/activation | empty data | **No response** |
| API: onboarding/cpanel/viewed/get | empty data | **No response** |
| All ApiNotifications | monitored all keys | Only `map-info` arrives (no quest notifications) |

### What `[0, 100, 1]` Means

Best theory: `[currentProgress, maxProgress, level]` for the resource miner (not quests). The "Missions" system appears to be the miner/resource system, not the quest system.

### QUEST PROTOCOL CRACKED! (2026-02-10)

**The key was `actionId=0`** - we had tried 1-30 but never tried 0!

#### QuestsActionRequestPacket Protocol

| ActionId | Action | Request Data | Response |
|----------|--------|-------------|----------|
| 0 | List all quests | `nil` | `[[questId, level, "name"], ..., [-9, null, null]]` |
| 1 | Get quest details | `questId` (int) | `[id, "name", "description", "conditions_DSL", {rewards}, status]` |
| 2 | Accept quest | `questId` (int) | `0`=success, `-1`=fail (also auto-sends actionId=1 response) |
| 3 | Complete/claim quest | `questId` (int) | `0`=success, `-1`=fail (also auto-sends actionId=1 response) |
| 4-10 | (no response) | - | - |

#### Quest Details Response Format

```json
[
  90,                           // questId
  "Map orientation",            // quest name
  "Fine, you have next...",     // description text
  "{\n\t[1:FT:10:10]\n\t...}", // conditions DSL string
  {"items":[                    // rewards
    {"type":2,"subtype":0,"amount":2000},
    {"type":3,"subtype":0,"amount":10},
    {"type":0,"subtype":0,"amount":4000},
    {"type":1,"subtype":0,"amount":30}
  ]},
  0                             // status: 0=available, 1=accepted
]
```

#### Quest Conditions DSL

Uses `QuestConditionsParser` format:
- `[id:FT:x:y]` = Fly To coordinates (x*100, y*100)
- `[id:C:materialType:amount]` = Collect material
- `[id:K:npcType:amount]` = Kill NPCs (theory)
- `<m:mapName>` = On specific map
- `{}` = condition group

#### Known Quest IDs (Level 1-3)

| ID | Level | Name |
|----|-------|------|
| 90 | 1 | Map orientation |
| 91 | 1 | Gaining Mercury |
| 92 | 1 | Cargo Boxes |
| 93 | 1 | Empty storage |
| 66 | 1 | Warming-up |
| 94 | 2 | Get better equipment |
| 95 | 2 | Ammunition resupply |
| 96 | 2 | Bonus boxes |
| 97 | 2 | New aliens |
| 539 | 2 | Daily dividends |
| 98 | 3 | Hunting the hunters |
| 99 | 3 | Refine |

#### What Didn't Work
- API URIs (quest/list, quest/info, etc.) - all return empty (server catch-all)
- REST HTTP calls to API server - all 404
- `quest/prioritize` API - only works for priority ordering, not listing
- QuestsAction(1-30, nil) - actionId=1 needs a specific questId as data

---

## Quest Manager (Built 2026-02-10)

Replaced `quest_detector.go` (discovery-only) with `quest_manager.go` (full automation):

### What It Does
- **Lists quests** via `QuestsActionPacket(0, nil)` → parses `[[id, level, "name"], ...]`
- **Fetches details** via `QuestsActionPacket(1, questId)` → parses conditions DSL, rewards, status
- **Parses conditions DSL**: `[id:FT:x:y]` (FlyTo), `[id:K:type:count]` (Kill), `[id:C:type:amount]` (Collect)
- **Accepts quests** via `QuestsActionPacket(2, questId)` → up to 5 at a time, sorted by level
- **Executes FlyTo** conditions - pauses kill controller via `questnav` trigger, flies to each waypoint
- **Completes quests** via `QuestsActionPacket(3, questId)` → claims rewards
- **Cycles every 90s** - re-checks progress, accepts new quests, retries completion
- **Logs quest info with extra fields** to discover progress data format

### Map Name Resolution
- DSL uses internal names like `f1`, resolved to display names like `U-1`
- Known aliases: f1→U-1, f2→U-2, f3→U-3, f4→U-4
- Quests with unknown map constraints are skipped

### Response Channel Design
- Separate buffered channels for list/info/action responses
- `HandleQuestsResponse` routes by actionId
- Channels drained before each request to handle stale/auto-sent responses
- Accept/Complete auto-send an actionId=1 response → goes to info channel, drained on next fetch

---

## Files Changed

| File | Change |
|------|--------|
| `internal/bot/autobuy_detector.go` | Equipment buying + catalog dump + type fixes |
| `internal/bot/equip_manager.go` | **NEW** - auto-mount equipment |
| `internal/bot/quest_manager.go` | **NEW** - quest automation (list, accept, FlyTo, complete) |
| `internal/bot/quest_detector.go` | **DELETED** - replaced by quest_manager.go |
| `internal/bot/engine.go` | Wire QuestManager, questnav trigger, clean up discovery routing |
| `internal/config/config.go` | `EquipmentAutobuySettings` struct |
| `internal/game/packets.go` | Equip/Quest payload structs, ShopItem type fixes |
| `internal/game/actions.go` | Equip/Quest packet builders |
| `internal/manager/manager.go` | `StartAutoStartWithLogDrainer` |
| `cmd/wubot/main.go` | Headless log draining |
| `config.yaml` | Equipment autobuy config (updated titles) |
| `wupacket/.../Main.java` | Added EquipMove + Quest + Mission to endpointClassMap |
| `wupacket.jar` | Rebuilt |

## TODO / Next Steps

### High Priority
1. **Quest progress tracking** - Run the bot and observe what the server returns when re-querying accepted quests (actionId=1). The response may include progress in extra fields or modified conditions DSL. Need to discover the format.
2. **QuestProgressUpdate packet** - Add `QuestProgressUpdate` class to the JAR bridge (`endpointClassMap`). The game client has this class and the server likely sends it when quest conditions are partially met (e.g., killed 3/5 NPCs). Currently unhandled.
3. **Test quest automation** - Run the bot headless and check logs to see:
   - Are quests being listed and accepted?
   - Do FlyTo conditions complete successfully?
   - Are kill/collect quests tracking progress?
   - Does completion claim rewards?

### Medium Priority
4. **Ship upgrade** - Auto-buy Zephyrus (90k credits) when affordable, for more equipment slots (current Shuttle = 1 laser + 1 gen)
5. **Quest condition filtering** - Currently accepts all quests with parsed conditions. Should filter by what the bot can actually complete (e.g., skip quests requiring items we don't have)
6. **Kill quest integration** - When a kill quest like "Strike the Hydro's" is accepted, verify the bot's kill targets match the quest's NPC type. Map quest condition NPC type IDs to NPC names.
7. **Collect quest integration** - Similarly verify collect quests match box types we're collecting

### Low Priority
8. **TUI quest view** - Show accepted quests and progress in the TUI
9. **Quest config** - Add config.yaml settings for quest automation (enabled/disabled, max quests, etc.)
10. **Better map aliases** - Discover all map internal names → display names mappings
