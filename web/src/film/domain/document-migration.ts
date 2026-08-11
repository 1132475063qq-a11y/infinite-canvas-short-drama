import { createFilmNodeState, isFilmNodeKind, type FilmNodeKind } from "@/film/domain/types";
import type { CanvasNodeData, CanvasProjectDocument } from "@/types/canvas";

const LEGACY_WORKFLOW_KIND_MAP: Record<string, FilmNodeKind> = {
    script: "script",
    story_input: "story",
    character: "character",
    scene: "scene",
    storyboard: "script",
    shot: "shot",
    final: "result",
};

export function migrateCanvasProjectDocument<T extends CanvasProjectDocument>(project: T): T {
    const nodes = project.nodes.map(migrateCanvasNode);
    return {
        ...project,
        schemaVersion: Math.max(2, project.schemaVersion || 1),
        layout: { gridSize: project.layout?.gridSize || 8 },
        nodes,
    };
}

export function migrateCanvasNode(node: CanvasNodeData): CanvasNodeData {
    const legacyKind = node.metadata?.workflowKind ? LEGACY_WORKFLOW_KIND_MAP[node.metadata.workflowKind] : undefined;
    const filmKind = isFilmNodeKind(node.filmKind) ? node.filmKind : legacyKind;
    if (!filmKind) return node;
    return {
        ...node,
        filmKind,
        filmState: createFilmNodeState(node.filmState),
        layout: node.layout || { mode: node.metadata?.locked ? "pinned" : "manual" },
    };
}
