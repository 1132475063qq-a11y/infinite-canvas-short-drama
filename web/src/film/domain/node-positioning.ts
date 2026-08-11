import { snapCanvasPosition } from "@/lib/canvas/layout/snap-engine";
import type { CanvasNodeData, Position } from "@/types/canvas";

/** Move a visual projection without touching its production identity. */
export function moveFilmNodeProjection(node: CanvasNodeData, position: Position): CanvasNodeData {
    if (!node.filmKind) return { ...node, position: snapCanvasPosition(position) };
    return {
        ...node,
        position: snapCanvasPosition(position),
        layout: { ...node.layout, mode: "manual" },
    };
}
