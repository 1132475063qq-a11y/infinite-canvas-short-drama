import { layoutFilmSceneLanes } from "@/lib/canvas/layout/film-layout-presets";
import { snapCanvasPosition } from "@/lib/canvas/layout/snap-engine";
import type { CanvasNodeData } from "@/types/canvas";

export function applyFilmAutoLayout(nodes: CanvasNodeData[]): CanvasNodeData[] {
    const positions = layoutFilmSceneLanes(nodes);
    if (!positions.size) return nodes;
    return nodes.map((node) => {
        const position = positions.get(node.id);
        return position ? { ...node, position: snapCanvasPosition(position) } : node;
    });
}
