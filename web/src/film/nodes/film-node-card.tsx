import { Badge, Tag } from "antd";

import type { CanvasNodeData } from "@/types/canvas";

const labels: Record<string, string> = { scene: "SCENE", shot: "SHOT", character: "CHARACTER", location: "LOCATION", prop: "PROP", acting: "ACTING", prompt_pack: "PROMPT PACK", needs_you: "NEEDS YOU" };

export function FilmNodeCard({ node }: { node: CanvasNodeData }) {
    const state = node.filmState;
    return (
        <div className="flex h-full w-full flex-col justify-between p-4">
            <div>
                <div className="mb-2 flex items-center justify-between gap-2"><span className="text-[var(--fs-tiny)] font-semibold tracking-[0.12em] opacity-60">{labels[node.filmKind || ""] || node.filmKind}</span><Badge status={state?.attention === "error" ? "error" : state?.attention === "warning" ? "warning" : "default"} /></div>
                <div className="text-base font-semibold">{node.title}</div>
                <div className="mt-2 text-[var(--fs-label)] opacity-65">{node.domainRef?.shotId || node.domainRef?.assetId || node.domainRef?.sceneId || "尚未绑定生产对象"}</div>
            </div>
            <div className="flex flex-wrap gap-1"><Tag>{state?.lifecycle || "draft"}</Tag><Tag>{state?.production || "not_started"}</Tag></div>
        </div>
    );
}
