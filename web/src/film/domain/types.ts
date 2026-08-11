export type FilmNodeKind =
    | "project"
    | "story"
    | "script"
    | "scene"
    | "character"
    | "location"
    | "prop"
    | "acting"
    | "prompt_pack"
    | "generation"
    | "result"
    | "qc"
    | "retry"
    | "needs_you"
    | "agent_task";

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
    return typeof value === "string" && ["project", "story", "script", "scene", "character", "location", "prop", "acting", "prompt_pack", "generation", "result", "qc", "retry", "needs_you", "agent_task"].includes(value);
}
