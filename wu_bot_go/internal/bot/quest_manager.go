package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	"wu_bot_go/internal/game"
)

// Quest action IDs (discovered through probing - actionId=0 was the key!)
const (
	QuestActionList     = 0 // Returns [[questId, level, "name"], ...]
	QuestActionInfo     = 1 // Get quest details: [id, "name", "desc", "DSL", {rewards}, status]
	QuestActionAccept   = 2 // Accept quest (data=questId), returns 0=success, -1=fail
	QuestActionComplete = 3 // Complete/claim quest (data=questId), returns 0=success, -1=fail

	MaxAcceptedQuests = 5
)

// Quest condition types parsed from DSL
const (
	CondFlyTo   = "FT" // [id:FT:x:y] - Fly to coordinates (x*100, y*100)
	CondKill    = "K"  // [id:K:npcType:amount] - Kill NPCs
	CondCollect = "C"  // [id:C:materialType:amount] - Collect materials
)

// Map name aliases: DSL internal names → display names
var mapAliases = map[string]string{
	"f1": "U-1",
	"f2": "U-2",
	"f3": "U-3",
	"f4": "U-4",
}

// QuestCondition represents a single parsed condition from the DSL.
type QuestCondition struct {
	Index  int
	Type   string // FT, K, C, or unknown
	Param1 int    // FT: x (raw), K: npc type, C: material type
	Param2 int    // FT: y (raw), K: count, C: amount
}

// Quest holds all known info about a quest.
type Quest struct {
	ID            int
	Level         int
	Name          string
	Description   string
	ConditionsDSL string
	Conditions    []QuestCondition
	MapName       string // resolved display name (e.g., "U-1")
	MapNameRaw    string // raw DSL name (e.g., "f1")
	Rewards       json.RawMessage
	Status        int // 0=available, 1=accepted
}

// QuestManager handles quest automation: listing, accepting, tracking, completing.
type QuestManager struct {
	scene  *game.Scene
	state  *State
	sendCh chan<- game.OutboundPacket
	log    func(string)

	mu        sync.Mutex
	quests    map[int]*Quest  // All known quests by ID
	completed map[int]bool    // Quests completed this session (don't re-accept)

	// Response channels for synchronizing requests with packet dispatcher
	listResponseCh   chan json.RawMessage
	infoResponseCh   chan json.RawMessage
	actionResponseCh chan json.RawMessage
}

// NewQuestManager creates a new quest automation manager.
func NewQuestManager(scene *game.Scene, state *State, sendCh chan<- game.OutboundPacket, log func(string)) *QuestManager {
	return &QuestManager{
		scene:            scene,
		state:            state,
		sendCh:           sendCh,
		log:              log,
		quests:           make(map[int]*Quest),
		completed:        make(map[int]bool),
		listResponseCh:   make(chan json.RawMessage, 1),
		infoResponseCh:   make(chan json.RawMessage, 4), // Buffer for auto-sent info responses
		actionResponseCh: make(chan json.RawMessage, 1),
	}
}

// HandleQuestsResponse routes quest action responses to the appropriate channel.
func (qm *QuestManager) HandleQuestsResponse(payload *game.QuestsActionResponsePayload) {
	switch payload.ActionID {
	case QuestActionList:
		select {
		case qm.listResponseCh <- payload.Data:
		default:
		}
	case QuestActionInfo:
		select {
		case qm.infoResponseCh <- payload.Data:
		default:
		}
	case QuestActionAccept, QuestActionComplete:
		select {
		case qm.actionResponseCh <- payload.Data:
		default:
		}
	}
}

// Run starts the quest automation loop.
func (qm *QuestManager) Run(ctx context.Context) {
	// Initial delay - let the bot stabilize first
	select {
	case <-time.After(30 * time.Second):
	case <-ctx.Done():
		return
	}

	qm.log("Quest manager started")

	for {
		if ctx.Err() != nil {
			return
		}

		qm.questCycle(ctx)

		// Wait before next cycle
		select {
		case <-time.After(90 * time.Second):
		case <-ctx.Done():
			return
		}
	}
}

// questCycle runs one full cycle of quest management.
func (qm *QuestManager) questCycle(ctx context.Context) {
	// Step 1: List all available quests
	questIDs, err := qm.listQuests(ctx)
	if err != nil {
		qm.log(fmt.Sprintf("Quest list failed: %v", err))
		return
	}

	// Step 2: Fetch details for quests we haven't loaded yet
	for _, qid := range questIDs {
		if ctx.Err() != nil {
			return
		}
		qm.mu.Lock()
		q, known := qm.quests[qid]
		needsInfo := !known || len(q.Conditions) == 0
		qm.mu.Unlock()
		if needsInfo {
			qm.fetchQuestInfo(ctx, qid)
			time.Sleep(300 * time.Millisecond) // Don't spam the server
		}
	}

	// Step 3: Re-fetch details for accepted quests to check progress
	qm.mu.Lock()
	var acceptedIDs []int
	for _, q := range qm.quests {
		if q.Status == 1 {
			acceptedIDs = append(acceptedIDs, q.ID)
		}
	}
	qm.mu.Unlock()

	for _, qid := range acceptedIDs {
		if ctx.Err() != nil {
			return
		}
		qm.fetchQuestInfo(ctx, qid)
		time.Sleep(300 * time.Millisecond)
	}

	// Step 4: Try to complete accepted quests
	for _, qid := range acceptedIDs {
		if ctx.Err() != nil {
			return
		}
		qm.tryCompleteQuest(ctx, qid)
		time.Sleep(300 * time.Millisecond)
	}

	// Step 5: Accept new quests if we have room
	qm.mu.Lock()
	acceptedCount := 0
	for _, q := range qm.quests {
		if q.Status == 1 {
			acceptedCount++
		}
	}
	qm.mu.Unlock()

	if acceptedCount < MaxAcceptedQuests {
		qm.acceptNewQuests(ctx, MaxAcceptedQuests-acceptedCount)
	}

	// Step 6: Execute FlyTo conditions for accepted quests
	qm.executeFlyToConditions(ctx)

	// Step 7: Try completing again after executing conditions
	qm.mu.Lock()
	acceptedIDs = nil
	for _, q := range qm.quests {
		if q.Status == 1 {
			acceptedIDs = append(acceptedIDs, q.ID)
		}
	}
	qm.mu.Unlock()

	for _, qid := range acceptedIDs {
		if ctx.Err() != nil {
			return
		}
		qm.tryCompleteQuest(ctx, qid)
		time.Sleep(300 * time.Millisecond)
	}
}

// listQuests sends a quest list request and parses the response.
func (qm *QuestManager) listQuests(ctx context.Context) ([]int, error) {
	drainChannel(qm.listResponseCh)
	qm.sendCh <- game.BuildQuestsActionPacket(QuestActionList, nil)

	select {
	case data := <-qm.listResponseCh:
		return qm.parseQuestList(data)
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (qm *QuestManager) parseQuestList(data json.RawMessage) ([]int, error) {
	var rawList []json.RawMessage
	if err := json.Unmarshal(data, &rawList); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	var ids []int
	for _, entry := range rawList {
		var arr []json.RawMessage
		if err := json.Unmarshal(entry, &arr); err != nil {
			continue
		}
		if len(arr) < 3 {
			continue
		}

		var questID int
		if err := json.Unmarshal(arr[0], &questID); err != nil {
			continue
		}
		if questID < 0 { // Skip terminators (e.g., -9)
			continue
		}

		var level int
		var name string
		json.Unmarshal(arr[1], &level)
		json.Unmarshal(arr[2], &name)

		qm.mu.Lock()
		if _, exists := qm.quests[questID]; !exists {
			qm.quests[questID] = &Quest{
				ID:    questID,
				Level: level,
				Name:  name,
			}
		}
		qm.mu.Unlock()

		ids = append(ids, questID)
	}

	qm.log(fmt.Sprintf("Quests: %d available", len(ids)))
	return ids, nil
}

// fetchQuestInfo requests and parses details for a specific quest.
func (qm *QuestManager) fetchQuestInfo(ctx context.Context, questID int) {
	drainChannel(qm.infoResponseCh)
	qm.sendCh <- game.BuildQuestsActionPacket(QuestActionInfo, questID)

	select {
	case data := <-qm.infoResponseCh:
		qm.parseQuestInfo(questID, data)
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
	}
}

func (qm *QuestManager) parseQuestInfo(questID int, data json.RawMessage) {
	// Response: [id, "name", "description", "conditions_DSL", {rewards}, status, ...?]
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return
	}
	if len(arr) < 6 {
		return
	}

	var id int
	var name, description, conditionsDSL string
	var status int

	json.Unmarshal(arr[0], &id)
	json.Unmarshal(arr[1], &name)
	json.Unmarshal(arr[2], &description)
	json.Unmarshal(arr[3], &conditionsDSL)
	json.Unmarshal(arr[5], &status)

	conditions, mapNameRaw := parseConditionsDSL(conditionsDSL)
	mapName := resolveMapName(mapNameRaw)

	qm.mu.Lock()
	q, exists := qm.quests[id]
	if !exists {
		q = &Quest{ID: id}
		qm.quests[id] = q
	}
	q.Name = name
	q.Description = description
	q.ConditionsDSL = conditionsDSL
	q.Conditions = conditions
	q.MapName = mapName
	q.MapNameRaw = mapNameRaw
	q.Rewards = arr[4]
	q.Status = status
	qm.mu.Unlock()

	// Log full response for progress discovery (truncate long DSL)
	dslShort := conditionsDSL
	if len(dslShort) > 100 {
		dslShort = dslShort[:100] + "..."
	}
	statusStr := "available"
	if status == 1 {
		statusStr = "accepted"
	}

	// Log extra fields if present (might contain progress data)
	extra := ""
	if len(arr) > 6 {
		extraBytes, _ := json.Marshal(arr[6:])
		extra = fmt.Sprintf(" extra=%s", string(extraBytes))
	}

	qm.log(fmt.Sprintf("Quest #%d %q: status=%s conditions=%q map=%s%s",
		id, name, statusStr, dslShort, mapName, extra))
}

// tryCompleteQuest attempts to complete/claim a quest.
func (qm *QuestManager) tryCompleteQuest(ctx context.Context, questID int) {
	drainChannel(qm.actionResponseCh)
	qm.sendCh <- game.BuildQuestsActionPacket(QuestActionComplete, questID)

	select {
	case data := <-qm.actionResponseCh:
		var result int
		if err := json.Unmarshal(data, &result); err == nil {
			if result == 0 {
				qm.mu.Lock()
				qm.completed[questID] = true
				name := ""
				if q, ok := qm.quests[questID]; ok {
					name = q.Name
				}
				delete(qm.quests, questID)
				qm.mu.Unlock()
				qm.log(fmt.Sprintf("Quest COMPLETED: %q (#%d)", name, questID))
			}
			// result == -1 means not completable yet, that's normal
		}
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
	}
}

// acceptNewQuests accepts available quests up to the given count.
func (qm *QuestManager) acceptNewQuests(ctx context.Context, count int) {
	qm.mu.Lock()
	currentMap, _, _ := qm.scene.GetMapInfo()

	var candidates []*Quest
	for _, q := range qm.quests {
		if q.Status != 0 || qm.completed[q.ID] {
			continue
		}
		if len(q.Conditions) == 0 {
			continue // No parsed conditions yet
		}
		// Check map constraint
		if q.MapName != "" && q.MapName != currentMap {
			continue
		}
		candidates = append(candidates, q)
	}
	qm.mu.Unlock()

	// Sort by level ascending (easier quests first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Level < candidates[j].Level
	})

	accepted := 0
	for _, q := range candidates {
		if accepted >= count || ctx.Err() != nil {
			break
		}
		if qm.acceptQuest(ctx, q.ID) {
			accepted++
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// acceptQuest sends an accept request for a single quest.
func (qm *QuestManager) acceptQuest(ctx context.Context, questID int) bool {
	drainChannel(qm.actionResponseCh)
	qm.sendCh <- game.BuildQuestsActionPacket(QuestActionAccept, questID)

	select {
	case data := <-qm.actionResponseCh:
		var result int
		if err := json.Unmarshal(data, &result); err == nil && result == 0 {
			qm.mu.Lock()
			if q, ok := qm.quests[questID]; ok {
				q.Status = 1
				qm.log(fmt.Sprintf("Quest accepted: %q (#%d)", q.Name, q.ID))
			}
			qm.mu.Unlock()
			return true
		}
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
	}
	return false
}

// executeFlyToConditions navigates to FlyTo coordinates for accepted quests.
func (qm *QuestManager) executeFlyToConditions(ctx context.Context) {
	qm.mu.Lock()
	var flyToQuests []*Quest
	for _, q := range qm.quests {
		if q.Status != 1 {
			continue
		}
		hasFlyTo := false
		for _, c := range q.Conditions {
			if c.Type == CondFlyTo {
				hasFlyTo = true
				break
			}
		}
		if hasFlyTo {
			cp := *q
			flyToQuests = append(flyToQuests, &cp)
		}
	}
	qm.mu.Unlock()

	if len(flyToQuests) == 0 {
		return
	}

	// Pause kill/collect controllers during FlyTo navigation
	qm.state.SetBoolTrigger("questnav", true)
	defer qm.state.SetBoolTrigger("questnav", false)

	// Brief delay for controllers to stop
	select {
	case <-time.After(500 * time.Millisecond):
	case <-ctx.Done():
		return
	}

	for _, q := range flyToQuests {
		if ctx.Err() != nil {
			return
		}
		qm.executeFlyToForQuest(ctx, q)
	}
}

func (qm *QuestManager) executeFlyToForQuest(ctx context.Context, q *Quest) {
	for _, cond := range q.Conditions {
		if cond.Type != CondFlyTo || ctx.Err() != nil {
			continue
		}

		targetX := cond.Param1 * 100
		targetY := cond.Param2 * 100

		px, py := qm.scene.GetPosition()
		dist := game.Distance(px, py, targetX, targetY)

		if dist < 200 {
			continue // Already near this waypoint
		}

		qm.log(fmt.Sprintf("Quest %q: Flying to (%d, %d)", q.Name, targetX, targetY))
		qm.scene.MoveAndWait(ctx, qm.sendCh, targetX, targetY)

		// Brief pause at destination for server to register
		select {
		case <-time.After(2 * time.Second):
		case <-ctx.Done():
			return
		}
	}
}

// --- DSL Parser ---

var conditionRe = regexp.MustCompile(`\[(\d+):(\w+):(\d+):(\d+)\]`)
var mapRe = regexp.MustCompile(`<m:([a-zA-Z0-9_-]+)>`)

// parseConditionsDSL parses the quest conditions DSL string.
// Examples:
//
//	{[1:FT:10:10][2:FT:140:90]<m:f1>}
//	{[1:K:1:5]<m:f1>}
//	{[1:C:4:100]}
func parseConditionsDSL(dsl string) ([]QuestCondition, string) {
	var conditions []QuestCondition

	matches := conditionRe.FindAllStringSubmatch(dsl, -1)
	for _, m := range matches {
		idx, _ := strconv.Atoi(m[1])
		p1, _ := strconv.Atoi(m[3])
		p2, _ := strconv.Atoi(m[4])
		conditions = append(conditions, QuestCondition{
			Index:  idx,
			Type:   m[2],
			Param1: p1,
			Param2: p2,
		})
	}

	var mapName string
	mapMatch := mapRe.FindStringSubmatch(dsl)
	if len(mapMatch) > 1 {
		mapName = mapMatch[1]
	}

	return conditions, mapName
}

// resolveMapName converts a DSL internal map name to display name.
func resolveMapName(raw string) string {
	if raw == "" {
		return ""
	}
	if display, ok := mapAliases[raw]; ok {
		return display
	}
	return raw // Unknown alias, return raw
}

// drainChannel empties a buffered channel.
func drainChannel(ch chan json.RawMessage) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
