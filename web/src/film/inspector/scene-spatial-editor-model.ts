import type { ProjectScene } from "@/services/api/projects";

export type SpatialPointDraft = Record<string, unknown> & { x: number; y: number; z: number };
export type SceneManifestDraft = Record<string, unknown> & {
    sceneId: string;
    sceneName: string;
    parentLocation: string;
    interiorExterior: string;
    storyEvents: string[];
    characters: string[];
    requiredProps: string[];
    entrances: string[];
    exits: string[];
    previousScene: string;
    nextScene: string;
    continuityPriority: string;
};
export type TopologyNodeDraft = Record<string, unknown> & { id: string; kind: string; label: string; position: SpatialPointDraft };
export type TopologyEdgeDraft = Record<string, unknown> & {
    from: string;
    to: string;
    direction: string;
    distance: number;
    visibility: string;
    elevation: string;
    transition: string;
    connectionType: string;
    doorRelation: string;
    windowRelation: string;
};
export type FloorZoneDraft = Record<string, unknown> & { id: string; topologyNodeId: string; label: string; minX: number; minY: number; maxX: number; maxY: number; purpose: string };
export type FloorOpeningDraft = Record<string, unknown> & { id: string; kind: string; position: SpatialPointDraft; orientation: string; state: string; connectsTo: string; continuityLock: boolean };
export type FloorPlanDraft = Record<string, unknown> & { coordinateSystem: string; origin: SpatialPointDraft; width: number; depth: number; zones: FloorZoneDraft[]; openings: FloorOpeningDraft[] };
export type SceneAnchorDraft = Record<string, unknown> & { anchorId: string; sceneId: string; objectId: string; anchorType: string; position: SpatialPointDraft; orientation: string; mobility: string; continuityLock: boolean; evidence: string[] };
export type CameraAnchorDraft = Record<string, unknown> & {
    cameraId: string;
    sceneId: string;
    topologyNodeId: string;
    anchorId: string;
    position: SpatialPointDraft;
    height: number;
    direction: string;
    facing: string;
    target: string;
    lensClass: string;
    shotSizeRange: string[];
    allowedFovRange: number[];
    movementConstraints: string[];
    axisSide: string;
    reverseOf: string;
};
export type ViewpointDraft = Record<string, unknown> & { viewId: string; cameraAnchorId: string; facing: string; purpose: string; required: boolean; assetReference: string; requiredVisibleObjectIds: string[]; forbiddenObjectIds: string[] };
export type SceneAssetReferenceDraft = Record<string, unknown> & { assetId: string; uri: string; role: string; viewId: string; evidenceRef: string };
export type SceneLookCardDraft = Record<string, unknown> & {
    lookFamilyId: string;
    inheritsFrom: string;
    basePalette: string[];
    accentPalette: string[];
    shadowPalette: string[];
    neonPalette: string[];
    lightingDirection: string;
    lightingTemperature: string;
    contrastLevel: string;
    timeOfDay: string;
    weather: string;
    materialRules: string[];
    renderingRules: string[];
    forbiddenRendering: string[];
    continuityLock: boolean;
};
export type EvidenceDraft = Record<string, unknown> & { id: string; kind: string; sourceRef: string; verified: boolean };
export type RequiredPathDraft = Record<string, unknown> & { id: string; from: string; to: string; via: string[]; purpose: string };
export type AuthorityConflictDraft = Record<string, unknown> & { field: string; sources: string[]; description: string };

export type SceneAssetPackDraft = Record<string, unknown> & {
    schemaVersion: number;
    sceneId: string;
    sceneManifestRef: string;
    sceneManifest: SceneManifestDraft;
    sceneTopologyGraph: Record<string, unknown> & { nodes: TopologyNodeDraft[]; edges: TopologyEdgeDraft[] };
    spatialFloorPlan: FloorPlanDraft;
    fixedAnchors: SceneAnchorDraft[];
    movableAnchors: SceneAnchorDraft[];
    cameraAnchorPlan: Record<string, unknown> & { actionAxis: string; anchors: CameraAnchorDraft[] };
    viewpointCoverageMatrix: Record<string, unknown> & { views: ViewpointDraft[] };
    masterSceneAsset: SceneAssetReferenceDraft;
    viewAssets: SceneAssetReferenceDraft[];
    interiorLookCard?: SceneLookCardDraft;
    exteriorLookCard?: SceneLookCardDraft;
    forbiddenChanges: string[];
    authorityConflicts: AuthorityConflictDraft[];
    evidence: EvidenceDraft[];
    requiredPaths: RequiredPathDraft[];
};

const LEGACY_LOCATION_FIELDS = ["canonicalGeometry", "doors", "windows", "anchors", "fixedProps", "movableProps", "lightingLogic", "masterView", "reverseViews"];

export function createSceneAssetPackDraft(scene: ProjectScene): SceneAssetPackDraft {
    const interiorExterior = scene.interiorExterior || "";
    const rootNodeId = "scene-main";
    const masterEvidenceId = scene.locationAssetId ? "master-scene-evidence" : "";
    return {
        schemaVersion: 1,
        sceneId: scene.id,
        sceneManifestRef: "",
        sceneManifest: {
            sceneId: scene.id,
            sceneName: scene.title,
            parentLocation: "",
            interiorExterior,
            storyEvents: scene.description.trim() ? [scene.description.trim()] : [],
            characters: [],
            requiredProps: [],
            entrances: [],
            exits: [],
            previousScene: "",
            nextScene: "",
            continuityPriority: "",
        },
        sceneTopologyGraph: { nodes: [{ id: rootNodeId, kind: "scene", label: scene.title, position: point() }], edges: [] },
        spatialFloorPlan: {
            coordinateSystem: "local-xy",
            origin: point(),
            width: 10,
            depth: 10,
            zones: [{ id: "main-zone", topologyNodeId: rootNodeId, label: scene.title, minX: 0, minY: 0, maxX: 10, maxY: 10, purpose: "主表演区" }],
            openings: [],
        },
        fixedAnchors: [],
        movableAnchors: [],
        cameraAnchorPlan: { actionAxis: "", anchors: [] },
        viewpointCoverageMatrix: { views: [] },
        masterSceneAsset: { assetId: scene.locationAssetId || "", uri: "", role: "master_scene", viewId: "", evidenceRef: masterEvidenceId },
        viewAssets: [],
        interiorLookCard: interiorExterior === "interior" || interiorExterior === "mixed" ? lookCard() : undefined,
        exteriorLookCard: interiorExterior === "exterior" || interiorExterior === "mixed" ? lookCard() : undefined,
        forbiddenChanges: [],
        authorityConflicts: [],
        evidence: masterEvidenceId ? [{ id: masterEvidenceId, kind: "master_scene", sourceRef: scene.locationAssetId || "", verified: false }] : [],
        requiredPaths: [],
    };
}

export function parseSceneAssetPackDraft(payloadJson: string | undefined, scene: ProjectScene): SceneAssetPackDraft {
    if (!payloadJson) return createSceneAssetPackDraft(scene);
    try {
        const parsed: unknown = JSON.parse(payloadJson);
        return normalizeSceneAssetPackDraft(record(parsed), scene);
    } catch {
        return createSceneAssetPackDraft(scene);
    }
}

export function buildSceneAssetPackPayload(draft: SceneAssetPackDraft, scene: ProjectScene): Record<string, unknown> {
    return {
        ...draft,
        schemaVersion: 1,
        sceneId: scene.id,
        sceneManifest: { ...draft.sceneManifest, sceneId: scene.id, sceneName: draft.sceneManifest.sceneName || scene.title },
        fixedAnchors: draft.fixedAnchors.map((anchor) => ({ ...anchor, sceneId: scene.id })),
        movableAnchors: draft.movableAnchors.map((anchor) => ({ ...anchor, sceneId: scene.id })),
        cameraAnchorPlan: { ...draft.cameraAnchorPlan, anchors: draft.cameraAnchorPlan.anchors.map((camera) => ({ ...camera, sceneId: scene.id })) },
    };
}

export function removeLegacyLocationFields(draft: SceneAssetPackDraft): SceneAssetPackDraft {
    const next = { ...draft };
    for (const field of LEGACY_LOCATION_FIELDS) delete next[field];
    return next;
}

export function hasLegacyLocationFields(draft: SceneAssetPackDraft) {
    return LEGACY_LOCATION_FIELDS.some((field) => Object.prototype.hasOwnProperty.call(draft, field));
}

export function createSceneLookCardDraft() {
    return lookCard();
}

export function listToLines(values: string[] | undefined) {
    return (values || []).join("\n");
}
export function linesToList(value: string) {
    return value
        .split(/\n|、/)
        .map((item) => item.trim())
        .filter(Boolean);
}
export function numberListToText(values: number[] | undefined) {
    return (values || []).join("\n");
}
export function textToNumberList(value: string) {
    return value
        .split(/\n|、/)
        .map((item) => Number(item.trim()))
        .filter((item) => Number.isFinite(item));
}

export function nextSpatialId(prefix: string, existing: string[]) {
    const occupied = new Set(existing.filter(Boolean));
    for (let index = 1; ; index += 1) {
        const candidate = `${prefix}-${index}`;
        if (!occupied.has(candidate)) return candidate;
    }
}

export function spatialPanelForPath(path: string | undefined): "overview" | "structure" | "coverage" | "assets" | "gate" {
    if (!path) return "gate";
    if (path.startsWith("sceneManifest") || path === "sceneId" || path === "status") return "overview";
    if (path.startsWith("sceneTopologyGraph") || path.startsWith("spatialFloorPlan") || path.startsWith("fixedAnchors") || path.startsWith("movableAnchors")) return "structure";
    if (path.startsWith("cameraAnchorPlan") || path.startsWith("viewpointCoverageMatrix") || path.startsWith("requiredPaths")) return "coverage";
    if (path.startsWith("masterSceneAsset") || path.startsWith("viewAssets") || path.startsWith("asset[") || path.startsWith("evidence") || path.startsWith("interiorLookCard") || path.startsWith("exteriorLookCard") || path.startsWith("lookCards"))
        return "assets";
    return "gate";
}

function normalizeSceneAssetPackDraft(input: Record<string, unknown>, scene: ProjectScene): SceneAssetPackDraft {
    const defaults = createSceneAssetPackDraft(scene);
    const manifest = record(input.sceneManifest);
    const graph = record(input.sceneTopologyGraph);
    const plan = record(input.spatialFloorPlan);
    const cameraPlan = record(input.cameraAnchorPlan);
    const coverage = record(input.viewpointCoverageMatrix);
    const sceneMode = text(manifest.interiorExterior, defaults.sceneManifest.interiorExterior);
    return {
        ...input,
        schemaVersion: number(input.schemaVersion, 1),
        sceneId: scene.id,
        sceneManifestRef: text(input.sceneManifestRef),
        sceneManifest: manifestDraft(manifest, scene, sceneMode),
        sceneTopologyGraph: { ...graph, nodes: array(graph.nodes).map(topologyNode), edges: array(graph.edges).map(topologyEdge) },
        spatialFloorPlan: floorPlan(plan, defaults.spatialFloorPlan),
        fixedAnchors: array(input.fixedAnchors).map((value) => anchor(value, scene.id, true)),
        movableAnchors: array(input.movableAnchors).map((value) => anchor(value, scene.id, false)),
        cameraAnchorPlan: { ...cameraPlan, actionAxis: text(cameraPlan.actionAxis), anchors: array(cameraPlan.anchors).map((value) => camera(value, scene.id)) },
        viewpointCoverageMatrix: { ...coverage, views: array(coverage.views).map(viewpoint) },
        masterSceneAsset: assetReference(record(input.masterSceneAsset), defaults.masterSceneAsset),
        viewAssets: array(input.viewAssets).map((value) => assetReference(record(value))),
        interiorLookCard: input.interiorLookCard ? lookCard(record(input.interiorLookCard)) : defaults.interiorLookCard,
        exteriorLookCard: input.exteriorLookCard ? lookCard(record(input.exteriorLookCard)) : defaults.exteriorLookCard,
        forbiddenChanges: strings(input.forbiddenChanges),
        authorityConflicts: array(input.authorityConflicts).map(authorityConflict),
        evidence: array(input.evidence).map(evidence),
        requiredPaths: array(input.requiredPaths).map(requiredPath),
    };
}

function manifestDraft(value: Record<string, unknown>, scene: ProjectScene, interiorExterior: string): SceneManifestDraft {
    return {
        ...value,
        sceneId: scene.id,
        sceneName: text(value.sceneName, scene.title),
        parentLocation: text(value.parentLocation),
        interiorExterior,
        storyEvents: strings(value.storyEvents),
        characters: strings(value.characters),
        requiredProps: strings(value.requiredProps),
        entrances: strings(value.entrances),
        exits: strings(value.exits),
        previousScene: text(value.previousScene),
        nextScene: text(value.nextScene),
        continuityPriority: text(value.continuityPriority),
    };
}
function topologyNode(value: unknown): TopologyNodeDraft {
    const item = record(value);
    return { ...item, id: text(item.id), kind: text(item.kind), label: text(item.label), position: point(item.position) };
}
function topologyEdge(value: unknown): TopologyEdgeDraft {
    const item = record(value);
    return {
        ...item,
        from: text(item.from),
        to: text(item.to),
        direction: text(item.direction),
        distance: number(item.distance),
        visibility: text(item.visibility),
        elevation: text(item.elevation),
        transition: text(item.transition),
        connectionType: text(item.connectionType),
        doorRelation: text(item.doorRelation),
        windowRelation: text(item.windowRelation),
    };
}
function floorPlan(value: Record<string, unknown>, fallback: FloorPlanDraft): FloorPlanDraft {
    return {
        ...value,
        coordinateSystem: text(value.coordinateSystem, fallback.coordinateSystem),
        origin: point(value.origin),
        width: number(value.width, fallback.width),
        depth: number(value.depth, fallback.depth),
        zones: array(value.zones).map(zone),
        openings: array(value.openings).map(opening),
    };
}
function zone(value: unknown): FloorZoneDraft {
    const item = record(value);
    return { ...item, id: text(item.id), topologyNodeId: text(item.topologyNodeId), label: text(item.label), minX: number(item.minX), minY: number(item.minY), maxX: number(item.maxX), maxY: number(item.maxY), purpose: text(item.purpose) };
}
function opening(value: unknown): FloorOpeningDraft {
    const item = record(value);
    return { ...item, id: text(item.id), kind: text(item.kind, "door"), position: point(item.position), orientation: text(item.orientation), state: text(item.state), connectsTo: text(item.connectsTo), continuityLock: boolean(item.continuityLock) };
}
function anchor(value: unknown, sceneId: string, fixed: boolean): SceneAnchorDraft {
    const item = record(value);
    return {
        ...item,
        anchorId: text(item.anchorId),
        sceneId,
        objectId: text(item.objectId),
        anchorType: text(item.anchorType),
        position: point(item.position),
        orientation: text(item.orientation),
        mobility: text(item.mobility, fixed ? "fixed" : "movable"),
        continuityLock: boolean(item.continuityLock, fixed),
        evidence: strings(item.evidence),
    };
}
function camera(value: unknown, sceneId: string): CameraAnchorDraft {
    const item = record(value);
    return {
        ...item,
        cameraId: text(item.cameraId),
        sceneId,
        topologyNodeId: text(item.topologyNodeId),
        anchorId: text(item.anchorId),
        position: point(item.position),
        height: number(item.height),
        direction: text(item.direction),
        facing: text(item.facing),
        target: text(item.target),
        lensClass: text(item.lensClass),
        shotSizeRange: strings(item.shotSizeRange),
        allowedFovRange: numbers(item.allowedFovRange),
        movementConstraints: strings(item.movementConstraints),
        axisSide: text(item.axisSide),
        reverseOf: text(item.reverseOf),
    };
}
function viewpoint(value: unknown): ViewpointDraft {
    const item = record(value);
    return {
        ...item,
        viewId: text(item.viewId),
        cameraAnchorId: text(item.cameraAnchorId),
        facing: text(item.facing),
        purpose: text(item.purpose),
        required: boolean(item.required),
        assetReference: text(item.assetReference),
        requiredVisibleObjectIds: strings(item.requiredVisibleObjectIds),
        forbiddenObjectIds: strings(item.forbiddenObjectIds),
    };
}
function assetReference(value: Record<string, unknown>, fallback?: SceneAssetReferenceDraft): SceneAssetReferenceDraft {
    return {
        ...value,
        assetId: text(value.assetId, fallback?.assetId),
        uri: text(value.uri, fallback?.uri),
        role: text(value.role, fallback?.role),
        viewId: text(value.viewId, fallback?.viewId),
        evidenceRef: text(value.evidenceRef, fallback?.evidenceRef),
    };
}
function lookCard(value: Record<string, unknown> = {}): SceneLookCardDraft {
    return {
        ...value,
        lookFamilyId: text(value.lookFamilyId),
        inheritsFrom: text(value.inheritsFrom),
        basePalette: strings(value.basePalette),
        accentPalette: strings(value.accentPalette),
        shadowPalette: strings(value.shadowPalette),
        neonPalette: strings(value.neonPalette),
        lightingDirection: text(value.lightingDirection),
        lightingTemperature: text(value.lightingTemperature),
        contrastLevel: text(value.contrastLevel),
        timeOfDay: text(value.timeOfDay),
        weather: text(value.weather),
        materialRules: strings(value.materialRules),
        renderingRules: strings(value.renderingRules),
        forbiddenRendering: strings(value.forbiddenRendering),
        continuityLock: boolean(value.continuityLock),
    };
}
function evidence(value: unknown): EvidenceDraft {
    const item = record(value);
    return { ...item, id: text(item.id), kind: text(item.kind), sourceRef: text(item.sourceRef), verified: boolean(item.verified) };
}
function requiredPath(value: unknown): RequiredPathDraft {
    const item = record(value);
    return { ...item, id: text(item.id), from: text(item.from), to: text(item.to), via: strings(item.via), purpose: text(item.purpose) };
}
function authorityConflict(value: unknown): AuthorityConflictDraft {
    const item = record(value);
    return { ...item, field: text(item.field), sources: strings(item.sources), description: text(item.description) };
}
function point(value: unknown = {}): SpatialPointDraft {
    const item = record(value);
    return { ...item, x: number(item.x), y: number(item.y), z: number(item.z) };
}
function record(value: unknown): Record<string, unknown> {
    return value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : {};
}
function array(value: unknown): unknown[] {
    return Array.isArray(value) ? value : [];
}
function strings(value: unknown): string[] {
    return array(value)
        .map((item) => text(item))
        .filter(Boolean);
}
function numbers(value: unknown): number[] {
    return array(value)
        .map((item) => Number(item))
        .filter((item) => Number.isFinite(item));
}
function text(value: unknown, fallback = ""): string {
    return typeof value === "string" || typeof value === "number" ? String(value) : fallback;
}
function number(value: unknown, fallback = 0): number {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : fallback;
}
function boolean(value: unknown, fallback = false): boolean {
    return typeof value === "boolean" ? value : fallback;
}
