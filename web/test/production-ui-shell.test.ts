import { describe, expect, test } from "bun:test";

import { buildProductionShellModel, formatShotCode } from "../src/film/panels/production-shell-model";
import type { ProjectDetail, ProjectShot } from "../src/services/api/projects";
import type { GenerationTask } from "../src/services/api/task-center";
import { CanvasNodeType, type CanvasNodeData } from "../src/types/canvas";

const NOW = "2026-08-12T00:00:00.000Z";

function fixture(): ProjectDetail {
    return {
        project: { id: "project-01", userId: "user-01", name: "寄生广告", type: "short_drama", aspectRatio: "9:16", sourceType: "script", description: "第一场测试", stylePresetId: "", status: "active", revision: 3, createdAt: NOW, updatedAt: NOW },
        units: [{ id: "unit-01", projectId: "project-01", kind: "chapter", title: "第一章", sourceText: "", status: "draft", position: 0, createdAt: NOW, updatedAt: NOW }],
        canvases: [],
        canvasUnitLinks: [],
        assets: [
            { id: "character-01", title: "大头", mediaType: "image", category: "character", status: "ready", versionCount: 1, usages: [], updatedAt: NOW },
            { id: "location-01", title: "诊所", mediaType: "image", category: "environment", status: "ready", versionCount: 1, usages: [], updatedAt: NOW },
            { id: "audio-01", title: "旁白", mediaType: "audio", category: "voice", status: "ready", versionCount: 1, usages: [], updatedAt: NOW },
        ],
        workflows: [],
        scenes: [
            { id: "scene-01", projectId: "project-01", code: "SC01", title: "诊所", description: "", position: 0, status: "draft", createdAt: NOW, updatedAt: NOW },
            { id: "scene-02", projectId: "project-01", code: "SC02", title: "走廊", description: "", position: 1, status: "draft", createdAt: NOW, updatedAt: NOW },
        ],
        shots: [shot("shot-01", "scene-01", "SC01-SH001 · 推门", 0, "approved"), shot("shot-02", "scene-01", "SH002 · 看诊", 1, "qc_failed"), shot("shot-03", "scene-02", "走廊跟拍", 0, "draft")],
        shotReferences: [],
        assetCandidates: [],
    };
}

function shot(id: string, sceneId: string, title: string, position: number, status: string): ProjectShot {
    return { id, projectId: "project-01", sceneId, title, description: "", position, durationMs: 5000, status, createdAt: NOW, updatedAt: NOW };
}

function node(id: string, filmKind: CanvasNodeData["filmKind"], domainRef: CanvasNodeData["domainRef"], attention: NonNullable<CanvasNodeData["filmState"]>["attention"] = "none"): CanvasNodeData {
    return {
        id,
        type: CanvasNodeType.Script,
        filmKind,
        domainRef,
        filmState: { lifecycle: "draft", production: filmKind === "qc" ? "qc_failed" : "not_started", evidence: "recorded", attention },
        title: id,
        position: { x: 0, y: 0 },
        width: 320,
        height: 180,
    };
}

describe("Production UI Shell model", () => {
    test("选中镜头决定当前 Scene，并汇总真实生产状态", () => {
        const detail = fixture();
        const selected = node("shot-node-03", "shot", { projectId: "project-01", sceneId: "scene-02", shotId: "shot-03" });
        const nodes = [
            selected,
            node("qc-node-02", "qc", { projectId: "project-01", sceneId: "scene-01", shotId: "shot-02" }),
            node("needs-you", "needs_you", { projectId: "project-01" }, "human_required"),
            node("agent-task", "agent_task", { projectId: "project-01", agentId: "director" }),
        ];
        const tasks = [{ id: "task-01", type: "video", status: "running", prompt: "", attempts: 1, createdAt: NOW, updatedAt: NOW }] as GenerationTask[];

        const model = buildProductionShellModel(detail, nodes, tasks, selected);

        expect(model.currentScene?.id).toBe("scene-02");
        expect(model.shots.map((item) => item.id)).toEqual(["shot-03"]);
        expect(model.approvedShots).toBe(1);
        expect(model.runningTasks).toBe(1);
        expect(model.qcFailed).toBe(1);
        expect(model.needsYou).toBe(1);
        expect(model.navigationCounts).toMatchObject({ scenes: 2, shots: 3, characters: 1, locations: 1, audio: 1, agents: 1, needs_you: 1 });
    });

    test("Shot Strip 优先显示合同中的显式镜头号，并为普通标题生成序号", () => {
        const detail = fixture();
        expect(formatShotCode(detail.shots[0], 0)).toBe("SC01-SH001");
        expect(formatShotCode(detail.shots[1], 1)).toBe("SH002");
        expect(formatShotCode(detail.shots[2], 2)).toBe("SH003");
    });
});
