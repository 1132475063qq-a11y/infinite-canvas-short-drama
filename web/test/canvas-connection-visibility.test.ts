import { describe, expect, test } from "bun:test";

import { defaultCanvasConnectionVisibilityMode, nextCanvasConnectionVisibilityMode, visibleCanvasConnectionIds } from "@/lib/canvas/canvas-connection-visibility";
import { CanvasNodeType, type CanvasConnection, type CanvasNodeData } from "@/types/canvas";

const connections: CanvasConnection[] = [
    { id: "scene-shot", fromNodeId: "scene", toNodeId: "shot" },
    { id: "shot-character", fromNodeId: "shot", toNodeId: "character" },
    { id: "location-prop", fromNodeId: "location", toNodeId: "prop" },
];

function node(patch: Partial<CanvasNodeData> = {}): CanvasNodeData {
    return {
        id: "node",
        type: CanvasNodeType.Text,
        title: "节点",
        position: { x: 0, y: 0 },
        width: 320,
        height: 180,
        ...patch,
    };
}

describe("canvas connection visibility", () => {
    test("film production canvases default to focused connections", () => {
        expect(defaultCanvasConnectionVisibilityMode([node({ filmKind: "scene", domainRef: { projectId: "project-1", sceneId: "scene-1" } })])).toBe("focus");
        expect(defaultCanvasConnectionVisibilityMode([node({ filmKind: "scene" })])).toBe("all");
        expect(defaultCanvasConnectionVisibilityMode([node()])).toBe("all");
    });

    test("focus mode includes direct and explicitly selected connections", () => {
        expect([...visibleCanvasConnectionIds(connections, "focus", "shot", null)]).toEqual(["scene-shot", "shot-character"]);
        expect([...visibleCanvasConnectionIds(connections, "focus", null, "location-prop")]).toEqual(["location-prop"]);
        expect([...visibleCanvasConnectionIds(connections, "focus", null, null)]).toEqual([]);
    });

    test("all and hidden modes are deterministic", () => {
        expect([...visibleCanvasConnectionIds(connections, "all", null, null)]).toEqual(connections.map((connection) => connection.id));
        expect([...visibleCanvasConnectionIds(connections, "hidden", "shot", "scene-shot")]).toEqual([]);
    });

    test("visibility control cycles focus, all, hidden", () => {
        expect(nextCanvasConnectionVisibilityMode("focus")).toBe("all");
        expect(nextCanvasConnectionVisibilityMode("all")).toBe("hidden");
        expect(nextCanvasConnectionVisibilityMode("hidden")).toBe("focus");
    });
});
