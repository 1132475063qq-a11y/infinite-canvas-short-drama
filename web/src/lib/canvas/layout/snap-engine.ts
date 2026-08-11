import type { Position } from "@/types/canvas";

export const DEFAULT_FILM_GRID_SIZE = 8;

export function snapCanvasPosition(position: Position, gridSize = DEFAULT_FILM_GRID_SIZE): Position {
    if (!Number.isFinite(gridSize) || gridSize <= 1) return position;
    return {
        x: Math.round(position.x / gridSize) * gridSize,
        y: Math.round(position.y / gridSize) * gridSize,
    };
}
