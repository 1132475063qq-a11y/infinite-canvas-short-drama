import type { CanvasNodeData, Position } from "@/types/canvas";

export const FILM_SCENE_LANE = {
    sceneWidth: 420,
    sceneHeight: 240,
    sceneGap: 96,
    shotOffsetX: 48,
    shotOffsetY: 288,
    shotGap: 32,
    shotHeight: 180,
} as const;

type SceneLane = {
    position: Position;
    nextShotY: number;
};

// 仅整理显式标记为 auto 的影视节点；用户手动摆放的节点不会被布局引擎覆盖。
export function layoutFilmSceneLanes(nodes: CanvasNodeData[], origin: Position = { x: 0, y: 0 }): Map<string, Position> {
    const positions = new Map<string, Position>();
    const sceneNodes = nodes
        .filter((node) => node.filmKind === "scene")
        .sort(compareFilmLayoutOrder);
    const lanes = new Map<string, SceneLane>();
    const laneOrigin = sceneNodes[0]?.position || origin;

    sceneNodes.forEach((scene, index) => {
        const position = {
            x: laneOrigin.x,
            y: laneOrigin.y + index * (FILM_SCENE_LANE.sceneHeight + FILM_SCENE_LANE.sceneGap),
        };
        const scenePosition = scene.layout?.mode === "auto" ? position : scene.position;
        if (scene.layout?.mode === "auto") positions.set(scene.id, scenePosition);
        if (scene.domainRef?.sceneId) {
            lanes.set(scene.domainRef.sceneId, {
                position: scenePosition,
                nextShotY: scenePosition.y + FILM_SCENE_LANE.shotOffsetY,
            });
        }
    });

    nodes
        .filter((node) => node.filmKind === "shot" && node.layout?.mode === "auto")
        .sort(compareFilmLayoutOrder)
        .forEach((shot) => {
            const lane = shot.domainRef?.sceneId ? lanes.get(shot.domainRef.sceneId) : undefined;
            if (!lane) return;
            positions.set(shot.id, { x: lane.position.x + FILM_SCENE_LANE.shotOffsetX, y: lane.nextShotY });
            lane.nextShotY += shot.height + FILM_SCENE_LANE.shotGap;
        });

    return positions;
}

function compareFilmLayoutOrder(first: CanvasNodeData, second: CanvasNodeData) {
    return (first.layout?.order ?? Number.MAX_SAFE_INTEGER) - (second.layout?.order ?? Number.MAX_SAFE_INTEGER)
        || first.id.localeCompare(second.id);
}
