import { describe, expect, test } from "bun:test";

import { calculateNodeAlignment, createNodeAlignmentContext } from "../src/lib/canvas/layout/alignment-guides";
import { alignCanvasNodes } from "../src/lib/canvas/layout/distribution";
import { applyFilmAutoLayout, applyFilmResultLayout } from "../src/lib/canvas/layout/layout-engine";
import { applyCanvasLayoutTransaction } from "../src/lib/canvas/layout/layout-transaction";
import { snapCanvasPosition } from "../src/lib/canvas/layout/snap-engine";
import { CanvasSpatialIndex } from "../src/lib/canvas/layout/spatial-index";
import { canvasNodeRect, canvasRectsOverlap } from "../src/lib/canvas/layout/collision";
import { CanvasNodeType, type CanvasNodeData } from "../src/types/canvas";

function node(id: string, x: number, y: number, width = 100, height = 80): CanvasNodeData {
    return {
        id,
        type: CanvasNodeType.Text,
        title: id,
        position: { x, y },
        width,
        height,
    };
}

describe("Phase 2 layout acceptance", () => {
    test("中心线相差六个屏幕像素时吸附到同一中心", () => {
        const fixed = node("fixed", 0, 300, 100, 80);
        const moving = node("moving", 200, 0, 100, 80);
        const context = createNodeAlignmentContext([fixed, moving], [{ id: moving.id, ...moving.position }]);
        const aligned = calculateNodeAlignment(context, { x: -194, y: 0 }, 6, 8);

        expect(aligned.offset.x).toBe(-200);
        expect(moving.position.x + moving.width / 2 + aligned.offset.x).toBe(fixed.position.x + fixed.width / 2);
        expect(aligned.snappedAxes.x).toBe(true);
    });

    test("中间节点在八像素阈值内吸附为相等间距并产生双箭头引导", () => {
        const left = node("left", 0, 0);
        const moving = node("moving", 205, 0);
        const right = node("right", 400, 0);
        const context = createNodeAlignmentContext([left, moving, right], [{ id: moving.id, ...moving.position }]);
        const aligned = calculateNodeAlignment(context, { x: 0, y: 0 }, 6, 8);

        expect(aligned.offset.x).toBe(-5);
        expect(aligned.guides.equalGapX).toMatchObject({
            axis: "x",
            beforeStart: 100,
            beforeEnd: 200,
            afterStart: 300,
            afterEnd: 400,
        });
        expect(aligned.snappedAxes.x).toBe(true);
    });

    test("十个 Shot 横向分布后间距一致", () => {
        const shots = Array.from({ length: 10 }, (_, index) => node(`shot-${index}`, index === 9 ? 1800 : index * (110 + index), 0, 100, 80));
        const positions = alignCanvasNodes(shots, "distributeX");
        const distributed = applyCanvasLayoutTransaction(shots, positions, { gridSize: 0 });
        const gaps = distributed.slice(1).map((item, index) => item.position.x - (distributed[index].position.x + distributed[index].width));

        gaps.forEach((gap) => expect(gap).toBeCloseTo(gaps[0], 8));
    });

    test("八个 Result 避开 pinned 障碍且互不重叠", () => {
        const pinned = { ...node("pinned", 0, 0, 120, 80), filmKind: "result" as const, layout: { mode: "pinned" as const } };
        const results = Array.from({ length: 8 }, (_, index) => ({
            ...node(`result-${index}`, 0, 0, 120, 80),
            filmKind: "result" as const,
            layout: { mode: "auto" as const, order: index },
        }));
        const positioned = applyFilmResultLayout([pinned, ...results]);
        const automatic = positioned.filter((item) => item.id.startsWith("result-"));

        automatic.forEach((item, index) => {
            expect(canvasRectsOverlap(canvasNodeRect(item), canvasNodeRect(pinned), 32)).toBe(false);
            automatic.slice(index + 1).forEach((other) => expect(canvasRectsOverlap(canvasNodeRect(item), canvasNodeRect(other), 32)).toBe(false));
        });
    });

    test("Tidy Scene 只移动 auto 节点，manual 和 pinned 均保持原位", () => {
        const scene = {
            ...node("scene", 13, 21, 420, 240),
            filmKind: "scene" as const,
            domainRef: { projectId: "project", sceneId: "scene" },
            layout: { mode: "auto" as const, order: 0 },
        };
        const automatic = {
            ...node("shot-auto", 900, 900, 320, 180),
            filmKind: "shot" as const,
            domainRef: { projectId: "project", sceneId: "scene", shotId: "shot-auto" },
            layout: { mode: "auto" as const, order: 0 },
        };
        const manual = { ...node("manual", 333, 444), filmKind: "character" as const, layout: { mode: "manual" as const } };
        const pinned = { ...node("pinned", 555, 666), filmKind: "location" as const, layout: { mode: "pinned" as const } };

        const positioned = applyFilmAutoLayout([scene, automatic, manual, pinned]);

        expect(positioned.find((item) => item.id === automatic.id)?.position).toEqual({ x: 64, y: 312 });
        expect(positioned.find((item) => item.id === manual.id)?.position).toEqual(manual.position);
        expect(positioned.find((item) => item.id === pinned.id)?.position).toEqual(pinned.position);
    });
});

describe("Phase 2 layout primitives", () => {
    test("网格支持 Off、8、16、24，并让对齐轴优先于网格", () => {
        expect(snapCanvasPosition({ x: 13, y: 21 }, 0)).toEqual({ x: 13, y: 21 });
        expect(snapCanvasPosition({ x: 13, y: 21 }, 8)).toEqual({ x: 16, y: 24 });
        expect(snapCanvasPosition({ x: 13, y: 21 }, 16)).toEqual({ x: 16, y: 16 });
        expect(snapCanvasPosition({ x: 13, y: 21 }, 24)).toEqual({ x: 24, y: 24 });
        expect(snapCanvasPosition({ x: 13, y: 21 }, 8, { x: true })).toEqual({ x: 13, y: 24 });
    });

    test("自动布局事务保持锁定和非 auto 节点，只返回一次完整提交", () => {
        const automatic = { ...node("auto", 0, 0), filmKind: "shot" as const, layout: { mode: "auto" as const } };
        const manual = { ...node("manual", 10, 10), filmKind: "shot" as const, layout: { mode: "manual" as const } };
        const locked = { ...node("locked", 20, 20), metadata: { locked: true }, layout: { mode: "auto" as const } };
        const positions = new Map([
            [automatic.id, { x: 101, y: 203 }],
            [manual.id, { x: 303, y: 405 }],
            [locked.id, { x: 505, y: 607 }],
        ]);

        const source = [automatic, manual, locked];
        const committed = applyCanvasLayoutTransaction(source, positions, { automatic: true, gridSize: 8 });

        expect(committed).not.toBe(source);
        expect(committed[0].position).toEqual({ x: 104, y: 200 });
        expect(committed[1]).toBe(manual);
        expect(committed[2]).toBe(locked);
    });

    test("空间索引只返回查询区域内的节点", () => {
        const index = CanvasSpatialIndex.from(
            Array.from({ length: 1_000 }, (_, itemIndex) => ({
                id: `node-${itemIndex}`,
                minX: itemIndex * 100,
                minY: 0,
                maxX: itemIndex * 100 + 80,
                maxY: 80,
            })),
            256,
        );

        expect(index.search({ minX: 50_000, minY: 0, maxX: 50_080, maxY: 80 }).map((item) => item.id)).toEqual(["node-500"]);
    });
});
