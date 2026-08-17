import { describe, expect, test } from "bun:test";

import {
    filmGenerationResultConnectionId,
    filmGenerationResultNodeId,
    projectFilmGenerationResultNode,
    reconcileFilmGenerationResultConnections,
    videoResultMediaFromGenerationTask,
} from "../src/lib/canvas/canvas-film-generation-result";
import { isFilmGenerationResultMediaProjection } from "../src/film/domain/node-projection";
import type { GenerationTask } from "../src/services/api/task-center";
import { CanvasNodeType, type CanvasNodeData } from "../src/types/canvas";

function videoTask(overrides: Partial<GenerationTask> = {}): GenerationTask {
    return {
        id: "task-video-1",
        type: "canvas_video",
        status: "succeeded",
        prompt: "短剧镜头",
        attempts: 1,
        createdAt: "2026-08-17T00:00:00.000Z",
        updatedAt: "2026-08-17T00:00:01.000Z",
        inputJson: JSON.stringify({ canvasNodeId: "request-source" }),
        resultJson: JSON.stringify({
            mode: "video",
            video: {
                url: "https://media.example.test/task-video-1.mp4",
                storageKey: "resource:video-resource-1",
                mimeType: "video/mp4",
                width: 1080,
                height: 1920,
                durationMs: 5000,
            },
        }),
        ...overrides,
    };
}

function generationNode(id = "request-source"): CanvasNodeData {
    return {
        id,
        type: CanvasNodeType.Text,
        filmKind: "generation",
        domainRef: {
            projectId: "project-1",
            artifactId: "generation-request-1",
            artifactVersion: "1",
        },
        title: "镜头生成请求",
        position: { x: 120, y: 80 },
        width: 320,
        height: 180,
    };
}

describe("Film generation result projection", () => {
    test("uses the immutable Task input source and creates a playable video projection", () => {
        const task = videoTask();
        const source = generationNode();
        source.domainRef = {
            projectId: "project-1",
            artifactId: "generation-request-1",
            artifactVersion: "1",
            generationAttemptId: "server-attempt",
            providerJobId: "server-job",
        };
        const projected = projectFilmGenerationResultNode([source, generationNode("stale-polling-binding")], task, "stale-polling-binding");
        const result = projected.find((node) => node.id === filmGenerationResultNodeId(task.id));
        if (!result) throw new Error("expected a result node");

        expect(result.type).toBe(CanvasNodeType.Video);
        expect(result.filmKind).toBe("result");
        expect(result.metadata?.content).toBe("https://media.example.test/task-video-1.mp4");
        expect(result.metadata?.generationSourceNodeId).toBe("request-source");
        expect(result.domainRef?.taskId).toBe(task.id);
        expect(result.domainRef?.generationAttemptId).toBeUndefined();
        expect(result.domainRef?.providerJobId).toBeUndefined();
        expect(isFilmGenerationResultMediaProjection(result)).toBe(true);
        expect(reconcileFilmGenerationResultConnections(projected, [])).toEqual([
            {
                id: filmGenerationResultConnectionId(task.id),
                fromNodeId: "request-source",
                toNodeId: result.id,
                edgeType: "derivation",
                filmPorts: { from: "generation_job", to: "generation_job" },
            },
        ]);
    });

    test("accepts a string video URL in the Task result", () => {
        const media = videoResultMediaFromGenerationTask(videoTask({ resultJson: JSON.stringify({ video: " https://media.example.test/string-result.mp4 " }) }));
        expect(media).toEqual({ content: "https://media.example.test/string-result.mp4", mimeType: "video/mp4" });
    });

    test("preserves a manually placed result node while refreshing media", () => {
        const task = videoTask();
        const resultId = filmGenerationResultNodeId(task.id);
        const existing: CanvasNodeData = {
            id: resultId,
            type: CanvasNodeType.Video,
            filmKind: "result",
            domainRef: { projectId: "project-1", taskId: task.id },
            title: "导演确认版",
            position: { x: 900, y: 420 },
            width: 640,
            height: 360,
            layout: { mode: "manual", lane: "shot_pipeline" },
            metadata: { content: "https://media.example.test/old.mp4" },
        };

        const projected = projectFilmGenerationResultNode([generationNode(), existing], task, "request-source");
        const result = projected.find((node) => node.id === resultId);
        if (!result) throw new Error("expected an existing result node");

        expect(result.title).toBe("导演确认版");
        expect(result.position).toEqual({ x: 900, y: 420 });
        expect(result.width).toBe(640);
        expect(result.height).toBe(360);
        expect(result.layout).toEqual({ mode: "manual", lane: "shot_pipeline" });
        expect(result.metadata?.content).toBe("https://media.example.test/task-video-1.mp4");
    });

    test("does not create or retain a system edge after the recorded source is deleted", () => {
        const task = videoTask({ inputJson: JSON.stringify({ canvasNodeId: "deleted-generation-node" }) });
        const nodes = [generationNode("current-polling-binding")];
        expect(projectFilmGenerationResultNode(nodes, task, "current-polling-binding")).toBe(nodes);
        expect(reconcileFilmGenerationResultConnections(nodes, [{
            id: filmGenerationResultConnectionId(task.id),
            fromNodeId: "deleted-generation-node",
            toNodeId: filmGenerationResultNodeId(task.id),
            edgeType: "derivation",
        }])).toEqual([]);
    });
});
