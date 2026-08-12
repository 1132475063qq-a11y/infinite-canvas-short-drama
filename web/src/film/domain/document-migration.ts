import { createFilmNodeState, isFilmNodeKind, type FilmNodeKind } from "@/film/domain/types";
import { normalizeFilmSceneTitle } from "@/film/domain/scene-projection";
import { validateFilmConnection } from "@/film/domain/edge-contract";
import { normalizeCanvasGridSize } from "@/lib/canvas/layout/layout-types";
import { CANVAS_DOCUMENT_SCHEMA_VERSION, CanvasNodeType, type CanvasConnection, type CanvasDocumentGroup, type CanvasDocumentV2, type CanvasNodeData, type ViewportTransform } from "@/types/canvas";

const LEGACY_WORKFLOW_KIND_MAP: Record<string, FilmNodeKind> = {
    script: "script",
    story_input: "story",
    character: "character",
    scene: "scene",
    storyboard: "script",
    shot: "shot",
    final: "result",
};

type CanvasDocumentMigrationInput = {
    schemaVersion?: number;
    projectId?: string;
    nodes?: CanvasNodeData[];
    connections?: CanvasConnection[];
    edges?: CanvasConnection[];
    viewport?: Partial<ViewportTransform>;
    groups?: CanvasDocumentGroup[];
    layout?: { gridSize?: unknown; grid?: unknown };
};

const DEFAULT_VIEWPORT: ViewportTransform = { x: 0, y: 0, k: 1 };

export function migrateCanvasProjectDocument<T extends CanvasDocumentMigrationInput>(project: T): Omit<T, "edges" | "schemaVersion" | "nodes" | "connections" | "viewport" | "groups" | "layout"> & CanvasDocumentV2 {
    const nodes = (project.nodes || []).map(migrateCanvasNode);
    const sourceConnections = project.connections || project.edges || [];
    const connections = sourceConnections.map((connection) => migrateFilmConnection(connection, nodes));
    const {
        edges: _legacyEdges,
        schemaVersion: _legacySchemaVersion,
        nodes: _legacyNodes,
        connections: _legacyConnections,
        viewport: _legacyViewport,
        groups: _legacyGroups,
        layout: _legacyLayout,
        ...rest
    } = project;
    return {
        ...rest,
        // Future documents keep their higher version so a newer client can still identify them.
        schemaVersion: Math.max(CANVAS_DOCUMENT_SCHEMA_VERSION, finiteNumber(project.schemaVersion, 1)),
        layout: { gridSize: normalizeCanvasGridSize(project.layout?.gridSize ?? project.layout?.grid) },
        nodes,
        connections,
        viewport: normalizeCanvasViewport(project.viewport),
        groups: deriveCanvasDocumentGroups(nodes, normalizeCanvasDocumentGroups(project.groups)),
    };
}

export function deriveCanvasDocumentGroups(nodes: readonly CanvasNodeData[], existingGroups: readonly CanvasDocumentGroup[] = []): CanvasDocumentGroup[] {
    const groupsById = new Map(existingGroups.filter((group) => group.kind === "custom").map((group) => [group.id, { ...group, nodeIds: [...group.nodeIds] }]));
    const childrenByParent = new Map<string, string[]>();
    nodes.forEach((node) => {
        if (!node.parentId) return;
        const children = childrenByParent.get(node.parentId) || [];
        children.push(node.id);
        childrenByParent.set(node.parentId, children);
    });
    nodes.forEach((node) => {
        if (node.type !== CanvasNodeType.Frame && !childrenByParent.has(node.id)) return;
        groupsById.set(node.id, {
            id: node.id,
            kind: node.filmKind === "scene" ? "scene" : "frame",
            nodeIds: childrenByParent.get(node.id) || [],
            title: node.title,
            collapsed: node.metadata?.frame?.collapsed,
        });
    });
    return [...groupsById.values()];
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

function normalizeCanvasViewport(viewport?: Partial<ViewportTransform>): ViewportTransform {
    const x = finiteNumber(viewport?.x, DEFAULT_VIEWPORT.x);
    const y = finiteNumber(viewport?.y, DEFAULT_VIEWPORT.y);
    const k = finiteNumber(viewport?.k, DEFAULT_VIEWPORT.k);
    return { x, y, k: k > 0 ? k : DEFAULT_VIEWPORT.k };
}

function normalizeCanvasDocumentGroups(groups?: CanvasDocumentGroup[]): CanvasDocumentGroup[] {
    if (!Array.isArray(groups)) return [];
    return groups.filter((group) => group && typeof group.id === "string" && Array.isArray(group.nodeIds)).map((group) => ({ ...group, nodeIds: group.nodeIds.filter((id) => typeof id === "string") }));
}

function finiteNumber(value: unknown, fallback: number): number {
    return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}
