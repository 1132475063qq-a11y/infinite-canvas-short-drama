import type { CanvasNodeData } from "@/types/canvas";

/**
 * Film meaning and canvas presentation are deliberately independent.
 *
 * A database-backed production projection always carries a project domain
 * reference. Legacy workflow nodes may gain Film semantics during migration,
 * but must keep their native editor or media renderer until they are linked to
 * a real production object.
 */
export function isFilmProductionProjection(node: Pick<CanvasNodeData, "filmKind" | "domainRef"> | undefined): boolean {
    return Boolean(node?.filmKind && node.domainRef?.projectId);
}
