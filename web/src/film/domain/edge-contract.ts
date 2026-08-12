import type { CanvasNodeData } from "@/types/canvas";

import { isFilmProductionProjection } from "./node-projection";
import type { FilmConnectionPorts, FilmEdgeType, FilmNodeKind, FilmNodePort } from "./types";

const REFERENCE_KINDS = new Set<FilmNodeKind>(["character", "location", "prop"]);

type FilmConnectionSemantic = { edgeType: FilmEdgeType; filmPorts: FilmConnectionPorts };

export type FilmConnectionValidation =
    | { allowed: true; semantic?: FilmConnectionSemantic }
    | { allowed: false; reason: string };

type FilmConnectionRule = FilmConnectionSemantic & { from: FilmNodeKind; to: FilmNodeKind };

const FILM_CONNECTION_RULES: FilmConnectionRule[] = [
    rule("project", "story", "dependency", "scene_context", "scene_context"),
    rule("project", "script", "dependency", "scene_context", "shot_contract"),
    rule("project", "scene", "continuity", "scene_context", "scene_context"),
    rule("story", "script", "dependency", "scene_context", "shot_contract"),
    rule("story", "scene", "continuity", "scene_context", "scene_context"),
    rule("script", "scene", "continuity", "shot_contract", "scene_context"),
    rule("script", "shot", "dependency", "shot_contract", "shot_contract"),
    rule("scene", "shot", "continuity", "scene_context", "scene_context"),
    rule("scene", "acting", "continuity", "scene_context", "acting_direction"),
    rule("shot", "acting", "dependency", "shot_contract", "shot_contract"),
    rule("shot", "prompt_pack", "dependency", "shot_contract", "shot_contract"),
    rule("shot", "qc", "dependency", "shot_contract", "shot_contract"),
    rule("character", "shot", "reference", "asset_reference", "asset_reference"),
    rule("location", "shot", "reference", "asset_reference", "asset_reference"),
    rule("prop", "shot", "reference", "asset_reference", "asset_reference"),
    rule("character", "prompt_pack", "reference", "asset_reference", "asset_reference"),
    rule("location", "prompt_pack", "reference", "asset_reference", "asset_reference"),
    rule("prop", "prompt_pack", "reference", "asset_reference", "asset_reference"),
    rule("acting", "prompt_pack", "dependency", "acting_direction", "acting_direction"),
    rule("prompt_pack", "generation", "dependency", "prompt", "prompt"),
    rule("generation", "result", "derivation", "generation_job", "generation_job"),
    rule("result", "qc", "derivation", "result", "result"),
    rule("qc", "retry", "authority", "qc_decision", "qc_decision"),
    rule("qc", "needs_you", "authority", "qc_decision", "qc_decision"),
    rule("retry", "generation", "dependency", "retry_request", "retry_request"),
    rule("needs_you", "retry", "authority", "generic", "retry_request"),
    rule("agent_task", "needs_you", "authority", "generic", "generic"),
];

const RULES_BY_PAIR = new Map(FILM_CONNECTION_RULES.map((item) => [`${item.from}->${item.to}`, item]));

export function filmOutputPort(kind: FilmNodeKind): FilmNodePort {
    if (["project", "story", "scene"].includes(kind)) return "scene_context";
    if (["script", "shot"].includes(kind)) return "shot_contract";
    if (REFERENCE_KINDS.has(kind)) return "asset_reference";
    if (kind === "acting") return "acting_direction";
    if (kind === "prompt_pack") return "prompt";
    if (kind === "generation") return "generation_job";
    if (kind === "result") return "result";
    if (kind === "qc") return "qc_decision";
    if (kind === "retry") return "retry_request";
    return "generic";
}

export function filmInputPort(kind: FilmNodeKind): FilmNodePort {
    if (["scene", "shot"].includes(kind)) return "scene_context";
    if (REFERENCE_KINDS.has(kind)) return "asset_reference";
    if (kind === "acting") return "acting_direction";
    if (kind === "prompt_pack") return "shot_contract";
    if (kind === "generation") return "prompt";
    if (kind === "result") return "generation_job";
    if (kind === "qc") return "result";
    if (kind === "retry") return "qc_decision";
    return "generic";
}

export function filmEdgeType(from: FilmNodeKind, to: FilmNodeKind): FilmEdgeType {
    const ruleMatch = RULES_BY_PAIR.get(`${from}->${to}`);
    if (ruleMatch) return ruleMatch.edgeType;
    if (REFERENCE_KINDS.has(from)) return "reference";
    if (from === "generation" || from === "result") return "derivation";
    if (from === "qc" || from === "needs_you") return "authority";
    if (from === "scene" && to === "shot") return "continuity";
    return "dependency";
}

/**
 * Legacy or mixed nodes keep the host canvas' free-form connection behavior.
 * Only two database-backed Film projections enter the production rule matrix.
 */
export function validateFilmConnection(from?: CanvasNodeData, to?: CanvasNodeData): FilmConnectionValidation {
    if (!isFilmProductionProjection(from) || !isFilmProductionProjection(to)) return { allowed: true };
    if (from.domainRef?.projectId !== to.domainRef?.projectId) {
        return { allowed: false, reason: "影视生产节点不能跨项目连接" };
    }
    if (from.filmKind === "scene" && to.filmKind === "shot" && from.domainRef?.sceneId && to.domainRef?.sceneId && from.domainRef.sceneId !== to.domainRef.sceneId) {
        return { allowed: false, reason: "镜头只能连接到自己所属的场景" };
    }
    const ruleMatch = RULES_BY_PAIR.get(`${from.filmKind}->${to.filmKind}`);
    if (!ruleMatch) {
        return { allowed: false, reason: `${filmKindLabel(from.filmKind)}不能连接到${filmKindLabel(to.filmKind)}` };
    }
    return {
        allowed: true,
        semantic: {
            edgeType: ruleMatch.edgeType,
            filmPorts: ruleMatch.filmPorts,
        },
    };
}

export function describeFilmConnection(from?: CanvasNodeData, to?: CanvasNodeData): FilmConnectionSemantic | undefined {
    const validation = validateFilmConnection(from, to);
    return validation.allowed ? validation.semantic : undefined;
}

function rule(from: FilmNodeKind, to: FilmNodeKind, edgeType: FilmEdgeType, output: FilmNodePort, input: FilmNodePort): FilmConnectionRule {
    return { from, to, edgeType, filmPorts: { from: output, to: input } };
}

function filmKindLabel(kind: FilmNodeKind) {
    return ({
        project: "项目",
        story: "故事",
        script: "剧本",
        scene: "场景",
        shot: "镜头",
        character: "角色",
        location: "场地",
        prop: "道具",
        acting: "表演",
        prompt_pack: "提示词包",
        generation: "生成任务",
        result: "生成结果",
        qc: "质检",
        retry: "重试",
        needs_you: "需要人工处理",
        agent_task: "Agent 任务",
    } satisfies Record<FilmNodeKind, string>)[kind];
}
