import type { CanvasNodeData } from "@/types/canvas";

/**
 * The persisted scene title may already include its production code (older
 * projects used "SC01 未命名场景"). Keep the canvas label stable for both new
 * and existing projects.
 */
export function formatFilmSceneTitle(code: string, title: string): string {
    const normalizedCode = code.trim();
    const normalizedTitle = title.trim();
    if (!normalizedCode) return normalizedTitle || "未命名场景";

    const titleWithoutCode = normalizedTitle.startsWith(normalizedCode)
        ? normalizedTitle.slice(normalizedCode.length).replace(/^[\s·:：-]+/, "").trim()
        : normalizedTitle;

    return titleWithoutCode ? `${normalizedCode} · ${titleWithoutCode}` : normalizedCode;
}

export function normalizeFilmSceneTitle(title: string): string {
    const normalizedTitle = title.trim();
    const match = normalizedTitle.match(/^(SC\d+)\s*[·:：-]?\s*(.*)$/i);
    return match ? formatFilmSceneTitle(match[1], match[2]) : normalizedTitle;
}

export function hasFilmSceneProjection(nodes: CanvasNodeData[], sceneId: string): boolean {
    return nodes.some((node) => node.filmKind === "scene" && node.domainRef?.sceneId === sceneId);
}
