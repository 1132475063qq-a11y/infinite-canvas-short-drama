import { applyFilmResultLayout } from "@/lib/canvas/layout/layout-engine";
import type { GenerationTask } from "@/services/api/task-center";
import { CanvasNodeType, type CanvasConnection, type CanvasNodeData, type CanvasNodeMetadata, type Position } from "@/types/canvas";
import type { CanvasGridSize } from "@/lib/canvas/layout/layout-types";

// Keep this browser-side immediate projection aligned with the durable server
// projection in backend/internal/model/models_project.go.
export const FILM_GENERATION_RESULT_NODE_ID_PREFIX = "film-generation-result:";
export const FILM_GENERATION_RESULT_EDGE_ID_PREFIX = "film-generation-result-edge:";

type VideoResultMedia = {
    content: string;
    storageKey?: string;
    mimeType?: string;
    width?: number;
    height?: number;
    bytes?: number;
    durationMs?: number;
};

export function filmGenerationResultNodeId(taskId: string) {
    return `${FILM_GENERATION_RESULT_NODE_ID_PREFIX}${taskId}`;
}

export function filmGenerationResultConnectionId(taskId: string) {
    return `${FILM_GENERATION_RESULT_EDGE_ID_PREFIX}${taskId}`;
}

export function isFilmGenerationResultMediaNode(node: CanvasNodeData | undefined) {
    return Boolean(node?.filmKind === "result" && node.type === CanvasNodeType.Video && node.metadata?.content);
}

export function hasFilmGenerationResultMedia(nodes: readonly CanvasNodeData[], taskId: string) {
    return isFilmGenerationResultMediaNode(nodes.find((node) => node.id === filmGenerationResultNodeId(taskId)));
}

/** Read the original request node from the immutable Task input before falling back to its current binding. */
export function filmGenerationTaskCanvasNodeId(task: GenerationTask) {
    const input = parseRecord(task.inputJson);
    const metadata = asRecord(input?.metadata);
    return firstString(input?.canvasNodeId, metadata?.canvasNodeId, metadata?.nodeId);
}

export function videoResultMediaFromGenerationTask(task: GenerationTask): VideoResultMedia | null {
    const result = parseRecord(task.resultJson);
    if (!result) return null;
    if (task.type !== "canvas_video" && result.mode !== "video" && result.video === undefined) return null;

    const rawVideo = result.video === undefined ? result : result.video;
    if (typeof rawVideo === "string") {
        const content = rawVideo.trim();
        return content ? { content, mimeType: "video/mp4" } : null;
    }
    const video = asRecord(rawVideo);
    if (!video) return null;

    const content = firstString(video.dataUrl, video.url, video.content, video.resultUrl, video.outputUrl);
    if (!content) return null;
    return {
        content,
        storageKey: firstString(video.storageKey),
        mimeType: firstString(video.mimeType) || "video/mp4",
        width: nonNegativeInteger(video.width),
        height: nonNegativeInteger(video.height),
        bytes: nonNegativeInteger(video.bytes),
        durationMs: nonNegativeInteger(video.durationMs),
    };
}

/**
 * Create or refresh the visual result projection only after a Task has already
 * reached its terminal success state. This function deliberately has no
 * knowledge of Attempt, ProviderJob, Result, or Resource identifiers: those
 * remain server-owned facts and arrive with the next Canvas read projection.
 */
export function projectFilmGenerationResultNode(nodes: CanvasNodeData[], task: GenerationTask, fallbackGenerationNodeId: string, gridSize: CanvasGridSize = 8) {
    if (task.status !== "succeeded") return nodes;
    const media = videoResultMediaFromGenerationTask(task);
    if (!media) return nodes;

    const sourceNodeId = filmGenerationTaskCanvasNodeId(task) || fallbackGenerationNodeId;
    const generationNode = nodes.find((node) => node.id === sourceNodeId && node.filmKind === "generation");
    if (!generationNode?.domainRef?.projectId) return nodes;

    const resultNodeId = filmGenerationResultNodeId(task.id);
    const existing = nodes.find((node) => node.id === resultNodeId);
    const nextResult = createFilmGenerationResultNode(generationNode, task, media, existing, resultNodeId);
    if (existing) {
        return nodes.map((node) => (node.id === resultNodeId ? nextResult : node));
    }

    return applyFilmResultLayout([...nodes, nextResult], resultOrigin(generationNode), gridSize);
}

export function ensureFilmGenerationResultConnection(connections: CanvasConnection[], taskId: string, generationNodeId: string) {
    if (!taskId.trim() || !generationNodeId.trim()) return connections;
    const resultNodeId = filmGenerationResultNodeId(taskId);
    const canonicalId = filmGenerationResultConnectionId(taskId);
    const canonical = {
        id: canonicalId,
        fromNodeId: generationNodeId,
        toNodeId: resultNodeId,
        edgeType: "derivation" as const,
        filmPorts: { from: "generation_job" as const, to: "generation_job" as const },
    };
    const matchingIndex = connections.findIndex((connection) => connection.id === canonicalId || (connection.fromNodeId === generationNodeId && connection.toNodeId === resultNodeId));
    if (matchingIndex < 0) return [...connections, canonical];

    const current = connections[matchingIndex];
    const next = { ...current, ...canonical, id: current.id || canonicalId };
    if (
        current.id === next.id
        && current.fromNodeId === next.fromNodeId
        && current.toNodeId === next.toNodeId
        && current.edgeType === next.edgeType
        && current.filmPorts?.from === next.filmPorts.from
        && current.filmPorts?.to === next.filmPorts.to
    ) return connections;
    return connections.map((connection, index) => (index === matchingIndex ? next : connection));
}

/**
 * Keep only browser/system-managed result edges in sync with visible nodes.
 * A result never guesses a replacement source when its recorded generation
 * node is gone, so a delayed task response cannot attach to another request.
 */
export function reconcileFilmGenerationResultConnections(nodes: readonly CanvasNodeData[], connections: CanvasConnection[]) {
    const sourcesByTaskId = new Map<string, string>();
    for (const resultNode of nodes) {
        if (!isFilmGenerationResultMediaNode(resultNode)) continue;
        const taskId = firstString(resultNode.domainRef?.taskId, resultNode.metadata?.taskId);
        if (!taskId || sourcesByTaskId.has(taskId)) continue;

        const sourceNodeId = filmGenerationResultSourceNodeId(nodes, resultNode, taskId);
        if (sourceNodeId) sourcesByTaskId.set(taskId, sourceNodeId);
    }

    let next = connections.filter((connection) => {
        const taskId = taskIdFromFilmGenerationResultConnection(connection.id);
        return !taskId || sourcesByTaskId.has(taskId);
    });
    for (const [taskId, sourceNodeId] of sourcesByTaskId) {
        next = ensureFilmGenerationResultConnection(next, taskId, sourceNodeId);
    }
    return next;
}

function createFilmGenerationResultNode(generationNode: CanvasNodeData, task: GenerationTask, media: VideoResultMedia, existing: CanvasNodeData | undefined, resultNodeId: string): CanvasNodeData {
    const defaultSize = videoNodeSize(media);
    const title = existing?.title?.trim() || `${generationNode.title || "视频生成"} · 视频结果`;
    const metadata: CanvasNodeMetadata = {
        ...existing?.metadata,
        content: media.content,
        storageKey: media.storageKey,
        status: "success",
        mimeType: media.mimeType || "video/mp4",
        taskId: task.id,
        taskStatus: "succeeded",
        taskProgress: 100,
        taskStage: task.stage || "任务完成",
        taskCreatedAt: task.createdAt || task.created_at,
        taskUpdatedAt: task.updatedAt || task.updated_at,
        generationSourceNodeId: generationNode.id,
        naturalWidth: media.width,
        naturalHeight: media.height,
        bytes: media.bytes,
        durationMs: media.durationMs,
        errorDetails: undefined,
        generationErrorCode: undefined,
        failedPromptFingerprint: undefined,
    };
    // A first browser projection only knows a Task and media display data.
    // Runtime execution identities may be retained solely from an existing
    // server projection; they must never be copied from a generation node.
    const inherited = existing?.domainRef;
    const domainRef = {
        projectId: generationNode.domainRef.projectId,
        unitId: generationNode.domainRef.unitId,
        sceneId: generationNode.domainRef.sceneId,
        shotId: generationNode.domainRef.shotId,
        artifactId: generationNode.domainRef.artifactId,
        artifactVersion: generationNode.domainRef.artifactVersion,
        taskId: task.id,
        // Preserve real IDs from a server projection, but never manufacture any
        // execution facts merely because an URL or storage key looks familiar.
        generationAttemptId: inherited?.generationAttemptId,
        providerJobId: inherited?.providerJobId,
        resultId: inherited?.resultId,
        resourceId: inherited?.resourceId,
    };

    return {
        id: resultNodeId,
        type: CanvasNodeType.Video,
        filmKind: "result",
        domainRef,
        filmState: { lifecycle: "locked", production: "generated", evidence: "recorded", attention: "none" },
        layout: existing?.layout || { mode: "auto", lane: "shot_pipeline" },
        title,
        position: existing?.position || resultOrigin(generationNode),
        width: existing?.width || defaultSize.width,
        height: existing?.height || defaultSize.height,
        ...(existing?.parentId ? { parentId: existing.parentId } : {}),
        metadata,
    };
}

function resultOrigin(generationNode: CanvasNodeData): Position {
    return { x: generationNode.position.x + generationNode.width + 64, y: generationNode.position.y };
}

function filmGenerationResultSourceNodeId(nodes: readonly CanvasNodeData[], resultNode: CanvasNodeData, taskId: string) {
    const explicitSourceNodeId = firstString(resultNode.metadata?.generationSourceNodeId);
    if (explicitSourceNodeId) {
        return nodes.some((node) => node.id === explicitSourceNodeId && node.filmKind === "generation") ? explicitSourceNodeId : "";
    }
    return nodes.find((node) => node.filmKind === "generation" && firstString(node.domainRef?.taskId, node.metadata?.taskId) === taskId)?.id || "";
}

function taskIdFromFilmGenerationResultConnection(connectionId: string) {
    return connectionId.startsWith(FILM_GENERATION_RESULT_EDGE_ID_PREFIX)
        ? connectionId.slice(FILM_GENERATION_RESULT_EDGE_ID_PREFIX.length).trim()
        : "";
}

function videoNodeSize(media: VideoResultMedia) {
    if (media.width && media.height) {
        const width = Math.min(480, media.width);
        const height = Math.round(width * media.height / media.width);
        if (height > 0) return { width, height };
    }
    return { width: 360, height: 203 };
}

function parseRecord(value: string | undefined) {
    if (!value?.trim()) return null;
    try {
        return asRecord(JSON.parse(value));
    } catch {
        return null;
    }
}

function asRecord(value: unknown): Record<string, unknown> | null {
    return value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : null;
}

function firstString(...values: unknown[]) {
    for (const value of values) {
        if (typeof value === "string" && value.trim()) return value.trim();
    }
    return "";
}

function nonNegativeInteger(value: unknown) {
    const number = typeof value === "number" ? value : typeof value === "string" ? Number(value) : NaN;
    return Number.isFinite(number) && number > 0 ? Math.round(number) : undefined;
}
