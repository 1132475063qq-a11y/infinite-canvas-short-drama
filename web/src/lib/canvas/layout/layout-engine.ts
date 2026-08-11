import { layoutFilmSceneLanes } from "@/lib/canvas/layout/film-layout-presets";
import { canvasNodeRect, findAvailableCanvasPosition } from "@/lib/canvas/layout/collision";
import { snapCanvasPosition } from "@/lib/canvas/layout/snap-engine";
import type { CanvasNodeData, Position } from "@/types/canvas";

export function applyFilmAutoLayout(nodes: CanvasNodeData[]): CanvasNodeData[] {
    const positions = layoutFilmSceneLanes(nodes);
    if (!positions.size) return nodes;
    return nodes.map((node) => {
        const position = positions.get(node.id);
        return position ? { ...node, position: snapCanvasPosition(position) } : node;
    });
}

/** Arrange only auto Result projections and reserve every existing manual/pinned node. */
export function applyFilmResultLayout(nodes: CanvasNodeData[], origin: Position = { x: 0, y: 0 }): CanvasNodeData[] {
    const occupied = nodes
        .filter((node) => node.filmKind !== "result" || node.layout?.mode !== "auto")
        .map(canvasNodeRect);
    let autoIndex = 0;

    return nodes.map((node) => {
        if (node.filmKind !== "result" || node.layout?.mode !== "auto") return node;
        const columns = 4;
        const candidate = {
            x: origin.x + (autoIndex % columns) * (node.width + 32),
            y: origin.y + Math.floor(autoIndex / columns) * (node.height + 32),
        };
        autoIndex += 1;
        const position = findAvailableCanvasPosition(candidate, node, occupied);
        occupied.push({ x: position.x, y: position.y, width: node.width, height: node.height });
        return { ...node, position: snapCanvasPosition(position) };
    });
}
