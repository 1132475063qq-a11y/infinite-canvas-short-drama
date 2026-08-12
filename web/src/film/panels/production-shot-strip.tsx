import { CheckCircle2, CircleDashed, CircleDot, Play, TriangleAlert } from "lucide-react";

import type { ProjectShot } from "@/services/api/projects";
import type { ProductionShellModel } from "./production-shell-model";
import { formatShotCode } from "./production-shell-model";

type ProductionShotStripProps = {
    model: ProductionShellModel;
    selectedShotId?: string;
    onSelectShot: (shot: ProjectShot) => void;
};

export function ProductionShotStrip({ model, selectedShotId, onSelectShot }: ProductionShotStripProps) {
    const sceneLabel = model.currentScene ? `${model.currentScene.code || "SC"} ${model.currentScene.title}` : "全部镜头";
    return (
        <section className="relative z-[var(--z-panel)] flex h-24 shrink-0 border-t border-border/80 bg-background/94 backdrop-blur-xl" aria-label="当前场景镜头条">
            <div className="flex w-44 shrink-0 flex-col justify-center border-r border-border/70 px-4">
                <span className="text-[var(--fs-tiny)] font-medium uppercase tracking-widest text-foreground/35">Shot Strip</span>
                <strong className="mt-1 truncate text-xs font-semibold text-foreground/78" title={sceneLabel}>
                    {sceneLabel}
                </strong>
                <span className="mt-1 text-[var(--fs-micro)] tabular-nums text-foreground/38">
                    {model.shots.length} 镜头 · {formatDuration(model.shots)}
                </span>
            </div>
            <div className="thin-scrollbar flex min-w-0 flex-1 items-center gap-2 overflow-x-auto px-3 py-2">
                {model.shots.length ? (
                    model.shots.map((shot, index) => (
                        <button
                            key={shot.id}
                            type="button"
                            className={`group flex h-16 w-36 shrink-0 flex-col justify-between rounded-lg border px-2.5 py-2 text-left transition ${selectedShotId === shot.id ? "border-[var(--workspace-accent)] bg-[var(--workspace-accent-soft)]" : "border-border/70 bg-foreground/[.025] hover:border-foreground/25 hover:bg-foreground/[.05]"}`}
                            onClick={() => onSelectShot(shot)}
                            aria-pressed={selectedShotId === shot.id}
                        >
                            <span className="flex items-center justify-between gap-2">
                                <strong className="text-[var(--fs-caption)] font-semibold text-foreground/78">{formatShotCode(shot, index)}</strong>
                                <ShotStatusIcon status={shot.status} />
                            </span>
                            <span className="truncate text-[var(--fs-micro)] text-foreground/42" title={shot.title}>
                                {shot.title || "未命名镜头"}
                            </span>
                            <span className="flex items-center gap-1 text-[var(--fs-micro)] tabular-nums text-foreground/35">
                                <Play className="size-2.5" />
                                {Math.max(0, shot.durationMs / 1000).toFixed(1)}s
                            </span>
                        </button>
                    ))
                ) : (
                    <div className="flex h-full items-center px-3 text-xs text-foreground/38">当前场景还没有镜头，创建 Shot 后会显示在这里。</div>
                )}
            </div>
        </section>
    );
}

function ShotStatusIcon({ status }: { status: string }) {
    const normalized = status.toLocaleLowerCase();
    if (["approved", "completed", "succeeded"].includes(normalized)) return <CheckCircle2 className="size-3.5 text-emerald-400" aria-label="已通过" />;
    if (["failed", "qc_failed"].includes(normalized)) return <TriangleAlert className="size-3.5 text-red-400" aria-label="质检失败" />;
    if (["running", "generating"].includes(normalized)) return <CircleDot className="size-3.5 text-sky-400" aria-label="生产中" />;
    return <CircleDashed className="size-3.5 text-foreground/30" aria-label="未开始" />;
}

function formatDuration(shots: ProjectShot[]) {
    const seconds = shots.reduce((total, shot) => total + Math.max(0, shot.durationMs), 0) / 1000;
    return `${seconds.toFixed(seconds >= 10 ? 0 : 1)} 秒`;
}
