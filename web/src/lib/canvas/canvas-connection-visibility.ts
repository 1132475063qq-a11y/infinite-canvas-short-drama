import { isFilmProductionProjection } from "@/film/domain/node-projection";
import type { CanvasConnection, CanvasNodeData } from "@/types/canvas";

export type CanvasConnectionVisibilityMode = "all" | "focus" | "hidden";

/**
 * Film production canvases prioritize the current production context, while
 * legacy free-form canvases keep their existing all-connections behaviour.
 */
export function defaultCanvasConnectionVisibilityMode(nodes: CanvasNodeData[]): CanvasConnectionVisibilityMode {
    return nodes.some(isFilmProductionProjection) ? "focus" : "all";
}

/**
 * Resolve the connections that are allowed into the render model. Filtering
 * here keeps the visual layer and its hit-testing layer in sync.
 */
export function visibleCanvasConnectionIds(connections: CanvasConnection[], mode: CanvasConnectionVisibilityMode, activeNodeId: string | null, selectedConnectionId: string | null): Set<string> {
    if (mode === "all") return new Set(connections.map((connection) => connection.id));
    if (mode === "hidden") return new Set<string>();

    const visible = new Set<string>();
    connections.forEach((connection) => {
        if (connection.id === selectedConnectionId || (activeNodeId && (connection.fromNodeId === activeNodeId || connection.toNodeId === activeNodeId))) {
            visible.add(connection.id);
        }
    });
    return visible;
}

export function nextCanvasConnectionVisibilityMode(mode: CanvasConnectionVisibilityMode): CanvasConnectionVisibilityMode {
    if (mode === "focus") return "all";
    if (mode === "all") return "hidden";
    return "focus";
}
