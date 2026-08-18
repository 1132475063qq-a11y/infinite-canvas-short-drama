package service

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestValidateSceneAssetPackBlackMarketClinicPasses(t *testing.T) {
	pack := blackMarketClinicSpatialPack()
	gate := ValidateSceneAssetPack(pack, nil)
	if gate.Status != SceneSpatialGatePass {
		t.Fatalf("black market clinic fixture status = %s, issues = %#v", gate.Status, gate.Issues)
	}
	if len(gate.Issues) != 0 {
		t.Fatalf("valid fixture should have no issues: %#v", gate.Issues)
	}
}

func TestValidateSceneAssetPackDetectsFixedAnchorAndDoorDrift(t *testing.T) {
	baseline := blackMarketClinicSpatialPack()
	current := blackMarketClinicSpatialPack()
	current.FixedAnchors[0].Position.X += 1
	current.SpatialFloorPlan.Openings[0].ConnectsTo = "main-street"
	gate := ValidateSceneAssetPack(current, &baseline)
	if gate.Status != SceneSpatialGateFail {
		t.Fatalf("drift status = %s, issues = %#v", gate.Status, gate.Issues)
	}
	assertSpatialIssueCode(t, gate.Issues, "FIXED_ANCHOR_DRIFT")
	assertSpatialIssueCode(t, gate.Issues, "DOOR_WINDOW_RELATION_CONFLICT")
}

func TestValidateSceneAssetPackDetectsStructuralBaselineDrift(t *testing.T) {
	tests := []struct {
		name      string
		issueCode string
		mutate    func(*SceneAssetPack)
	}{
		{
			name:      "scene manifest",
			issueCode: "SCENE_REDESIGN_DETECTED",
			mutate: func(pack *SceneAssetPack) {
				pack.SceneManifest.Entrances = []string{"service-door"}
			},
		},
		{
			name:      "topology graph",
			issueCode: "SCENE_TOPOLOGY_CONFLICT",
			mutate: func(pack *SceneAssetPack) {
				pack.SceneTopologyGraph.Edges[0].Direction = "west"
			},
		},
		{
			name:      "floor plan",
			issueCode: "SCENE_REDESIGN_DETECTED",
			mutate: func(pack *SceneAssetPack) {
				pack.SpatialFloorPlan.Width++
			},
		},
		{
			name:      "camera anchor plan",
			issueCode: "CAMERA_GEOMETRY_CONFLICT",
			mutate: func(pack *SceneAssetPack) {
				pack.CameraAnchorPlan.Anchors[0].Direction = "north"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			baseline := blackMarketClinicSpatialPack()
			current := blackMarketClinicSpatialPack()
			test.mutate(&current)

			gate := ValidateSceneAssetPack(current, &baseline)
			if gate.Status != SceneSpatialGateFail {
				t.Fatalf("structural drift status = %s, issues = %#v", gate.Status, gate.Issues)
			}
			assertSpatialIssueCode(t, gate.Issues, test.issueCode)
		})
	}
}

func TestValidateSceneAssetPackAllowsStructuralCollectionReordering(t *testing.T) {
	baseline := blackMarketClinicSpatialPack()
	current := blackMarketClinicSpatialPack()

	baseline.SceneManifest.Entrances = []string{"clinic-door", "service-door"}
	baseline.SceneManifest.Exits = []string{"clinic-door", "service-door"}
	current.SceneManifest.Entrances = []string{"service-door", "clinic-door"}
	current.SceneManifest.Exits = []string{"service-door", "clinic-door"}

	extraZone := SpatialZone{ID: "storage-zone", MinX: 7, MinY: 1, MaxX: 10, MaxY: 3, Purpose: "storage"}
	extraOpening := SpatialOpening{ID: "service-window", Kind: "window", Position: SpatialPoint{X: 8, Y: 0}, Orientation: "north"}
	baseline.SpatialFloorPlan.Zones = append(baseline.SpatialFloorPlan.Zones, extraZone)
	baseline.SpatialFloorPlan.Openings = append(baseline.SpatialFloorPlan.Openings, extraOpening)
	current.SpatialFloorPlan.Zones = append(current.SpatialFloorPlan.Zones, extraZone)
	current.SpatialFloorPlan.Openings = append(current.SpatialFloorPlan.Openings, extraOpening)

	slices.Reverse(current.SceneTopologyGraph.Nodes)
	slices.Reverse(current.SceneTopologyGraph.Edges)
	slices.Reverse(current.SpatialFloorPlan.Zones)
	slices.Reverse(current.SpatialFloorPlan.Openings)
	slices.Reverse(current.CameraAnchorPlan.Anchors)

	gate := ValidateSceneAssetPack(current, &baseline)
	if gate.Status != SceneSpatialGatePass {
		t.Fatalf("collection reordering should preserve baseline: status=%s issues=%#v", gate.Status, gate.Issues)
	}
}

func TestBlackMarketClinicFixtureCoversFullSpatialSequence(t *testing.T) {
	pack := blackMarketClinicSpatialPack()
	requiredNodes := []string{"clinic-inside", "clinic-door", "clinic-entrance", "clinic-exterior-wall", "street-corner", "main-street", "right-street-extension"}
	nodes := make(map[string]struct{}, len(pack.SceneTopologyGraph.Nodes))
	for _, node := range pack.SceneTopologyGraph.Nodes {
		nodes[node.ID] = struct{}{}
	}
	for _, nodeID := range requiredNodes {
		if _, exists := nodes[nodeID]; !exists {
			t.Fatalf("black market clinic fixture is missing topology node %q", nodeID)
		}
	}
	requiredObjects := []string{"operating-console", "patient-chair", "bridge-chair", "garbage-bin", "wall-sign", "pipe-line"}
	objects := make(map[string]struct{}, len(pack.FixedAnchors))
	for _, anchor := range pack.FixedAnchors {
		objects[anchor.ObjectID] = struct{}{}
	}
	for _, objectID := range requiredObjects {
		if _, exists := objects[objectID]; !exists {
			t.Fatalf("black market clinic fixture is missing fixed object %q", objectID)
		}
	}
	if len(pack.CameraAnchorPlan.Anchors) != 8 || len(pack.ViewpointCoverageMatrix.Views) != 8 {
		t.Fatalf("fixture must cover eight production viewpoints: cameras=%d views=%d", len(pack.CameraAnchorPlan.Anchors), len(pack.ViewpointCoverageMatrix.Views))
	}
	if len(pack.RequiredPaths) != 1 || pack.RequiredPaths[0].From != "clinic-inside" || pack.RequiredPaths[0].To != "right-street-extension" {
		t.Fatalf("fixture must preserve the complete clinic-to-street path: %#v", pack.RequiredPaths)
	}
}

func TestValidateSceneAssetPackRejectsInvalidCoverageAndCameraGeometry(t *testing.T) {
	tests := []struct {
		name      string
		issueCode string
		mutate    func(*SceneAssetPack)
	}{
		{
			name:      "missing view asset",
			issueCode: "VIEW_COVERAGE_MISSING",
			mutate: func(pack *SceneAssetPack) {
				pack.ViewpointCoverageMatrix.Views[1].AssetReference = "missing-asset"
			},
		},
		{
			name:      "unknown required object",
			issueCode: "VIEW_COVERAGE_MISSING",
			mutate: func(pack *SceneAssetPack) {
				pack.ViewpointCoverageMatrix.Views[0].RequiredVisibleObjectIDs = []string{"unknown-object"}
			},
		},
		{
			name:      "reverse camera crosses axis",
			issueCode: "REVERSE_CAMERA_CONFLICT",
			mutate: func(pack *SceneAssetPack) {
				pack.CameraAnchorPlan.Anchors[1].AxisSide = "south"
			},
		},
		{
			name:      "camera outside topology zone",
			issueCode: "CAMERA_GEOMETRY_CONFLICT",
			mutate: func(pack *SceneAssetPack) {
				pack.CameraAnchorPlan.Anchors[0].TopologyNodeID = "street-corner"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pack := blackMarketClinicSpatialPack()
			test.mutate(&pack)
			gate := ValidateSceneAssetPack(pack, nil)
			if gate.Status != SceneSpatialGateFail {
				t.Fatalf("invalid spatial contract status = %s, issues = %#v", gate.Status, gate.Issues)
			}
			assertSpatialIssueCode(t, gate.Issues, test.issueCode)
		})
	}
}

func TestValidateSceneAssetPackDetectsLockedLookAndNewAnchorDrift(t *testing.T) {
	tests := []struct {
		name      string
		issueCode string
		mutate    func(*SceneAssetPack)
	}{
		{
			name:      "accent palette",
			issueCode: "LOOK_CARD_DRIFT",
			mutate: func(pack *SceneAssetPack) {
				pack.InteriorLookCard.AccentPalette[0] = "warm-white"
			},
		},
		{
			name:      "new locked prop",
			issueCode: "FIXED_ANCHOR_DRIFT",
			mutate: func(pack *SceneAssetPack) {
				pack.FixedAnchors = append(pack.FixedAnchors, SceneAnchor{AnchorID: "new-machine-anchor", SceneID: pack.SceneID, ObjectID: "new-machine", AnchorType: "fixed-prop", Position: SpatialPoint{X: 6, Y: 6}, Mobility: "fixed", ContinuityLock: true})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			baseline := blackMarketClinicSpatialPack()
			current := blackMarketClinicSpatialPack()
			test.mutate(&current)
			gate := ValidateSceneAssetPack(current, &baseline)
			if gate.Status != SceneSpatialGateFail {
				t.Fatalf("locked continuity drift status = %s, issues = %#v", gate.Status, gate.Issues)
			}
			assertSpatialIssueCode(t, gate.Issues, test.issueCode)
		})
	}
}

func TestValidateSceneAssetPackMissingEvidenceIsUncertain(t *testing.T) {
	pack := blackMarketClinicSpatialPack()
	pack.MasterSceneAsset = nil
	pack.ViewpointCoverageMatrix = nil
	gate := ValidateSceneAssetPack(pack, nil)
	if gate.Status != SceneSpatialGateUncertain {
		t.Fatalf("missing evidence status = %s, issues = %#v", gate.Status, gate.Issues)
	}
	assertSpatialIssueCode(t, gate.Issues, "VIEW_COVERAGE_MISSING")
}

func TestValidateSceneAssetPackAuthorityConflictNeedsYou(t *testing.T) {
	pack := blackMarketClinicSpatialPack()
	pack.AuthorityConflicts = []SceneAuthorityConflict{{Field: "clinicDoor.orientation", Sources: []string{"story-bible:v2", "location-bible:v1"}}}
	gate := ValidateSceneAssetPack(pack, nil)
	if gate.Status != SceneSpatialGateNeedsYou {
		t.Fatalf("authority conflict status = %s, issues = %#v", gate.Status, gate.Issues)
	}
	assertSpatialIssueCode(t, gate.Issues, "AUTHORITY_CONFLICT")
}

func TestDecodeLegacyLocationDefinitionRemainsReadableButUncertain(t *testing.T) {
	pack, err := DecodeSceneAssetPackPayload([]byte(`{"sceneId":"scene-clinic","canonicalGeometry":"自由文本旧定义","doors":"clinic-door"}`))
	if err != nil {
		t.Fatal(err)
	}
	gate := ValidateSceneAssetPack(pack, nil)
	if !gate.Legacy || gate.Status != SceneSpatialGateUncertain {
		t.Fatalf("legacy payload should remain readable and uncertain: legacy=%v status=%s issues=%#v", gate.Legacy, gate.Status, gate.Issues)
	}
	assertSpatialIssueCode(t, gate.Issues, "LEGACY_LOCATION_DEFINITION")
}

func TestValidateSceneAssetPackRejectsImpossiblePath(t *testing.T) {
	pack := blackMarketClinicSpatialPack()
	pack.RequiredPaths = []ScenePathRequirement{{ID: "escape", From: "clinic-inside", To: "roof-access"}}
	gate := ValidateSceneAssetPack(pack, nil)
	if gate.Status != SceneSpatialGateFail {
		t.Fatalf("impossible path status = %s, issues = %#v", gate.Status, gate.Issues)
	}
	assertSpatialIssueCode(t, gate.Issues, "PATH_IMPOSSIBLE")
}

func TestNormalizeSceneSpatialDiagnosticUsesExistingGateVocabulary(t *testing.T) {
	issue := NormalizeSceneSpatialDiagnostic(SceneDiagnosticDoorPositionDrift, "shot[7].door", "门在反打镜头中漂移")
	if issue.Code != "DOOR_WINDOW_RELATION_CONFLICT" || issue.Severity != SceneSpatialIssueError || issue.Path != "shot[7].door" {
		t.Fatalf("unexpected normalized spatial diagnostic: %#v", issue)
	}
	unknown := NormalizeSceneSpatialDiagnostic(SceneSpatialDiagnosticCode("future-code"), "", "尚未定义的观察")
	if unknown.Code != "SCENE_DIAGNOSTIC_UNMAPPED" || unknown.Severity != SceneSpatialIssueWarning {
		t.Fatalf("unknown diagnostics must remain uncertain: %#v", unknown)
	}
}

func blackMarketClinicSpatialPack() SceneAssetPack {
	return SceneAssetPack{
		SchemaVersion: SceneSpatialContractSchemaVersion,
		SceneID:       "scene-clinic",
		SceneManifest: &SceneManifest{
			SceneID: "scene-clinic", SceneName: "黑市诊所连续空间", ParentLocation: "black-market-clinic", InteriorExterior: "mixed",
			StoryEvents: []string{"小皮在操作台前拆线", "富豪从诊所门进入", "人物冲出诊所", "沿外墙移动", "到达街角并向右转", "继续沿主街延伸"},
			Characters:  []string{"xiaopi", "tycoon"}, RequiredProps: []string{"operating-console", "patient-chair", "bridge-chair", "garbage-bin", "wall-sign", "pipe-line"},
			Entrances: []string{"clinic-door"}, Exits: []string{"clinic-door"}, PreviousScene: "scene-market", NextScene: "scene-main-street", ContinuityPriority: "high",
		},
		SceneTopologyGraph: &SceneTopologyGraph{
			Nodes: []SceneTopologyNode{
				{ID: "clinic-inside", Kind: "room", Position: &SpatialPoint{X: 7, Y: 4}},
				{ID: "clinic-door", Kind: "door", Position: &SpatialPoint{X: 14, Y: 6}},
				{ID: "clinic-entrance", Kind: "entrance", Position: &SpatialPoint{X: 15, Y: 6}},
				{ID: "clinic-exterior-wall", Kind: "building-edge", Position: &SpatialPoint{X: 20, Y: 5}},
				{ID: "clinic-window", Kind: "window", Position: &SpatialPoint{X: 20, Y: 10}},
				{ID: "street-corner", Kind: "street-corner", Position: &SpatialPoint{X: 25, Y: 6}},
				{ID: "main-street", Kind: "street", Position: &SpatialPoint{X: 30, Y: 6}},
				{ID: "right-street-extension", Kind: "street", Position: &SpatialPoint{X: 36, Y: 6}},
			},
			Edges: []SceneTopologyEdge{
				{From: "clinic-inside", To: "clinic-door", Direction: "east", Distance: 4, Transition: "door-crossing", ConnectionType: "door", DoorRelation: "clinic-door"},
				{From: "clinic-door", To: "clinic-entrance", Direction: "east", Distance: 1, Transition: "threshold", ConnectionType: "transition", DoorRelation: "clinic-door"},
				{From: "clinic-entrance", To: "clinic-exterior-wall", Direction: "east", Distance: 5, Transition: "walk-along-wall", ConnectionType: "exterior-path"},
				{From: "clinic-exterior-wall", To: "clinic-window", Direction: "north", Distance: 5, Visibility: "same-facade", ConnectionType: "window-sightline", WindowRelation: "clinic-window"},
				{From: "clinic-exterior-wall", To: "street-corner", Direction: "east", Distance: 5, Transition: "walk-to-corner", ConnectionType: "exterior-path"},
				{From: "street-corner", To: "main-street", Direction: "east", Distance: 5, Transition: "right-turn", ConnectionType: "street-turn"},
				{From: "main-street", To: "right-street-extension", Direction: "east", Distance: 6, Transition: "continue-forward", ConnectionType: "street-extension"},
			},
		},
		SpatialFloorPlan: &SpatialFloorPlan{
			CoordinateSystem: "local-xyz", Width: 40, Depth: 12,
			Zones: []SpatialZone{
				{ID: "operation-zone", TopologyNodeID: "clinic-inside", MinX: 1, MinY: 1, MaxX: 5, MaxY: 7, Purpose: "小皮操作台"},
				{ID: "patient-zone", TopologyNodeID: "clinic-inside", MinX: 5, MinY: 1, MaxX: 9, MaxY: 7, Purpose: "患者椅"},
				{ID: "bridge-zone", TopologyNodeID: "clinic-inside", MinX: 9, MinY: 1, MaxX: 13, MaxY: 7, Purpose: "桥接椅"},
				{ID: "entrance-zone", TopologyNodeID: "clinic-entrance", MinX: 13, MinY: 1, MaxX: 16, MaxY: 11, Purpose: "诊所门口"},
				{ID: "exterior-wall-zone", TopologyNodeID: "clinic-exterior-wall", MinX: 16, MinY: 1, MaxX: 23, MaxY: 11, Purpose: "诊所外墙"},
				{ID: "street-corner-zone", TopologyNodeID: "street-corner", MinX: 23, MinY: 1, MaxX: 27, MaxY: 11, Purpose: "街角"},
				{ID: "main-street-zone", TopologyNodeID: "main-street", MinX: 27, MinY: 1, MaxX: 34, MaxY: 11, Purpose: "右侧主街"},
				{ID: "right-street-zone", TopologyNodeID: "right-street-extension", MinX: 34, MinY: 1, MaxX: 40, MaxY: 11, Purpose: "街道延伸"},
			},
			Openings: []SpatialOpening{
				{ID: "clinic-door", Kind: "door", Position: SpatialPoint{X: 14, Y: 6}, Orientation: "east", ConnectsTo: "clinic-entrance", ContinuityLock: true},
				{ID: "clinic-window", Kind: "window", Position: SpatialPoint{X: 20, Y: 10}, Orientation: "north", ConnectsTo: "clinic-exterior-wall", ContinuityLock: true},
			},
		},
		FixedAnchors: []SceneAnchor{
			{AnchorID: "clinic-door-anchor", SceneID: "scene-clinic", ObjectID: "clinic-door", AnchorType: "door", Position: SpatialPoint{X: 14, Y: 6}, Orientation: "east", Mobility: "fixed", ContinuityLock: true},
			{AnchorID: "console-anchor", SceneID: "scene-clinic", ObjectID: "operating-console", AnchorType: "fixed-prop", Position: SpatialPoint{X: 2, Y: 2}, Mobility: "fixed", ContinuityLock: true},
			{AnchorID: "patient-chair-anchor", SceneID: "scene-clinic", ObjectID: "patient-chair", AnchorType: "fixed-prop", Position: SpatialPoint{X: 7, Y: 4}, Mobility: "fixed", ContinuityLock: true},
			{AnchorID: "bridge-chair-anchor", SceneID: "scene-clinic", ObjectID: "bridge-chair", AnchorType: "fixed-prop", Position: SpatialPoint{X: 11, Y: 4}, Mobility: "fixed", ContinuityLock: true},
			{AnchorID: "garbage-bin-anchor", SceneID: "scene-clinic", ObjectID: "garbage-bin", AnchorType: "fixed-prop", Position: SpatialPoint{X: 18, Y: 2}, Mobility: "fixed", ContinuityLock: true},
			{AnchorID: "wall-sign-anchor", SceneID: "scene-clinic", ObjectID: "wall-sign", AnchorType: "fixed-prop", Position: SpatialPoint{X: 20, Y: 6}, Mobility: "fixed", ContinuityLock: true},
			{AnchorID: "pipe-anchor", SceneID: "scene-clinic", ObjectID: "pipe-line", AnchorType: "fixed-prop", Position: SpatialPoint{X: 21, Y: 8}, Mobility: "fixed", ContinuityLock: true},
			{AnchorID: "clinic-window-anchor", SceneID: "scene-clinic", ObjectID: "clinic-window", AnchorType: "window", Position: SpatialPoint{X: 20, Y: 10}, Orientation: "north", Mobility: "fixed", ContinuityLock: true},
		},
		MovableAnchors: []SceneAnchor{
			{AnchorID: "xiaopi-anchor", SceneID: "scene-clinic", ObjectID: "xiaopi", AnchorType: "character", Position: SpatialPoint{X: 4, Y: 4}, Mobility: "movable"},
			{AnchorID: "tycoon-anchor", SceneID: "scene-clinic", ObjectID: "tycoon", AnchorType: "character", Position: SpatialPoint{X: 8, Y: 4}, Mobility: "movable"},
		},
		CameraAnchorPlan: &CameraAnchorPlan{ActionAxis: "east-west", Anchors: []CameraAnchor{
			{CameraID: "C1", SceneID: "scene-clinic", TopologyNodeID: "clinic-inside", AnchorID: "console-anchor", Position: SpatialPoint{X: 3, Y: 2}, Direction: "east", Facing: "tycoon", LensClass: "35mm", AxisSide: "north", ReverseOf: "C2"},
			{CameraID: "C2", SceneID: "scene-clinic", TopologyNodeID: "clinic-inside", AnchorID: "bridge-chair-anchor", Position: SpatialPoint{X: 11, Y: 4}, Direction: "west", Facing: "xiaopi", LensClass: "50mm", AxisSide: "north", ReverseOf: "C1"},
			{CameraID: "C3", SceneID: "scene-clinic", TopologyNodeID: "clinic-inside", AnchorID: "patient-chair-anchor", Position: SpatialPoint{X: 7, Y: 1}, Direction: "north", Facing: "patient", LensClass: "50mm", AxisSide: "south"},
			{CameraID: "C4", SceneID: "scene-clinic", TopologyNodeID: "clinic-entrance", AnchorID: "clinic-door-anchor", Position: SpatialPoint{X: 15, Y: 6}, Direction: "west", Facing: "inside", LensClass: "24mm", AxisSide: "south"},
			{CameraID: "C5", SceneID: "scene-clinic", TopologyNodeID: "clinic-inside", AnchorID: "clinic-door-anchor", Position: SpatialPoint{X: 12, Y: 6}, Direction: "east", Facing: "clinic-door", LensClass: "24mm", AxisSide: "south"},
			{CameraID: "C6", SceneID: "scene-clinic", TopologyNodeID: "clinic-exterior-wall", AnchorID: "garbage-bin-anchor", Position: SpatialPoint{X: 19, Y: 5}, Direction: "east", Facing: "street-corner", LensClass: "35mm", AxisSide: "north"},
			{CameraID: "C7", SceneID: "scene-clinic", TopologyNodeID: "street-corner", AnchorID: "wall-sign-anchor", Position: SpatialPoint{X: 25, Y: 6}, Direction: "east", Facing: "main-street", LensClass: "28mm", AxisSide: "north"},
			{CameraID: "C8", SceneID: "scene-clinic", TopologyNodeID: "right-street-extension", AnchorID: "wall-sign-anchor", Position: SpatialPoint{X: 35, Y: 6}, Direction: "east", Facing: "street-extension", LensClass: "35mm", AxisSide: "north"},
		}},
		ViewpointCoverageMatrix: &ViewpointCoverageMatrix{Views: []ViewpointCoverage{
			{ViewID: "V01", CameraAnchorID: "C1", Facing: "east", Purpose: "小皮看富豪", Required: true, AssetReference: "clinic-master", RequiredVisibleObjectIDs: []string{"operating-console", "bridge-chair"}},
			{ViewID: "V02", CameraAnchorID: "C2", Facing: "west", Purpose: "富豪反打看操作台", Required: true, AssetReference: "clinic-reverse", RequiredVisibleObjectIDs: []string{"operating-console"}},
			{ViewID: "V03", CameraAnchorID: "C3", Facing: "north", Purpose: "患者椅正面", Required: true, AssetReference: "clinic-patient-view", RequiredVisibleObjectIDs: []string{"patient-chair"}},
			{ViewID: "V04", CameraAnchorID: "C4", Facing: "west", Purpose: "门外看诊所内部", Required: true, AssetReference: "clinic-door-exterior", RequiredVisibleObjectIDs: []string{"clinic-door", "operating-console"}},
			{ViewID: "V05", CameraAnchorID: "C5", Facing: "east", Purpose: "室内看诊所门外", Required: true, AssetReference: "clinic-inside-door", RequiredVisibleObjectIDs: []string{"clinic-door", "wall-sign"}},
			{ViewID: "V06", CameraAnchorID: "C6", Facing: "east", Purpose: "沿外墙移动", Required: true, AssetReference: "clinic-wall-run", RequiredVisibleObjectIDs: []string{"garbage-bin", "wall-sign", "pipe-line"}},
			{ViewID: "V07", CameraAnchorID: "C7", Facing: "east", Purpose: "街角向右转", Required: true, AssetReference: "clinic-corner", RequiredVisibleObjectIDs: []string{"wall-sign"}},
			{ViewID: "V08", CameraAnchorID: "C8", Facing: "east", Purpose: "继续看到右侧街道延伸", Required: true, AssetReference: "clinic-right-street", ForbiddenObjectIDs: []string{"operating-console"}},
		}},
		MasterSceneAsset: &SceneAssetReference{AssetID: "clinic-master", ViewID: "V01", Role: "master", EvidenceRef: "e-master"},
		ViewAssets: []SceneAssetReference{
			{AssetID: "clinic-reverse", ViewID: "V02", EvidenceRef: "e-reverse"},
			{AssetID: "clinic-patient-view", ViewID: "V03", EvidenceRef: "e-patient"},
			{AssetID: "clinic-door-exterior", ViewID: "V04", EvidenceRef: "e-door-exterior"},
			{AssetID: "clinic-inside-door", ViewID: "V05", EvidenceRef: "e-inside-door"},
			{AssetID: "clinic-wall-run", ViewID: "V06", EvidenceRef: "e-wall-run"},
			{AssetID: "clinic-corner", ViewID: "V07", EvidenceRef: "e-corner"},
			{AssetID: "clinic-right-street", ViewID: "V08", EvidenceRef: "e-right-street"},
		},
		Evidence: []SceneSpatialEvidence{
			{ID: "e-master", Kind: "approved-reference", SourceRef: "asset:clinic-master", Verified: true},
			{ID: "e-reverse", Kind: "approved-reference", SourceRef: "asset:clinic-reverse", Verified: true},
			{ID: "e-patient", Kind: "approved-reference", SourceRef: "asset:clinic-patient-view", Verified: true},
			{ID: "e-door-exterior", Kind: "approved-reference", SourceRef: "asset:clinic-door-exterior", Verified: true},
			{ID: "e-inside-door", Kind: "approved-reference", SourceRef: "asset:clinic-inside-door", Verified: true},
			{ID: "e-wall-run", Kind: "approved-reference", SourceRef: "asset:clinic-wall-run", Verified: true},
			{ID: "e-corner", Kind: "approved-reference", SourceRef: "asset:clinic-corner", Verified: true},
			{ID: "e-right-street", Kind: "approved-reference", SourceRef: "asset:clinic-right-street", Verified: true},
		},
		InteriorLookCard: &SceneLookCard{LookFamilyID: "black-market-clinic-night", InheritsFrom: "exteriorLookCard", BasePalette: []string{"dirty-gray", "blue-gray"}, AccentPalette: []string{"cold-white"}, ShadowPalette: []string{"deep-blue"}, LightingDirection: "east", LightingTemperature: "cool", ContrastLevel: "high", TimeOfDay: "night", Weather: "dry", MaterialRules: []string{"hard-edged-2d"}, RenderingRules: []string{"flat-shadows"}, ForbiddenRendering: []string{"photoreal-3d", "soft-reflections"}, ContinuityLock: true},
		ExteriorLookCard: &SceneLookCard{LookFamilyID: "black-market-clinic-night", BasePalette: []string{"dirty-gray", "blue-gray"}, AccentPalette: []string{"neon-cyan", "neon-orange"}, ShadowPalette: []string{"deep-blue"}, NeonPalette: []string{"neon-cyan", "neon-pink"}, LightingDirection: "east", LightingTemperature: "cool-neon", ContrastLevel: "high", TimeOfDay: "night", Weather: "dry", MaterialRules: []string{"aged-concrete", "hard-edged-2d"}, RenderingRules: []string{"flat-shadows", "limited-reflection"}, ForbiddenRendering: []string{"photoreal-3d", "plastic-surfaces"}, ContinuityLock: true},
		ForbiddenChanges: []string{"door-position", "window-position", "console-position", "patient-chair-position", "garbage-bin-position", "wall-sign-position", "pipe-route", "camera-axis", "street-direction"},
		RequiredPaths:    []ScenePathRequirement{{ID: "clinic-escape", From: "clinic-inside", To: "right-street-extension", Via: []string{"clinic-door", "clinic-entrance", "clinic-exterior-wall", "street-corner", "main-street"}, Purpose: "人物冲出诊所并沿右侧街道延伸"}},
	}
}

func assertSpatialIssueCode(t *testing.T, issues []SceneSpatialIssue, code string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Code == code {
			return
		}
	}
	encoded, _ := json.Marshal(issues)
	t.Fatalf("expected spatial issue %s, got %s", code, encoded)
}
