import { Plus } from "lucide-react";
import { useCallback, useEffect, useRef } from "react";

import type { CanvasTheme } from "@/lib/canvas-theme";

export function CanvasConnectionSideRail({
    side,
    scale,
    visible,
    theme,
    onPointerDown,
}: {
    side: "left" | "right";
    scale: number;
    visible: boolean;
    theme: CanvasTheme;
    onPointerDown: (event: React.PointerEvent, anchorRatio: number) => void;
}) {
    const handleRef = useRef<HTMLSpanElement>(null);
    const anchorRatioRef = useRef(0.5);
    const inverseScale = 1 / Math.max(scale, 0.05);

    const resetAnchor = useCallback(() => {
        anchorRatioRef.current = 0.5;
        if (handleRef.current) handleRef.current.style.top = "50%";
    }, []);

    useEffect(() => {
        if (!visible) resetAnchor();
    }, [resetAnchor, visible]);

    const updateAnchor = (event: React.PointerEvent<HTMLButtonElement>) => {
        const railBounds = event.currentTarget.getBoundingClientRect();
        const nodeBounds = event.currentTarget.parentElement?.getBoundingClientRect() || railBounds;
        const screenPadding = Math.min(railBounds.height * 0.35, 12);
        const railRatio = Math.min(1 - screenPadding / Math.max(railBounds.height, 1), Math.max(screenPadding / Math.max(railBounds.height, 1), (event.clientY - railBounds.top) / Math.max(railBounds.height, 1)));
        const anchorY = railBounds.top + railBounds.height * railRatio;
        anchorRatioRef.current = Math.min(1, Math.max(0, (anchorY - nodeBounds.top) / Math.max(nodeBounds.height, 1)));
        if (handleRef.current) handleRef.current.style.top = `${railRatio * 100}%`;
    };

    return (
        <button
            type="button"
            className={`group absolute top-1/2 z-[var(--node-z-overlay)] touch-none -translate-y-1/2 outline-none transition-opacity duration-150 ${visible ? "pointer-events-auto opacity-100" : "pointer-events-none opacity-0"}`}
            style={{ width: 56 * inverseScale, height: `min(100%, ${72 * inverseScale}px)`, ...(side === "left" ? { right: "100%" } : { left: "100%" }) }}
            onPointerEnter={updateAnchor}
            onPointerMove={updateAnchor}
            onPointerLeave={resetAnchor}
            onPointerDown={(event) => onPointerDown(event, anchorRatioRef.current)}
            onMouseDown={(event) => {
                event.preventDefault();
                event.stopPropagation();
            }}
            aria-label={`${side === "left" ? "输入" : "输出"}连接点，单击创建节点或拖动连线`}
        >
            <span
                ref={handleRef}
                className="absolute grid -translate-y-1/2 place-items-center rounded-full border transition-[background-color,box-shadow] duration-150 group-hover:brightness-125 group-focus-visible:brightness-125"
                style={{
                    top: "50%",
                    width: 18 * inverseScale,
                    height: 18 * inverseScale,
                    ...(side === "left" ? { right: 6 * inverseScale } : { left: 6 * inverseScale }),
                    borderWidth: inverseScale,
                    background: theme.spatial.elevated,
                    borderColor: theme.node.activeStroke,
                    color: theme.node.activeStroke,
                    boxShadow: `0 4px 12px ${theme.spatial.shadow}`,
                }}
            >
                <Plus style={{ width: 10 * inverseScale, height: 10 * inverseScale }} strokeWidth={2} />
            </span>
        </button>
    );
}
