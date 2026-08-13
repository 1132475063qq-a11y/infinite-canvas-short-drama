import { describe, expect, test } from "bun:test";

import { migrateCanvasProjectDocument } from "../src/film/domain/document-migration";
import { describeFilmConnection, validateFilmConnection } from "../src/film/domain/edge-contract";
import { moveFilmNodeProjection } from "../src/film/domain/node-positioning";
import { isFilmProductionProjection } from "../src/film/domain/node-projection";
import { resolveFilmNode } from "../src/film/domain/film-node-resolver";
import { FILM_NODE_PORT_SCHEMAS, areFilmPortSchemasCompatible, findFilmPortSchema, missingRequiredFilmInputPorts } from "../src/film/domain/port-schema";
import { formatFilmSceneTitle, hasFilmSceneProjection, normalizeFilmSceneTitle } from "../src/film/domain/scene-projection";
import { FILM_NODE_KINDS } from "../src/film/domain/types";
import { applyFilmAutoLayout, applyFilmResultLayout } from "../src/lib/canvas/layout/layout-engine";
import { canvasNodeRect, canvasRectsOverlap } from "../src/lib/canvas/layout/collision";
import { reconcileFilmSceneProjections } from "../src/lib/canvas/layout/film-scene-projection";
import { snapCanvasPosition } from "../src/lib/canvas/layout/snap-engine";
import { normalizeConnection } from "../src/lib/canvas/canvas-project-domain";
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
        expect(isFilmProductionProjection(migrated.nodes[0])).toBe(false);
    });

    test("旧工作流语义不会把原生节点误判为数据库生产投影", () => {
        const nativeStoryboard = migrateCanvasProjectDocument({
            nodes: [
                {
                    ...filmNode("legacy-storyboard", undefined),
                    metadata: { workflowKind: "storyboard", content: "原分镜内容" },
                },
            ],
        }).nodes[0];
        const recordedShot = {
            ...filmNode("recorded-shot", "shot"),
            domainRef: { projectId: "project-01", sceneId: "scene-01", shotId: "shot-01" },
        };

        expect(nativeStoryboard.filmKind).toBe("script");
        expect(nativeStoryboard.metadata?.content).toBe("原分镜内容");
        expect(isFilmProductionProjection(nativeStoryboard)).toBe(false);
        expect(isFilmProductionProjection(recordedShot)).toBe(true);
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
        expect(positioned.find((node) => node.id === "scene-02")?.position).toEqual({ x: 16, y: 584 });
        expect(positioned.find((node) => node.id === "shot-001")?.position).toEqual({ x: 64, y: 312 });
        expect(positioned.find((node) => node.id === "manual-character")?.position).toEqual({ x: 101, y: 101 });
    });

    test("Shot 生产链按 Shot、Acting、Prompt Pack 横向排列", () => {
        const scene = { ...filmNode("scene-01", "scene", { x: 13, y: 21 }), domainRef: { projectId: "project-01", sceneId: "scene-01" }, layout: { mode: "auto" as const, order: 1 } };
        const shot = { ...filmNode("shot-001", "shot"), domainRef: { projectId: "project-01", sceneId: "scene-01", shotId: "shot-001" }, layout: { mode: "auto" as const, order: 1 } };
        const acting = { ...filmNode("acting-001", "acting"), width: 300, height: 180, domainRef: { projectId: "project-01", sceneId: "scene-01", shotId: "shot-001" }, layout: { mode: "auto" as const, order: 1 } };
        const prompt = { ...filmNode("prompt-001", "prompt_pack"), width: 340, height: 180, domainRef: { projectId: "project-01", sceneId: "scene-01", shotId: "shot-001" }, layout: { mode: "auto" as const, order: 2 } };

        const positioned = applyFilmAutoLayout([scene, shot, acting, prompt]);

        expect(positioned.find((node) => node.id === "shot-001")?.position).toEqual({ x: 64, y: 312 });
        expect(positioned.find((node) => node.id === "acting-001")?.position).toEqual({ x: 528, y: 312 });
        expect(positioned.find((node) => node.id === "prompt-001")?.position).toEqual({ x: 880, y: 312 });
    });

    test("旧镜头画布会补上所属场景投影，但不会重建生产对象", () => {
        const shot = {
            ...filmNode("shot-001", "shot", { x: 400, y: 600 }),
            domainRef: { projectId: "project-01", sceneId: "scene-01", shotId: "shot-001" },
            layout: { mode: "auto" as const, order: 1 },
        };
        const reconciled = reconcileFilmSceneProjections([shot], "project-01", [{ id: "scene-01", code: "SC01", title: "未命名场景", position: 0 }]);

        expect(reconciled).toHaveLength(2);
        expect(reconciled.find((node) => node.filmKind === "scene")).toMatchObject({
            title: "SC01 · 未命名场景",
            domainRef: { projectId: "project-01", sceneId: "scene-01" },
            position: { x: 352, y: 312 },
        });
        expect(reconciled.find((node) => node.id === "shot-001")?.position).toEqual({ x: 400, y: 600 });
        expect(reconcileFilmSceneProjections(reconciled, "project-01", [{ id: "scene-01", code: "SC01", title: "未命名场景", position: 0 }])).toBe(reconciled);
    });

    test("位置吸附到默认八像素网格", () => {
        expect(snapCanvasPosition({ x: 13, y: 21 })).toEqual({ x: 16, y: 24 });
    });

    test("整理 Scene Lane 时保持 pinned 投影的位置不变", () => {
        const pinnedScene = {
            ...filmNode("scene-pinned", "scene", { x: 333, y: 777 }),
            domainRef: { projectId: "project-01", sceneId: "scene-pinned" },
            layout: { mode: "pinned" as const, order: 1 },
        };
        const autoShot = {
            ...filmNode("shot-pinned-scene", "shot"),
            domainRef: { projectId: "project-01", sceneId: "scene-pinned", shotId: "shot-001" },
            layout: { mode: "auto" as const, order: 1 },
        };

        const positioned = applyFilmAutoLayout([pinnedScene, autoShot]);
        expect(positioned.find((node) => node.id === pinnedScene.id)?.position).toEqual(pinnedScene.position);
        expect(positioned.find((node) => node.id === autoShot.id)?.position).toEqual({ x: 384, y: 1064 });
    });
});

describe("Film semantic contracts", () => {
    test("影视节点解析器始终读取 Shot 当前合同版本", () => {
        const shotNode = { ...filmNode("shot-node", "shot"), domainRef: { projectId: "project-01", sceneId: "scene-01", shotId: "shot-01", artifactId: "contract-v2" } };
        const project = {
            project: { id: "project-01", userId: "user-01", name: "寄生广告", type: "short_drama", aspectRatio: "9:16", sourceType: "script", description: "", stylePresetId: "", status: "active", revision: 1, createdAt: "2026-08-12T00:00:00Z", updatedAt: "2026-08-12T00:00:00Z" },
            units: [], canvases: [], canvasUnitLinks: [], workflows: [], assets: [], assetCandidates: [], shotReferences: [],
            scenes: [{ id: "scene-01", projectId: "project-01", code: "SC01", title: "诊所", description: "", position: 0, status: "draft", createdAt: "2026-08-12T00:00:00Z", updatedAt: "2026-08-12T00:00:00Z" }],
            shots: [{ id: "shot-01", projectId: "project-01", sceneId: "scene-01", title: "SC01-SH001", description: "", position: 0, durationMs: 4200, status: "ready", contractArtifactId: "contract-v2", contractVersion: 2, createdAt: "2026-08-12T00:00:00Z", updatedAt: "2026-08-12T00:00:00Z" }],
            filmArtifacts: [
                { id: "contract-v1", projectId: "project-01", sceneId: "scene-01", shotId: "shot-01", artifactType: "shot_contract", objectVersion: 1, status: "draft", payloadJson: JSON.stringify({ shotSize: "MS" }), sourceRefsJson: "[]", authorityRefsJson: "[]", createdAt: "2026-08-12T00:00:00Z", updatedAt: "2026-08-12T00:00:00Z" },
                { id: "contract-v2", projectId: "project-01", sceneId: "scene-01", shotId: "shot-01", artifactType: "shot_contract", objectVersion: 2, status: "ready", payloadJson: JSON.stringify({ shotSize: "MCU" }), sourceRefsJson: "[]", authorityRefsJson: "[]", createdAt: "2026-08-12T00:01:00Z", updatedAt: "2026-08-12T00:01:00Z" },
                { id: "acting-v1", projectId: "project-01", sceneId: "scene-01", shotId: "shot-01", artifactType: "acting", objectVersion: 1, status: "draft", payloadJson: JSON.stringify({ objective: "conceal" }), sourceRefsJson: "[]", authorityRefsJson: "[]", createdAt: "2026-08-12T00:02:00Z", updatedAt: "2026-08-12T00:02:00Z" },
                { id: "acting-v2", projectId: "project-01", sceneId: "scene-01", shotId: "shot-01", artifactType: "acting", objectVersion: 2, status: "ready", payloadJson: JSON.stringify({ objective: "redirect" }), sourceRefsJson: "[]", authorityRefsJson: "[]", createdAt: "2026-08-12T00:03:00Z", updatedAt: "2026-08-12T00:03:00Z" },
            ],
        };

        const resolved = resolveFilmNode(shotNode, project);
        expect(resolved.artifact?.id).toBe("contract-v2");
        expect(resolved.contract.shotSize).toBe("MCU");
        expect(resolved.scene?.title).toBe("诊所");

        const actingNode = { ...filmNode("acting-node", "acting"), domainRef: { projectId: "project-01", sceneId: "scene-01", shotId: "shot-01", artifactId: "acting-v1" } };
        const actingResolved = resolveFilmNode(actingNode, project);
        expect(actingResolved.artifact?.id).toBe("acting-v2");
        expect(actingResolved.contract.objective).toBe("redirect");
    });

    test("FilmNodeKind v2 与每种节点的端口合同完整冻结", () => {
        // 规划枚举示例漏了 Phase 4 明确要求的 acting；保留现有 acting 并补齐细分类型。
        expect(FILM_NODE_KINDS).toHaveLength(26);
        expect(FILM_NODE_KINDS).toContain("acting");
        expect(FILM_NODE_KINDS).toContain("character_state");
        expect(FILM_NODE_KINDS).toContain("generation_attempt");
        expect(FILM_NODE_KINDS).toContain("continuity");
        expect(FILM_NODE_KINDS).toContain("delivery");
        FILM_NODE_KINDS.forEach((kind) => {
            expect(FILM_NODE_PORT_SCHEMAS[kind].length).toBeGreaterThan(0);
            FILM_NODE_PORT_SCHEMAS[kind].forEach((port) => {
                expect(typeof port.role).toBe("string");
                expect(typeof port.required).toBe("boolean");
                expect(typeof port.multiple).toBe("boolean");
                expect(Array.isArray(port.accepts)).toBe(true);
            });
        });
    });

    test("端口类型、必填输入和单连接容量形成可执行合同", () => {
        const scene = { ...filmNode("scene-01", "scene"), domainRef: { projectId: "project-01", sceneId: "scene-01" } };
        const shot = { ...filmNode("shot-001", "shot"), domainRef: { projectId: "project-01", sceneId: "scene-01", shotId: "shot-001" } };
        const sceneOutput = findFilmPortSchema("scene", "scene_context", "output");
        const shotInput = findFilmPortSchema("shot", "scene_context", "input");

        expect(sceneOutput && shotInput && areFilmPortSchemasCompatible(sceneOutput, shotInput)).toBe(true);
        expect(missingRequiredFilmInputPorts(shot, [])).toEqual(expect.arrayContaining([expect.objectContaining({ id: "scene_context" })]));
        expect(missingRequiredFilmInputPorts(shot, [{ id: "scene-shot", fromNodeId: scene.id, toNodeId: shot.id, filmPorts: { from: "scene_context", to: "scene_context" } }])).toEqual([]);
        expect(validateFilmConnection(scene, shot, { connections: [{ id: "existing", fromNodeId: scene.id, toNodeId: shot.id, filmPorts: { from: "scene_context", to: "scene_context" } }] })).toEqual({ allowed: false, reason: "scene_context 只允许一个输入连接" });
    });

    test("影视节点连线带有独立于画布句柄的端口和边语义", () => {
        const scene = { ...filmNode("scene-01", "scene"), type: CanvasNodeType.Frame, domainRef: { projectId: "project-01", sceneId: "scene-01" } };
        const shot = { ...filmNode("shot-001", "shot"), domainRef: { projectId: "project-01", sceneId: "scene-01", shotId: "shot-001" } };
        const character = { ...filmNode("character-01", "character"), domainRef: { projectId: "project-01", assetId: "asset-01" } };

        expect(describeFilmConnection(scene, shot)).toEqual({
            edgeType: "continuity",
            filmPorts: { from: "scene_context", to: "scene_context" },
        });
        expect(describeFilmConnection(character, shot)).toEqual({
            edgeType: "reference",
            filmPorts: { from: "asset_reference", to: "asset_reference" },
        });
        expect(describeFilmConnection(scene, filmNode("legacy", undefined))).toBeUndefined();
        expect(normalizeConnection(scene.id, shot.id, [scene, shot], "source")).toEqual({ fromNodeId: scene.id, toNodeId: shot.id });
    });

    test("不兼容、跨项目和跨场景的生产连线被拒绝，旧节点仍保持自由连线", () => {
        const shot = { ...filmNode("shot-001", "shot"), domainRef: { projectId: "project-01", sceneId: "scene-01", shotId: "shot-001" } };
        const qc = { ...filmNode("qc-001", "qc"), domainRef: { projectId: "project-01", sceneId: "scene-01", shotId: "shot-001" } };
        const character = { ...filmNode("character-01", "character"), domainRef: { projectId: "project-01", assetId: "asset-01" } };
        const otherProjectCharacter = { ...character, id: "character-02", domainRef: { projectId: "project-02", assetId: "asset-02" } };
        const otherScene = { ...filmNode("scene-02", "scene"), domainRef: { projectId: "project-01", sceneId: "scene-02" } };
        const legacyShot = { ...filmNode("legacy-shot", "shot"), metadata: { workflowKind: "shot" as const } };

        expect(validateFilmConnection(qc, character)).toMatchObject({ allowed: false });
        expect(validateFilmConnection(otherProjectCharacter, shot)).toEqual({ allowed: false, reason: "影视生产节点不能跨项目连接" });
        expect(validateFilmConnection(otherScene, shot)).toEqual({ allowed: false, reason: "镜头只能连接到自己所属的场景" });
        expect(validateFilmConnection(legacyShot, qc)).toEqual({ allowed: true });
    });

    test("文档迁移为合法的旧生产连线补充语义，但不改写旧工作流和错误连线", () => {
        const scene = { ...filmNode("scene-01", "scene"), domainRef: { projectId: "project-01", sceneId: "scene-01" } };
        const shot = { ...filmNode("shot-001", "shot"), domainRef: { projectId: "project-01", sceneId: "scene-01", shotId: "shot-001" } };
        const qc = { ...filmNode("qc-001", "qc"), domainRef: { projectId: "project-01", shotId: "shot-001" } };
        const legacy = { ...filmNode("legacy", undefined), metadata: { workflowKind: "story_input" as const } };
        const migrated = migrateCanvasProjectDocument({
            nodes: [scene, shot, qc, legacy],
            connections: [
                { id: "valid", fromNodeId: scene.id, toNodeId: shot.id },
                { id: "invalid", fromNodeId: qc.id, toNodeId: shot.id, edgeType: "authority", filmPorts: { from: "qc_decision", to: "scene_context" } },
                { id: "legacy", fromNodeId: legacy.id, toNodeId: shot.id },
            ],
        });

        expect(migrated.connections?.find((item) => item.id === "valid")).toMatchObject({
            edgeType: "continuity",
            filmPorts: { from: "scene_context", to: "scene_context" },
        });
        expect(migrated.connections?.find((item) => item.id === "invalid")?.edgeType).toBeUndefined();
        expect(migrated.connections?.find((item) => item.id === "legacy")?.edgeType).toBeUndefined();
    });

    test("拖动镜头仅改变画布投影，不改变生产对象引用或状态", () => {
        const shot = {
            ...filmNode("shot-001", "shot"),
            domainRef: { projectId: "project-01", sceneId: "scene-01", shotId: "shot-001" },
            filmState: { lifecycle: "review" as const, production: "ready" as const, evidence: "recorded" as const, attention: "none" as const },
            layout: { mode: "auto" as const, lane: "scene-01", order: 1 },
        };

        const moved = moveFilmNodeProjection(shot, { x: 101, y: 205 });

        expect(moved.position).toEqual({ x: 104, y: 208 });
        expect(moved.layout).toEqual({ mode: "manual", lane: "scene-01", order: 1 });
        expect(moved.domainRef).toEqual(shot.domainRef);
        expect(moved.filmState).toEqual(shot.filmState);
    });
});

describe("Film collision-aware layout", () => {
    test("八个自动结果避开 pinned 节点且彼此不重叠", () => {
        const pinned = {
            ...filmNode("pinned", "result", { x: 0, y: 0 }),
            width: 120,
            height: 80,
            layout: { mode: "pinned" as const },
        };
        const results = Array.from({ length: 8 }, (_, index) => ({
            ...filmNode(`result-${index}`, "result"),
            width: 120,
            height: 80,
            layout: { mode: "auto" as const, order: index },
        }));

        const positioned = applyFilmResultLayout([pinned, ...results]);
        const autoResults = positioned.filter((node) => node.id.startsWith("result-"));

        expect(positioned.find((node) => node.id === pinned.id)?.position).toEqual(pinned.position);
        autoResults.forEach((node, index) => {
            expect(canvasRectsOverlap(canvasNodeRect(node), canvasNodeRect(pinned), 32)).toBe(false);
            autoResults.slice(index + 1).forEach((other) => expect(canvasRectsOverlap(canvasNodeRect(node), canvasNodeRect(other), 32)).toBe(false));
        });
    });
});
