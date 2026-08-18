package service

// This file is the server-owned handoff between a locked scene asset pack and
// a Prompt Pack.  It is a projection, not another persisted Artifact: the
// source of truth remains the versioned scene_asset_pack plus its Gate.

import (
	"encoding/json"
	"fmt"
	"strings"

	"infinite-canvas/backend/internal/model"
)

const lockedSceneAssetPackMarker = "[LOCKED_SCENE_ASSET_PACK]"

// These persisted fields are rebuilt from the exact PASS Pack/Gate. The
// browser may propose camera/view IDs, but the server validates that selection
// and writes back the canonical values with server-owned provenance.
var serverOwnedPromptSpatialKeys = []string{
	"lockedSceneAssetPack",
	"spatialPackArtifactId",
	"spatialPackArtifactVersion",
	"spatialGateArtifactId",
	"spatialGateArtifactVersion",
	"spatialContractSchemaVersion",
	"cameraAnchorId",
	"viewId",
	"spatialBindingStatus",
}

// LockedSceneAssetPackProjection is the read-only, shot-specific view of a
// scene pack.  Keeping the selected camera/view alongside the full immutable
// spatial constraints lets image prompts, video direction and diagnostics
// consume one auditable snapshot without inventing a second Artifact type.
type LockedSceneAssetPackProjection struct {
	SchemaVersion       int                    `json:"schemaVersion"`
	SceneID             string                 `json:"sceneId"`
	ContinuityStatus    SceneSpatialGateStatus `json:"continuityStatus"`
	PackArtifactID      string                 `json:"packArtifactId"`
	PackArtifactVersion int                    `json:"packArtifactVersion"`
	GateArtifactID      string                 `json:"gateArtifactId"`
	GateArtifactVersion int                    `json:"gateArtifactVersion"`
	CameraAnchorID      string                 `json:"cameraAnchorId"`
	ViewID              string                 `json:"viewId"`
	SceneManifest       SceneManifest          `json:"sceneManifest"`
	SceneTopologyGraph  SceneTopologyGraph     `json:"sceneTopologyGraph"`
	SpatialFloorPlan    SpatialFloorPlan       `json:"spatialFloorPlan"`
	FixedAnchors        []SceneAnchor          `json:"fixedAnchors"`
	MovableAnchors      []SceneAnchor          `json:"movableAnchors"`
	CameraAnchor        CameraAnchor           `json:"cameraAnchor"`
	ReverseCamera       *CameraAnchor          `json:"reverseCamera,omitempty"`
	Viewpoint           ViewpointCoverage      `json:"viewpoint"`
	MasterSceneAsset    *SceneAssetReference   `json:"masterSceneAsset,omitempty"`
	ViewAsset           *SceneAssetReference   `json:"viewAsset,omitempty"`
	InteriorLookCard    *SceneLookCard         `json:"interiorLookCard,omitempty"`
	ExteriorLookCard    *SceneLookCard         `json:"exteriorLookCard,omitempty"`
	ForbiddenChanges    []string               `json:"forbiddenChanges"`
	RequiredPaths       []ScenePathRequirement `json:"requiredPaths"`
}

// BuildLockedSceneAssetPackProjection selects one declared view and camera
// from a PASS scene pack. Selection is deterministic: an omitted ID is only
// inferred when there is exactly one unambiguous candidate.
func BuildLockedSceneAssetPackProjection(pack SceneAssetPack, packArtifact model.FilmArtifact, gateArtifact model.FilmArtifact, gate SceneSpatialGate, cameraID string, viewID string) (LockedSceneAssetPackProjection, error) {
	if gate.Status != SceneSpatialGatePass {
		return LockedSceneAssetPackProjection{}, fmt.Errorf("spatial gate is %s", gate.Status)
	}
	if pack.SceneManifest == nil || pack.SceneTopologyGraph == nil || pack.SpatialFloorPlan == nil || pack.CameraAnchorPlan == nil || pack.ViewpointCoverageMatrix == nil {
		return LockedSceneAssetPackProjection{}, fmt.Errorf("scene pack is incomplete")
	}
	if packArtifact.ID == "" || packArtifact.ObjectVersion < 1 || gateArtifact.ID == "" || gateArtifact.ObjectVersion < 1 {
		return LockedSceneAssetPackProjection{}, fmt.Errorf("scene pack provenance is incomplete")
	}
	if gate.SceneID != pack.SceneID || gate.PackArtifactID != packArtifact.ID || gate.PackArtifactVersion != packArtifact.ObjectVersion || gate.PackVersion != packArtifact.ObjectVersion {
		return LockedSceneAssetPackProjection{}, fmt.Errorf("scene gate does not point to the selected pack")
	}

	view, camera, err := selectSceneViewAndCamera(*pack.ViewpointCoverageMatrix, *pack.CameraAnchorPlan, cameraID, viewID)
	if err != nil {
		return LockedSceneAssetPackProjection{}, err
	}
	projection := LockedSceneAssetPackProjection{
		SchemaVersion:       SceneSpatialContractSchemaVersion,
		SceneID:             pack.SceneID,
		ContinuityStatus:    gate.Status,
		PackArtifactID:      packArtifact.ID,
		PackArtifactVersion: packArtifact.ObjectVersion,
		GateArtifactID:      gateArtifact.ID,
		GateArtifactVersion: gateArtifact.ObjectVersion,
		CameraAnchorID:      view.CameraAnchorID,
		ViewID:              view.ViewID,
		SceneManifest:       *pack.SceneManifest,
		SceneTopologyGraph:  *pack.SceneTopologyGraph,
		SpatialFloorPlan:    *pack.SpatialFloorPlan,
		FixedAnchors:        append([]SceneAnchor(nil), pack.FixedAnchors...),
		MovableAnchors:      append([]SceneAnchor(nil), pack.MovableAnchors...),
		CameraAnchor:        camera,
		Viewpoint:           view,
		MasterSceneAsset:    pack.MasterSceneAsset,
		InteriorLookCard:    pack.InteriorLookCard,
		ExteriorLookCard:    pack.ExteriorLookCard,
		ForbiddenChanges:    append([]string(nil), pack.ForbiddenChanges...),
		RequiredPaths:       append([]ScenePathRequirement(nil), pack.RequiredPaths...),
	}
	for index := range pack.CameraAnchorPlan.Anchors {
		candidate := pack.CameraAnchorPlan.Anchors[index]
		if candidate.CameraID == camera.ReverseOf {
			copyOfCandidate := candidate
			projection.ReverseCamera = &copyOfCandidate
			break
		}
	}
	if pack.MasterSceneAsset != nil && sceneAssetMatchesView(*pack.MasterSceneAsset, view) {
		copyOfMasterAsset := *pack.MasterSceneAsset
		projection.ViewAsset = &copyOfMasterAsset
	}
	if projection.ViewAsset == nil {
		for index := range pack.ViewAssets {
			asset := pack.ViewAssets[index]
			if sceneAssetMatchesView(asset, view) {
				copyOfAsset := asset
				projection.ViewAsset = &copyOfAsset
				break
			}
		}
	}
	return projection, nil
}

func sceneAssetMatchesView(asset SceneAssetReference, view ViewpointCoverage) bool {
	assetReference := strings.TrimSpace(view.AssetReference)
	if assetReference != "" {
		return strings.TrimSpace(asset.AssetID) == assetReference || strings.TrimSpace(asset.URI) == assetReference
	}
	return strings.TrimSpace(asset.ViewID) == strings.TrimSpace(view.ViewID)
}

func selectSceneViewAndCamera(matrix ViewpointCoverageMatrix, plan CameraAnchorPlan, cameraID string, viewID string) (ViewpointCoverage, CameraAnchor, error) {
	cameraID = strings.TrimSpace(cameraID)
	viewID = strings.TrimSpace(viewID)
	views := make([]ViewpointCoverage, 0, len(matrix.Views))
	if viewID != "" {
		for _, view := range matrix.Views {
			if view.ViewID == viewID {
				views = append(views, view)
				break
			}
		}
		if len(views) == 0 {
			return ViewpointCoverage{}, CameraAnchor{}, fmt.Errorf("viewId %q is not declared by the scene pack", viewID)
		}
	} else {
		for _, view := range matrix.Views {
			if cameraID == "" || view.CameraAnchorID == cameraID {
				views = append(views, view)
			}
		}
		if len(views) != 1 {
			return ViewpointCoverage{}, CameraAnchor{}, fmt.Errorf("Prompt Pack 必须指定唯一的 viewId（当前有 %d 个候选视角）", len(views))
		}
	}
	view := views[0]
	if cameraID != "" && view.CameraAnchorID != cameraID {
		return ViewpointCoverage{}, CameraAnchor{}, fmt.Errorf("cameraAnchorId 与 viewId 不匹配")
	}
	for _, camera := range plan.Anchors {
		if camera.CameraID == view.CameraAnchorID {
			return view, camera, nil
		}
	}
	return ViewpointCoverage{}, CameraAnchor{}, fmt.Errorf("viewId %q 引用了未知 cameraAnchorId", view.ViewID)
}

// CompileLockedScenePrompt appends a deterministic, JSON-encoded spatial
// contract. JSON avoids lossy prose conversion of coordinates and makes the
// exact handoff inspectable by downstream directors and diagnostics.
func CompileLockedScenePrompt(creativePrompt string, projection LockedSceneAssetPackProjection) (string, error) {
	creativePrompt = strings.TrimSpace(creativePrompt)
	if creativePrompt == "" {
		return "", fmt.Errorf("creative prompt is empty")
	}
	encoded, err := json.Marshal(projection)
	if err != nil {
		return "", fmt.Errorf("locked scene projection: %w", err)
	}
	return creativePrompt + "\n\n" + lockedSceneAssetPackMarker + "\n" +
		"The following scene-space contract is immutable. Preserve every fixed anchor, opening, topology relation, camera axis, and look-card rule; do not redesign the scene.\n" +
		string(encoded), nil
}

func promptCreativeText(payload map[string]any) string {
	if creative := payloadString(payload, "creativePrompt"); creative != "" {
		if marker := strings.Index(creative, lockedSceneAssetPackMarker); marker >= 0 {
			creative = creative[:marker]
		}
		return strings.TrimSpace(creative)
	}
	compiled := payloadString(payload, "compiledPrompt")
	if marker := strings.Index(compiled, lockedSceneAssetPackMarker); marker >= 0 {
		compiled = compiled[:marker]
	}
	return strings.TrimSpace(compiled)
}

func decodeLockedSceneAssetPackProjection(payload map[string]any) (LockedSceneAssetPackProjection, error) {
	value, exists := payload["lockedSceneAssetPack"]
	if !exists || value == nil {
		return LockedSceneAssetPackProjection{}, fmt.Errorf("Generation Request 缺少锁定场景空间投影")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return LockedSceneAssetPackProjection{}, fmt.Errorf("锁定场景空间投影格式无效")
	}
	var projection LockedSceneAssetPackProjection
	if err := json.Unmarshal(encoded, &projection); err != nil {
		return LockedSceneAssetPackProjection{}, fmt.Errorf("锁定场景空间投影格式无效")
	}
	if projection.SchemaVersion != SceneSpatialContractSchemaVersion || strings.TrimSpace(projection.SceneID) == "" || strings.TrimSpace(projection.PackArtifactID) == "" || projection.PackArtifactVersion < 1 || strings.TrimSpace(projection.GateArtifactID) == "" || projection.GateArtifactVersion < 1 || strings.TrimSpace(projection.CameraAnchorID) == "" || strings.TrimSpace(projection.ViewID) == "" {
		return LockedSceneAssetPackProjection{}, fmt.Errorf("锁定场景空间投影缺少不可变来源字段")
	}
	if projection.ContinuityStatus != SceneSpatialGatePass {
		return LockedSceneAssetPackProjection{}, fmt.Errorf("锁定场景空间投影不是 PASS 状态")
	}
	if projection.Viewpoint.ViewID != projection.ViewID || projection.Viewpoint.CameraAnchorID != projection.CameraAnchorID || projection.CameraAnchor.CameraID != projection.CameraAnchorID {
		return LockedSceneAssetPackProjection{}, fmt.Errorf("锁定场景空间投影的视角与机位不一致")
	}
	return projection, nil
}

func spatialPromptFieldKeys() []string {
	return append([]string(nil), serverOwnedPromptSpatialKeys...)
}
