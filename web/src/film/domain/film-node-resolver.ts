import type { FilmArtifact, ProjectAsset, ProjectDetail, ProjectScene, ProjectShot } from "@/services/api/projects";
import type { CanvasNodeData } from "@/types/canvas";

export type ResolvedFilmNode = {
    scene?: ProjectScene;
    shot?: ProjectShot;
    asset?: ProjectAsset;
    artifact?: FilmArtifact;
    contract: Record<string, unknown>;
    definition: Record<string, unknown>;
    sceneShots: ProjectShot[];
    usedShotCount: number;
};

export function resolveFilmNode(node: CanvasNodeData, project?: ProjectDetail): ResolvedFilmNode {
    const ref = node.domainRef;
    const scenes = project?.scenes || [];
    const shots = project?.shots || [];
    const assets = project?.assets || [];
    const shotReferences = project?.shotReferences || [];
    const scene = ref?.sceneId ? scenes.find((item) => item.id === ref.sceneId) : undefined;
    const shot = ref?.shotId ? shots.find((item) => item.id === ref.shotId) : undefined;
    const asset = ref?.assetId ? assets.find((item) => item.id === ref.assetId) : undefined;
    const artifact = resolveArtifact(project, node.filmKind, ref?.artifactId || shot?.contractArtifactId, shot?.id);
    const versionIds = new Set([asset?.primaryVersionId, ref?.assetVersionId].filter(Boolean));

    return {
        scene,
        shot,
        asset,
        artifact,
        contract: parseObject(artifact?.payloadJson),
        definition: asset?.character?.definition || asset?.currentVersion?.definition || {},
        sceneShots: scene ? shots.filter((item) => item.sceneId === scene.id) : [],
        usedShotCount: shotReferences.filter((item) => versionIds.has(item.assetVersionId)).length,
    };
}

function resolveArtifact(project: ProjectDetail | undefined, filmKind: CanvasNodeData["filmKind"], artifactId: string | undefined, shotId: string | undefined) {
    if (!project) return undefined;
    const artifactType = artifactTypeForFilmNode(filmKind);
    if (artifactType && shotId) {
        const latest = (project.filmArtifacts || [])
            .filter((item) => item.shotId === shotId && item.artifactType === artifactType)
            .sort((left, right) => right.objectVersion - left.objectVersion)[0];
        if (latest) return latest;
    }
    if (artifactId) {
        const exact = project.filmArtifacts?.find((item) => item.id === artifactId);
        if (exact) return exact;
    }
    return undefined;
}

export function artifactTypeForFilmNode(filmKind: CanvasNodeData["filmKind"]) {
    if (filmKind === "shot") return "shot_contract";
    if (filmKind === "acting") return "acting";
    if (filmKind === "prompt_pack") return "video_prompt_pack";
    return undefined;
}

export function parseObject(value: string | undefined): Record<string, unknown> {
    if (!value) return {};
    try {
        const parsed: unknown = JSON.parse(value);
        return parsed && typeof parsed === "object" && !Array.isArray(parsed) ? (parsed as Record<string, unknown>) : {};
    } catch {
        return {};
    }
}

export function filmText(value: unknown) {
    if (Array.isArray(value)) return value.map(String).join("、");
    return typeof value === "string" || typeof value === "number" ? String(value) : "";
}
