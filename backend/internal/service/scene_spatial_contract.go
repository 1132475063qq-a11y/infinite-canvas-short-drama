package service

// This file contains the versioned, provider-independent spatial contract for
// a scene.  It deliberately has no media-generation concerns: image/video
// providers are consumers of a locked spatial model, never authors of it.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

const SceneSpatialContractSchemaVersion = 1

type SceneSpatialGateStatus string

const (
	SceneSpatialGatePass      SceneSpatialGateStatus = "PASS"
	SceneSpatialGateUncertain SceneSpatialGateStatus = "UNCERTAIN"
	SceneSpatialGateFail      SceneSpatialGateStatus = "FAIL"
	SceneSpatialGateNeedsYou  SceneSpatialGateStatus = "NEEDS_YOU"
)

type SceneSpatialIssueSeverity string

const (
	SceneSpatialIssueInfo    SceneSpatialIssueSeverity = "INFO"
	SceneSpatialIssueWarning SceneSpatialIssueSeverity = "WARNING"
	SceneSpatialIssueError   SceneSpatialIssueSeverity = "ERROR"
	SceneSpatialIssueBlocker SceneSpatialIssueSeverity = "BLOCKER"
)

// SpatialPoint is a local logical coordinate. X is left/right, Y is
// front/back and Z is height. The floor-plan origin is explicit so that a
// scene can be exported to different DCCs without changing anchor IDs.
type SpatialPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type SceneManifest struct {
	SceneID            string   `json:"sceneId"`
	SceneName          string   `json:"sceneName,omitempty"`
	ParentLocation     string   `json:"parentLocation,omitempty"`
	InteriorExterior   string   `json:"interiorExterior,omitempty"`
	StoryEvents        []string `json:"storyEvents,omitempty"`
	Characters         []string `json:"characters,omitempty"`
	RequiredProps      []string `json:"requiredProps,omitempty"`
	Entrances          []string `json:"entrances,omitempty"`
	Exits              []string `json:"exits,omitempty"`
	PreviousScene      string   `json:"previousScene,omitempty"`
	NextScene          string   `json:"nextScene,omitempty"`
	ContinuityPriority string   `json:"continuityPriority,omitempty"`
}

type SceneTopologyNode struct {
	ID       string        `json:"id"`
	Kind     string        `json:"kind,omitempty"`
	Label    string        `json:"label,omitempty"`
	Position *SpatialPoint `json:"position,omitempty"`
}

type SceneTopologyEdge struct {
	From           string  `json:"from"`
	To             string  `json:"to"`
	Direction      string  `json:"direction,omitempty"`
	Distance       float64 `json:"distance,omitempty"`
	Visibility     string  `json:"visibility,omitempty"`
	Elevation      string  `json:"elevation,omitempty"`
	Transition     string  `json:"transition,omitempty"`
	ConnectionType string  `json:"connectionType,omitempty"`
	DoorRelation   string  `json:"doorRelation,omitempty"`
	WindowRelation string  `json:"windowRelation,omitempty"`
}

type SceneTopologyGraph struct {
	Nodes []SceneTopologyNode `json:"nodes"`
	Edges []SceneTopologyEdge `json:"edges"`
}

type SpatialZone struct {
	ID             string  `json:"id"`
	TopologyNodeID string  `json:"topologyNodeId,omitempty"`
	Label          string  `json:"label,omitempty"`
	MinX           float64 `json:"minX"`
	MinY           float64 `json:"minY"`
	MaxX           float64 `json:"maxX"`
	MaxY           float64 `json:"maxY"`
	Purpose        string  `json:"purpose,omitempty"`
}

type SpatialOpening struct {
	ID             string       `json:"id"`
	Kind           string       `json:"kind"` // door or window
	Position       SpatialPoint `json:"position"`
	Orientation    string       `json:"orientation,omitempty"`
	State          string       `json:"state,omitempty"`
	ConnectsTo     string       `json:"connectsTo,omitempty"`
	ContinuityLock bool         `json:"continuityLock"`
}

type SpatialFloorPlan struct {
	CoordinateSystem string           `json:"coordinateSystem,omitempty"`
	Origin           SpatialPoint     `json:"origin"`
	Width            float64          `json:"width"`
	Depth            float64          `json:"depth"`
	Zones            []SpatialZone    `json:"zones,omitempty"`
	Openings         []SpatialOpening `json:"openings,omitempty"`
}

type SceneAnchor struct {
	AnchorID       string       `json:"anchorId"`
	SceneID        string       `json:"sceneId,omitempty"`
	ObjectID       string       `json:"objectId"`
	AnchorType     string       `json:"anchorType,omitempty"`
	Position       SpatialPoint `json:"position"`
	Orientation    string       `json:"orientation,omitempty"`
	Mobility       string       `json:"mobility,omitempty"`
	ContinuityLock bool         `json:"continuityLock"`
	Evidence       []string     `json:"evidence,omitempty"`
}

type CameraAnchor struct {
	CameraID            string       `json:"cameraId"`
	SceneID             string       `json:"sceneId,omitempty"`
	TopologyNodeID      string       `json:"topologyNodeId,omitempty"`
	AnchorID            string       `json:"anchorId,omitempty"`
	Position            SpatialPoint `json:"position"`
	Height              float64      `json:"height,omitempty"`
	Direction           string       `json:"direction,omitempty"`
	Facing              string       `json:"facing,omitempty"`
	Target              string       `json:"target,omitempty"`
	LensClass           string       `json:"lensClass,omitempty"`
	ShotSizeRange       []string     `json:"shotSizeRange,omitempty"`
	AllowedFOVRange     []float64    `json:"allowedFovRange,omitempty"`
	MovementConstraints []string     `json:"movementConstraints,omitempty"`
	AxisSide            string       `json:"axisSide,omitempty"`
	ReverseOf           string       `json:"reverseOf,omitempty"`
}

type CameraAnchorPlan struct {
	ActionAxis string         `json:"actionAxis,omitempty"`
	Anchors    []CameraAnchor `json:"anchors"`
}

type ViewpointCoverage struct {
	ViewID                   string   `json:"viewId"`
	CameraAnchorID           string   `json:"cameraAnchorId"`
	Facing                   string   `json:"facing,omitempty"`
	Purpose                  string   `json:"purpose,omitempty"`
	Required                 bool     `json:"required"`
	AssetReference           string   `json:"assetReference,omitempty"`
	RequiredVisibleObjectIDs []string `json:"requiredVisibleObjectIds,omitempty"`
	ForbiddenObjectIDs       []string `json:"forbiddenObjectIds,omitempty"`
}

type ViewpointCoverageMatrix struct {
	Views []ViewpointCoverage `json:"views"`
}

type SceneAssetReference struct {
	AssetID     string `json:"assetId,omitempty"`
	URI         string `json:"uri,omitempty"`
	Role        string `json:"role,omitempty"`
	ViewID      string `json:"viewId,omitempty"`
	EvidenceRef string `json:"evidenceRef,omitempty"`
}

type SceneLookCard struct {
	LookFamilyID        string   `json:"lookFamilyId,omitempty"`
	InheritsFrom        string   `json:"inheritsFrom,omitempty"`
	BasePalette         []string `json:"basePalette,omitempty"`
	AccentPalette       []string `json:"accentPalette,omitempty"`
	ShadowPalette       []string `json:"shadowPalette,omitempty"`
	NeonPalette         []string `json:"neonPalette,omitempty"`
	LightingDirection   string   `json:"lightingDirection,omitempty"`
	LightingTemperature string   `json:"lightingTemperature,omitempty"`
	ContrastLevel       string   `json:"contrastLevel,omitempty"`
	TimeOfDay           string   `json:"timeOfDay,omitempty"`
	Weather             string   `json:"weather,omitempty"`
	MaterialRules       []string `json:"materialRules,omitempty"`
	RenderingRules      []string `json:"renderingRules,omitempty"`
	ForbiddenRendering  []string `json:"forbiddenRendering,omitempty"`
	ContinuityLock      bool     `json:"continuityLock"`
}

type SceneAuthorityConflict struct {
	Field       string   `json:"field"`
	Sources     []string `json:"sources,omitempty"`
	Description string   `json:"description,omitempty"`
}

type SceneSpatialEvidence struct {
	ID        string `json:"id"`
	Kind      string `json:"kind,omitempty"`
	SourceRef string `json:"sourceRef,omitempty"`
	Verified  bool   `json:"verified"`
}

type ScenePathRequirement struct {
	ID      string   `json:"id"`
	From    string   `json:"from"`
	To      string   `json:"to"`
	Via     []string `json:"via,omitempty"`
	Purpose string   `json:"purpose,omitempty"`
}

// SceneAssetPack is the single spatial source consumed by camera planning and
// prompt compilation. Components are pointers so an omitted component can be
// distinguished from an intentionally empty component and reported as
// UNCERTAIN instead of silently passing.
type SceneAssetPack struct {
	SchemaVersion            int                      `json:"schemaVersion"`
	SceneID                  string                   `json:"sceneId"`
	SceneManifestRef         string                   `json:"sceneManifestRef,omitempty"`
	SceneManifest            *SceneManifest           `json:"sceneManifest,omitempty"`
	SceneTopologyGraph       *SceneTopologyGraph      `json:"sceneTopologyGraph,omitempty"`
	SpatialFloorPlan         *SpatialFloorPlan        `json:"spatialFloorPlan,omitempty"`
	FixedAnchors             []SceneAnchor            `json:"fixedAnchors,omitempty"`
	MovableAnchors           []SceneAnchor            `json:"movableAnchors,omitempty"`
	CameraAnchorPlan         *CameraAnchorPlan        `json:"cameraAnchorPlan,omitempty"`
	ViewpointCoverageMatrix  *ViewpointCoverageMatrix `json:"viewpointCoverageMatrix,omitempty"`
	MasterSceneAsset         *SceneAssetReference     `json:"masterSceneAsset,omitempty"`
	ViewAssets               []SceneAssetReference    `json:"viewAssets,omitempty"`
	InteriorLookCard         *SceneLookCard           `json:"interiorLookCard,omitempty"`
	ExteriorLookCard         *SceneLookCard           `json:"exteriorLookCard,omitempty"`
	ForbiddenChanges         []string                 `json:"forbiddenChanges,omitempty"`
	AuthorityConflicts       []SceneAuthorityConflict `json:"authorityConflicts,omitempty"`
	Evidence                 []SceneSpatialEvidence   `json:"evidence,omitempty"`
	RequiredPaths            []ScenePathRequirement   `json:"requiredPaths,omitempty"`
	LegacyLocationDefinition bool                     `json:"-"`
}

type SceneSpatialIssue struct {
	Code     string                    `json:"code"`
	Severity SceneSpatialIssueSeverity `json:"severity"`
	Path     string                    `json:"path,omitempty"`
	Message  string                    `json:"message"`
}

type SceneSpatialGate struct {
	SchemaVersion       int                    `json:"schemaVersion"`
	SceneID             string                 `json:"sceneId"`
	PackArtifactID      string                 `json:"packArtifactId,omitempty"`
	PackArtifactVersion int                    `json:"packArtifactVersion,omitempty"`
	PackVersion         int                    `json:"packVersion,omitempty"`
	Status              SceneSpatialGateStatus `json:"status"`
	Issues              []SceneSpatialIssue    `json:"issues"`
	ValidatedAt         time.Time              `json:"validatedAt"`
	Validator           string                 `json:"validator"`
	Legacy              bool                   `json:"legacy"`
}

// ValidateSceneAssetPack validates one candidate pack. A baseline may be
// supplied when a new version is created; continuity-locked fields are then
// compared deterministically against it.
func ValidateSceneAssetPack(pack SceneAssetPack, baseline *SceneAssetPack) SceneSpatialGate {
	gate := SceneSpatialGate{
		SchemaVersion: SceneSpatialContractSchemaVersion,
		SceneID:       strings.TrimSpace(pack.SceneID),
		Status:        SceneSpatialGatePass,
		Issues:        make([]SceneSpatialIssue, 0),
		ValidatedAt:   time.Now().UTC(),
		Validator:     "scene-spatial-gate/1",
	}
	add := func(code string, severity SceneSpatialIssueSeverity, path string, message string, status SceneSpatialGateStatus) {
		gate.Issues = append(gate.Issues, SceneSpatialIssue{Code: code, Severity: severity, Path: path, Message: message})
		gate.Status = worseSceneSpatialStatus(gate.Status, status)
	}

	if pack.SchemaVersion == 0 {
		add("SCENE_SCHEMA_UNSUPPORTED", SceneSpatialIssueWarning, "schemaVersion", "缺少场景空间合同版本", SceneSpatialGateUncertain)
	} else if pack.SchemaVersion != SceneSpatialContractSchemaVersion {
		add("SCENE_SCHEMA_UNSUPPORTED", SceneSpatialIssueError, "schemaVersion", "场景空间合同版本不受支持", SceneSpatialGateFail)
	}
	if strings.TrimSpace(pack.SceneID) == "" {
		add("SCENE_MANIFEST_INCOMPLETE", SceneSpatialIssueBlocker, "sceneId", "缺少 sceneId", SceneSpatialGateUncertain)
	}
	if pack.SceneManifest == nil || strings.TrimSpace(pack.SceneManifest.SceneID) == "" {
		add("SCENE_MANIFEST_INCOMPLETE", SceneSpatialIssueBlocker, "sceneManifest", "缺少包含 sceneId 的 Scene Manifest", SceneSpatialGateUncertain)
	} else if pack.SceneManifest.SceneID != pack.SceneID {
		add("SCENE_MANIFEST_INCOMPLETE", SceneSpatialIssueError, "sceneManifest.sceneId", "Scene Manifest 与场景 ID 不一致", SceneSpatialGateFail)
	} else if len(pack.SceneManifest.StoryEvents) == 0 {
		add("SCENE_MANIFEST_INCOMPLETE", SceneSpatialIssueWarning, "sceneManifest.storyEvents", "Scene Manifest 没有绑定剧情事件", SceneSpatialGateUncertain)
	}
	if pack.SceneManifest != nil {
		switch strings.ToLower(strings.TrimSpace(pack.SceneManifest.InteriorExterior)) {
		case "interior", "exterior", "mixed":
		default:
			add("SCENE_MANIFEST_INCOMPLETE", SceneSpatialIssueWarning, "sceneManifest.interiorExterior", "Scene Manifest 必须声明 interior、exterior 或 mixed", SceneSpatialGateUncertain)
		}
	}
	if pack.SceneTopologyGraph == nil {
		add("TOPOLOGY_NODE_MISSING", SceneSpatialIssueBlocker, "sceneTopologyGraph", "缺少 SceneTopologyGraph", SceneSpatialGateUncertain)
	} else {
		validateTopology(*pack.SceneTopologyGraph, add)
	}
	if pack.SpatialFloorPlan == nil {
		add("FLOORPLAN_ANCHOR_OUT_OF_BOUNDS", SceneSpatialIssueBlocker, "spatialFloorPlan", "缺少 SpatialFloorPlan", SceneSpatialGateUncertain)
	} else {
		validateFloorPlan(*pack.SpatialFloorPlan, pack, add)
	}
	if len(pack.FixedAnchors) == 0 {
		add("FLOORPLAN_ANCHOR_OUT_OF_BOUNDS", SceneSpatialIssueWarning, "fixedAnchors", "没有声明固定空间锚点；无法锁定墙、门或固定道具", SceneSpatialGateUncertain)
	}
	validateAnchors(pack.FixedAnchors, "fixedAnchors", pack.SceneID, pack.SpatialFloorPlan, add)
	validateAnchors(pack.MovableAnchors, "movableAnchors", pack.SceneID, pack.SpatialFloorPlan, add)
	validateAnchorSets(pack.FixedAnchors, pack.MovableAnchors, add)
	if pack.CameraAnchorPlan == nil || len(pack.CameraAnchorPlan.Anchors) == 0 {
		add("CAMERA_ANCHOR_UNKNOWN", SceneSpatialIssueBlocker, "cameraAnchorPlan", "缺少可拍机位锚点", SceneSpatialGateUncertain)
	} else {
		validateCameraPlan(*pack.CameraAnchorPlan, pack, add)
	}
	if pack.ViewpointCoverageMatrix == nil || len(pack.ViewpointCoverageMatrix.Views) == 0 {
		add("VIEW_COVERAGE_MISSING", SceneSpatialIssueBlocker, "viewpointCoverageMatrix", "缺少镜头需求驱动的视角覆盖矩阵", SceneSpatialGateUncertain)
	} else {
		validateCoverage(*pack.ViewpointCoverageMatrix, pack, add)
	}
	if pack.MasterSceneAsset == nil || (strings.TrimSpace(pack.MasterSceneAsset.AssetID) == "" && strings.TrimSpace(pack.MasterSceneAsset.URI) == "") {
		add("SCENE_MANIFEST_INCOMPLETE", SceneSpatialIssueWarning, "masterSceneAsset", "缺少可追溯的主场景资产证据", SceneSpatialGateUncertain)
	}
	validateLookCards(pack, add)
	validateEvidence(pack, add)
	if len(pack.AuthorityConflicts) > 0 {
		for index, conflict := range pack.AuthorityConflicts {
			add("AUTHORITY_CONFLICT", SceneSpatialIssueBlocker, fmt.Sprintf("authorityConflicts[%d]", index), conflict.DescriptionOrDefault(), SceneSpatialGateNeedsYou)
		}
	}
	validateRequiredPaths(pack, add)
	if baseline != nil {
		validateContinuityBaseline(*baseline, pack, add)
	}
	if pack.LegacyLocationDefinitionPresent() {
		add("LEGACY_LOCATION_DEFINITION", SceneSpatialIssueWarning, "", "检测到旧版自由文本 Location Definition；当前包可读取但尚未完成空间验证", SceneSpatialGateUncertain)
		gate.Legacy = true
	}

	sort.SliceStable(gate.Issues, func(i, j int) bool {
		if gate.Issues[i].Code != gate.Issues[j].Code {
			return gate.Issues[i].Code < gate.Issues[j].Code
		}
		return gate.Issues[i].Path < gate.Issues[j].Path
	})
	return gate
}

// LegacyLocationDefinitionPresent is intentionally conservative. It lets a
// migration/read path identify old free-text location payloads without
// treating those fields as a valid spatial model.
func (p SceneAssetPack) LegacyLocationDefinitionPresent() bool {
	return p.LegacyLocationDefinition
}

func DecodeSceneAssetPackPayload(payload []byte) (SceneAssetPack, error) {
	var pack SceneAssetPack
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&pack); err != nil {
		return SceneAssetPack{}, fmt.Errorf("scene asset pack payload: %w", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err == nil {
		for _, key := range []string{"canonicalGeometry", "doors", "windows", "anchors", "fixedProps", "movableProps", "lightingLogic", "masterView", "reverseViews"} {
			if _, exists := raw[key]; exists {
				pack.LegacyLocationDefinition = true
				break
			}
		}
	}
	return pack, nil
}

func EncodeSceneAssetPackPayload(pack SceneAssetPack) ([]byte, error) {
	if pack.SchemaVersion == 0 {
		pack.SchemaVersion = SceneSpatialContractSchemaVersion
	}
	return json.Marshal(pack)
}

func validateTopology(graph SceneTopologyGraph, add func(string, SceneSpatialIssueSeverity, string, string, SceneSpatialGateStatus)) {
	if len(graph.Nodes) == 0 {
		add("TOPOLOGY_NODE_MISSING", SceneSpatialIssueBlocker, "sceneTopologyGraph.nodes", "拓扑图没有节点", SceneSpatialGateUncertain)
		return
	}
	nodes := make(map[string]SceneTopologyNode, len(graph.Nodes))
	for i, node := range graph.Nodes {
		id := strings.TrimSpace(node.ID)
		if id == "" {
			add("TOPOLOGY_NODE_MISSING", SceneSpatialIssueError, fmt.Sprintf("sceneTopologyGraph.nodes[%d].id", i), "拓扑节点缺少 ID", SceneSpatialGateFail)
			continue
		}
		if _, exists := nodes[id]; exists {
			add("TOPOLOGY_NODE_MISSING", SceneSpatialIssueError, "sceneTopologyGraph.nodes", "拓扑节点 ID 重复", SceneSpatialGateFail)
			continue
		}
		nodes[id] = node
	}
	for i, edge := range graph.Edges {
		if _, ok := nodes[strings.TrimSpace(edge.From)]; !ok || edge.From == "" {
			add("TOPOLOGY_EDGE_INVALID", SceneSpatialIssueError, fmt.Sprintf("sceneTopologyGraph.edges[%d].from", i), "拓扑边的起点不存在", SceneSpatialGateFail)
		}
		if _, ok := nodes[strings.TrimSpace(edge.To)]; !ok || edge.To == "" {
			add("TOPOLOGY_EDGE_INVALID", SceneSpatialIssueError, fmt.Sprintf("sceneTopologyGraph.edges[%d].to", i), "拓扑边的终点不存在", SceneSpatialGateFail)
		}
		if edge.Distance < 0 || math.IsNaN(edge.Distance) || math.IsInf(edge.Distance, 0) {
			add("TOPOLOGY_EDGE_INVALID", SceneSpatialIssueError, fmt.Sprintf("sceneTopologyGraph.edges[%d].distance", i), "拓扑边距离必须是非负有限数", SceneSpatialGateFail)
		}
	}
	if len(graph.Nodes) > 1 && len(graph.Edges) == 0 {
		add("TOPOLOGY_EDGE_INVALID", SceneSpatialIssueWarning, "sceneTopologyGraph.edges", "多个拓扑节点之间没有声明空间关系", SceneSpatialGateUncertain)
	}
}

func validateFloorPlan(plan SpatialFloorPlan, pack SceneAssetPack, add func(string, SceneSpatialIssueSeverity, string, string, SceneSpatialGateStatus)) {
	if plan.Width <= 0 || plan.Depth <= 0 || math.IsNaN(plan.Width) || math.IsNaN(plan.Depth) || math.IsInf(plan.Width, 0) || math.IsInf(plan.Depth, 0) {
		add("FLOORPLAN_ANCHOR_OUT_OF_BOUNDS", SceneSpatialIssueError, "spatialFloorPlan", "平面图 width/depth 必须为正数", SceneSpatialGateFail)
		return
	}
	topologyNodes := make(map[string]struct{})
	if pack.SceneTopologyGraph != nil {
		for _, node := range pack.SceneTopologyGraph.Nodes {
			if id := strings.TrimSpace(node.ID); id != "" {
				topologyNodes[id] = struct{}{}
			}
		}
	}
	zoneIDs := make(map[string]struct{}, len(plan.Zones))
	for i, zone := range plan.Zones {
		zoneID := strings.TrimSpace(zone.ID)
		if zoneID == "" || zone.MaxX <= zone.MinX || zone.MaxY <= zone.MinY || zone.MinX < 0 || zone.MinY < 0 || zone.MaxX > plan.Width || zone.MaxY > plan.Depth {
			add("FLOORPLAN_ANCHOR_OUT_OF_BOUNDS", SceneSpatialIssueError, fmt.Sprintf("spatialFloorPlan.zones[%d]", i), "功能区超出平面图边界或几何无效", SceneSpatialGateFail)
		}
		if _, exists := zoneIDs[zoneID]; zoneID != "" && exists {
			add("FLOORPLAN_ANCHOR_OUT_OF_BOUNDS", SceneSpatialIssueError, fmt.Sprintf("spatialFloorPlan.zones[%d].id", i), "平面图功能区 ID 重复", SceneSpatialGateFail)
		}
		zoneIDs[zoneID] = struct{}{}
		if topologyNodeID := strings.TrimSpace(zone.TopologyNodeID); topologyNodeID != "" {
			if _, exists := topologyNodes[topologyNodeID]; !exists {
				add("SCENE_TOPOLOGY_CONFLICT", SceneSpatialIssueError, fmt.Sprintf("spatialFloorPlan.zones[%d].topologyNodeId", i), "平面图功能区引用了不存在的拓扑节点", SceneSpatialGateFail)
			}
		}
	}
	openingIDs := make(map[string]struct{}, len(plan.Openings))
	for i, opening := range plan.Openings {
		openingID := strings.TrimSpace(opening.ID)
		if openingID == "" || (opening.Kind != "door" && opening.Kind != "window") || !pointInFloorPlan(opening.Position, plan) {
			add("FLOORPLAN_ANCHOR_OUT_OF_BOUNDS", SceneSpatialIssueError, fmt.Sprintf("spatialFloorPlan.openings[%d]", i), "门窗锚点无效或超出平面图边界", SceneSpatialGateFail)
		}
		if _, exists := openingIDs[openingID]; openingID != "" && exists {
			add("DOOR_WINDOW_RELATION_CONFLICT", SceneSpatialIssueError, fmt.Sprintf("spatialFloorPlan.openings[%d].id", i), "平面图门窗 ID 重复", SceneSpatialGateFail)
		}
		openingIDs[openingID] = struct{}{}
		connectsTo := strings.TrimSpace(opening.ConnectsTo)
		if connectsTo == "" {
			continue
		}
		if _, exists := topologyNodes[connectsTo]; !exists {
			add("DOOR_WINDOW_RELATION_CONFLICT", SceneSpatialIssueError, fmt.Sprintf("spatialFloorPlan.openings[%d].connectsTo", i), "门窗连接了不存在的拓扑节点", SceneSpatialGateFail)
			continue
		}
		if _, openingIsNode := topologyNodes[openingID]; !openingIsNode {
			add("DOOR_WINDOW_RELATION_CONFLICT", SceneSpatialIssueWarning, fmt.Sprintf("spatialFloorPlan.openings[%d].id", i), "门窗没有同 ID 的拓扑节点，无法证明室内外连接关系", SceneSpatialGateUncertain)
			continue
		}
		if pack.SceneTopologyGraph == nil || !topologyConnectionExists(*pack.SceneTopologyGraph, openingID, connectsTo) {
			add("DOOR_WINDOW_RELATION_CONFLICT", SceneSpatialIssueError, fmt.Sprintf("spatialFloorPlan.openings[%d].connectsTo", i), "门窗关系与场景拓扑边不一致", SceneSpatialGateFail)
		}
	}
}

func validateAnchors(anchors []SceneAnchor, field string, sceneID string, plan *SpatialFloorPlan, add func(string, SceneSpatialIssueSeverity, string, string, SceneSpatialGateStatus)) {
	seen := make(map[string]struct{}, len(anchors))
	for i, anchor := range anchors {
		path := fmt.Sprintf("%s[%d]", field, i)
		id := strings.TrimSpace(anchor.AnchorID)
		if id == "" || strings.TrimSpace(anchor.ObjectID) == "" {
			add("FLOORPLAN_ANCHOR_OUT_OF_BOUNDS", SceneSpatialIssueError, path, "空间锚点必须包含 anchorId 和 objectId", SceneSpatialGateFail)
			continue
		}
		if _, exists := seen[id]; exists {
			add("FLOORPLAN_ANCHOR_OUT_OF_BOUNDS", SceneSpatialIssueError, path, "空间锚点 ID 重复", SceneSpatialGateFail)
		}
		seen[id] = struct{}{}
		if anchor.SceneID != "" && anchor.SceneID != sceneID {
			add("FLOORPLAN_ANCHOR_OUT_OF_BOUNDS", SceneSpatialIssueError, path+".sceneId", "空间锚点属于其他场景", SceneSpatialGateFail)
		}
		if field == "fixedAnchors" && !anchor.ContinuityLock {
			add("FIXED_ANCHOR_DRIFT", SceneSpatialIssueWarning, path+".continuityLock", "固定锚点必须开启连续性锁，才能阻止后续版本静默漂移", SceneSpatialGateUncertain)
		}
		if plan != nil && !pointInFloorPlan(anchor.Position, *plan) {
			add("FLOORPLAN_ANCHOR_OUT_OF_BOUNDS", SceneSpatialIssueError, path+".position", "空间锚点超出平面图边界", SceneSpatialGateFail)
		}
	}
}

func validateAnchorSets(fixed []SceneAnchor, movable []SceneAnchor, add func(string, SceneSpatialIssueSeverity, string, string, SceneSpatialGateStatus)) {
	fixedIDs := make(map[string]struct{}, len(fixed))
	for _, anchor := range fixed {
		fixedIDs[anchor.AnchorID] = struct{}{}
	}
	for index, anchor := range movable {
		if _, exists := fixedIDs[anchor.AnchorID]; exists && anchor.AnchorID != "" {
			add("FIXED_ANCHOR_DRIFT", SceneSpatialIssueError, fmt.Sprintf("movableAnchors[%d].anchorId", index), "同一锚点不能同时声明为固定和可移动", SceneSpatialGateFail)
		}
	}
}

func validateCameraPlan(plan CameraAnchorPlan, pack SceneAssetPack, add func(string, SceneSpatialIssueSeverity, string, string, SceneSpatialGateStatus)) {
	seen := make(map[string]struct{}, len(plan.Anchors))
	camerasByID := make(map[string]CameraAnchor, len(plan.Anchors))
	anchorIDs := make(map[string]struct{}, len(pack.FixedAnchors)+len(pack.MovableAnchors))
	for _, anchor := range append(append([]SceneAnchor{}, pack.FixedAnchors...), pack.MovableAnchors...) {
		anchorIDs[anchor.AnchorID] = struct{}{}
	}
	topologyNodes := make(map[string]struct{})
	if pack.SceneTopologyGraph != nil {
		for _, node := range pack.SceneTopologyGraph.Nodes {
			topologyNodes[node.ID] = struct{}{}
		}
	}
	zonesByTopologyNode := make(map[string][]SpatialZone)
	if pack.SpatialFloorPlan != nil {
		for _, zone := range pack.SpatialFloorPlan.Zones {
			if zone.TopologyNodeID != "" {
				zonesByTopologyNode[zone.TopologyNodeID] = append(zonesByTopologyNode[zone.TopologyNodeID], zone)
			}
		}
	}
	for _, camera := range plan.Anchors {
		if camera.CameraID != "" {
			camerasByID[camera.CameraID] = camera
		}
	}
	for i, camera := range plan.Anchors {
		path := fmt.Sprintf("cameraAnchorPlan.anchors[%d]", i)
		if strings.TrimSpace(camera.CameraID) == "" {
			add("CAMERA_ANCHOR_UNKNOWN", SceneSpatialIssueError, path, "机位缺少 cameraId", SceneSpatialGateFail)
			continue
		}
		if _, exists := seen[camera.CameraID]; exists {
			add("CAMERA_ANCHOR_UNKNOWN", SceneSpatialIssueError, path, "cameraId 重复", SceneSpatialGateFail)
		}
		seen[camera.CameraID] = struct{}{}
		if strings.TrimSpace(camera.SceneID) == "" {
			add("CAMERA_ANCHOR_UNKNOWN", SceneSpatialIssueWarning, path+".sceneId", "机位没有声明所属场景，无法形成完整空间来源", SceneSpatialGateUncertain)
		} else if camera.SceneID != pack.SceneID {
			add("CAMERA_ANCHOR_UNKNOWN", SceneSpatialIssueError, path+".sceneId", "机位属于其他场景", SceneSpatialGateFail)
		}
		topologyNodeID := strings.TrimSpace(camera.TopologyNodeID)
		if topologyNodeID == "" {
			add("CAMERA_ANCHOR_UNKNOWN", SceneSpatialIssueWarning, path+".topologyNodeId", "机位没有绑定拓扑节点，无法验证该视角所属空间", SceneSpatialGateUncertain)
		} else if _, exists := topologyNodes[topologyNodeID]; !exists {
			add("SCENE_TOPOLOGY_CONFLICT", SceneSpatialIssueError, path+".topologyNodeId", "机位引用了不存在的拓扑节点", SceneSpatialGateFail)
		} else if zones := zonesByTopologyNode[topologyNodeID]; len(zones) > 0 && !pointInAnyZone(camera.Position, zones) {
			add("CAMERA_GEOMETRY_CONFLICT", SceneSpatialIssueError, path+".position", "机位坐标不在其拓扑节点对应的平面区域内", SceneSpatialGateFail)
		}
		if camera.AnchorID != "" {
			if _, exists := anchorIDs[camera.AnchorID]; !exists {
				add("CAMERA_ANCHOR_UNKNOWN", SceneSpatialIssueError, path+".anchorId", "机位引用了不存在的空间锚点", SceneSpatialGateFail)
			}
		}
		if pack.SpatialFloorPlan != nil && !pointInFloorPlan(camera.Position, *pack.SpatialFloorPlan) {
			add("CAMERA_ANCHOR_UNKNOWN", SceneSpatialIssueError, path+".position", "机位超出平面图边界", SceneSpatialGateFail)
		}
		if len(camera.AllowedFOVRange) > 0 {
			if len(camera.AllowedFOVRange) != 2 || camera.AllowedFOVRange[0] <= 0 || camera.AllowedFOVRange[1] < camera.AllowedFOVRange[0] || math.IsNaN(camera.AllowedFOVRange[0]) || math.IsNaN(camera.AllowedFOVRange[1]) || math.IsInf(camera.AllowedFOVRange[0], 0) || math.IsInf(camera.AllowedFOVRange[1], 0) {
				add("CAMERA_ANCHOR_UNKNOWN", SceneSpatialIssueError, path+".allowedFovRange", "机位允许视场范围必须是正数且按升序排列的两个有限值", SceneSpatialGateFail)
			}
		}
		if camera.ReverseOf != "" {
			// A reverse camera may be declared later; the complete pair is
			// checked in the second pass below.
		}
	}
	for i, camera := range plan.Anchors {
		if camera.ReverseOf == "" {
			continue
		}
		candidate, found := camerasByID[camera.ReverseOf]
		if !found {
			add("REVERSE_CAMERA_CONFLICT", SceneSpatialIssueError, fmt.Sprintf("cameraAnchorPlan.anchors[%d].reverseOf", i), "反打机位引用不存在", SceneSpatialGateFail)
			continue
		}
		if camera.ReverseOf == camera.CameraID {
			add("REVERSE_CAMERA_CONFLICT", SceneSpatialIssueError, fmt.Sprintf("cameraAnchorPlan.anchors[%d].reverseOf", i), "机位不能把自身声明为反打", SceneSpatialGateFail)
			continue
		}
		if candidate.ReverseOf != camera.CameraID {
			add("REVERSE_CAMERA_CONFLICT", SceneSpatialIssueError, fmt.Sprintf("cameraAnchorPlan.anchors[%d].reverseOf", i), "正反打关系必须由两侧机位互相声明", SceneSpatialGateFail)
		}
		if strings.TrimSpace(plan.ActionAxis) == "" || strings.TrimSpace(camera.AxisSide) == "" || strings.TrimSpace(candidate.AxisSide) == "" {
			add("REVERSE_CAMERA_CONFLICT", SceneSpatialIssueWarning, fmt.Sprintf("cameraAnchorPlan.anchors[%d].axisSide", i), "正反打机位缺少动作轴或轴线侧，无法验证左右关系", SceneSpatialGateUncertain)
		} else if camera.AxisSide != candidate.AxisSide {
			add("REVERSE_CAMERA_CONFLICT", SceneSpatialIssueError, fmt.Sprintf("cameraAnchorPlan.anchors[%d].axisSide", i), "正反打机位跨越动作轴，左右关系会翻转", SceneSpatialGateFail)
		}
		if opposite := oppositeCardinalDirection(camera.Direction); opposite != "" && strings.ToLower(strings.TrimSpace(candidate.Direction)) != opposite {
			add("REVERSE_CAMERA_CONFLICT", SceneSpatialIssueError, fmt.Sprintf("cameraAnchorPlan.anchors[%d].direction", i), "正反打机位朝向不是相反方向", SceneSpatialGateFail)
		}
	}
}

func validateCoverage(matrix ViewpointCoverageMatrix, pack SceneAssetPack, add func(string, SceneSpatialIssueSeverity, string, string, SceneSpatialGateStatus)) {
	cameraCount := 0
	if pack.CameraAnchorPlan != nil {
		cameraCount = len(pack.CameraAnchorPlan.Anchors)
	}
	cameras := make(map[string]struct{}, cameraCount)
	if pack.CameraAnchorPlan != nil {
		for _, camera := range pack.CameraAnchorPlan.Anchors {
			cameras[camera.CameraID] = struct{}{}
		}
	}
	assets := make(map[string]SceneAssetReference, len(pack.ViewAssets)+1)
	registerAsset := func(asset SceneAssetReference) {
		if asset.AssetID != "" {
			assets[asset.AssetID] = asset
		}
		if asset.URI != "" {
			assets[asset.URI] = asset
		}
	}
	if pack.MasterSceneAsset != nil {
		registerAsset(*pack.MasterSceneAsset)
	}
	for _, asset := range pack.ViewAssets {
		registerAsset(asset)
	}
	objects := make(map[string]struct{}, len(pack.FixedAnchors)+len(pack.MovableAnchors))
	for _, anchor := range append(append([]SceneAnchor{}, pack.FixedAnchors...), pack.MovableAnchors...) {
		if anchor.ObjectID != "" {
			objects[anchor.ObjectID] = struct{}{}
		}
	}
	seen := make(map[string]struct{}, len(matrix.Views))
	for i, view := range matrix.Views {
		path := fmt.Sprintf("viewpointCoverageMatrix.views[%d]", i)
		if strings.TrimSpace(view.ViewID) == "" || strings.TrimSpace(view.CameraAnchorID) == "" {
			add("VIEW_COVERAGE_MISSING", SceneSpatialIssueError, path, "视角覆盖必须包含 viewId 和 cameraAnchorId", SceneSpatialGateFail)
			continue
		}
		if _, exists := seen[view.ViewID]; exists {
			add("VIEW_COVERAGE_MISSING", SceneSpatialIssueError, path, "viewId 重复", SceneSpatialGateFail)
		}
		seen[view.ViewID] = struct{}{}
		if _, exists := cameras[view.CameraAnchorID]; !exists {
			add("CAMERA_ANCHOR_UNKNOWN", SceneSpatialIssueError, path+".cameraAnchorId", "视角覆盖引用了未知机位", SceneSpatialGateFail)
		}
		if view.Required && strings.TrimSpace(view.Purpose) == "" {
			add("VIEW_COVERAGE_MISSING", SceneSpatialIssueWarning, path+".purpose", "必需视角没有说明剧情用途", SceneSpatialGateUncertain)
		}
		if view.Required && strings.TrimSpace(view.AssetReference) == "" {
			add("VIEW_COVERAGE_MISSING", SceneSpatialIssueWarning, path+".assetReference", "必需视角尚未绑定视角资产", SceneSpatialGateUncertain)
		} else if assetReference := strings.TrimSpace(view.AssetReference); assetReference != "" {
			asset, exists := assets[assetReference]
			if !exists {
				add("VIEW_COVERAGE_MISSING", SceneSpatialIssueError, path+".assetReference", "视角引用了不存在的场景资产", SceneSpatialGateFail)
			} else if asset.ViewID != "" && asset.ViewID != view.ViewID {
				add("VIEW_COVERAGE_MISSING", SceneSpatialIssueError, path+".assetReference", "视角资产属于其他 viewId", SceneSpatialGateFail)
			}
		}
		requiredObjects := make(map[string]struct{}, len(view.RequiredVisibleObjectIDs))
		for objectIndex, objectID := range view.RequiredVisibleObjectIDs {
			objectID = strings.TrimSpace(objectID)
			requiredObjects[objectID] = struct{}{}
			if _, exists := objects[objectID]; objectID == "" || !exists {
				add("VIEW_COVERAGE_MISSING", SceneSpatialIssueError, fmt.Sprintf("%s.requiredVisibleObjectIds[%d]", path, objectIndex), "必需可见对象没有对应的场景锚点", SceneSpatialGateFail)
			}
		}
		for objectIndex, objectID := range view.ForbiddenObjectIDs {
			objectID = strings.TrimSpace(objectID)
			if _, exists := objects[objectID]; objectID == "" || !exists {
				add("VIEW_COVERAGE_MISSING", SceneSpatialIssueError, fmt.Sprintf("%s.forbiddenObjectIds[%d]", path, objectIndex), "禁止出现对象没有对应的场景锚点", SceneSpatialGateFail)
			}
			if _, required := requiredObjects[objectID]; required && objectID != "" {
				add("VIEW_COVERAGE_MISSING", SceneSpatialIssueError, fmt.Sprintf("%s.forbiddenObjectIds[%d]", path, objectIndex), "同一对象不能同时要求可见和禁止出现", SceneSpatialGateFail)
			}
		}
	}
}

func validateLookCards(pack SceneAssetPack, add func(string, SceneSpatialIssueSeverity, string, string, SceneSpatialGateStatus)) {
	interiorExterior := ""
	if pack.SceneManifest != nil {
		interiorExterior = strings.ToLower(strings.TrimSpace(pack.SceneManifest.InteriorExterior))
	}
	if (interiorExterior == "interior" || interiorExterior == "mixed") && pack.InteriorLookCard == nil {
		add("LOOK_CARD_DRIFT", SceneSpatialIssueWarning, "interiorLookCard", "室内场景缺少独立 SceneLookCard", SceneSpatialGateUncertain)
	}
	if (interiorExterior == "exterior" || interiorExterior == "mixed") && pack.ExteriorLookCard == nil {
		add("LOOK_CARD_DRIFT", SceneSpatialIssueWarning, "exteriorLookCard", "室外场景缺少独立 SceneLookCard", SceneSpatialGateUncertain)
	}
	if interiorExterior == "mixed" && pack.InteriorLookCard != nil && pack.ExteriorLookCard != nil {
		interiorFamily := strings.TrimSpace(pack.InteriorLookCard.LookFamilyID)
		exteriorFamily := strings.TrimSpace(pack.ExteriorLookCard.LookFamilyID)
		if interiorFamily == "" || exteriorFamily == "" {
			add("LOOK_CARD_DRIFT", SceneSpatialIssueWarning, "lookCards.lookFamilyId", "混合内外景的 Look Card 必须声明同一 lookFamilyId", SceneSpatialGateUncertain)
		} else if interiorFamily != exteriorFamily {
			add("LOOK_CARD_DRIFT", SceneSpatialIssueError, "lookCards.lookFamilyId", "室内外 Look Card 不属于同一视觉族", SceneSpatialGateFail)
		}
	}
	for path, card := range map[string]*SceneLookCard{"interiorLookCard": pack.InteriorLookCard, "exteriorLookCard": pack.ExteriorLookCard} {
		if card != nil && (len(card.BasePalette) == 0 || strings.TrimSpace(card.LightingDirection) == "") {
			add("LOOK_CARD_DRIFT", SceneSpatialIssueWarning, path, "SceneLookCard 缺少基础色板或光源方向", SceneSpatialGateUncertain)
		}
		if card == nil || strings.TrimSpace(card.InheritsFrom) == "" {
			continue
		}
		validParent := (path == "interiorLookCard" && card.InheritsFrom == "exteriorLookCard") || (path == "exteriorLookCard" && card.InheritsFrom == "interiorLookCard")
		if !validParent {
			add("LOOK_CARD_DRIFT", SceneSpatialIssueError, path+".inheritsFrom", "Look Card 只能继承同一场景的另一张内/外景 Look Card", SceneSpatialGateFail)
		}
	}
}

func validateEvidence(pack SceneAssetPack, add func(string, SceneSpatialIssueSeverity, string, string, SceneSpatialGateStatus)) {
	evidence := make(map[string]bool, len(pack.Evidence))
	for _, item := range pack.Evidence {
		if item.ID != "" {
			evidence[item.ID] = item.Verified
		}
	}
	refs := make([]SceneAssetReference, 0, len(pack.ViewAssets)+1)
	if pack.MasterSceneAsset != nil {
		refs = append(refs, *pack.MasterSceneAsset)
	}
	refs = append(refs, pack.ViewAssets...)
	for i, ref := range refs {
		if strings.TrimSpace(ref.AssetID) == "" && strings.TrimSpace(ref.URI) == "" {
			add("SCENE_MANIFEST_INCOMPLETE", SceneSpatialIssueWarning, fmt.Sprintf("asset[%d]", i), "场景资产引用缺少 assetId 或 uri", SceneSpatialGateUncertain)
		}
		if ref.EvidenceRef != "" && !evidence[ref.EvidenceRef] {
			add("SCENE_MANIFEST_INCOMPLETE", SceneSpatialIssueWarning, fmt.Sprintf("asset[%d].evidenceRef", i), "场景资产证据不存在或未验证", SceneSpatialGateUncertain)
		} else if ref.EvidenceRef == "" {
			add("SCENE_MANIFEST_INCOMPLETE", SceneSpatialIssueWarning, fmt.Sprintf("asset[%d].evidenceRef", i), "场景资产引用没有绑定验证证据", SceneSpatialGateUncertain)
		}
	}
	for i, item := range pack.Evidence {
		if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.SourceRef) == "" {
			add("SCENE_MANIFEST_INCOMPLETE", SceneSpatialIssueWarning, fmt.Sprintf("evidence[%d]", i), "证据记录缺少 ID 或 sourceRef", SceneSpatialGateUncertain)
		}
	}
}

func validateRequiredPaths(pack SceneAssetPack, add func(string, SceneSpatialIssueSeverity, string, string, SceneSpatialGateStatus)) {
	if len(pack.RequiredPaths) == 0 || pack.SceneTopologyGraph == nil {
		return
	}
	adjacency := make(map[string]map[string]struct{}, len(pack.SceneTopologyGraph.Nodes))
	for _, node := range pack.SceneTopologyGraph.Nodes {
		adjacency[node.ID] = make(map[string]struct{})
	}
	for _, edge := range pack.SceneTopologyGraph.Edges {
		if _, ok := adjacency[edge.From]; ok {
			adjacency[edge.From][edge.To] = struct{}{}
		}
	}
	for i, path := range pack.RequiredPaths {
		if !graphReachable(adjacency, path.From, path.To, path.Via) {
			add("PATH_IMPOSSIBLE", SceneSpatialIssueError, fmt.Sprintf("requiredPaths[%d]", i), "声明的人物/道具路径在拓扑图中不可达", SceneSpatialGateFail)
		}
	}
}
