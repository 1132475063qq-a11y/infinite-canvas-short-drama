import { isFrameNode } from "@/lib/canvas/canvas-frame";
import type { CanvasNodeData, Position } from "@/types/canvas";

import type { CanvasAlignmentGuides, CanvasEqualGapGuide, CanvasRect, CanvasSnappedAxes } from "./layout-types";
import { CanvasSpatialIndex, type CanvasSpatialItem } from "./spatial-index";

type AlignmentTarget = CanvasSpatialItem & {
    rect: CanvasRect;
    x: [number, number, number];
    y: [number, number, number];
};

export type NodeAlignmentContext = {
    movingBounds: { left: number; top: number; right: number; bottom: number };
    targetBounds: Omit<CanvasSpatialItem, "id">;
    targets: Map<string, AlignmentTarget>;
    spatialIndex: CanvasSpatialIndex;
};

export function createNodeAlignmentContext(nodes: CanvasNodeData[], initialPositions: Array<{ id: string; x: number; y: number }>): NodeAlignmentContext | null {
    const movingIds = new Set(initialPositions.map((item) => item.id));
    const initialById = new Map(initialPositions.map((item) => [item.id, item]));
    const movingNodes = nodes.filter((node) => movingIds.has(node.id));
    if (!movingNodes.length) return null;

    const nodeById = new Map(nodes.map((node) => [node.id, node]));
    const left = Math.min(...movingNodes.map((node) => initialById.get(node.id)?.x ?? node.position.x));
    const top = Math.min(...movingNodes.map((node) => initialById.get(node.id)?.y ?? node.position.y));
    const right = Math.max(...movingNodes.map((node) => (initialById.get(node.id)?.x ?? node.position.x) + node.width));
    const bottom = Math.max(...movingNodes.map((node) => (initialById.get(node.id)?.y ?? node.position.y) + node.height));
    const targets = new Map<string, AlignmentTarget>();

    nodes.forEach((node) => {
        if (movingIds.has(node.id)) return;
        const batchRoot = node.metadata?.batchRootId ? nodeById.get(node.metadata.batchRootId) : null;
        if (batchRoot && !batchRoot.metadata?.imageBatchExpanded) return;
        const parent = node.parentId ? nodeById.get(node.parentId) : null;
        if (parent && isFrameNode(parent) && parent.metadata?.frame?.collapsed) return;
        const rect = { x: node.position.x, y: node.position.y, width: node.width, height: node.height };
        targets.set(node.id, {
            id: node.id,
            minX: rect.x,
            minY: rect.y,
            maxX: rect.x + rect.width,
            maxY: rect.y + rect.height,
            rect,
            x: [rect.x, rect.x + rect.width / 2, rect.x + rect.width],
            y: [rect.y, rect.y + rect.height / 2, rect.y + rect.height],
        });
    });

    const targetItems = Array.from(targets.values());
    const targetBounds = targetItems.length
        ? {
              minX: Math.min(...targetItems.map((item) => item.minX)),
              minY: Math.min(...targetItems.map((item) => item.minY)),
              maxX: Math.max(...targetItems.map((item) => item.maxX)),
              maxY: Math.max(...targetItems.map((item) => item.maxY)),
          }
        : { minX: left, minY: top, maxX: right, maxY: bottom };

    return {
        movingBounds: { left, top, right, bottom },
        targetBounds,
        targets,
        spatialIndex: CanvasSpatialIndex.from(targetItems),
    };
}

export function calculateNodeAlignment(context: NodeAlignmentContext | null, rawOffset: Position, alignmentThreshold: number, equalGapThreshold = alignmentThreshold) {
    const emptyGuides: CanvasAlignmentGuides = {};
    const emptyAxes: CanvasSnappedAxes = { x: false, y: false };
    if (!context) return { offset: rawOffset, guides: emptyGuides, snappedAxes: emptyAxes };

    const current = movedBounds(context.movingBounds, rawOffset);
    const targetIds = alignmentCandidateIds(context, current, Math.max(alignmentThreshold, equalGapThreshold));
    const targets = Array.from(targetIds, (id) => context.targets.get(id)).filter((target): target is AlignmentTarget => Boolean(target));
    const movingX = [current.left, (current.left + current.right) / 2, current.right];
    const movingY = [current.top, (current.top + current.bottom) / 2, current.bottom];
    let bestXDelta: number | undefined;
    let bestXGuide: number | undefined;
    let bestYDelta: number | undefined;
    let bestYGuide: number | undefined;

    targets.forEach((target) => {
        movingX.forEach((value, anchorIndex) => {
            const delta = target.x[anchorIndex] - value;
            if (Math.abs(delta) <= alignmentThreshold && (bestXDelta === undefined || Math.abs(delta) < Math.abs(bestXDelta))) {
                bestXDelta = delta;
                bestXGuide = target.x[anchorIndex];
            }
        });
        movingY.forEach((value, anchorIndex) => {
            const delta = target.y[anchorIndex] - value;
            if (Math.abs(delta) <= alignmentThreshold && (bestYDelta === undefined || Math.abs(delta) < Math.abs(bestYDelta))) {
                bestYDelta = delta;
                bestYGuide = target.y[anchorIndex];
            }
        });
    });

    const equalX = bestXDelta === undefined ? findHorizontalEqualGap(current, targets, equalGapThreshold) : undefined;
    const equalY = bestYDelta === undefined ? findVerticalEqualGap(current, targets, equalGapThreshold) : undefined;
    const xDelta = bestXDelta ?? equalX?.delta ?? 0;
    const yDelta = bestYDelta ?? equalY?.delta ?? 0;
    return {
        offset: { x: rawOffset.x + xDelta, y: rawOffset.y + yDelta },
        guides: {
            vertical: bestXGuide,
            horizontal: bestYGuide,
            equalGapX: equalX?.guide,
            equalGapY: equalY?.guide,
        } satisfies CanvasAlignmentGuides,
        snappedAxes: { x: bestXDelta !== undefined || Boolean(equalX), y: bestYDelta !== undefined || Boolean(equalY) },
    };
}

function alignmentCandidateIds(context: NodeAlignmentContext, current: ReturnType<typeof movedBounds>, threshold: number) {
    const ids = new Set<string>();
    const queries = [
        { minX: current.left - threshold, minY: context.targetBounds.minY, maxX: current.right + threshold, maxY: context.targetBounds.maxY },
        { minX: context.targetBounds.minX, minY: current.top - threshold, maxX: context.targetBounds.maxX, maxY: current.bottom + threshold },
    ];
    queries.forEach((query) => context.spatialIndex.search(query).forEach((item) => ids.add(item.id)));
    return ids;
}

function findHorizontalEqualGap(current: ReturnType<typeof movedBounds>, targets: AlignmentTarget[], threshold: number) {
    const rowTargets = targets.filter((target) => rangesOverlap(current.top, current.bottom, target.minY, target.maxY));
    const left = rowTargets.filter((target) => target.maxX <= current.left).sort((a, b) => b.maxX - a.maxX)[0];
    const right = rowTargets.filter((target) => target.minX >= current.right).sort((a, b) => a.minX - b.minX)[0];
    if (!left || !right) return undefined;
    const width = current.right - current.left;
    const targetLeft = (left.maxX + right.minX - width) / 2;
    const delta = targetLeft - current.left;
    if (Math.abs(delta) > threshold) return undefined;
    const correctedRight = targetLeft + width;
    return {
        delta,
        guide: {
            axis: "x",
            beforeStart: left.maxX,
            beforeEnd: targetLeft,
            afterStart: correctedRight,
            afterEnd: right.minX,
            cross: (Math.max(current.top, left.minY, right.minY) + Math.min(current.bottom, left.maxY, right.maxY)) / 2,
        } satisfies CanvasEqualGapGuide,
    };
}

function findVerticalEqualGap(current: ReturnType<typeof movedBounds>, targets: AlignmentTarget[], threshold: number) {
    const columnTargets = targets.filter((target) => rangesOverlap(current.left, current.right, target.minX, target.maxX));
    const above = columnTargets.filter((target) => target.maxY <= current.top).sort((a, b) => b.maxY - a.maxY)[0];
    const below = columnTargets.filter((target) => target.minY >= current.bottom).sort((a, b) => a.minY - b.minY)[0];
    if (!above || !below) return undefined;
    const height = current.bottom - current.top;
    const targetTop = (above.maxY + below.minY - height) / 2;
    const delta = targetTop - current.top;
    if (Math.abs(delta) > threshold) return undefined;
    const correctedBottom = targetTop + height;
    return {
        delta,
        guide: {
            axis: "y",
            beforeStart: above.maxY,
            beforeEnd: targetTop,
            afterStart: correctedBottom,
            afterEnd: below.minY,
            cross: (Math.max(current.left, above.minX, below.minX) + Math.min(current.right, above.maxX, below.maxX)) / 2,
        } satisfies CanvasEqualGapGuide,
    };
}

function movedBounds(bounds: NodeAlignmentContext["movingBounds"], offset: Position) {
    return {
        left: bounds.left + offset.x,
        top: bounds.top + offset.y,
        right: bounds.right + offset.x,
        bottom: bounds.bottom + offset.y,
    };
}

function rangesOverlap(firstStart: number, firstEnd: number, secondStart: number, secondEnd: number) {
    return firstStart < secondEnd && firstEnd > secondStart;
}
