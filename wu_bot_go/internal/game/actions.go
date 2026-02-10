package game

import "fmt"

// Action IDs for UserActionsPacket
const (
	ActionMove         = 1
	ActionSelectTarget = 2
	ActionAttack       = 3
	ActionDeselect     = 4
	ActionSwitchConfig = 5
	ActionJumpPortal   = 6
	ActionSwitchAmmo   = 12
)

// Game event IDs
const (
	EventShipDestroyed = 11
	EventShipRevived   = 12
)

// Ship change IDs
const (
	ChangeNameID         = 12
	ChangeClanTagID      = 13
	ChangeCorporationID  = 14
	ChangePositionID     = 17
	ChangeTargetPosID    = 18
	ChangeSelectedID     = 20
	ChangeIsAttackingID  = 22
	ChangeInRangeID      = 23
	ChangeShipTypeID     = 24
	ChangeHealthID       = 25
	ChangeMaxHealthID    = 26
	ChangeShieldID       = 27
	ChangeMaxShieldID    = 28
	ChangeCargoID        = 29
	ChangeMaxCargoID     = 30
	ChangeSpeedID        = 31
	ChangeDroneArrayID   = 32
	ChangeHeartbeatID    = 42
)

// Enrichment module indices
const (
	EnrichLasers  = 0
	EnrichRockets = 1
	EnrichShields = 2
	EnrichSpeed   = 3
)

// BuildMoveAction creates a move action packet.
func BuildMoveAction(x, y int) OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointUserActions,
		Data: map[string]interface{}{
			"actions": []map[string]interface{}{
				{"actionId": ActionMove, "data": fmt.Sprintf("%d|%d", x, y)},
			},
		},
	}
}

// BuildSelectAction creates a select-target action packet.
func BuildSelectAction(shipID int) OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointUserActions,
		Data: map[string]interface{}{
			"actions": []map[string]interface{}{
				{"actionId": ActionSelectTarget, "data": shipID},
			},
		},
	}
}

// BuildAttackAction creates an attack action packet.
func BuildAttackAction() OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointUserActions,
		Data: map[string]interface{}{
			"actions": []map[string]interface{}{
				{"actionId": ActionAttack},
			},
		},
	}
}

// BuildDeselectAction creates a deselect action packet.
func BuildDeselectAction() OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointUserActions,
		Data: map[string]interface{}{
			"actions": []map[string]interface{}{
				{"actionId": ActionDeselect},
			},
		},
	}
}

// BuildSwitchConfigAction creates a config-switch action packet.
func BuildSwitchConfigAction() OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointUserActions,
		Data: map[string]interface{}{
			"actions": []map[string]interface{}{
				{"actionId": ActionSwitchConfig},
			},
		},
	}
}

// BuildJumpPortalAction creates a portal-jump action packet.
func BuildJumpPortalAction() OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointUserActions,
		Data: map[string]interface{}{
			"actions": []map[string]interface{}{
				{"actionId": ActionJumpPortal},
			},
		},
	}
}

// BuildSwitchAmmoAction creates an ammo-switch action packet.
func BuildSwitchAmmoAction(ammoType int) OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointUserActions,
		Data: map[string]interface{}{
			"actions": []map[string]interface{}{
				{"actionId": ActionSwitchAmmo, "data": ammoType},
			},
		},
	}
}

// BuildRocketSwitchPacket creates a rocket-switch packet.
func BuildRocketSwitchPacket(rocketID int) OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointRocketSwitch,
		Data: map[string]interface{}{
			"rocketId": rocketID,
		},
	}
}

// BuildCollectPacket creates a collect-box packet.
func BuildCollectPacket(id int) OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointCollectableCollect,
		Data: map[string]interface{}{
			"id": id,
		},
	}
}

// BuildRepairPacket creates a repair/revive packet.
func BuildRepairPacket() OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointRepairRequest,
		Data:     nil,
	}
}

// BuildStartClientPacket creates the initial JAR connection packet.
func BuildStartClientPacket(host string, port int) OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointStartClient,
		Data: map[string]interface{}{
			"host": host,
			"port": port,
		},
	}
}

// BuildStopClientPacket creates a stop-client packet.
func BuildStopClientPacket() OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointStopClient,
		Data:     map[string]interface{}{},
	}
}

// BuildApiRequestPacket creates an API request packet.
// For ApiRequestPacket, nested objects in requestDataJson must be double-encoded (JSON string).
func BuildApiRequestPacket(requestID int, uri string, requestData interface{}) OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointApiRequest,
		Data: map[string]interface{}{
			"requestId":       requestID,
			"uri":             uri,
			"requestDataJson": requestData,
		},
	}
}

// BuildResourcesActionPacket creates a resources action packet.
func BuildResourcesActionPacket(actionID int, data []int) OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointResourcesAction,
		Data: map[string]interface{}{
			"actionId": actionID,
			"data":     data,
		},
	}
}

// BuildResourcesRequestPacket creates an empty resources request packet (to query resources).
func BuildResourcesRequestPacket() OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointResourcesAction,
		Data:     map[string]interface{}{},
	}
}

// BuildEquipRequestPacket creates an equipment list request for a given config.
func BuildEquipRequestPacket(confi int) OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointEquipRequest,
		Data: map[string]interface{}{
			"confi": confi,
		},
	}
}

// BuildEquipMovePacket creates a packet to mount/unmount a single equipment item.
func BuildEquipMovePacket(item EquipmentItem, confi int, toHangar bool) OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointEquipMoveRequest,
		Data: map[string]interface{}{
			"item": map[string]interface{}{
				"id":        item.ID,
				"type":      item.Type,
				"subtype":   item.Subtype,
				"price":     item.Price,
				"sellPrice": item.SellPrice,
				"elite":     item.Elite,
			},
			"toHangar": toHangar,
			"confi":    confi,
			"drone":    false,
		},
	}
}

// BuildQuestsActionPacket creates a quest action request.
func BuildQuestsActionPacket(actionID int, data interface{}) OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointQuestsAction,
		Data: map[string]interface{}{
			"actionId": actionID,
			"data":     data,
		},
	}
}

// BuildMissionsActionPacket creates a missions action request.
func BuildMissionsActionPacket(actionID int, data interface{}) OutboundPacket {
	return OutboundPacket{
		Endpoint: EndpointMissionsAction,
		Data: map[string]interface{}{
			"actionId": actionID,
			"data":     data,
		},
	}
}
