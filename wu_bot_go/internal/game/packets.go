package game

import "encoding/json"

// Inbound packet types (from JAR/game server)
const (
	PacketGameStateResponse      = "GameStateResponsePacket"
	PacketUserInfoResponse       = "UserInfoResponsePacket"
	PacketApiNotification        = "ApiNotification"
	PacketApiResponse            = "ApiResponsePacket"
	PacketAuthAnswer             = "AuthAnswerPacket"
	PacketGameEvent              = "GameEvent"
	PacketResourcesInfo          = "ResourcesInfoResponsePacket"
	PacketJAREvent               = "event"
	PacketEquipResponse          = "EquipResponsePacket"
	PacketEquipMoveResponse      = "EquipMoveResponsePacket"
	PacketQuestsActionResponse   = "QuestsActionResponsePacket"
	PacketMissionsActionResponse = "MissionsActionResponsePacket"
)

// Outbound packet endpoints (to JAR)
const (
	EndpointStartClient          = "startClient"
	EndpointStopClient           = "stopClient"
	EndpointApiRequest           = "ApiRequestPacket"
	EndpointUserActions          = "UserActionsPacket"
	EndpointCollectableCollect   = "CollectableCollectRequest"
	EndpointRepairRequest        = "RepairRequestPacket"
	EndpointRocketSwitch         = "RocketSwitchRequest"
	EndpointResourcesAction      = "ResourcesActionRequestPacket"
	EndpointEquipRequest         = "EquipRequestPacket"
	EndpointEquipMoveRequest     = "EquipMoveRequestPacket"
	EndpointQuestsAction         = "QuestsActionRequestPacket"
	EndpointMissionsAction       = "MissionsActionRequestPacket"
)

// InboundPacket represents a packet received from the JAR bridge.
type InboundPacket struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	Event   string          `json:"event,omitempty"`
}

// OutboundPacket represents a packet to send to the JAR bridge.
type OutboundPacket struct {
	Endpoint string
	Data     interface{}
}

// GameStateResponsePayload is the payload of a GameStateResponsePacket.
type GameStateResponsePayload struct {
	PlayerID     int              `json:"playerId"`
	SafeZone     bool             `json:"safeZone"`
	Ships        []ShipUpdate     `json:"ships"`
	Collectables []CollectableUpdate `json:"collectables"`
	MapChanges   json.RawMessage  `json:"mapChanges,omitempty"`
	Confi        *int             `json:"confi,omitempty"`
}

type ShipUpdate struct {
	ID        int            `json:"id"`
	Destroyed bool           `json:"destroyed"`
	Changes   []ShipChange   `json:"changes"`
}

type ShipChange struct {
	ID   int             `json:"id"`
	Data json.RawMessage `json:"data"`
}

type CollectableUpdate struct {
	ID         int  `json:"id"`
	Type       int  `json:"type"`
	X          int  `json:"x"`
	Y          int  `json:"y"`
	ExistOnMap bool `json:"existOnMap"`
}

// UserInfoResponsePayload is the payload of a UserInfoResponsePacket.
type UserInfoResponsePayload struct {
	Params []UserInfoParam `json:"params"`
}

type UserInfoParam struct {
	ID   int             `json:"id"`
	Type int             `json:"type"`
	Data json.RawMessage `json:"data"`
}

// ApiNotificationPayload is the payload of an ApiNotification.
type ApiNotificationPayload struct {
	Key                    string `json:"key"`
	NotificationJsonString string `json:"notificationJsonString"`
}

// MapInfoNotification is parsed from ApiNotification with key "map-info".
type MapInfoNotification struct {
	Name       string          `json:"name"`
	Width      int             `json:"width"`
	Height     int             `json:"height"`
	MapObjects json.RawMessage `json:"mapObjects"`
}

// ApiResponsePayload is the payload of an ApiResponsePacket.
type ApiResponsePayload struct {
	RequestID        int    `json:"requestId"`
	URI              string `json:"uri"`
	ResponseDataJson string `json:"responseDataJson"`
}

// AuthAnswerPayload is the payload of an AuthAnswerPacket.
type AuthAnswerPayload struct {
	Success bool `json:"success"`
}

// GameEventPayload is the payload of a GameEvent.
type GameEventPayload struct {
	ID int `json:"id"`
}

// ResourcesInfoPayload is the payload of a ResourcesInfoResponsePacket.
type ResourcesInfoPayload struct {
	Resources []ResourceInfo `json:"resources"`
}

type ResourceInfo struct {
	Amount int `json:"amount"`
}

// Shop types
type ShopItemsResponse struct {
	ItemsDataList []ShopItem `json:"itemsDataList"`
}

type ShopItem struct {
	ItemID         int     `json:"itemId"`
	ItemKindID     string  `json:"itemKindId"`
	Title          string  `json:"title"`
	Price          float64 `json:"price"`
	Quantity       float64 `json:"quantity"`
	CurrencyKindID string  `json:"currencyKindId"`
}

// Equipment types
const (
	EquipTypeLaser    = 1
	EquipTypeSpeedGen = 2
	EquipTypeShieldGen = 3
	EquipTypeExtension = 4
	EquipTypeDroneCover = 5
)

// EquipResponsePayload is the payload of an EquipResponsePacket.
type EquipResponsePayload struct {
	OnShip     []EquipmentItem `json:"onShip"`
	Equip      []EquipmentItem `json:"equip"`
	LaserSlots int             `json:"laserSlots"`
	GenSlots   int             `json:"genSlots"`
	ExtSlots   int             `json:"extSlots"`
}

type EquipmentItem struct {
	ID        int  `json:"id"`
	Type      int  `json:"type"`
	Subtype   int  `json:"subtype"`
	Price     int  `json:"price"`
	SellPrice int  `json:"sellPrice"`
	Elite     bool `json:"elite"`
}

type EquipSlots struct {
	LaserSlots int
	GenSlots   int
	ExtSlots   int
}

// EquipMoveResponsePayload is the payload of an EquipMoveResponsePacket.
type EquipMoveResponsePayload struct {
	Status int `json:"status"`
}

// QuestsActionResponsePayload is the payload of a QuestsActionResponsePacket.
type QuestsActionResponsePayload struct {
	ActionID int             `json:"actionId"`
	Data     json.RawMessage `json:"data"`
}

// MissionsActionResponsePayload is the payload of a MissionsActionResponsePacket.
type MissionsActionResponsePayload struct {
	ActionID int             `json:"actionId"`
	Data     json.RawMessage `json:"data"`
}
