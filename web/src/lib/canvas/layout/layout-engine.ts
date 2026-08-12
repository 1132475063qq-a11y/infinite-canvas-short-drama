import { layoutFilmSceneLanes } from "@/lib/canvas/layout/film-layout-presets";
import { canvasNodeRect, findAvailableCanvasPosition } from "@/lib/canvas/layout/collision";
import type { CanvasNodeData, Position } from "@/types/canvas";
import { applyCanvasLayoutTransaction } from "./layout-transaction";
import type { CanvasGridSize } from "./layout-types";

export function applyFilmAutoLayout(nodes: CanvasNodeData[], gridSize: CanvasGridSize = 8): CanvasNodeData[] {
    const positions = layoutFilmSceneLanes(nodes);
    if (!positions.size) return nodes;
    return applyCanvasLayoutTransaction(nodes, positions, { automatic: true, gridSize });
}

/** Arrange only auto Result projections and reserve every existing manual/pinned node. */
export function applyFilmResultLayout(nodes: CanvasNodeData[], origin: Position = { x: 0, y: 0 }, gridSize: CanvasGridSize = 8): CanvasNodeData[] {
    const occupied = nodes
        .filter((node) => node.filmKind !== "result" || node.layout?.mode !== "auto")
        .map(canvasNodeRect);
    let autoIndex = 0;

    const positions = new Map<string, Position>();
    nodes.forEach((node) => {
        if (node.filmKind !== "result" || node.layout?.mode !== "auto") return;
        const columns = 4;
        const candidate = {
            x: origin.x + (autoIndex % columns) * (node.width + 32),
            y: origin.y + Math.floor(autoIndex / columns) * (node.height + 32),
        };
        autoIndex += 1;
        const position = findAvailableCanvasPosition(candidate, node, occupied);
        occupied.push({ x: position.x, y: position.y, width: node.width, height: node.height });
        positions.set(node.id, position);
    });
    return applyCanvasLayoutTransaction(nodes, positions, { automatic: true, gridSize });
}
