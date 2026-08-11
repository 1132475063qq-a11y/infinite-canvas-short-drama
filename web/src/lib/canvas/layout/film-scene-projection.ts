import { formatFilmSceneTitle, hasFilmSceneProjection } from "@/film/domain/scene-projection";
import { createFilmCanvasNode } from "@/lib/canvas/canvas-project-domain";
import { CanvasNodeType, type CanvasNodeData } from "@/types/canvas";

import { applyFilmAutoLayout } from "./layout-engine";
import { FILM_SCENE_LANE } from "./film-layout-presets";

type SceneProjectionRecord = {
    id: string;
    code: string;
    title: string;
    position: number;
};

/**
 * Old project canvases can contain a Shot projection without the matching
 * Scene projection. Rebuild the missing visual projection from the project
 * record, without creating another Scene or changing the underlying Shot.
 */
export function reconcileFilmSceneProjections(nodes: CanvasNodeData[], projectId: string, scenes: SceneProjectionRecord[]): CanvasNodeData[] {
    const scenesById = new Map(scenes.map((scene) => [scene.id, scene]));
    const missingSceneIds = Array.from(
        new Set(
            nodes
                .filter((node) => node.filmKind === "shot" && Boolean(node.domainRef?.sceneId))
                .map((node) => node.domainRef!.sceneId!)
                .filter((sceneId) => !hasFilmSceneProjection(nodes, sceneId) && scenesById.has(sceneId)),
        ),
    );
    if (!missingSceneIds.length) return nodes;

    const projectedScenes = missingSceneIds.map((sceneId) => {
        const scene = scenesById.get(sceneId)!;
        const firstShot = nodes.find((node) => node.filmKind === "shot" && node.domainRef?.sceneId === sceneId)!;
        const node = createFilmCanvasNode(
            CanvasNodeType.Frame,
            "scene",
            {
                x: firstShot.position.x - FILM_SCENE_LANE.shotOffsetX,
                y: firstShot.position.y - FILM_SCENE_LANE.shotOffsetY,
            },
            { projectId, sceneId: scene.id },
            { workflowKind: "scene", workflowTitle: scene.code || "场景" },
        );
        node.title = formatFilmSceneTitle(scene.code, scene.title);
        node.width = FILM_SCENE_LANE.sceneWidth;
        node.height = FILM_SCENE_LANE.sceneHeight;
        node.position = {
            x: firstShot.position.x - FILM_SCENE_LANE.shotOffsetX,
            y: firstShot.position.y - FILM_SCENE_LANE.shotOffsetY,
        };
        node.layout = { mode: "auto", lane: "scene", order: scene.position };
        return node;
    });

    return applyFilmAutoLayout([...nodes, ...projectedScenes]);
}
