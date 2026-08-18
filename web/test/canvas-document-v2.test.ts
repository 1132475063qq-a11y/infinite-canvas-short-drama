import { describe, expect, test } from "bun:test";

import { deriveCanvasDocumentGroups, migrateCanvasProjectDocument } from "../src/film/domain/document-migration";
import { CanvasNodeType, type CanvasNodeData } from "../src/types/canvas";

function node(id: string, type = CanvasNodeType.Text): CanvasNodeData {
    return { id, type, title: id, position: { x: 0, y: 0 }, width: 320, height: 180 };
}

describe("CanvasDocumentV2 migration", () => {
    test("将 edges、layout.grid 和缺省字段归一为唯一 V2 文档合同", () => {
        const legacyConnection = { id: "edge-01", fromNodeId: "node-01", toNodeId: "node-02" };
        const migrated = migrateCanvasProjectDocument({
            schemaVersion: 1,
            projectId: "project-01",
            nodes: [node("node-01"), node("node-02")],
            edges: [legacyConnection],
            layout: { grid: 16 },
            viewport: { x: 12, y: -8, k: 0 },
        });

        expect(migrated).toMatchObject({
            schemaVersion: 2,
            projectId: "project-01",
            layout: { gridSize: 16 },
            connections: [legacyConnection],
            viewport: { x: 12, y: -8, k: 1 },
            groups: [],
        });
        expect("edges" in migrated).toBe(false);
    });

    test("Frame 和 Scene parent 关系生成稳定 group projection", () => {
        const frame = { ...node("scene-frame", CanvasNodeType.Frame), filmKind: "scene" as const, metadata: { frame: { collapsed: true, expandedWidth: 640, expandedHeight: 480 } } };
        const child = { ...node("shot-01"), parentId: frame.id };

        expect(deriveCanvasDocumentGroups([frame, child])).toEqual([
            { id: frame.id, kind: "scene", nodeIds: [child.id], title: frame.title, collapsed: true },
        ]);
    });

    test("未知未来影视节点和更高 schemaVersion 原样保留", () => {
        const futureNode = { ...node("future-node"), filmKind: "future_film_kind", metadata: { content: "未来版本数据" } } as unknown as CanvasNodeData;
        const migrated = migrateCanvasProjectDocument({ schemaVersion: 7, nodes: [futureNode] });

        expect(migrated.schemaVersion).toBe(7);
        expect(migrated.nodes[0]).toEqual(futureNode);
    });

    test("已标准化文档重复迁移不会改变数据语义", () => {
        const first = migrateCanvasProjectDocument({ nodes: [node("plain")] });
        const second = migrateCanvasProjectDocument(first);

        expect(second).toEqual(first);
    });
});
