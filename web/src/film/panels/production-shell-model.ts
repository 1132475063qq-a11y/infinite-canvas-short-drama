import type { ProjectDetail, ProjectScene, ProjectShot } from "@/services/api/projects";
import type { GenerationTask } from "@/services/api/task-center";
import type { CanvasNodeData } from "@/types/canvas";

export type ProductionNavigationKey = "overview" | "story" | "script" | "scenes" | "characters" | "locations" | "props" | "storyboard" | "shots" | "assets" | "audio" | "qc" | "agents" | "needs_you" | "delivery";

export type ProductionNavigationCounts = Record<ProductionNavigationKey, number>;

export const EMPTY_PRODUCTION_NAVIGATION_COUNTS: ProductionNavigationCounts = {
    overview: 0,
    story: 0,
    script: 0,
    scenes: 0,
    characters: 0,
    locations: 0,
    props: 0,
    storyboard: 0,
    shots: 0,
    assets: 0,
    audio: 0,
    qc: 0,
    agents: 0,
    needs_you: 0,
    delivery: 0,
};

export type ProductionShellModel = {
    projectName: string;
    currentScene?: ProjectScene;
    shots: ProjectShot[];
    totalShots: number;
    approvedShots: number;
    runningTasks: number;
    qcFailed: number;
    needsYou: number;
    navigationCounts: ProductionNavigationCounts;
};

const APPROVED_STATUSES = new Set(["approved", "completed", "succeeded", "done"]);
const QC_FAILED_STATUSES = new Set(["qc_failed", "failed"]);

export function buildProductionShellModel(detail: ProjectDetail, nodes: readonly CanvasNodeData[], tasks: readonly GenerationTask[], selectedNode?: CanvasNodeData | null): ProductionShellModel {
    const scenes = detail.scenes.slice().sort((left, right) => left.position - right.position);
    const allShots = detail.shots.slice().sort((left, right) => left.position - right.position);
    const selectedSceneId = selectedNode?.domainRef?.sceneId || (selectedNode?.domainRef?.shotId ? allShots.find((shot) => shot.id === selectedNode.domainRef?.shotId)?.sceneId : undefined);
    const currentScene = scenes.find((scene) => scene.id === selectedSceneId) || scenes[0];
    const shots = currentScene ? allShots.filter((shot) => shot.sceneId === currentScene.id) : allShots;
    const qcFailedShotIds = new Set(allShots.filter((shot) => QC_FAILED_STATUSES.has(shot.status.toLocaleLowerCase())).map((shot) => shot.id));
    nodes.forEach((node) => {
        if (node.filmKind === "qc" && node.filmState?.production === "qc_failed" && node.domainRef?.shotId) qcFailedShotIds.add(node.domainRef.shotId);
    });
    const needsYou = nodes.filter((node) => node.filmKind === "needs_you" || node.filmState?.attention === "human_required").length;
    const assetCounts = {
        characters: detail.assets.filter((asset) => asset.category === "character").length,
        locations: detail.assets.filter((asset) => asset.category === "environment" || asset.category === "location").length,
        props: detail.assets.filter((asset) => asset.category === "prop").length,
        audio: detail.assets.filter((asset) => asset.mediaType === "audio").length,
    };

    return {
        projectName: detail.project.name,
        currentScene,
        shots,
        totalShots: allShots.length,
        approvedShots: allShots.filter((shot) => APPROVED_STATUSES.has(shot.status.toLocaleLowerCase())).length,
        runningTasks: tasks.filter((task) => task.status === "queued" || task.status === "running").length,
        qcFailed: qcFailedShotIds.size + nodes.filter((node) => node.filmKind === "qc" && node.filmState?.production === "qc_failed" && !node.domainRef?.shotId).length,
        needsYou,
        navigationCounts: {
            overview: 1,
            story: detail.project.description.trim() ? 1 : 0,
            script: detail.units.length,
            scenes: scenes.length,
            characters: assetCounts.characters,
            locations: assetCounts.locations,
            props: assetCounts.props,
            storyboard: nodes.filter((node) => node.filmKind === "storyboard").length,
            shots: allShots.length,
            assets: detail.assets.length,
            audio: assetCounts.audio,
            qc: nodes.filter((node) => node.filmKind === "qc").length,
            agents: nodes.filter((node) => node.filmKind === "agent_task").length,
            needs_you: needsYou,
            delivery: nodes.filter((node) => node.filmKind === "delivery").length,
        },
    };
}

export function formatShotCode(shot: ProjectShot, index: number): string {
    const explicitCode = shot.title.match(/(?:SC\d+[-_])?SH\d+/i)?.[0];
    return explicitCode?.toUpperCase().replace("_", "-") || `SH${String(index + 1).padStart(3, "0")}`;
}
