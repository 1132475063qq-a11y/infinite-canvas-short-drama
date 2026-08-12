import type { FilmNodeDomainRef, FilmNodeKind } from "./types";
import type { CanvasNodeData } from "@/types/canvas";

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
