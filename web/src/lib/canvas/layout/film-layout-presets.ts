import type { CanvasNodeData, Position } from "@/types/canvas";

export const FILM_SCENE_LANE = {
    sceneWidth: 420,
    sceneHeight: 240,
    sceneGap: 96,
    shotOffsetX: 48,
    shotOffsetY: 288,
    shotGap: 32,
    shotHeight: 180,
    pipelineGap: 48,
    actingWidth: 300,
    promptPackWidth: 340,
} as const;

type SceneLane = {
    position: Position;
    shotNodes: CanvasNodeData[];
};

// 仅整理显式标记为 auto 的影视节点；用户手动摆放的节点不会被布局引擎覆盖。
export function layoutFilmSceneLanes(nodes: CanvasNodeData[], origin: Position = { x: 0, y: 0 }): Map<string, Position> {
    const positions = new Map<string, Position>();
    const sceneNodes = nodes
        .filter((node) => node.filmKind === "scene")
        .sort(compareFilmLayoutOrder);
    const lanes = new Map<string, SceneLane>();
    const laneOrigin = sceneNodes[0]?.position || origin;
    let nextSceneY = laneOrigin.y;

    sceneNodes.forEach((scene) => {
        const shotNodes = nodes
            .filter((node) => node.filmKind === "shot" && node.layout?.mode === "auto" && node.domainRef?.sceneId === scene.domainRef?.sceneId)
            .sort(compareFilmLayoutOrder);
        const position = {
            x: laneOrigin.x,
            y: nextSceneY,
        };
        const scenePosition = scene.layout?.mode === "auto" ? position : scene.position;
        if (scene.layout?.mode === "auto") positions.set(scene.id, scenePosition);
        if (scene.domainRef?.sceneId) {
            lanes.set(scene.domainRef.sceneId, {
                position: scenePosition,
                shotNodes,
            });
        }
        const sceneBottom = scenePosition.y + FILM_SCENE_LANE.sceneHeight;
        const shotsBottom = shotNodes.length
            ? scenePosition.y + FILM_SCENE_LANE.shotOffsetY + shotNodes.reduce((total, shot) => total + shot.height, 0) + Math.max(0, shotNodes.length - 1) * FILM_SCENE_LANE.shotGap
            : sceneBottom;
        nextSceneY = Math.max(sceneBottom, shotsBottom) + FILM_SCENE_LANE.sceneGap;
    });

    lanes.forEach((lane) => {
        let nextShotY = lane.position.y + FILM_SCENE_LANE.shotOffsetY;
        lane.shotNodes.forEach((shot) => {
            positions.set(shot.id, { x: lane.position.x + FILM_SCENE_LANE.shotOffsetX, y: nextShotY });
            nextShotY += shot.height + FILM_SCENE_LANE.shotGap;
        });
    });

    // The per-shot production chain grows horizontally so the scene and shot
    // ordering remain readable: Shot -> Acting -> Prompt Pack -> Generation Request. Only auto nodes
    // participate; manual and pinned projections preserve the user's layout.
    nodes.filter((node) => node.filmKind === "shot").forEach((shot) => {
        const shotPosition = positions.get(shot.id) || shot.position;
        const actingNodes = pipelineNodes(nodes, shot, "acting");
        const promptNodes = pipelineNodes(nodes, shot, "prompt_pack");
        const generationNodes = pipelineNodes(nodes, shot, "generation");
        actingNodes.forEach((node, index) => positions.set(node.id, {
            x: shotPosition.x + shot.width + FILM_SCENE_LANE.pipelineGap,
            y: shotPosition.y + index * (node.height + FILM_SCENE_LANE.shotGap),
        }));
        promptNodes.forEach((node, index) => positions.set(node.id, {
            x: shotPosition.x + shot.width + FILM_SCENE_LANE.pipelineGap + FILM_SCENE_LANE.actingWidth + FILM_SCENE_LANE.pipelineGap,
            y: shotPosition.y + index * (node.height + FILM_SCENE_LANE.shotGap),
        }));
        generationNodes.forEach((node, index) => positions.set(node.id, {
            x: shotPosition.x + shot.width + FILM_SCENE_LANE.pipelineGap + FILM_SCENE_LANE.actingWidth + FILM_SCENE_LANE.pipelineGap + FILM_SCENE_LANE.promptPackWidth + FILM_SCENE_LANE.pipelineGap,
            y: shotPosition.y + index * (node.height + FILM_SCENE_LANE.shotGap),
        }));
    });

    return positions;
}

function pipelineNodes(nodes: CanvasNodeData[], shot: CanvasNodeData, kind: "acting" | "prompt_pack" | "generation") {
    return nodes
        .filter((node) => node.filmKind === kind && node.layout?.mode === "auto" && node.domainRef?.shotId === shot.domainRef?.shotId)
        .sort(compareFilmLayoutOrder);
}

function compareFilmLayoutOrder(first: CanvasNodeData, second: CanvasNodeData) {
    return (first.layout?.order ?? Number.MAX_SAFE_INTEGER) - (second.layout?.order ?? Number.MAX_SAFE_INTEGER)
        || first.id.localeCompare(second.id);
}
