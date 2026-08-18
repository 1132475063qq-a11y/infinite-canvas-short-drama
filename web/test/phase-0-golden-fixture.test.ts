import { describe, expect, test } from "bun:test";

import { parasiteAdSc01Fixture } from "./fixtures/parasite-ad-sc01";

describe("Phase 0 Golden Fixture", () => {
    test("固定《寄生广告》SC01 的最小生产事实，不依赖 Provider 或 Agent Runtime", () => {
        const { project, scene, shot, document, expectedReferenceAssets } = parasiteAdSc01Fixture;
        const sceneNode = document.nodes.find((node) => node.filmKind === "scene");
        const shotNode = document.nodes.find((node) => node.filmKind === "shot");

        expect(project.name).toBe("寄生广告");
        expect(scene).toMatchObject({ code: "SC01", title: "黑市诊所" });
        expect(shot).toMatchObject({ title: "SC01-SH005", durationMs: 5000 });
        expect(expectedReferenceAssets).toEqual(["大头 Base", "小皮 Base", "富豪", "Clinic Location", "Patient Chair", "Bridge Chair", "VR Helmet", "Console"]);
        expect(document).toMatchObject({ schemaVersion: 2, layout: { gridSize: 8 } });
        expect(sceneNode?.domainRef).toEqual({ projectId: project.id, sceneId: scene.id });
        expect(shotNode?.domainRef).toEqual({ projectId: project.id, sceneId: scene.id, shotId: shot.id });
    });
});
