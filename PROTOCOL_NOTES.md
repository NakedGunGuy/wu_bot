# Wu Bot - Protocol & Feature Notes

## Current State (2026-02-09)

### Improvements Implemented
- Health detector: 1000ms -> 200ms polling
- Health interrupts combat (calls KillManager.resetState() before Recover.start())
- Kill loop respects escape/recover states (early returns + skips)
- Portal matching: exact coords -> distance < 100 (recover, escape, navigation)
- Orbit movement: new startOrbiting/stopOrbiting in navigation.js (replaces antibanMove)
- Navigation following: faster updates (200-500ms), tighter prediction
- Client UI: attack lines, direction indicators, orbit/detection circles, bot state panel, log tab, enhanced stats

---

## Shop System (READY TO BUILD)

### List Items
- **Endpoint:** `shop/items/v2` (GET, empty request body)
- **Response:**
```json
{
  "itemsDataList": [
    {
      "itemId": 123,
      "itemKindId": "key_1",
      "title": "RLX-1",
      "shortTitle": "RLX-1",
      "description": "Laser weapon",
      "itemProperties": {},
      "currencyKindId": "currency_1",
      "price": 1000,
      "priceString": "1000",
      "category": "weapons",
      "quantity": 1
    }
  ]
}
```

### Buy Item
- **Endpoint:** `shop/buy` (POST)
- **Request:**
```json
{
  "itemId": 123,
  "quantity": 10000,
  "price": 500
}
```
- **Response:** ApiMessageResponse with status (NORMAL, SUCCESS, ERROR, DELAYED)
- **Already used by:** `modules/detectors/autobuy.js` for ammo purchases

### TODO: Build auto-buy for LG-3 lasers and SG3N generators
- Need to find itemIds for LG-3 and SG3N-B03 from shop catalog
- Buy until we have enough to fill all ship slots

---

## Equipment System (READY TO BUILD)

### Move/Equip Items
- **Endpoint:** `MoveEquipmentEndpoint`
- **Operations:** MOUNT, UNMOUNT
- **Vehicles:** SHIP, DRONE
- **Equipment Types:**
  - LASER = 1
  - SPEEDGEN = 2
  - SHIELDGEN = 3
  - EXTENSION = 4
  - DRONE_COVER = 5

### Mount Request
```json
{
  "items": [
    {
      "id": 123,
      "type": 1,
      "subtype": 2,
      "price": 500,
      "sellPrice": 250,
      "elite": false
    }
  ],
  "operation": "MOUNT",
  "vehicle": "SHIP",
  "confi": 1
}
```

### Equipment Settings (Get/Set)
- **Get:** `EquipmentSettingsGetEndpoint` - request: `{ "equipmentSettingsType": "AUTOMATIC_COMPRESSOR" }`
- **Set:** `EquipmentSettingsSetEndpoint` - request: `{ "equipmentSettingsType": "AUTOMATIC_COMPRESSOR", "equipmentSettingsJson": "{...}" }`

### Item Info
- **Endpoint:** `ItemInfoEndpoint`
- **Request:** `{ "id": "laser_rlx_1" }`
- **Response:** `{ "itemProperties": { "damage": "250", ... } }`

### TODO: Build auto-equip module
- After buying LG-3/SG3N, mount them to ship
- Need to know: how many laser slots + generator slots the ship has
- Or just buy and let user equip manually first

---

## Quest System (NEEDS PACKET SNIFFING)

### What Exists
- `PrioritizeQuestEndpoint` - only sets a quest as active/prioritized
  - Request: `{ "questId": <int> }`
- `QuestsActionRequestPacket` (legacy/deprecated): `{ "actionId": <int>, "data": <object> }`
- Quest progress tracked via `QuestProgressUpdate` with conditions

### What's Missing
- **No endpoint to list available quests**
- **No endpoint to accept quests**
- Quest acceptance likely happens through game client UI or undocumented packets
- Would need to sniff packets from the actual WarUniverse game client to reverse engineer

### Quest Conditions Format
- Complex nested format: `{[condition]:type:arg1:arg2}`
- Supports groups `{}`, conditions `[]`, subconditions `<>`, group conditions `()`
- Flags: `O` (ordered), `OC` (oneOfCond), `^` (antiCond)

### Goal
- Accept only NPC kill quests (no PvP, no collection)
- Auto-complete and claim rewards
- Need to figure out quest listing + acceptance packets first

---

## API Pattern Reference

All modern endpoints use `ApiRequestPacket`:
```js
client.sendPacket("ApiRequestPacket", {
  requestId: <unique_int>,
  uri: "shop/buy",
  requestDataJson: JSON.stringify({ itemId: 123, quantity: 1, price: 500 })
});
```

Responses come back as `ApiResponsePacket`:
```js
// Listened via kryo_packet event
// type: "ApiResponsePacket"
// payload: { uri: "shop/buy", responseDataJson: "..." }
```

Response statuses: NORMAL, SUCCESS, ERROR, DELAYED

---

## Files Modified in This Session

| File | Changes |
|------|---------|
| `modules/detectors/health.js` | Polling 1000ms -> 200ms |
| `modules/general/controller.js` | Health interrupts combat, force-stop kills |
| `modules/controllers/kill.js` | Escape/recover checks, orbiting, removed antibanMove |
| `modules/controllers/killcollect.js` | Escape/recover checks, removed antibanMove call |
| `modules/modifiers/navigation.js` | Orbit methods, faster following, distance-based portal |
| `modules/state/stateManager.js` | Added orbiting state |
| `modules/behaviour/recover.js` | Distance-based portal check |
| `modules/behaviour/escape.js` | Distance-based portal check |
| `client/renderer.js` | Attack lines, direction indicators, bot state panel, log tab, stats |
| `client/index.html` | Added Log tab |

## Next Steps
1. Build shop auto-buy for LG-3 lasers and SG3N generators
2. Build auto-equip module (mount items to ship)
3. Packet sniff quest system from game client to figure out accept/list
4. Build quest auto-accept for NPC kill quests only
