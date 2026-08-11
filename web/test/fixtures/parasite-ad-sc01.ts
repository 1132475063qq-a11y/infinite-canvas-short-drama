import { CanvasNodeType, type CanvasNodeData, type CanvasProjectDocument } from "../../src/types/canvas";

const projectId = "golden-parasite-ad";
const sceneId = "golden-parasite-ad-sc01";

export const parasiteAdSc01Fixture = {
    project: { id: projectId, name: "寄生广告" },
    scene: { id: sceneId, code: "SC01", title: "黑市诊所", position: 0 },
    shot: { id: "golden-parasite-ad-sc01-sh005", title: "SC01-SH005", position: 4, durationMs: 5000 },
    expectedReferenceAssets: ["大头 Base", "小皮 Base", "富豪", "Clinic Location", "Patient Chair", "Bridge Chair", "VR Helmet", "Console"],
    document: {
        schemaVersion: 2,
        layout: { gridSize: 8 },
        nodes: [
            {
                id: "golden-sc01-node",
                type: CanvasNodeType.Frame,
                filmKind: "scene",
                domainRef: { projectId, sceneId },
                filmState: { lifecycle: "draft", production: "not_started", evidence: "recorded", attention: "none" },
                layout: { mode: "auto", lane: "scene", order: 0 },
                title: "SC01 · 黑市诊所",
                position: { x: 0, y: 0 },
                width: 420,
                height: 240,
            },
            {
                id: "golden-sc01-sh005-node",
                type: CanvasNodeType.Text,
                filmKind: "shot",
                domainRef: { projectId, sceneId, shotId: "golden-parasite-ad-sc01-sh005" },
                filmState: { lifecycle: "draft", production: "not_started", evidence: "recorded", attention: "none" },
                layout: { mode: "auto", lane: "shot", order: 4 },
                title: "SC01-SH005 · Shot Contract",
                position: { x: 48, y: 288 },
                width: 320,
                height: 180,
            },
        ] satisfies CanvasNodeData[],
    } satisfies CanvasProjectDocument,
} as const;
