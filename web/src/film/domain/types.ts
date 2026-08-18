export const FILM_NODE_KINDS = [
    "project",
    "story",
    "script",
    "scene",
    "character",
    "character_state",
    "location",
    "location_view",
    "prop",
    "voice_profile",
    "acting",
    "acting_profile",
    "scene_acting",
    "storyboard",
    "shot",
    "shot_reference_pack",
    "prompt_pack",
    "generation",
    "generation_attempt",
    "result",
    "qc",
    "continuity",
    "retry",
    "needs_you",
    "agent_task",
    "delivery",
] as const;

export type FilmNodeKind = (typeof FILM_NODE_KINDS)[number];

export type FilmNodeDomainRef = {
    projectId: string;
    unitId?: string;
    sceneId?: string;
    shotId?: string;
    artifactId?: string;
    artifactVersion?: string;
    assetId?: string;
    assetVersionId?: string;
    taskId?: string;
    generationAttemptId?: string;
    providerJobId?: string;
    resultId?: string;
    resourceId?: string;
    agentId?: string;
    skillId?: string;
};

export type FilmNodeState = {
    lifecycle: "draft" | "review" | "locked" | "superseded" | "archived";
    production: "not_started" | "ready" | "running" | "blocked" | "generated" | "qc_failed" | "approved";
    evidence: "recorded" | "inferred" | "unknown";
    attention: "none" | "warning" | "error" | "human_required";
};

export type FilmNodeLayout = {
    mode: "auto" | "manual" | "pinned";
    lane?: string;
    order?: number;
};

export type FilmEdgeType = "dependency" | "reference" | "derivation" | "continuity" | "authority";

export type FilmPortType =
    | "story"
    | "script"
    | "scene"
    | "character"
    | "location"
    | "prop"
    | "acting"
    | "reference"
    | "prompt"
    | "image"
    | "video"
    | "audio"
    | "generation"
    | "qc"
    | "continuity"
    | "decision"
    | "artifact";

// Film ports describe production meaning. They deliberately sit beside the canvas
// renderer handles so the existing free-form connection UI stays backwards compatible.
export type FilmNodePort =
    | "scene_context"
    | "shot_contract"
    | "asset_reference"
    | "acting_direction"
    | "prompt"
    | "generation_job"
    | "result"
    | "qc_decision"
    | "retry_request"
    | "generic";

export type FilmPortSchema = {
    id: FilmNodePort;
    direction: "input" | "output";
    type: FilmPortType;
    role: string;
    required: boolean;
    multiple: boolean;
    accepts: readonly FilmPortType[];
};

export type FilmConnectionPorts = {
    from: FilmNodePort;
    to: FilmNodePort;
};

export const DEFAULT_FILM_NODE_STATE: FilmNodeState = {
    lifecycle: "draft",
    production: "not_started",
    evidence: "unknown",
    attention: "none",
};

export function createFilmNodeState(patch: Partial<FilmNodeState> = {}): FilmNodeState {
    return { ...DEFAULT_FILM_NODE_STATE, ...patch };
}

export function isFilmNodeKind(value: unknown): value is FilmNodeKind {
    return typeof value === "string" && (FILM_NODE_KINDS as readonly string[]).includes(value);
}
