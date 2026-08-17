package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

func validateContinuityBaseline(baseline SceneAssetPack, current SceneAssetPack, add func(string, SceneSpatialIssueSeverity, string, string, SceneSpatialGateStatus)) {
	if !sameSceneManifestStructure(baseline.SceneManifest, current.SceneManifest) {
		add("SCENE_REDESIGN_DETECTED", SceneSpatialIssueError, "sceneManifest", "场景父级、内外景或出入口发生变化；必须通过明确的变更请求创建新空间基线", SceneSpatialGateFail)
	}
	if !sameSceneTopologyGraph(baseline.SceneTopologyGraph, current.SceneTopologyGraph) {
		add("SCENE_TOPOLOGY_CONFLICT", SceneSpatialIssueError, "sceneTopologyGraph", "场景拓扑节点、方向或相邻空间关系发生变化", SceneSpatialGateFail)
	}
	if !sameSpatialFloorPlan(baseline.SpatialFloorPlan, current.SpatialFloorPlan) {
		add("SCENE_REDESIGN_DETECTED", SceneSpatialIssueError, "spatialFloorPlan", "场景平面图范围、功能区或门窗结构发生变化；必须通过明确的变更请求创建新空间基线", SceneSpatialGateFail)
	}
	if !sameCameraAnchorPlan(baseline.CameraAnchorPlan, current.CameraAnchorPlan) {
		add("CAMERA_GEOMETRY_CONFLICT", SceneSpatialIssueError, "cameraAnchorPlan", "锁定机位的位置、朝向、轴线或反打关系发生变化", SceneSpatialGateFail)
	}
	baseAnchors := lockedAnchorMap(append(append([]SceneAnchor{}, baseline.FixedAnchors...), baseline.MovableAnchors...))
	currentAnchors := lockedAnchorMap(append(append([]SceneAnchor{}, current.FixedAnchors...), current.MovableAnchors...))
	for id, previous := range baseAnchors {
		next, ok := currentAnchors[id]
		if !ok {
			add("FIXED_ANCHOR_DRIFT", SceneSpatialIssueError, "fixedAnchors", "连续性锁定锚点被删除："+id, SceneSpatialGateFail)
			continue
		}
		if !sameSpatialPoint(previous.Position, next.Position) || previous.Orientation != next.Orientation || previous.ObjectID != next.ObjectID || previous.AnchorType != next.AnchorType || previous.Mobility != next.Mobility || previous.SceneID != next.SceneID {
			add("FIXED_ANCHOR_DRIFT", SceneSpatialIssueError, "fixedAnchors."+id, "连续性锁定锚点位置、朝向或对象发生漂移", SceneSpatialGateFail)
		}
	}
	for id := range currentAnchors {
		if _, existed := baseAnchors[id]; !existed {
			add("FIXED_ANCHOR_DRIFT", SceneSpatialIssueError, "fixedAnchors."+id, "新增了连续性锁定锚点；必须通过明确的空间版本变更引入", SceneSpatialGateFail)
		}
	}
	baseOpenings := lockedOpeningMap(baseline.SpatialFloorPlan)
	currentOpenings := lockedOpeningMap(current.SpatialFloorPlan)
	for id, previous := range baseOpenings {
		next, ok := currentOpenings[id]
		if !ok || !sameSpatialPoint(previous.Position, next.Position) || previous.Orientation != next.Orientation || previous.ConnectsTo != next.ConnectsTo {
			add("DOOR_WINDOW_RELATION_CONFLICT", SceneSpatialIssueError, "spatialFloorPlan.openings."+id, "连续性锁定门窗关系发生变化", SceneSpatialGateFail)
		}
	}
	compareLookCard("interiorLookCard", baseline.InteriorLookCard, current.InteriorLookCard, add)
	compareLookCard("exteriorLookCard", baseline.ExteriorLookCard, current.ExteriorLookCard, add)
}

func sameSceneManifestStructure(left *SceneManifest, right *SceneManifest) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.SceneID == right.SceneID &&
		left.ParentLocation == right.ParentLocation &&
		left.InteriorExterior == right.InteriorExterior &&
		sameStringSet(left.Entrances, right.Entrances) &&
		sameStringSet(left.Exits, right.Exits)
}

func sameSceneTopologyGraph(left *SceneTopologyGraph, right *SceneTopologyGraph) bool {
	if left == nil || right == nil {
		return left == right
	}
	leftCopy, rightCopy := *left, *right
	leftCopy.Nodes = append([]SceneTopologyNode(nil), left.Nodes...)
	rightCopy.Nodes = append([]SceneTopologyNode(nil), right.Nodes...)
	leftCopy.Edges = append([]SceneTopologyEdge(nil), left.Edges...)
	rightCopy.Edges = append([]SceneTopologyEdge(nil), right.Edges...)
	sort.Slice(leftCopy.Nodes, func(i, j int) bool { return leftCopy.Nodes[i].ID < leftCopy.Nodes[j].ID })
	sort.Slice(rightCopy.Nodes, func(i, j int) bool { return rightCopy.Nodes[i].ID < rightCopy.Nodes[j].ID })
	sort.Slice(leftCopy.Edges, func(i, j int) bool {
		return topologyEdgeSortKey(leftCopy.Edges[i]) < topologyEdgeSortKey(leftCopy.Edges[j])
	})
	sort.Slice(rightCopy.Edges, func(i, j int) bool {
		return topologyEdgeSortKey(rightCopy.Edges[i]) < topologyEdgeSortKey(rightCopy.Edges[j])
	})
	return sameJSONValue(leftCopy, rightCopy)
}

func topologyEdgeSortKey(edge SceneTopologyEdge) string {
	return strings.Join([]string{
		edge.From,
		edge.To,
		edge.ConnectionType,
		edge.Direction,
		fmt.Sprintf("%.17g", edge.Distance),
		edge.Visibility,
		edge.Elevation,
		edge.Transition,
		edge.DoorRelation,
		edge.WindowRelation,
	}, "\x00")
}

func sameSpatialFloorPlan(left *SpatialFloorPlan, right *SpatialFloorPlan) bool {
	if left == nil || right == nil {
		return left == right
	}
	leftCopy, rightCopy := *left, *right
	leftCopy.Zones = append([]SpatialZone(nil), left.Zones...)
	rightCopy.Zones = append([]SpatialZone(nil), right.Zones...)
	leftCopy.Openings = append([]SpatialOpening(nil), left.Openings...)
	rightCopy.Openings = append([]SpatialOpening(nil), right.Openings...)
	sort.Slice(leftCopy.Zones, func(i, j int) bool { return leftCopy.Zones[i].ID < leftCopy.Zones[j].ID })
	sort.Slice(rightCopy.Zones, func(i, j int) bool { return rightCopy.Zones[i].ID < rightCopy.Zones[j].ID })
	sort.Slice(leftCopy.Openings, func(i, j int) bool { return leftCopy.Openings[i].ID < leftCopy.Openings[j].ID })
	sort.Slice(rightCopy.Openings, func(i, j int) bool { return rightCopy.Openings[i].ID < rightCopy.Openings[j].ID })
	return sameJSONValue(leftCopy, rightCopy)
}

func sameCameraAnchorPlan(left *CameraAnchorPlan, right *CameraAnchorPlan) bool {
	if left == nil || right == nil {
		return left == right
	}
	leftCopy, rightCopy := *left, *right
	leftCopy.Anchors = append([]CameraAnchor(nil), left.Anchors...)
	rightCopy.Anchors = append([]CameraAnchor(nil), right.Anchors...)
	sort.Slice(leftCopy.Anchors, func(i, j int) bool { return leftCopy.Anchors[i].CameraID < leftCopy.Anchors[j].CameraID })
	sort.Slice(rightCopy.Anchors, func(i, j int) bool { return rightCopy.Anchors[i].CameraID < rightCopy.Anchors[j].CameraID })
	return sameJSONValue(leftCopy, rightCopy)
}

func sameJSONValue(left any, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func compareLookCard(path string, previous *SceneLookCard, current *SceneLookCard, add func(string, SceneSpatialIssueSeverity, string, string, SceneSpatialGateStatus)) {
	if previous == nil || !previous.ContinuityLock {
		return
	}
	if current == nil || !sameJSONValue(*previous, *current) {
		add("LOOK_CARD_DRIFT", SceneSpatialIssueError, path, "连续性锁定的色卡或光源规则发生漂移", SceneSpatialGateFail)
	}
}

func lockedAnchorMap(anchors []SceneAnchor) map[string]SceneAnchor {
	result := make(map[string]SceneAnchor)
	for _, anchor := range anchors {
		if anchor.ContinuityLock && anchor.AnchorID != "" {
			result[anchor.AnchorID] = anchor
		}
	}
	return result
}

func lockedOpeningMap(plan *SpatialFloorPlan) map[string]SpatialOpening {
	result := make(map[string]SpatialOpening)
	if plan == nil {
		return result
	}
	for _, opening := range plan.Openings {
		if opening.ContinuityLock && opening.ID != "" {
			result[opening.ID] = opening
		}
	}
	return result
}

func pointInFloorPlan(point SpatialPoint, plan SpatialFloorPlan) bool {
	return point.X >= 0 && point.X <= plan.Width && point.Y >= 0 && point.Y <= plan.Depth && point.Z >= 0 && !math.IsNaN(point.X) && !math.IsNaN(point.Y) && !math.IsNaN(point.Z) && !math.IsInf(point.X, 0) && !math.IsInf(point.Y, 0) && !math.IsInf(point.Z, 0)
}

func pointInAnyZone(point SpatialPoint, zones []SpatialZone) bool {
	for _, zone := range zones {
		if point.X >= zone.MinX && point.X <= zone.MaxX && point.Y >= zone.MinY && point.Y <= zone.MaxY {
			return true
		}
	}
	return false
}

func topologyConnectionExists(graph SceneTopologyGraph, from string, to string) bool {
	for _, edge := range graph.Edges {
		if (edge.From == from && edge.To == to) || (edge.From == to && edge.To == from) {
			return true
		}
	}
	return false
}

func oppositeCardinalDirection(direction string) string {
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "north":
		return "south"
	case "south":
		return "north"
	case "east":
		return "west"
	case "west":
		return "east"
	default:
		return ""
	}
}

func sameSpatialPoint(left SpatialPoint, right SpatialPoint) bool {
	const epsilon = 0.0001
	return math.Abs(left.X-right.X) <= epsilon && math.Abs(left.Y-right.Y) <= epsilon && math.Abs(left.Z-right.Z) <= epsilon
}

func sameStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	return sameStrings(leftCopy, rightCopy)
}

func (c SceneAuthorityConflict) DescriptionOrDefault() string {
	if strings.TrimSpace(c.Description) != "" {
		return c.Description
	}
	if strings.TrimSpace(c.Field) != "" {
		return "权威来源对字段存在冲突：" + c.Field
	}
	return "场景空间合同存在未裁定的权威冲突"
}

func worseSceneSpatialStatus(current SceneSpatialGateStatus, candidate SceneSpatialGateStatus) SceneSpatialGateStatus {
	rank := map[SceneSpatialGateStatus]int{
		SceneSpatialGatePass: 0, SceneSpatialGateUncertain: 1, SceneSpatialGateFail: 2, SceneSpatialGateNeedsYou: 3,
	}
	if rank[candidate] > rank[current] {
		return candidate
	}
	return current
}
