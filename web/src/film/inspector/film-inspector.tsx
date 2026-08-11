import { Badge, Empty, Tag } from "antd";
import type { ReactNode } from "react";

import type { ProjectDetail } from "@/services/api/projects";
import type { CanvasNodeData } from "@/types/canvas";

type FilmInspectorProps = {
    node: CanvasNodeData | null;
    project?: ProjectDetail;
};

const kindLabels: Record<string, string> = {
    scene: "场景", shot: "镜头", character: "角色", location: "场地", prop: "道具", acting: "表演", prompt_pack: "提示词包", needs_you: "需要人工决定",
};

export function FilmInspector({ node, project }: FilmInspectorProps) {
    if (!node?.filmKind) return <aside className="w-80 shrink-0 border-l p-4" aria-label="影视生产检查器"><Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="选择影视节点查看生产信息" /></aside>;
    const ref = node.domainRef;
    const shot = ref?.shotId ? project?.shots.find((item) => item.id === ref.shotId) : undefined;
    const asset = ref?.assetId ? project?.assets.find((item) => item.id === ref.assetId) : undefined;
    const state = node.filmState;
    return (
        <aside className="w-80 shrink-0 overflow-y-auto border-l p-4" aria-label="影视生产检查器">
            <div className="mb-4 flex items-start justify-between gap-2">
                <div><div className="text-xs text-muted-foreground">{kindLabels[node.filmKind] || node.filmKind}</div><h2 className="m-0 text-base">{node.title}</h2></div>
                <Badge status={state?.attention === "error" ? "error" : state?.attention === "warning" ? "warning" : "processing"} />
            </div>
            <InspectorSection title="领域关联">
                <InspectorRow label="项目 ID" value={ref?.projectId || "未绑定"} />
                <InspectorRow label="Scene ID" value={ref?.sceneId || "未绑定"} />
                <InspectorRow label="Shot ID" value={ref?.shotId || "未绑定"} />
                <InspectorRow label="Asset" value={asset ? asset.title : ref?.assetId || "未绑定"} />
                <InspectorRow label="资产版本" value={ref?.assetVersionId || "未固定"} />
            </InspectorSection>
            {shot ? <InspectorSection title="Shot Contract"><InspectorRow label="名称" value={shot.title} /><InspectorRow label="时长" value={`${Math.round(shot.durationMs / 1000)} 秒`} /><InspectorRow label="状态" value={shot.status} /></InspectorSection> : null}
            <InspectorSection title="生产状态">
                <div className="flex flex-wrap gap-1"><Tag>{state?.lifecycle || "draft"}</Tag><Tag>{state?.production || "not_started"}</Tag><Tag>{state?.evidence || "unknown"}</Tag></div>
                <InspectorRow label="布局" value={node.layout?.mode || "manual"} />
            </InspectorSection>
        </aside>
    );
}

function InspectorSection({ title, children }: { title: string; children: ReactNode }) {
    return <section className="mb-5"><h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">{title}</h3><div className="space-y-2">{children}</div></section>;
}

function InspectorRow({ label, value }: { label: string; value: string }) {
    return <div className="grid grid-cols-[88px_minmax(0,1fr)] gap-2 text-sm"><span className="text-muted-foreground">{label}</span><span className="break-all">{value}</span></div>;
}
