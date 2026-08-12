import { createFilmNodeState, isFilmNodeKind, type FilmNodeKind } from "@/film/domain/types";
import { normalizeFilmSceneTitle } from "@/film/domain/scene-projection";
import { validateFilmConnection } from "@/film/domain/edge-contract";
import type { CanvasConnection, CanvasNodeData, CanvasProjectDocument } from "@/types/canvas";

const LEGACY_WORKFLOW_KIND_MAP: Record<string, FilmNodeKind> = {
    script: "script",
    story_input: "story",
    character: "character",
    scene: "scene",
    storyboard: "script",
    shot: "shot",
    final: "result",
};

export function migrateCanvasProjectDocument<T extends CanvasProjectDocument & { connections?: CanvasConnection[] }>(project: T): T {
    const nodes = project.nodes.map(migrateCanvasNode);
    const connections = project.connections?.map((connection) => migrateFilmConnection(connection, nodes));
    return {
        ...project,
        schemaVersion: Math.max(2, project.schemaVersion || 1),
        layout: { gridSize: project.layout?.gridSize || 8 },
        nodes,
        ...(connections ? { connections } : {}),
    };
}

export function migrateFilmConnection(connection: CanvasConnection, nodes: CanvasNodeData[]): CanvasConnection {
    const from = nodes.find((node) => node.id === connection.fromNodeId);
    const to = nodes.find((node) => node.id === connection.toNodeId);
    const validation = validateFilmConnection(from, to);
    if (!validation.allowed) {
        const { edgeType: _edgeType, filmPorts: _filmPorts, ...freeformConnection } = connection;
        return freeformConnection;
    }
    if (!validation.semantic) return connection;
    return { ...connection, ...validation.semantic };
}

export function migrateCanvasNode(node: CanvasNodeData): CanvasNodeData {
    const legacyKind = node.metadata?.workflowKind ? LEGACY_WORKFLOW_KIND_MAP[node.metadata.workflowKind] : undefined;
    const filmKind = isFilmNodeKind(node.filmKind) ? node.filmKind : legacyKind;
    if (!filmKind) return node;
    return {
        ...node,
        title: filmKind === "scene" ? normalizeFilmSceneTitle(node.title) : node.title,
        filmKind,
        filmState: createFilmNodeState(node.filmState),
        layout: node.layout || { mode: node.metadata?.locked ? "pinned" : "manual" },
    };
}
