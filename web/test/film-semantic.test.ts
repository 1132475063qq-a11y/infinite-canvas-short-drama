import { describe, expect, test } from "bun:test";

import { migrateCanvasProjectDocument } from "../src/film/domain/document-migration";
import { formatFilmSceneTitle, hasFilmSceneProjection, normalizeFilmSceneTitle } from "../src/film/domain/scene-projection";
import { applyFilmAutoLayout } from "../src/lib/canvas/layout/layout-engine";
import { snapCanvasPosition } from "../src/lib/canvas/layout/snap-engine";
import { CanvasNodeType, type CanvasNodeData } from "../src/types/canvas";

function filmNode(id: string, filmKind: CanvasNodeData["filmKind"], position = { x: 0, y: 0 }): CanvasNodeData {
    return {
        id,
        type: CanvasNodeType.Script,
        filmKind,
        title: id,
        position,
        width: 420,
        height: filmKind === "shot" ? 180 : 240,
    };
}

describe("Film Semantic migration", () => {
    test("将旧工作流节点升级为具有默认状态和八像素网格的影视文档", () => {
        const migrated = migrateCanvasProjectDocument({
            nodes: [
                {
                    ...filmNode("legacy-shot", undefined),
                    metadata: { workflowKind: "shot", locked: true },
                },
            ],
        });

        expect(migrated.schemaVersion).toBe(2);
        expect(migrated.layout).toEqual({ gridSize: 8 });
        expect(migrated.nodes[0]).toMatchObject({
            filmKind: "shot",
            layout: { mode: "pinned" },
            filmState: {
                lifecycle: "draft",
                production: "not_started",
                evidence: "unknown",
                attention: "none",
            },
        });
    });

    test("不具备影视语义的旧节点保持原样", () => {
        const node = filmNode("plain", undefined, { x: 13, y: 21 });
        const migrated = migrateCanvasProjectDocument({ nodes: [node] });

        expect(migrated.nodes[0]).toEqual(node);
    });
});

describe("Film layout", () => {
    test("镜头进入新画布时识别缺失的场景投影，并兼容旧场景标题", () => {
        const scene = {
            ...filmNode("scene-01", "scene"),
            domainRef: { projectId: "project-01", sceneId: "scene-01" },
        };

        expect(hasFilmSceneProjection([scene], "scene-01")).toBe(true);
        expect(hasFilmSceneProjection([], "scene-01")).toBe(false);
        expect(formatFilmSceneTitle("SC01", "SC01 未命名场景")).toBe("SC01 · 未命名场景");
        expect(formatFilmSceneTitle("SC02", "诊所外景")).toBe("SC02 · 诊所外景");
        expect(normalizeFilmSceneTitle("SC01 · SC01 未命名场景")).toBe("SC01 · 未命名场景");
    });

    test("自动布局按场次建立纵向 lane，并将镜头放入对应场次", () => {
        const sceneOne = {
            ...filmNode("scene-01", "scene", { x: 13, y: 21 }),
            domainRef: { projectId: "project-01", sceneId: "scene-01" },
            layout: { mode: "auto", order: 1 },
        };
        const sceneTwo = {
            ...filmNode("scene-02", "scene", { x: 200, y: 500 }),
            domainRef: { projectId: "project-01", sceneId: "scene-02" },
            layout: { mode: "auto", order: 2 },
        };
        const shot = {
            ...filmNode("shot-001", "shot"),
            domainRef: { projectId: "project-01", sceneId: "scene-01", shotId: "shot-001" },
            layout: { mode: "auto", order: 1 },
        };
        const manuallyPlaced = {
            ...filmNode("manual-character", "character", { x: 101, y: 101 }),
            layout: { mode: "manual" },
        };

        const positioned = applyFilmAutoLayout([sceneOne, sceneTwo, shot, manuallyPlaced]);

        expect(positioned.find((node) => node.id === "scene-01")?.position).toEqual({ x: 16, y: 24 });
        expect(positioned.find((node) => node.id === "scene-02")?.position).toEqual({ x: 16, y: 360 });
        expect(positioned.find((node) => node.id === "shot-001")?.position).toEqual({ x: 64, y: 312 });
        expect(positioned.find((node) => node.id === "manual-character")?.position).toEqual({ x: 101, y: 101 });
    });

    test("位置吸附到默认八像素网格", () => {
        expect(snapCanvasPosition({ x: 13, y: 21 })).toEqual({ x: 16, y: 24 });
    });
});
