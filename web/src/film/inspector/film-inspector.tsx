import { useEffect, useMemo, useState, type ReactNode } from "react";
import { AlertTriangle, Box, CheckCircle2, CircleDot, FileQuestion, ShieldCheck } from "lucide-react";

import type { FilmNodeKind, FilmNodeState } from "@/film/domain/types";
import type { ProductionNavigationKey } from "@/film/panels/production-shell-model";
import type { ProjectDetail } from "@/services/api/projects";
import type { CanvasNodeData } from "@/types/canvas";

type FilmInspectorProps = {
    node: CanvasNodeData | null;
    project?: ProjectDetail;
    activeSection?: ProductionNavigationKey;
};

const KIND_LABELS: Record<FilmNodeKind, string> = {
    project: "项目",
    story: "故事",
    script: "剧本",
    scene: "场景",
    character: "角色",
    character_state: "角色状态",
    location: "场地",
    location_view: "场地视图",
    prop: "道具",
    voice_profile: "声音档案",
    acting: "表演",
    acting_profile: "表演档案",
    scene_acting: "场景表演",
    storyboard: "分镜",
    shot: "镜头",
    shot_reference_pack: "镜头参考包",
    prompt_pack: "提示词包",
    generation: "生成任务",
    generation_attempt: "生成尝试",
    result: "生成结果",
    qc: "质量检查",
    continuity: "连续性",
    retry: "重试",
    needs_you: "需要人工决定",
    agent_task: "Agent 任务",
    delivery: "交付",
};

const SHOT_TABS = ["Overview", "Script", "Blocking", "Acting", "References", "Camera", "Prompt", "Generation", "QC", "Versions"] as const;
const CHARACTER_TABS = ["Overview", "References", "Acting", "Voice", "Usage", "Versions"] as const;
const DEFAULT_TABS = ["Overview", "References", "Production", "Versions"] as const;

export function FilmInspector({ node, project, activeSection = "overview" }: FilmInspectorProps) {
    const tabs = useMemo(() => (node?.filmKind === "shot" ? SHOT_TABS : node?.filmKind === "character" ? CHARACTER_TABS : DEFAULT_TABS), [node?.filmKind]);
    const [activeTab, setActiveTab] = useState<string>(tabs[0]);

    useEffect(() => setActiveTab(tabs[0]), [node?.id, tabs]);

    return (
        <aside className="relative z-[var(--z-panel)] hidden w-72 shrink-0 flex-col border-l border-border bg-background/94 backdrop-blur-xl lg:flex 2xl:w-80" aria-label="影视生产检查器">
            {node?.filmKind ? <NodeInspector node={node} project={project} tabs={tabs} activeTab={activeTab} onTabChange={setActiveTab} /> : <ProjectInspector project={project} activeSection={activeSection} />}
        </aside>
    );
}

function ProjectInspector({ project, activeSection }: { project?: ProjectDetail; activeSection: ProductionNavigationKey }) {
    const sceneCount = project?.scenes.length || 0;
    const shotCount = project?.shots.length || 0;
    const assetCount = project?.assets.length || 0;
    return (
        <>
            <InspectorHeader eyebrow="PROJECT INSPECTOR" title={project?.project.name || "项目检查器"} subtitle={`当前区域 · ${navigationLabel(activeSection)}`} />
            <div className="thin-scrollbar min-h-0 flex-1 overflow-y-auto p-4">
                <InspectorSection title="项目事实源">
                    <InspectorRow label="Project ID" value={project?.project.id || "未关联"} />
                    <InspectorRow label="状态" value={project?.project.status || "unknown"} />
                    <InspectorRow label="版本" value={project?.project.revision ? `r${project.project.revision}` : "未记录"} />
                </InspectorSection>
                <div className="grid grid-cols-3 gap-2">
                    <ProjectMetric label="场景" value={sceneCount} />
                    <ProjectMetric label="镜头" value={shotCount} />
                    <ProjectMetric label="资产" value={assetCount} />
                </div>
                <div className="mt-5 rounded-lg border border-dashed border-border/80 p-4 text-center">
                    <Box className="mx-auto size-7 text-foreground/22" />
                    <p className="mt-2 text-xs leading-5 text-foreground/45">选择画布中的影视节点，查看 DomainRef、生产状态和版本信息。</p>
                </div>
            </div>
        </>
    );
}

function NodeInspector({ node, project, tabs, activeTab, onTabChange }: { node: CanvasNodeData; project?: ProjectDetail; tabs: readonly string[]; activeTab: string; onTabChange: (tab: string) => void }) {
    const ref = node.domainRef;
    const shot = ref?.shotId ? project?.shots.find((item) => item.id === ref.shotId) : undefined;
    const asset = ref?.assetId ? project?.assets.find((item) => item.id === ref.assetId) : undefined;
    const state = node.filmState;
    const objectId = ref?.shotId || ref?.assetId || ref?.artifactId || ref?.sceneId || node.id;
    const version = ref?.artifactVersion || ref?.assetVersionId || "未固定";
    const responsible = ref?.agentId || "未分配";

    return (
        <>
            <InspectorHeader eyebrow={KIND_LABELS[node.filmKind!] || node.filmKind!} title={node.title} subtitle={objectId} attention={state?.attention} />
            <div className="border-b border-border/70 px-3 py-2">
                <div className="thin-scrollbar flex gap-1 overflow-x-auto" role="tablist" aria-label="检查器标签">
                    {tabs.map((tab) => (
                        <button
                            key={tab}
                            type="button"
                            role="tab"
                            aria-selected={activeTab === tab}
                            onClick={() => onTabChange(tab)}
                            className={`h-7 shrink-0 rounded px-2 text-[var(--fs-micro)] font-medium ${activeTab === tab ? "bg-[var(--workspace-accent-soft)] text-[var(--workspace-accent)]" : "text-foreground/42 hover:bg-foreground/[.05] hover:text-foreground"}`}
                        >
                            {tab}
                        </button>
                    ))}
                </div>
            </div>
            <div className="thin-scrollbar min-h-0 flex-1 overflow-y-auto p-4">
                {activeTab === "Overview" ? (
                    <>
                        <InspectorSection title="统一对象头">
                            <InspectorRow label="Object ID" value={objectId} />
                            <InspectorRow label="Version" value={version} />
                            <InspectorRow label="Responsible" value={responsible} />
                            <InspectorRow label="Status" value={`${state?.lifecycle || "draft"} / ${state?.production || "not_started"}`} />
                            <InspectorRow label="Evidence" value={state?.evidence || "unknown"} />
                        </InspectorSection>
                        <InspectorSection title="领域关联">
                            <InspectorRow label="Project ID" value={ref?.projectId || "未绑定"} />
                            <InspectorRow label="Scene ID" value={ref?.sceneId || "未绑定"} />
                            <InspectorRow label="Shot ID" value={ref?.shotId || "未绑定"} />
                            <InspectorRow label="Asset" value={asset?.title || ref?.assetId || "未绑定"} />
                        </InspectorSection>
                        {shot ? (
                            <InspectorSection title="Shot Contract">
                                <InspectorRow label="名称" value={shot.title} />
                                <InspectorRow label="时长" value={`${Math.round(shot.durationMs / 100) / 10} 秒`} />
                                <InspectorRow label="状态" value={shot.status} />
                            </InspectorSection>
                        ) : null}
                        <InspectorSection title="画布投影">
                            <InspectorRow label="节点 ID" value={node.id} />
                            <InspectorRow label="布局" value={node.layout?.mode || "manual"} />
                        </InspectorSection>
                    </>
                ) : (
                    <InspectorTabPlaceholder tab={activeTab} />
                )}
            </div>
        </>
    );
}

function InspectorHeader({ eyebrow, title, subtitle, attention }: { eyebrow: string; title: string; subtitle: string; attention?: FilmNodeState["attention"] }) {
    const AttentionIcon = attention === "error" ? AlertTriangle : attention === "warning" || attention === "human_required" ? CircleDot : CheckCircle2;
    const tone = attention === "error" ? "text-red-400" : attention === "warning" || attention === "human_required" ? "text-amber-400" : "text-emerald-400";
    return (
        <header className="shrink-0 border-b border-border/70 px-4 py-3">
            <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                    <div className="text-[var(--fs-tiny)] font-semibold uppercase tracking-widest text-foreground/35">{eyebrow}</div>
                    <h2 className="mt-1 truncate text-sm font-semibold text-foreground/82" title={title}>
                        {title}
                    </h2>
                    <div className="mt-1 truncate font-mono text-[var(--fs-micro)] text-foreground/32" title={subtitle}>
                        {subtitle}
                    </div>
                </div>
                {attention ? <AttentionIcon className={`mt-1 size-4 shrink-0 ${tone}`} aria-label={`注意状态 ${attention}`} /> : null}
            </div>
        </header>
    );
}

function InspectorTabPlaceholder({ tab }: { tab: string }) {
    return (
        <div className="rounded-lg border border-dashed border-border/80 p-4 text-center">
            <FileQuestion className="mx-auto size-7 text-foreground/22" />
            <h3 className="mt-2 text-xs font-semibold text-foreground/62">{tab}</h3>
            <p className="mt-1 text-[var(--fs-caption)] leading-5 text-foreground/40">标签结构已建立；具体编辑能力将在对应影视节点阶段接入事实源。</p>
        </div>
    );
}

function InspectorSection({ title, children }: { title: string; children: ReactNode }) {
    return (
        <section className="mb-5">
            <h3 className="mb-2 flex items-center gap-1.5 text-[var(--fs-tiny)] font-semibold uppercase tracking-widest text-foreground/38">
                <ShieldCheck className="size-3.5" />
                {title}
            </h3>
            <div className="space-y-2">{children}</div>
        </section>
    );
}

function InspectorRow({ label, value }: { label: string; value: string }) {
    return (
        <div className="grid grid-cols-[5.5rem_minmax(0,1fr)] gap-2 text-[var(--fs-caption)]">
            <span className="text-foreground/38">{label}</span>
            <span className="break-all text-foreground/68">{value}</span>
        </div>
    );
}

function ProjectMetric({ label, value }: { label: string; value: number }) {
    return (
        <div className="rounded-lg border border-border/70 bg-foreground/[.025] p-2 text-center">
            <strong className="block text-sm tabular-nums text-foreground/75">{value}</strong>
            <span className="mt-0.5 block text-[var(--fs-micro)] text-foreground/36">{label}</span>
        </div>
    );
}

function navigationLabel(section: ProductionNavigationKey) {
    return (
        {
            overview: "总览",
            story: "故事",
            script: "剧本",
            scenes: "场景",
            characters: "角色",
            locations: "场地",
            props: "道具",
            storyboard: "分镜",
            shots: "镜头",
            assets: "资产",
            audio: "声音",
            qc: "QC",
            agents: "Agent",
            needs_you: "待处理",
            delivery: "交付",
        } satisfies Record<ProductionNavigationKey, string>
    )[section];
}
