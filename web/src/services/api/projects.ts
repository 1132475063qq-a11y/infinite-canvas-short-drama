import { apiClient, request } from "@/services/api/request";
import type { GenerationTask } from "@/services/api/task-center";

const api = apiClient;

export type Project = {
    id: string;
    userId: string;
    name: string;
    type: string;
    aspectRatio: string;
    sourceType: string;
    description: string;
    stylePresetId: string;
    styleProfileJson?: string;
    status: "active" | "archived" | string;
    revision: number;
    createdAt: string;
    updatedAt: string;
};

export type ProjectCanvas = {
    id: string;
    projectId?: string;
    title: string;
    createdAt: string;
    updatedAt: string;
};

export type CanvasUnitLink = {
    id: string;
    projectId: string;
    canvasId: string;
    unitId: string;
    role: string;
    createdAt: string;
};

export type ProjectUnit = {
    id: string;
    projectId: string;
    kind: "chapter" | "episode" | string;
    title: string;
    sourceText: string;
    status: "draft" | "ready" | "completed" | string;
    position: number;
    createdAt: string;
    updatedAt: string;
};

export type ProjectAsset = {
    id: string;
    title: string;
    mediaType: string;
    category: string;
    status: string;
    primaryVersionId?: string;
    versionCount: number;
    usages: string[];
    updatedAt: string;
    currentVersion?: ProjectAssetVersionSummary;
    character?: CharacterCardSummary;
};

export type ProjectAssetVersionSummary = {
    id: string;
    version: number;
    status: string;
    definition: Record<string, unknown>;
    note: string;
    updatedAt: string;
};

export type CharacterRepresentation = {
    id: string;
    resourceId: string;
    mediaType: string;
    role: "primary" | "front" | "side" | "back" | "turnaround_sheet" | "expression_sheet" | string;
};

export type VoiceProfile = {
    id: string;
    name: string;
    provider: string;
    voiceKey: string;
    language: string;
    timbre: string;
    sampleResourceId?: string;
    compatibleModels: string[];
    status: string;
};

export type CharacterCardSummary = {
    versionId: string;
    version: number;
    definition: Record<string, unknown>;
    representations: CharacterRepresentation[];
    voice?: { profile: VoiceProfile; instructions: string };
    visualStatus: "missing" | "partial" | "ready" | string;
    voiceStatus: "missing" | "ready" | "unavailable" | string;
};

export type ProjectCharacterDetail = {
    asset: ProjectAsset;
    character: CharacterCardSummary;
};

export type ProjectAssetCandidate = {
    id: string;
    projectId: string;
    unitId?: string;
    shotId?: string;
    name: string;
    category: string;
    status: "pending_confirmation" | "confirmed" | "ignored" | string;
    detailsJson: string;
    resolvedAssetId?: string;
    createdAt: string;
    updatedAt: string;
};

export type ProjectShot = {
    id: string;
    projectId: string;
    unitId?: string;
    sceneId?: string;
    title: string;
    description: string;
    position: number;
    durationMs: number;
    status: string;
    contractArtifactId?: string;
    contractVersion?: number;
    createdAt: string;
    updatedAt: string;
};

export type ProjectScene = {
    id: string;
    projectId: string;
    unitId?: string;
    code: string;
    title: string;
    description: string;
    interiorExterior?: "interior" | "exterior" | "mixed" | "" | string;
    timeOfDay?: "day" | "night" | "dawn" | "dusk" | "continuous" | "" | string;
    locationAssetId?: string;
    position: number;
    status: string;
    createdAt: string;
    updatedAt: string;
};

export type FilmArtifact = {
    id: string;
    projectId: string;
    unitId?: string;
    sceneId?: string;
    shotId?: string;
    scope?: "project" | "unit" | "scene" | "shot" | string;
    scopeId?: string;
    artifactType: string;
    objectVersion: number;
    status: string;
    responsibleAgentId?: string;
    payloadJson: string;
    sourceRefsJson: string;
    authorityRefsJson: string;
    createdAt: string;
    updatedAt: string;
};

export type SceneSpatialGateStatus = "PASS" | "UNCERTAIN" | "FAIL" | "NEEDS_YOU" | string;

export type SceneSpatialIssue = {
    code: string;
    severity: string;
    path?: string;
    message: string;
};

export type SceneSpatialGate = {
    schemaVersion: number;
    sceneId: string;
    packArtifactId?: string;
    packArtifactVersion?: number;
    packVersion?: number;
    status: SceneSpatialGateStatus;
    issues: SceneSpatialIssue[];
    validatedAt: string;
    validator: string;
    legacy: boolean;
};

export type SceneAssetPackDetail = {
    packArtifact?: FilmArtifact;
    gateArtifact?: FilmArtifact;
    gate: SceneSpatialGate;
};

export type LockedSceneAssetPackProjection = {
    schemaVersion: number;
    sceneId: string;
    continuityStatus: SceneSpatialGateStatus;
    packArtifactId: string;
    packArtifactVersion: number;
    gateArtifactId: string;
    gateArtifactVersion: number;
    cameraAnchorId: string;
    viewId: string;
    sceneManifest?: Record<string, unknown>;
    sceneTopologyGraph?: Record<string, unknown>;
    spatialFloorPlan?: Record<string, unknown>;
    fixedAnchors?: Array<Record<string, unknown>>;
    movableAnchors?: Array<Record<string, unknown>>;
    cameraAnchor?: Record<string, unknown>;
    reverseCamera?: Record<string, unknown>;
    viewpoint?: Record<string, unknown>;
    masterSceneAsset?: Record<string, unknown>;
    viewAsset?: Record<string, unknown>;
    interiorLookCard?: Record<string, unknown>;
    exteriorLookCard?: Record<string, unknown>;
    forbiddenChanges?: string[];
    requiredPaths?: Array<Record<string, unknown>>;
};

export type SceneAssetPackSaveResult = {
    packArtifact: FilmArtifact;
    gateArtifact: FilmArtifact;
    gate: SceneSpatialGate;
};

// This is a read-only Provider Gateway contract, not a queued Task. It never
// contains a channel configuration, a model credential, or a provider job ID.
export type FilmGenerationTaskDraft = {
    schemaVersion: number;
    generationRequestArtifactId: string;
    generationRequestArtifactVersion: number;
    generationRequestStatus: string;
    projectId: string;
    unitId?: string;
    sceneId?: string;
    shotId: string;
    taskType: "canvas_image" | "canvas_video" | "canvas_audio" | string;
    operation: string;
    requestReady: boolean;
    providerRouteResolved: boolean;
    submissionAllowed: boolean;
    submissionState: string;
    blockers: string[] | null;
    requestFingerprint: string;
    spatialGateReady: boolean;
    spatialContinuityGate?: SceneSpatialGate;
    spatialGateArtifactId?: string;
    spatialGateArtifactVersion?: number;
    spatialPackArtifactId?: string;
    spatialPackArtifactVersion?: number;
    cameraAnchorId?: string;
    viewId?: string;
    spatialContext?: LockedSceneAssetPackProjection;
    gatewayInput: {
        schemaVersion: number;
        mode: "image" | "video" | "audio" | string;
        prompt: string;
        aspectRatio?: string;
        durationMs?: number;
        outputIntent: string;
        generationRequestArtifactId: string;
        generationRequestArtifactVersion: number;
        promptArtifactId: string;
        promptArtifactVersion: number;
        sourceRefs: string[];
        spatialPackArtifactId?: string;
        spatialPackArtifactVersion?: number;
        spatialGateArtifactId?: string;
        spatialGateArtifactVersion?: number;
        cameraAnchorId?: string;
        viewId?: string;
        spatialContext?: LockedSceneAssetPackProjection;
    };
};

// Provider route discovery is deliberately read-only. It exposes only
// backend-managed system-channel readiness, never a base URL or credential.
export type FilmGenerationProviderRoute = {
    channelId: string;
    channelName: string;
    model: string;
    modelDisplayName: string;
    capability: "image" | "video" | "audio" | string;
    protocol: string;
    billingMode: string;
    unitPriceMicrocredits?: number;
    priceConfigured: boolean;
    capabilityVersion?: number;
    providerReady: boolean;
    billingReady: boolean;
    routeReady: boolean;
    blockers: string[] | null;
};

export type FilmGenerationProviderRouteCatalog = {
    schemaVersion: number;
    generationRequestArtifactId: string;
    generationRequestArtifactVersion: number;
    mode: "image" | "video" | "audio" | string;
    requestReady: boolean;
    state: string;
    blockers: string[] | null;
    requestFingerprint: string;
    routes: FilmGenerationProviderRoute[];
};

export type FilmProviderJobExecution = {
    id: string;
    providerRequestId: string;
    status: "accepted" | "running" | "succeeded" | "failed" | "cancellation_requested" | "cancelled" | "uncertain" | string;
    providerStatus?: string;
    pollStage?: string;
    lastError?: string;
    firstObservedAt: string;
    lastObservedAt: string;
    completedAt?: string;
};

export type FilmGenerationResult = {
    id: string;
    kind: "film_generation_result" | string;
    url?: string;
    payload: string;
    createdAt: string;
};

export type FilmGenerationAttemptExecution = {
    id: string;
    taskId: string;
    attemptNumber: number;
    generationRequestArtifactVersion: number;
    canvasId: string;
    canvasNodeId: string;
    unitId?: string;
    sceneId?: string;
    shotId?: string;
    requestFingerprint: string;
    billingOrderId?: string;
    channelId: string;
    channelModelId: string;
    model: string;
    capability: string;
    protocol: string;
    capabilityVersion: number;
    priceVersion: number;
    status: "queued" | "running" | "succeeded" | "failed" | "cancelled" | "uncertain" | string;
    error?: string;
    startedAt?: string;
    completedAt?: string;
    createdAt: string;
    updatedAt: string;
    providerJobs: FilmProviderJobExecution[];
    results: FilmGenerationResult[];
};

export type FilmGenerationExecutionHistory = {
    schemaVersion: number;
    projectId: string;
    generationRequestArtifactId: string;
    generationRequestArtifactVersion: number;
    attempts: FilmGenerationAttemptExecution[];
};

export type FilmGenerationTaskSubmission = {
    schemaVersion: number;
    generationRequestArtifactId: string;
    generationRequestArtifactVersion: number;
    requestFingerprint: string;
    canvasId: string;
    canvasNodeId: string;
    projectionRevision: number;
    created: boolean;
    idempotentReplay: boolean;
    task: GenerationTask;
};

export type EcommerceArtifact = {
    id: string;
    projectId: string;
    artifactKey: string;
    artifactType: string;
    schemaVersion: number;
    revision: number;
    lifecycle: "draft" | "review" | "finalized" | "superseded" | "archived" | string;
    evidence: "recorded" | "inferred" | "unknown" | string;
    responsibleAgentId?: string;
    skillRef?: string;
    payloadJson: string;
    sourceRefsJson: string;
    authorityRefsJson: string;
    createdAt: string;
    updatedAt: string;
};

export type ShotAssetReference = {
    id: string;
    shotId: string;
    assetVersionId: string;
    role: "reference" | "start_frame" | "end_frame" | "keyframe" | "storyboard" | "output" | string;
    status: string;
    createdAt: string;
};

export type WorkflowStep = {
    id: string;
    workflowInstanceId: string;
    stepKey: string;
    name: string;
    position: number;
    status: "pending" | "ready" | "running" | "review" | "completed" | "failed" | "skipped" | string;
    error?: string;
    updatedAt: string;
};

export type ProjectWorkflow = {
    instance: { id: string; projectId: string; unitId?: string; scope: string; status: string; revision: number };
    steps: WorkflowStep[];
};

export type ProjectSummary = {
    project: Project;
    canvasCount: number;
    assetCount: number;
    unitCount: number;
    completedUnitCount: number;
};

export type ProjectDetail = {
    project: Project;
    units: ProjectUnit[];
    canvases: ProjectCanvas[];
    canvasUnitLinks: CanvasUnitLink[];
    assets: ProjectAsset[];
    workflows: ProjectWorkflow[];
    scenes: ProjectScene[];
    shots: ProjectShot[];
    filmArtifacts?: FilmArtifact[];
    ecommerceArtifacts?: EcommerceArtifact[];
    shotReferences: ShotAssetReference[];
    assetCandidates: ProjectAssetCandidate[];
};

export function listProjects() {
    return request<{ projects: ProjectSummary[] }>(api.get("/projects"));
}

export function getProject(id: string) {
    return request<ProjectDetail>(api.get(`/projects/${encodeURIComponent(id)}`));
}

export function createProject(input: { name: string; type: string; aspectRatio: string; sourceType: string; description?: string; stylePresetId?: string; styleProfileJson?: string }) {
    return request<{ project: Project }>(api.post("/projects", input));
}

export function updateProject(projectId: string, input: Partial<Pick<Project, "name" | "type" | "aspectRatio" | "sourceType" | "description" | "stylePresetId" | "styleProfileJson" | "status">>) {
    return request<{ project: Project }>(api.patch(`/projects/${encodeURIComponent(projectId)}`, input));
}

export function deleteProject(projectId: string) {
    return request<{ id: string }>(api.delete(`/projects/${encodeURIComponent(projectId)}`));
}

export function createProjectUnit(projectId: string, input: { kind: string; title: string; sourceText?: string; position?: number }) {
    return request<{ unit: ProjectUnit }>(api.post(`/projects/${encodeURIComponent(projectId)}/units`, input));
}

export function getProjectUnit(projectId: string, unitId: string) {
    return request<{ unit: ProjectUnit }>(api.get(`/projects/${encodeURIComponent(projectId)}/units/${encodeURIComponent(unitId)}`));
}

export function importProjectUnits(projectId: string, units: Array<{ kind: string; title: string; sourceText?: string }>) {
    return request<{ units: ProjectUnit[] }>(api.post(`/projects/${encodeURIComponent(projectId)}/units/import`, { units }));
}

export function reorderProjectUnits(projectId: string, unitIds: string[]) {
    return request<{ unitIds: string[] }>(api.patch(`/projects/${encodeURIComponent(projectId)}/units/reorder`, { unitIds }));
}

export function updateProjectUnit(projectId: string, unitId: string, input: { title?: string; sourceText: string; status?: ProjectUnit["status"] }) {
    return request<{ unit: ProjectUnit }>(api.patch(`/projects/${encodeURIComponent(projectId)}/units/${encodeURIComponent(unitId)}`, input));
}

export function deleteProjectUnit(projectId: string, unitId: string) {
    return request<{ id: string }>(api.delete(`/projects/${encodeURIComponent(projectId)}/units/${encodeURIComponent(unitId)}`));
}

export function linkCanvasUnit(projectId: string, input: { canvasId: string; unitId: string; role?: string }) {
    return request<{ link: { id: string; projectId: string; canvasId: string; unitId: string; role: string } }>(api.post(`/projects/${encodeURIComponent(projectId)}/canvas-links`, input));
}

export function unlinkCanvasUnit(projectId: string, canvasId: string, unitId: string) {
    return request<{ canvasId: string; unitId: string }>(api.delete(`/projects/${encodeURIComponent(projectId)}/canvas-links/${encodeURIComponent(canvasId)}/units/${encodeURIComponent(unitId)}`));
}

export function unlinkCanvasProject(projectId: string, canvasId: string) {
    return request<{ canvasId: string }>(api.delete(`/projects/${encodeURIComponent(projectId)}/canvases/${encodeURIComponent(canvasId)}`));
}

export function linkProjectAsset(projectId: string, input: { assetId: string; category: string }) {
    return request<{ asset: ProjectAsset }>(api.post(`/projects/${encodeURIComponent(projectId)}/assets`, input));
}

export function unlinkProjectAsset(projectId: string, assetId: string) {
    return request<{ id: string }>(api.delete(`/projects/${encodeURIComponent(projectId)}/assets/${encodeURIComponent(assetId)}`));
}

export function updateProjectAssetCategory(projectId: string, assetId: string, category: string) {
    return request<{ asset: ProjectAsset }>(api.patch(`/projects/${encodeURIComponent(projectId)}/assets/${encodeURIComponent(assetId)}`, { category }));
}

export function createProjectAssetVersion(projectId: string, assetId: string, input: { title?: string; prompt?: string; definitionJson?: string; note?: string }) {
    return request<{ version: { id: string; assetId: string; version: number; status: string; definitionJson: string; note: string } }>(api.post(`/projects/${encodeURIComponent(projectId)}/assets/${encodeURIComponent(assetId)}/versions`, input));
}

export function listVoiceProfiles() {
    return request<{ profiles: VoiceProfile[] }>(api.get("/voice-profiles"));
}

export function createProjectCharacter(projectId: string, input: { name: string; definition?: Record<string, unknown> }) {
    return request<ProjectCharacterDetail>(api.post(`/projects/${encodeURIComponent(projectId)}/characters`, input));
}

export function getProjectCharacter(projectId: string, assetId: string) {
    return request<ProjectCharacterDetail>(api.get(`/projects/${encodeURIComponent(projectId)}/characters/${encodeURIComponent(assetId)}`));
}

export function updateProjectCharacter(projectId: string, assetId: string, input: { name: string; definition: Record<string, unknown> }) {
    return request<ProjectCharacterDetail>(api.patch(`/projects/${encodeURIComponent(projectId)}/characters/${encodeURIComponent(assetId)}`, input));
}

export function replaceProjectCharacterRepresentations(projectId: string, assetId: string, representations: Array<{ role: string; resourceId: string; metadata?: Record<string, unknown> }>) {
    return request<ProjectCharacterDetail>(api.put(`/projects/${encodeURIComponent(projectId)}/characters/${encodeURIComponent(assetId)}/representations`, { representations }));
}

export function bindProjectCharacterVoice(projectId: string, assetId: string, input: { voiceProfileId: string; instructions?: string }) {
    return request<ProjectCharacterDetail>(api.put(`/projects/${encodeURIComponent(projectId)}/characters/${encodeURIComponent(assetId)}/voice`, input));
}

export function unbindProjectCharacterVoice(projectId: string, assetId: string) {
    return request<ProjectCharacterDetail>(api.delete(`/projects/${encodeURIComponent(projectId)}/characters/${encodeURIComponent(assetId)}/voice`));
}

export function createUnitWorkflow(projectId: string, unitId: string) {
    return request<{ workflow: ProjectWorkflow }>(api.post(`/projects/${encodeURIComponent(projectId)}/workflows`, { unitId }));
}

export function saveProjectShot(projectId: string, input: { id?: string; unitId?: string; sceneId?: string; title: string; description?: string; position?: number; durationMs?: number; status?: string; contract?: Record<string, unknown> }) {
    return request<{ shot: ProjectShot }>(api.post(`/projects/${encodeURIComponent(projectId)}/shots`, input));
}

export function saveProjectFilmArtifact(projectId: string, input: { shotId: string; artifactType: "acting" | "video_prompt_pack" | "generation_request"; status?: string; responsibleAgentId?: string; payload: Record<string, unknown> }) {
    return request<{ artifact: FilmArtifact }>(api.post(`/projects/${encodeURIComponent(projectId)}/film-artifacts`, input));
}

export function getProjectFilmGenerationTaskDraft(projectId: string, artifactId: string) {
    return request<{ taskDraft: FilmGenerationTaskDraft }>(api.get(`/projects/${encodeURIComponent(projectId)}/film-generation-requests/${encodeURIComponent(artifactId)}/task-draft`));
}

export function getProjectFilmGenerationProviderRoutes(projectId: string, artifactId: string) {
    return request<{ providerRoutes: FilmGenerationProviderRouteCatalog }>(api.get(`/projects/${encodeURIComponent(projectId)}/film-generation-requests/${encodeURIComponent(artifactId)}/provider-routes`));
}

export function submitProjectFilmGenerationTask(projectId: string, artifactId: string, input: { canvasId: string; canvasNodeId: string; requestFingerprint: string; channelId: string; model: string }) {
    return request<{ submission: FilmGenerationTaskSubmission }>(api.post(`/projects/${encodeURIComponent(projectId)}/film-generation-requests/${encodeURIComponent(artifactId)}/tasks`, input));
}

export function getProjectFilmGenerationExecutions(projectId: string, artifactId: string) {
    return request<{ executionHistory: FilmGenerationExecutionHistory }>(api.get(`/projects/${encodeURIComponent(projectId)}/film-generation-requests/${encodeURIComponent(artifactId)}/executions`));
}

export function listProjectEcommerceArtifacts(projectId: string) {
    return request<{ artifacts: EcommerceArtifact[] }>(api.get(`/projects/${encodeURIComponent(projectId)}/ecommerce-artifacts`));
}

export function saveProjectEcommerceArtifact(
    projectId: string,
    input: {
        artifactKey: string;
        artifactType: string;
        schemaVersion: number;
        lifecycle?: EcommerceArtifact["lifecycle"];
        evidence?: EcommerceArtifact["evidence"];
        responsibleAgentId?: string;
        skillRef?: string;
        payload: Record<string, unknown>;
        sourceRefs: string[];
        authorityRefs?: string[];
    },
) {
    return request<{ artifact: EcommerceArtifact }>(api.post(`/projects/${encodeURIComponent(projectId)}/ecommerce-artifacts`, input));
}

export function saveProjectScene(projectId: string, input: { id?: string; unitId?: string; code?: string; title: string; description?: string; interiorExterior?: string; timeOfDay?: string; locationAssetId?: string; position?: number; status?: string }) {
    return request<{ scene: ProjectScene }>(api.post(`/projects/${encodeURIComponent(projectId)}/scenes`, input));
}

export function getProjectSceneAssetPack(projectId: string, sceneId: string) {
    return request<{ sceneAssetPack: SceneAssetPackDetail }>(api.get(`/projects/${encodeURIComponent(projectId)}/scenes/${encodeURIComponent(sceneId)}/asset-pack`));
}

export function saveProjectSceneAssetPack(projectId: string, sceneId: string, input: { status?: string; responsibleAgentId?: string; expectedVersion: number; payload: Record<string, unknown> }) {
    return request<{ sceneAssetPack: SceneAssetPackSaveResult }>(api.post(`/projects/${encodeURIComponent(projectId)}/scenes/${encodeURIComponent(sceneId)}/asset-pack`, input));
}

export function replaceProjectUnitShots(projectId: string, unitId: string, shots: Array<{ title: string; description: string; durationMs: number }>) {
    return request<{ shots: ProjectShot[] }>(api.put(`/projects/${encodeURIComponent(projectId)}/units/${encodeURIComponent(unitId)}/shots`, { shots }));
}

export function linkShotAsset(projectId: string, shotId: string, input: { assetVersionId: string; role: ShotAssetReference["role"] }) {
    return request<{ reference: ShotAssetReference }>(api.post(`/projects/${encodeURIComponent(projectId)}/shots/${encodeURIComponent(shotId)}/assets`, input));
}

export function createProjectAssetCandidates(projectId: string, candidates: Array<{ unitId?: string; shotId?: string; name: string; category: string; details?: Record<string, unknown> }>) {
    return request<{ candidates: ProjectAssetCandidate[] }>(api.post(`/projects/${encodeURIComponent(projectId)}/asset-candidates`, { candidates }));
}

export function confirmProjectAssetCandidate(projectId: string, candidateId: string, assetId?: string) {
    return request<{ asset: ProjectAsset }>(api.post(`/projects/${encodeURIComponent(projectId)}/asset-candidates/${encodeURIComponent(candidateId)}/confirm`, { assetId: assetId || "" }));
}

export function updateWorkflowStep(projectId: string, stepId: string, input: { status: string; outputJson?: string; error?: string }) {
    return request<{ step: WorkflowStep }>(api.patch(`/projects/${encodeURIComponent(projectId)}/workflow-steps/${encodeURIComponent(stepId)}`, input));
}

export function registerProjectTaskOutput(projectId: string, stepId: string, input: { taskId: string; assetVersionId?: string; resourceId?: string; mediaType?: string; role?: string; metadataJson?: string; outputJson?: string }) {
    return request<{ step: WorkflowStep }>(api.post(`/projects/${encodeURIComponent(projectId)}/workflow-steps/${encodeURIComponent(stepId)}/task-output`, input));
}
