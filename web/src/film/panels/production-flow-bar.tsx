import { Check, CircleDot, Clapperboard, PackageCheck, ScrollText, Sparkles } from "lucide-react";
import type { LucideIcon } from "lucide-react";

import type { ProductionNavigationKey, ProductionShellModel } from "./production-shell-model";

type FlowStageKey = Extract<ProductionNavigationKey, "story" | "scenes" | "storyboard" | "shots" | "delivery">;
type FlowStageState = "complete" | "current" | "pending";

type ProductionFlowBarProps = {
    model: ProductionShellModel;
    activeSection: ProductionNavigationKey;
    onNavigate: (key: ProductionNavigationKey) => void;
};

type FlowStage = {
    key: FlowStageKey;
    label: string;
    detail: string;
    icon: LucideIcon;
    complete: boolean;
};

export function ProductionFlowBar({ model, activeSection, onNavigate }: ProductionFlowBarProps) {
    const stages = buildFlowStages(model);
    const currentIndex = stages.findIndex((stage) => !stage.complete);
    const activeIndex = currentIndex >= 0 ? currentIndex : stages.length - 1;
    const nextAction = describeNextAction(model, stages);

    return (
        <section data-canvas-no-zoom className="pointer-events-none absolute left-1/2 top-[calc(var(--canvas-topbar-offset)+var(--space-2))] z-[var(--z-toolbar)] w-[min(820px,calc(100%_-_24px))] -translate-x-1/2" aria-label="影视制作流程">
            <div className="pointer-events-auto rounded-lg border border-border/75 bg-background/88 px-2 py-1.5 shadow-[var(--workspace-overlay-shadow)] backdrop-blur-2xl">
                <div className="flex min-w-0 items-center justify-between gap-3 px-1.5 pb-1.5">
                    <span className="inline-flex shrink-0 items-center gap-1.5 text-[var(--fs-tiny)] font-semibold text-foreground/62">
                        <Clapperboard className="size-3.5 text-[var(--workspace-accent)]" />
                        制作流程
                    </span>
                    <span className="min-w-0 truncate text-right text-[var(--fs-micro)] text-foreground/45" title={nextAction.detail}>
                        <strong className="font-medium text-foreground/68">{nextAction.label}</strong>
                        <span className="hidden sm:inline"> · {nextAction.detail}</span>
                    </span>
                </div>
                <div className="hide-scrollbar flex min-w-0 items-center overflow-x-auto" role="list">
                    {stages.map((stage, index) => {
                        const state: FlowStageState = stage.complete ? "complete" : index === activeIndex ? "current" : "pending";
                        const Icon = stage.icon;
                        const active = activeSection === stage.key;
                        return (
                            <div key={stage.key} className="flex min-w-0 shrink-0 items-center" role="listitem">
                                {index ? <span className={`mx-1 h-px w-4 shrink-0 ${stages[index - 1].complete ? "bg-[var(--workspace-accent)]/70" : "bg-border/80"}`} aria-hidden /> : null}
                                <button
                                    type="button"
                                    className={`group flex h-8 min-w-[112px] items-center gap-2 rounded-md px-2 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--workspace-accent)] ${active ? "bg-[var(--workspace-accent-soft)]" : "hover:bg-foreground/[.05]"}`}
                                    onClick={() => onNavigate(stage.key)}
                                    aria-current={active ? "page" : undefined}
                                    title={`${stage.label}：${stage.detail}`}
                                >
                                    <span
                                        className={`grid size-5 shrink-0 place-items-center rounded-full border ${state === "complete" ? "border-[var(--workspace-accent)] bg-[var(--workspace-accent)] text-white" : state === "current" ? "border-[var(--workspace-accent)] text-[var(--workspace-accent)]" : "border-border text-foreground/32"}`}
                                    >
                                        {state === "complete" ? <Check className="size-3" /> : <Icon className="size-3" />}
                                    </span>
                                    <span className="min-w-0">
                                        <span className={`block truncate text-[var(--fs-caption)] font-semibold ${state === "current" || active ? "text-[var(--workspace-accent)]" : "text-foreground/68"}`}>{stage.label}</span>
                                        <span className="block truncate text-[var(--fs-micro)] text-foreground/38">{stage.detail}</span>
                                    </span>
                                </button>
                            </div>
                        );
                    })}
                </div>
            </div>
        </section>
    );
}

function buildFlowStages(model: ProductionShellModel): FlowStage[] {
    const counts = model.navigationCounts;
    const storyComplete = counts.story > 0 || counts.script > 0;
    const scenesComplete = counts.scenes > 0;
    const storyboardComplete = counts.storyboard > 0 || model.totalShots > 0;
    const shotsComplete = model.totalShots > 0 && model.approvedShots >= model.totalShots && model.qcFailed === 0;
    const deliveryComplete = counts.delivery > 0;

    return [
        { key: "story", label: "故事", detail: storyComplete ? `${Math.max(counts.story, counts.script)} 项` : "待建立", icon: Sparkles, complete: storyComplete },
        { key: "scenes", label: "场景", detail: scenesComplete ? `${counts.scenes} 个` : "待创建", icon: Clapperboard, complete: scenesComplete },
        { key: "storyboard", label: "分镜", detail: storyboardComplete ? `${counts.storyboard || model.totalShots} 个` : "待拆分", icon: ScrollText, complete: storyboardComplete },
        { key: "shots", label: "镜头", detail: model.totalShots ? `${model.approvedShots}/${model.totalShots} 已通过` : "待建立", icon: CircleDot, complete: shotsComplete },
        { key: "delivery", label: "交付", detail: deliveryComplete ? `${counts.delivery} 项` : "待准备", icon: PackageCheck, complete: deliveryComplete },
    ];
}

function describeNextAction(model: ProductionShellModel, stages: FlowStage[]) {
    if (model.runningTasks > 0) return { label: "正在生成", detail: `${model.runningTasks} 个任务进行中` };
    if (model.qcFailed > 0) return { label: "下一步：处理 QC", detail: `${model.qcFailed} 个镜头需要修复` };
    if (model.needsYou > 0) return { label: "下一步：处理待确认", detail: `${model.needsYou} 项需要你的决定` };
    const next = stages.find((stage) => !stage.complete);
    if (!next) return { label: "制作流程已完成", detail: "可以检查交付内容" };
    return { label: `下一步：${next.label}`, detail: next.detail === "待建立" ? `先${next.label}` : `完善${next.label}` };
}
