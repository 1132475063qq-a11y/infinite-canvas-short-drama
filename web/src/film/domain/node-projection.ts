import type { FilmNodeDomainRef, FilmNodeKind } from "./types";
import { CanvasNodeType, type CanvasNodeData } from "@/types/canvas";

export type FilmProductionProjection = CanvasNodeData & {
    filmKind: FilmNodeKind;
    domainRef: FilmNodeDomainRef & { projectId: string };
};

/**
 * Film meaning and canvas presentation are deliberately independent.
 *
 * A database-backed production projection always carries a project domain
 * reference. Legacy workflow nodes may gain Film semantics during migration,
 * but must keep their native editor or media renderer until they are linked to
 * a real production object.
 */
export function isFilmProductionProjection(node: CanvasNodeData | undefined): node is FilmProductionProjection {
    return Boolean(node?.filmKind && node.domainRef?.projectId);
}

// Result nodes retain Film identity for provenance and Inspector selection, but
// completed media must keep the host canvas renderer so videos remain playable.
export function isFilmGenerationResultMediaProjection(node: CanvasNodeData | undefined) {
    return Boolean(node?.filmKind === "result" && node.type === CanvasNodeType.Video && node.metadata?.content);
}
