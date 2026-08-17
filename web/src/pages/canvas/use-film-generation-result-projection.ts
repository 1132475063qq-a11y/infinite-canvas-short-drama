import { useEffect, useMemo, useRef } from "react";
import type { Dispatch, SetStateAction } from "react";

import {
    hasFilmGenerationResultMedia,
    projectFilmGenerationResultNode,
    reconcileFilmGenerationResultConnections,
    videoResultMediaFromGenerationTask,
} from "@/lib/canvas/canvas-film-generation-result";
import type { CanvasGridSize } from "@/lib/canvas/layout/layout-types";
import { queryGenerationTask } from "@/services/api/task-center";
import type { CanvasConnection, CanvasNodeData } from "@/types/canvas";

type FilmGenerationBinding = { taskId: string; nodeId: string };

type UseFilmGenerationResultProjectionOptions = {
    projectLoaded: boolean;
    nodes: CanvasNodeData[];
    gridSize: CanvasGridSize;
    setNodes: Dispatch<SetStateAction<CanvasNodeData[]>>;
    setConnections: Dispatch<SetStateAction<CanvasConnection[]>>;
};

// This polling path intentionally queries a task without waiting on it. The
// wait helper treats AbortSignal cancellation as a request to cancel the Task,
// which is not appropriate when a Canvas page is unmounted or refreshed.
export function useFilmGenerationResultProjection({ projectLoaded, nodes, gridSize, setNodes, setConnections }: UseFilmGenerationResultProjectionOptions) {
    const terminalWithoutVideoRef = useRef(new Set<string>());
    const bindings = useMemo(() => collectFilmGenerationBindings(nodes), [nodes]);
    const bindingKey = bindings.map((binding) => `${binding.taskId}:${binding.nodeId}`).join("|");

    useEffect(() => {
        if (!projectLoaded) return;
        setConnections((current) => reconcileFilmGenerationResultConnections(nodes, current));
    }, [nodes, projectLoaded, setConnections]);

    useEffect(() => {
        if (!projectLoaded || !bindings.length) return;
        const candidates = bindings.filter((binding) => !hasFilmGenerationResultMedia(nodes, binding.taskId) && !terminalWithoutVideoRef.current.has(binding.taskId));
        if (!candidates.length) return;

        let active = true;
        let polling = false;
        const controller = new AbortController();
        const poll = async () => {
            if (polling) return;
            polling = true;
            try {
                const results = await Promise.all(candidates.map(async (binding) => ({ binding, task: await queryGenerationTask(binding.taskId, { signal: controller.signal }).catch(() => null) })));
                if (!active) return;
                for (const { binding, task } of results) {
                    if (!task) continue;
                    if (task.status === "failed" || task.status === "cancelled") {
                        terminalWithoutVideoRef.current.add(task.id);
                        continue;
                    }
                    if (task.status !== "succeeded") continue;
                    if (!videoResultMediaFromGenerationTask(task)) {
                        terminalWithoutVideoRef.current.add(task.id);
                        continue;
                    }
                    setNodes((current) => {
                        return projectFilmGenerationResultNode(current, task, binding.nodeId, gridSize);
                    });
                }
            } finally {
                polling = false;
            }
        };

        void poll();
        const timer = window.setInterval(() => void poll(), 2_000);
        return () => {
            active = false;
            controller.abort();
            window.clearInterval(timer);
        };
    }, [bindingKey, bindings, gridSize, nodes, projectLoaded, setNodes]);
}

function collectFilmGenerationBindings(nodes: CanvasNodeData[]) {
    const unique = new Map<string, FilmGenerationBinding>();
    for (const node of nodes) {
        if (node.filmKind !== "generation") continue;
        const taskId = node.domainRef?.taskId || node.metadata?.taskId;
        if (taskId?.trim() && !unique.has(taskId)) unique.set(taskId, { taskId, nodeId: node.id });
    }
    return [...unique.values()];
}
