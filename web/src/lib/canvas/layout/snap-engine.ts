import type { Position } from "@/types/canvas";
import type { CanvasGridSize, CanvasSnappedAxes } from "./layout-types";

export const DEFAULT_FILM_GRID_SIZE: CanvasGridSize = 8;

export function snapCanvasPosition(position: Position, gridSize: CanvasGridSize = DEFAULT_FILM_GRID_SIZE, preserveAxes: Partial<CanvasSnappedAxes> = {}): Position {
    if (gridSize === 0) return position;
    return {
        x: preserveAxes.x ? position.x : Math.round(position.x / gridSize) * gridSize,
        y: preserveAxes.y ? position.y : Math.round(position.y / gridSize) * gridSize,
    };
}
