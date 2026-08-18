import { snapCanvasPosition } from "@/lib/canvas/layout/snap-engine";
import type { CanvasNodeData, Position } from "@/types/canvas";

import type { CanvasGridSize, CanvasSnappedAxes } from "./layout-types";

export type CanvasLayoutTransactionOptions = {
    automatic?: boolean;
    gridSize?: CanvasGridSize;
    preserveAlignedAxes?: CanvasSnappedAxes;
};

/** Apply a complete layout as one immutable node-array commit (one undo entry). */
export function applyCanvasLayoutTransaction(nodes: CanvasNodeData[], positions: Map<string, Position>, options: CanvasLayoutTransactionOptions = {}) {
    const gridSize = options.gridSize ?? 8;
    return nodes.map((node) => {
        const position = positions.get(node.id);
        if (!position || node.metadata?.locked || (options.automatic && node.layout?.mode !== "auto")) return node;
        const snapped = snapCanvasPosition(position, gridSize, options.preserveAlignedAxes);
        if (snapped.x === node.position.x && snapped.y === node.position.y) return node;
        return { ...node, position: snapped };
    });
}
