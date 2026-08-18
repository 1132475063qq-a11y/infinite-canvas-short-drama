import { AlertTriangle, CheckCircle2, CircleDot, Clapperboard, UserRoundCheck } from "lucide-react";
import type { ReactNode } from "react";

import type { ProductionShellModel } from "./production-shell-model";

export function ProjectProductionStatus({ model }: { model: ProductionShellModel }) {
    const sceneLabel = model.currentScene ? `${model.currentScene.code || "SC"} · ${model.currentScene.title}` : "全部场景";
    return (
        <div
            className="pointer-events-auto absolute left-1/2 top-2 flex h-9 max-w-xl -translate-x-1/2 items-center gap-1 rounded-full border border-border/70 bg-background/88 px-1.5 shadow-[var(--workspace-overlay-shadow)] backdrop-blur-2xl"
            aria-label="项目生产状态"
        >
            <span className="hidden min-w-0 items-center gap-1.5 px-2 text-[var(--fs-caption)] font-medium text-foreground/72 2xl:flex" title={sceneLabel}>
                <Clapperboard className="size-3.5 shrink-0 text-[var(--workspace-accent)]" />
                <span className="max-w-32 truncate">{sceneLabel}</span>
            </span>
            <StatusMetric icon={<CheckCircle2 className="size-3.5" />} label="镜头" value={`${model.approvedShots}/${model.totalShots}`} tone="normal" />
            <StatusMetric icon={<CircleDot className="size-3.5" />} label="运行" value={model.runningTasks} tone={model.runningTasks ? "active" : "normal"} />
            <StatusMetric icon={<AlertTriangle className="size-3.5" />} label="QC" value={model.qcFailed} tone={model.qcFailed ? "danger" : "normal"} />
            <StatusMetric icon={<UserRoundCheck className="size-3.5" />} label="待定" value={model.needsYou} tone={model.needsYou ? "warning" : "normal"} />
        </div>
    );
}

function StatusMetric({ icon, label, value, tone }: { icon: ReactNode; label: string; value: string | number; tone: "normal" | "active" | "warning" | "danger" }) {
    const toneClass = tone === "danger" ? "text-red-400" : tone === "warning" ? "text-amber-400" : tone === "active" ? "text-sky-400" : "text-foreground/55";
    return (
        <span className={`flex h-7 items-center gap-1 rounded-full px-2 text-[var(--fs-micro)] tabular-nums ${toneClass}`} title={`${label}: ${value}`}>
            {icon}
            <span className="hidden 2xl:inline">{label}</span>
            <strong className="font-semibold text-foreground/82">{value}</strong>
        </span>
    );
}
