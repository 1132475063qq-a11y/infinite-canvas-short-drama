import type { CanvasNodeData } from "@/types/canvas";

import type { FilmConnectionPorts, FilmEdgeType, FilmNodeKind, FilmNodePort } from "./types";

const REFERENCE_KINDS = new Set<FilmNodeKind>(["character", "location", "prop"]);

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
    if (REFERENCE_KINDS.has(from)) return "reference";
    if (from === "generation" || from === "result") return "derivation";
    if (from === "qc" || from === "needs_you") return "authority";
    if (from === "scene" && to === "shot") return "continuity";
    return "dependency";
}

/**
 * Returns semantic annotations only when both endpoints are Film nodes. Mixed
 * legacy edges intentionally remain free-form and untouched.
 */
export function describeFilmConnection(from?: CanvasNodeData, to?: CanvasNodeData): { edgeType: FilmEdgeType; filmPorts: FilmConnectionPorts } | undefined {
    if (!from?.filmKind || !to?.filmKind) return undefined;
    return {
        edgeType: filmEdgeType(from.filmKind, to.filmKind),
        filmPorts: {
            from: filmOutputPort(from.filmKind),
            to: filmInputPort(to.filmKind),
        },
    };
}
