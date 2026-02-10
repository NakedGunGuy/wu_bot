package game

import "math"

// Portal represents a teleport portal on a map.
type Portal struct {
	X  int
	Y  int
	To string
}

// MapRegions contains all portal definitions for each map.
var MapRegions = map[string][]Portal{
	"R-1": {
		{X: 1000, Y: 1000, To: "R-2"},
		{X: 1000, Y: 9000, To: "R-3"},
	},
	"R-2": {
		{X: 8000, Y: 9000, To: "R-3"},
		{X: 1000, Y: 1000, To: "J-VS"},
		{X: 15000, Y: 1000, To: "R-1"},
	},
	"R-3": {
		{X: 1000, Y: 11500, To: "E-3"},
		{X: 10000, Y: 11500, To: "T-1"},
		{X: 19000, Y: 11500, To: "J-SO"},
		{X: 1000, Y: 7500, To: "U-3"},
		{X: 19000, Y: 1000, To: "R-2"},
	},
	"R-5": {
		{X: 1000, Y: 11500, To: "R-7"},
		{X: 10000, Y: 11500, To: "R-6"},
		{X: 1000, Y: 1000, To: "G-1"},
		{X: 19000, Y: 1000, To: "T-1"},
	},
	"R-6": {
		{X: 1000, Y: 17700, To: "R-7"},
		{X: 29000, Y: 1000, To: "R-5"},
	},
	"R-7": {
		{X: 15000, Y: 9000, To: "R-6"},
		{X: 15000, Y: 1000, To: "R-5"},
	},
	"J-SO": {
		{X: 1000, Y: 1000, To: "R-3"},
		{X: 35000, Y: 21500, To: "E-2"},
	},
	"T-1": {
		{X: 16000, Y: 17000, To: "R-5"},
		{X: 4800, Y: 8500, To: "U-5"},
		{X: 27300, Y: 8500, To: "E-5"},
		{X: 16000, Y: 6000, To: "R-3"},
		{X: 14300, Y: 3000, To: "U-3"},
		{X: 17800, Y: 3000, To: "E-3"},
	},
	"G-1": {
		{X: 20000, Y: 8800, To: "T-1"},
	},
	"E-1": {
		{X: 15000, Y: 1000, To: "E-2"},
		{X: 1000, Y: 1000, To: "E-3"},
	},
	"E-2": {
		{X: 15000, Y: 5000, To: "E-1"},
		{X: 1000, Y: 9000, To: "E-3"},
		{X: 8000, Y: 1000, To: "J-SO"},
	},
	"E-3": {
		{X: 19000, Y: 11500, To: "E-2"},
		{X: 10000, Y: 1000, To: "R-3"},
		{X: 1000, Y: 11500, To: "J-VO"},
		{X: 1000, Y: 1000, To: "T-1"},
	},
	"E-5": {
		{X: 1000, Y: 1000, To: "T-1"},
		{X: 1000, Y: 11500, To: "G-1"},
		{X: 19000, Y: 11500, To: "E-6"},
		{X: 19000, Y: 6300, To: "E-7"},
	},
	"E-6": {
		{X: 1000, Y: 1000, To: "E-5"},
		{X: 29000, Y: 17700, To: "E-7"},
	},
	"E-7": {
		{X: 15000, Y: 1000, To: "E-5"},
		{X: 1000, Y: 1000, To: "E-6"},
	},
	"J-VO": {
		{X: 35000, Y: 21500, To: "E-3"},
		{X: 1000, Y: 1000, To: "U-2"},
	},
	"U-1": {
		{X: 1000, Y: 9000, To: "U-2"},
		{X: 15000, Y: 1000, To: "U-3"},
	},
	"U-2": {
		{X: 15000, Y: 9000, To: "J-VO"},
		{X: 15000, Y: 1000, To: "U-3"},
		{X: 1000, Y: 1000, To: "U-1"},
	},
	"U-3": {
		{X: 1000, Y: 11500, To: "U-2"},
		{X: 19000, Y: 11500, To: "T-1"},
		{X: 19000, Y: 6500, To: "R-3"},
		{X: 19000, Y: 1000, To: "J-VS"},
	},
	"J-VS": {
		{X: 1000, Y: 21500, To: "U-3"},
		{X: 35000, Y: 1000, To: "R-2"},
	},
	"U-5": {
		{X: 19000, Y: 11500, To: "T-1"},
		{X: 1000, Y: 11500, To: "G-1"},
		{X: 1000, Y: 6300, To: "U-6"},
		{X: 1000, Y: 1000, To: "U-7"},
	},
	"U-6": {
		{X: 29000, Y: 9300, To: "U-5"},
		{X: 1000, Y: 9300, To: "U-7"},
	},
	"U-7": {
		{X: 15000, Y: 9000, To: "U-6"},
		{X: 15000, Y: 1000, To: "U-5"},
	},
}

// PvPMaps are maps that should be avoided during escape/recovery.
var PvPMaps = map[string]bool{
	"T-1": true,
	"G-1": true,
}

// BFSPathStep represents one step in a BFS navigation path.
type BFSPathStep struct {
	Map    string
	Portal Portal
}

// FindPath uses BFS to find the shortest portal path from currentMap to destinationMap.
func FindPath(currentMap, destinationMap string) []BFSPathStep {
	type queueItem struct {
		mapName string
		path    []BFSPathStep
	}

	queue := []queueItem{{mapName: currentMap, path: nil}}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		if visited[item.mapName] {
			continue
		}
		visited[item.mapName] = true

		portals, ok := MapRegions[item.mapName]
		if !ok {
			continue
		}

		for _, portal := range portals {
			newPath := make([]BFSPathStep, len(item.path)+1)
			copy(newPath, item.path)
			newPath[len(item.path)] = BFSPathStep{Map: item.mapName, Portal: portal}

			if portal.To == destinationMap {
				return newPath
			}

			queue = append(queue, queueItem{mapName: portal.To, path: newPath})
		}
	}

	return nil
}

// FindClosestSafePortal finds the closest portal that doesn't lead to a PvP zone.
func FindClosestSafePortal(mapName string, x, y int) *Portal {
	portals, ok := MapRegions[mapName]
	if !ok {
		return nil
	}

	var closest *Portal
	shortestDist := math.MaxFloat64

	for i := range portals {
		p := &portals[i]
		if PvPMaps[p.To] {
			continue
		}
		dist := Distance(x, y, p.X, p.Y)
		if dist < shortestDist {
			shortestDist = dist
			closest = p
		}
	}

	return closest
}

// Distance calculates Euclidean distance between two points.
func Distance(x1, y1, x2, y2 int) float64 {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	return math.Sqrt(dx*dx + dy*dy)
}
