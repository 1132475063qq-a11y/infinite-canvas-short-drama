export type CanvasSpatialItem = {
    id: string;
    minX: number;
    minY: number;
    maxX: number;
    maxY: number;
};

/**
 * A deterministic uniform-grid index for layout queries. Leafer keeps its own
 * render index; this index contains only production/layout geometry.
 */
export class CanvasSpatialIndex {
    private readonly buckets = new Map<string, Set<string>>();
    private readonly items = new Map<string, CanvasSpatialItem>();

    constructor(private readonly cellSize = 512) {}

    static from(items: CanvasSpatialItem[], cellSize = 512) {
        const index = new CanvasSpatialIndex(cellSize);
        items.forEach((item) => index.insert(item));
        return index;
    }

    insert(item: CanvasSpatialItem) {
        if (this.items.has(item.id)) this.remove(item.id);
        const normalized = normalizeItem(item);
        this.items.set(normalized.id, normalized);
        this.forEachCell(normalized, (key) => {
            const bucket = this.buckets.get(key) || new Set<string>();
            bucket.add(normalized.id);
            this.buckets.set(key, bucket);
        });
    }

    remove(id: string) {
        const item = this.items.get(id);
        if (!item) return;
        this.forEachCell(item, (key) => {
            const bucket = this.buckets.get(key);
            bucket?.delete(id);
            if (!bucket?.size) this.buckets.delete(key);
        });
        this.items.delete(id);
    }

    search(bounds: Omit<CanvasSpatialItem, "id">): CanvasSpatialItem[] {
        const query = normalizeItem({ ...bounds, id: "query" });
        const ids = new Set<string>();
        this.forEachCell(query, (key) => this.buckets.get(key)?.forEach((id) => ids.add(id)));
        return Array.from(ids, (id) => this.items.get(id)!).filter((item) => spatialItemsOverlap(item, query));
    }

    private forEachCell(item: CanvasSpatialItem, visit: (key: string) => void) {
        const minColumn = Math.floor(item.minX / this.cellSize);
        const maxColumn = Math.floor(item.maxX / this.cellSize);
        const minRow = Math.floor(item.minY / this.cellSize);
        const maxRow = Math.floor(item.maxY / this.cellSize);
        for (let column = minColumn; column <= maxColumn; column += 1) {
            for (let row = minRow; row <= maxRow; row += 1) visit(`${column}:${row}`);
        }
    }
}

export function spatialItemsOverlap(first: Omit<CanvasSpatialItem, "id">, second: Omit<CanvasSpatialItem, "id">) {
    return first.minX <= second.maxX && first.maxX >= second.minX && first.minY <= second.maxY && first.maxY >= second.minY;
}

function normalizeItem<T extends CanvasSpatialItem>(item: T): T {
    return {
        ...item,
        minX: Math.min(item.minX, item.maxX),
        minY: Math.min(item.minY, item.maxY),
        maxX: Math.max(item.minX, item.maxX),
        maxY: Math.max(item.minY, item.maxY),
    };
}
