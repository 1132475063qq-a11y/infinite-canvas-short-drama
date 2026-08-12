import { snapCanvasPosition } from "@/lib/canvas/layout/snap-engine";
import type { CanvasGridSize, CanvasSnappedAxes } from "@/lib/canvas/layout/layout-types";
import type { CanvasNodeData, Position } from "@/types/canvas";

/** Move a visual projection without touching its production identity. */
export function moveFilmNodeProjection(node: CanvasNodeData, position: Position, gridSize: CanvasGridSize = 8, preserveAxes: Partial<CanvasSnappedAxes> = {}): CanvasNodeData {
    if (!node.filmKind) return { ...node, position: snapCanvasPosition(position, gridSize, preserveAxes) };
    return {
        ...node,
        position: snapCanvasPosition(position, gridSize, preserveAxes),
        layout: { ...node.layout, mode: "manual" },
    };
}
