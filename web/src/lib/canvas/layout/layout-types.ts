export const CANVAS_GRID_SIZES = [0, 8, 16, 24] as const;

export type CanvasGridSize = (typeof CANVAS_GRID_SIZES)[number];

export type CanvasRect = {
    x: number;
    y: number;
    width: number;
    height: number;
};

export type CanvasEqualGapGuide = {
    axis: "x" | "y";
    beforeStart: number;
    beforeEnd: number;
    afterStart: number;
    afterEnd: number;
    cross: number;
};

export type CanvasAlignmentGuides = {
    vertical?: number;
    horizontal?: number;
    equalGapX?: CanvasEqualGapGuide;
    equalGapY?: CanvasEqualGapGuide;
};

export type CanvasSnappedAxes = {
    x: boolean;
    y: boolean;
};

export function normalizeCanvasGridSize(value: unknown): CanvasGridSize {
    return CANVAS_GRID_SIZES.includes(value as CanvasGridSize) ? (value as CanvasGridSize) : 8;
}

export function hasCanvasAlignmentGuide(guides: CanvasAlignmentGuides) {
    return typeof guides.vertical === "number" || typeof guides.horizontal === "number" || Boolean(guides.equalGapX || guides.equalGapY);
}
