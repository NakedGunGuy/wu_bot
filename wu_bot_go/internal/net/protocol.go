package net

import (
	"encoding/json"
	"fmt"
	"strings"

	"wu_bot_go/internal/game"
)

// EncodePacket encodes an outbound packet into the wire format: endpoint|JSON\n
// For ApiRequestPacket, nested objects in requestDataJson must be double-encoded.
// For UserActionsPacket and ResourcesActionRequestPacket, use standard JSON.
func EncodePacket(pkt game.OutboundPacket) ([]byte, error) {
	if pkt.Data == nil {
		return []byte(pkt.Endpoint + "|\n"), nil
	}

	switch pkt.Endpoint {
	case game.EndpointUserActions, game.EndpointResourcesAction:
		// Standard JSON encoding
		jsonBytes, err := json.Marshal(pkt.Data)
		if err != nil {
			return nil, fmt.Errorf("marshal %s: %w", pkt.Endpoint, err)
		}
		return []byte(pkt.Endpoint + "|" + string(jsonBytes) + "\n"), nil
	default:
		// For ApiRequestPacket and others: nested objects become JSON strings (double-encoded)
		data, ok := pkt.Data.(map[string]interface{})
		if !ok {
			jsonBytes, err := json.Marshal(pkt.Data)
			if err != nil {
				return nil, fmt.Errorf("marshal %s: %w", pkt.Endpoint, err)
			}
			return []byte(pkt.Endpoint + "|" + string(jsonBytes) + "\n"), nil
		}

		encoded := make(map[string]interface{})
		for k, v := range data {
			switch val := v.(type) {
			case map[string]interface{}, []interface{}:
				// Double-encode: convert to JSON string
				inner, err := json.Marshal(val)
				if err != nil {
					return nil, fmt.Errorf("marshal nested %s.%s: %w", pkt.Endpoint, k, err)
				}
				encoded[k] = string(inner)
			default:
				encoded[k] = v
			}
		}

		jsonBytes, err := json.Marshal(encoded)
		if err != nil {
			return nil, fmt.Errorf("marshal %s: %w", pkt.Endpoint, err)
		}
		return []byte(pkt.Endpoint + "|" + string(jsonBytes) + "\n"), nil
	}
}

// DecodePacket decodes a line from the JAR into an InboundPacket.
// JAR sends JSON lines: {"type":"...", "payload":{...}}
func DecodePacket(line string) (*game.InboundPacket, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	var pkt game.InboundPacket
	if err := json.Unmarshal([]byte(line), &pkt); err != nil {
		return nil, fmt.Errorf("decode packet: %w (data: %s)", err, truncate(line, 200))
	}
	return &pkt, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
