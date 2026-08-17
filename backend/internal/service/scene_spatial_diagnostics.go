package service

// Diagnostic codes are a stable handoff contract for a future AI Shot
// Diagnostics executor. This package only normalizes a reported observation;
// it does not inspect pixels or claim that a provider result was analyzed.

type SceneSpatialDiagnosticCode string

const (
	SceneDiagnosticFixedPropDrift      SceneSpatialDiagnosticCode = "FIXED_PROP_DRIFT"
	SceneDiagnosticDoorPositionDrift   SceneSpatialDiagnosticCode = "DOOR_POSITION_DRIFT"
	SceneDiagnosticWindowPositionDrift SceneSpatialDiagnosticCode = "WINDOW_POSITION_DRIFT"
	SceneDiagnosticCameraGeometry      SceneSpatialDiagnosticCode = "CAMERA_GEOMETRY_CONFLICT"
	SceneDiagnosticReverseShot         SceneSpatialDiagnosticCode = "REVERSE_SHOT_CONFLICT"
	SceneDiagnosticTopology            SceneSpatialDiagnosticCode = "SCENE_TOPOLOGY_CONFLICT"
	SceneDiagnosticFloorPlan           SceneSpatialDiagnosticCode = "FLOORPLAN_CONFLICT"
	SceneDiagnosticLookDrift           SceneSpatialDiagnosticCode = "LOOK_DRIFT"
	SceneDiagnosticPathImpossible      SceneSpatialDiagnosticCode = "PATH_IMPOSSIBLE"
	SceneDiagnosticSceneRedesign       SceneSpatialDiagnosticCode = "SCENE_REDESIGN_DETECTED"
)

// NormalizeSceneSpatialDiagnostic maps an observation into the existing Gate
// issue vocabulary. Unknown observations stay UNCERTAIN instead of silently
// being treated as a pass.
func NormalizeSceneSpatialDiagnostic(code SceneSpatialDiagnosticCode, path string, message string) SceneSpatialIssue {
	issue := SceneSpatialIssue{Code: "SCENE_DIAGNOSTIC_UNMAPPED", Severity: SceneSpatialIssueWarning, Path: path, Message: message}
	switch code {
	case SceneDiagnosticFixedPropDrift:
		issue.Code, issue.Severity = "FIXED_ANCHOR_DRIFT", SceneSpatialIssueError
	case SceneDiagnosticDoorPositionDrift, SceneDiagnosticWindowPositionDrift:
		issue.Code, issue.Severity = "DOOR_WINDOW_RELATION_CONFLICT", SceneSpatialIssueError
	case SceneDiagnosticCameraGeometry:
		issue.Code, issue.Severity = "CAMERA_ANCHOR_UNKNOWN", SceneSpatialIssueError
	case SceneDiagnosticReverseShot:
		issue.Code, issue.Severity = "REVERSE_CAMERA_CONFLICT", SceneSpatialIssueError
	case SceneDiagnosticTopology:
		issue.Code, issue.Severity = "TOPOLOGY_EDGE_INVALID", SceneSpatialIssueError
	case SceneDiagnosticFloorPlan:
		issue.Code, issue.Severity = "FLOORPLAN_ANCHOR_OUT_OF_BOUNDS", SceneSpatialIssueError
	case SceneDiagnosticLookDrift:
		issue.Code, issue.Severity = "LOOK_CARD_DRIFT", SceneSpatialIssueError
	case SceneDiagnosticPathImpossible:
		issue.Code, issue.Severity = "PATH_IMPOSSIBLE", SceneSpatialIssueError
	case SceneDiagnosticSceneRedesign:
		issue.Code, issue.Severity = "SCENE_REDESIGN_DETECTED", SceneSpatialIssueError
	}
	return issue
}
