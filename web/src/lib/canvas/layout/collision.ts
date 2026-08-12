import type { CanvasNodeData, Position } from "@/types/canvas";
import type { CanvasRect } from "./layout-types";

export type { CanvasRect } from "./layout-types";

export function canvasNodeRect(node: Pick<CanvasNodeData, "position" | "width" | "height">): CanvasRect {
    return { x: node.position.x, y: node.position.y, width: node.width, height: node.height };
}

export function canvasRectsOverlap(first: CanvasRect, second: CanvasRect, padding = 0) {
    return first.x < second.x + second.width + padding
        && first.x + first.width + padding > second.x
        && first.y < second.y + second.height + padding
        && first.y + first.height + padding > second.y;
}

export function findAvailableCanvasPosition(candidate: Position, size: Pick<CanvasRect, "width" | "height">, occupied: CanvasRect[], options: { gap?: number; gridSize?: number; attempts?: number } = {}): Position {
    const gap = options.gap ?? 32;
    const gridSize = options.gridSize ?? 8;
    const attempts = options.attempts ?? 256;
    const origin = { x: snap(candidate.x, gridSize), y: snap(candidate.y, gridSize) };
    const canPlace = (position: Position) => !occupied.some((rect) => canvasRectsOverlap({ ...position, ...size }, rect, gap));
    if (canPlace(origin)) return origin;

    // Deterministic downward-first spiral keeps result groups readable and avoids
    // rewriting user positions merely because the layout runs again.
    for (let step = 1; step <= attempts; step += 1) {
        const distance = step * gridSize;
        const candidates = [
            { x: origin.x, y: origin.y + distance },
            { x: origin.x + distance, y: origin.y },
            { x: origin.x - distance, y: origin.y },
            { x: origin.x, y: origin.y - distance },
        ];
        const available = candidates.find(canPlace);
        if (available) return available;
    }
    return origin;
}

function snap(value: number, gridSize: number) {
    return Math.round(value / gridSize) * gridSize;
}
