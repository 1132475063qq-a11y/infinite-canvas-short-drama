import { useCallback, useEffect, useMemo, useState } from "react";
import { App, Button, Radio, Tag } from "antd";
import { Coins, History, RefreshCw, Send } from "lucide-react";

import { resolveFilmNode } from "@/film/domain/film-node-resolver";
import { formatCredits } from "@/constant/credits";
import {
    getProjectFilmGenerationExecutions,
    getProjectFilmGenerationProviderRoutes,
    getProjectFilmGenerationTaskDraft,
    submitProjectFilmGenerationTask,
    type FilmGenerationAttemptExecution,
    type FilmGenerationProviderRoute,
    type FilmGenerationProviderRouteCatalog,
    type FilmGenerationExecutionHistory,
    type FilmGenerationTaskDraft,
    type ProjectDetail,
} from "@/services/api/projects";
import type { CanvasNodeData } from "@/types/canvas";

type FilmGenerationExecutionPanelProps = {
    node: CanvasNodeData;
    project?: ProjectDetail;
    canvasId?: string;
    onProjectChanged?: () => Promise<unknown> | void;
    onProjectionChanged?: (nodeId: string, patch: Pick<CanvasNodeData, "title"> & { domainRef?: CanvasNodeData["domainRef"] }) => void;
};

export function FilmGenerationExecutionPanel({ node, project, canvasId, onProjectChanged, onProjectionChanged }: FilmGenerationExecutionPanelProps) {
    const { message, modal } = App.useApp();
    const resolved = resolveFilmNode(node, project);
    const projectId = project?.project.id || "";
    const artifactId = node.domainRef?.artifactId || resolved.artifact?.id || "";
    const taskId = node.domainRef?.taskId || "";
    const [taskDraft, setTaskDraft] = useState<FilmGenerationTaskDraft | null>(null);
    const [providerRoutes, setProviderRoutes] = useState<FilmGenerationProviderRouteCatalog | null>(null);
    const [executionHistory, setExecutionHistory] = useState<FilmGenerationExecutionHistory | null>(null);
    const [selectedRouteKey, setSelectedRouteKey] = useState("");
    const [loading, setLoading] = useState(false);
    const [submitting, setSubmitting] = useState(false);
    const [loadError, setLoadError] = useState("");

    const refresh = useCallback(async () => {
        if (!projectId || !artifactId) {
            setTaskDraft(null);
            setProviderRoutes(null);
            setExecutionHistory(null);
            return;
        }
        setLoading(true);
        setLoadError("");
        try {
            const [draftResponse, routesResponse, historyResponse] = await Promise.all([
                getProjectFilmGenerationTaskDraft(projectId, artifactId),
                getProjectFilmGenerationProviderRoutes(projectId, artifactId),
                getProjectFilmGenerationExecutions(projectId, artifactId),
            ]);
            setTaskDraft(draftResponse.taskDraft);
            setProviderRoutes(routesResponse.providerRoutes);
            setExecutionHistory(historyResponse.executionHistory);
            setSelectedRouteKey((current) => {
                if (current && routesResponse.providerRoutes.routes.some((route) => routeKey(route) === current && route.routeReady)) return current;
                const firstReady = routesResponse.providerRoutes.routes.find((route) => route.routeReady);
                return firstReady ? routeKey(firstReady) : "";
            });
        } catch (error) {
            const text = error instanceof Error ? error.message : "读取生成执行信息失败";
            setLoadError(text);
            message.error(text);
        } finally {
            setLoading(false);
        }
    }, [artifactId, message, projectId]);

    useEffect(() => {
        void refresh();
    }, [refresh]);

    const selectedRoute = useMemo(() => providerRoutes?.routes.find((route) => routeKey(route) === selectedRouteKey), [providerRoutes?.routes, selectedRouteKey]);
    const activeTaskId = taskId || executionHistory?.attempts[0]?.taskId || "";
    const requestFingerprintMatches = Boolean(taskDraft?.requestFingerprint && providerRoutes?.requestFingerprint === taskDraft.requestFingerprint);
    const submissionReady = Boolean(
        taskDraft?.requestReady
        && selectedRoute?.routeReady
        && requestFingerprintMatches
        && projectId
        && artifactId
        && canvasId
        && node.id
        && !activeTaskId,
    );

    const submit = () => {
        if (!submissionReady || !taskDraft || !selectedRoute || !canvasId) {
            message.error("请先确认请求已就绪、已选择可用渠道，并确认画布已保存");
            return;
        }
        modal.confirm({
            title: "确认提交影视生成任务",
            okText: "确认并提交",
            cancelText: "取消",
            content: <SubmissionConfirmation route={selectedRoute} taskDraft={taskDraft} />,
            onOk: async () => {
                setSubmitting(true);
                try {
                    const response = await submitProjectFilmGenerationTask(projectId, artifactId, {
                        canvasId,
                        canvasNodeId: node.id,
                        requestFingerprint: taskDraft.requestFingerprint,
                        channelId: selectedRoute.channelId,
                        model: selectedRoute.model,
                    });
                    const submission = response.submission;
                    onProjectionChanged?.(node.id, { title: node.title, domainRef: { ...node.domainRef, projectId, taskId: submission.task.id } });
                    await onProjectChanged?.();
                    await refresh();
                    message.success(submission.idempotentReplay ? "已恢复已有影视生成任务" : "影视生成任务已提交并进入队列");
                } catch (error) {
                    message.error(error instanceof Error ? error.message : "提交影视生成任务失败");
                    throw error;
                } finally {
                    setSubmitting(false);
                }
            },
        });
    };

    if (!projectId || !artifactId) return <ExecutionHint text="请先保存 Generation Request，才能读取任务合同和执行历史。" />;

    return <div className="space-y-5">
        <ExecutionSection title="提交边界" icon={<Send className="size-3.5" />}>
            <ExecutionRow label="请求状态" value={taskDraft ? `${taskDraft.generationRequestStatus} · ${taskDraft.submissionState}` : "读取中"} />
            <ExecutionRow label="提交条件" value={taskDraft && providerRoutes ? submissionReady ? "已满足" : activeTaskId ? "已有任务" : "未满足" : "读取中"} />
            <ExecutionHint text="读取任务合同、渠道和执行历史都是只读操作，不创建 Task、不冻结积分，也不会调用 Provider。只有点击下方确认按钮才会进入可能计费的提交路径。" />
            {!taskDraft?.requestReady ? (taskDraft?.blockers || []).map((blocker) => <ExecutionHint key={blocker} text={blocker} />) : null}
            {taskDraft && providerRoutes && !requestFingerprintMatches ? <ExecutionHint text="任务合同与渠道目录的请求版本不一致，请刷新后再提交。" /> : null}
        </ExecutionSection>

        <ExecutionSection title="Provider 渠道" icon={<Coins className="size-3.5" />}>
            {providerRoutes?.routes.length ? <Radio.Group value={selectedRouteKey} onChange={(event) => setSelectedRouteKey(event.target.value)} className="w-full">
                <div className="space-y-2">
                    {providerRoutes.routes.map((route) => <RouteOption key={routeKey(route)} route={route} taskDraft={taskDraft} disabled={Boolean(activeTaskId)} />)}
                </div>
            </Radio.Group> : <ExecutionHint text={providerRoutes ? (providerRoutes.blockers || []).join("；") || "当前没有可用的后端渠道。" : "读取渠道目录中"} />}
            {selectedRoute ? <div className="mt-3 rounded-md border border-amber-500/25 bg-amber-500/[.06] p-2.5 text-[var(--fs-caption)] text-foreground/65"><div className="flex items-center gap-1.5 font-medium text-amber-500"><Coins className="size-3.5" />提交前费用确认</div><p className="mt-1 leading-5">目录价格和计费数量会在服务端事务中重新复核；确认后可能立即冻结积分并进入队列。</p></div> : null}
            <Button block type="primary" icon={<Send className="size-3.5" />} disabled={!submissionReady || loading || submitting} loading={submitting} onClick={submit}>{activeTaskId ? "任务已提交" : "确认费用并提交"}</Button>
            {(providerRoutes?.blockers || []).map((blocker) => <ExecutionHint key={blocker} text={blocker} />)}
        </ExecutionSection>

        <ExecutionSection title="执行历史" icon={<History className="size-3.5" />}>
            <div className="flex items-center justify-between gap-2"><ExecutionRow label="请求版本" value={executionHistory ? `v${executionHistory.generationRequestArtifactVersion}` : "读取中"} /><Button type="text" size="small" icon={<RefreshCw className={`size-3.5 ${loading ? "animate-spin" : ""}`} />} onClick={() => void refresh()} disabled={loading}>刷新</Button></div>
            {loadError ? <ExecutionHint text={loadError} /> : null}
            {executionHistory?.attempts.length ? <div className="space-y-2">{executionHistory.attempts.map((attempt) => <AttemptCard key={attempt.id} attempt={attempt} />)}</div> : <ExecutionHint text="尚无执行记录。提交后，Task、Attempt、ProviderJob 和 Result 会按事实链逐步出现在这里。" />}
        </ExecutionSection>
    </div>;
}

function SubmissionConfirmation({ route, taskDraft }: { route: FilmGenerationProviderRoute; taskDraft: FilmGenerationTaskDraft }) {
    const quantity = route.billingMode === "per_second" ? Math.max(1, Math.round((taskDraft.gatewayInput.durationMs || 0) / 1000)) : 1;
    const unitLabel = route.billingMode === "per_second" ? "秒" : "次";
    const estimatedBase = (route.unitPriceMicrocredits || 0) * quantity;
    return <div className="space-y-2 text-sm"><p>即将使用后端渠道 <strong>{route.channelName || "系统渠道"}</strong> 的 <strong>{route.modelDisplayName || route.model}</strong>。</p><ExecutionRow label="计费方式" value={`${formatCredits(route.unitPriceMicrocredits || 0)} 积分 / ${unitLabel}`} /><ExecutionRow label="请求数量" value={`${quantity} ${unitLabel}`} /><ExecutionRow label="目录估算" value={`${formatCredits(estimatedBase)} 积分`} /><p className="text-xs leading-5 text-foreground/55">服务端会在同一事务中再次核对路由、价格、配额和余额；提交成功后 Worker 可能立即开始调用 Provider。取消或上游状态不明时，费用状态以后端账务事实为准。</p></div>;
}

function RouteOption({ route, taskDraft, disabled }: { route: FilmGenerationProviderRoute; taskDraft: FilmGenerationTaskDraft | null; disabled: boolean }) {
    const quantity = route.billingMode === "per_second" ? Math.max(1, Math.round((taskDraft?.gatewayInput.durationMs || 0) / 1000)) : 1;
    const price = route.unitPriceMicrocredits || 0;
    return <label className={`flex items-start gap-2 rounded-md border p-2.5 ${route.routeReady ? "border-border/80 hover:border-[var(--workspace-accent)]" : "border-border/50 opacity-55"}`}><Radio value={routeKey(route)} disabled={!route.routeReady || disabled} /><span className="min-w-0 flex-1"><span className="flex items-center justify-between gap-2"><span className="truncate text-xs font-medium text-foreground/75">{route.channelName || "系统渠道"} · {route.modelDisplayName || route.model}</span><Tag color={route.routeReady ? "green" : "default"}>{route.routeReady ? "可提交" : "待配置"}</Tag></span><span className="mt-1 block text-[var(--fs-micro)] text-foreground/42">{route.billingMode === "per_second" ? `${formatCredits(price)} 积分/秒 · 本请求 ${quantity} 秒` : `${formatCredits(price)} 积分/次`} · {route.protocol}</span>{(route.blockers || []).map((blocker) => <span key={blocker} className="mt-1 block text-[var(--fs-micro)] text-amber-500/80">{blocker}</span>)}</span></label>;
}

function AttemptCard({ attempt }: { attempt: FilmGenerationAttemptExecution }) {
    return <div className="rounded-md border border-border/70 bg-foreground/[.025] p-2.5"><div className="flex items-center justify-between gap-2"><span className="text-xs font-medium text-foreground/75">Attempt #{attempt.attemptNumber}</span><Tag color={attemptStatusColor(attempt.status)}>{attemptStatusLabel(attempt.status)}</Tag></div><div className="mt-2 grid grid-cols-2 gap-x-2 gap-y-1"><ExecutionRow label="Task" value={attempt.taskId} /><ExecutionRow label="模型" value={attempt.model || "未记录"} /><ExecutionRow label="开始" value={formatExecutionTime(attempt.startedAt || attempt.createdAt)} /><ExecutionRow label="更新" value={formatExecutionTime(attempt.updatedAt)} /></div>{attempt.error ? <ExecutionHint text={attempt.error} /> : null}{attempt.providerJobs.map((job) => <div key={job.id} className="mt-2 border-t border-border/60 pt-2"><ExecutionRow label="ProviderJob" value={`${job.providerRequestId} · ${attemptStatusLabel(job.status)}`} />{job.lastError ? <ExecutionHint text={job.lastError} /> : null}</div>)}{attempt.results.map((result) => <div key={result.id} className="mt-2 border-t border-border/60 pt-2"><ExecutionRow label="Result" value={result.url ? `${result.id} · 已有媒体地址` : result.id} /></div>)}</div>;
}

function routeKey(route: FilmGenerationProviderRoute) { return `${route.channelId}:${route.model}`; }
function formatExecutionTime(value?: string) { return value ? new Date(value).toLocaleString("zh-CN", { month: "numeric", day: "numeric", hour: "2-digit", minute: "2-digit" }) : "未开始"; }
function attemptStatusLabel(status: string) { return ({ queued: "排队中", running: "运行中", succeeded: "已成功", failed: "明确失败", cancelled: "已取消", uncertain: "状态不明" } as Record<string, string>)[status] || status; }
function attemptStatusColor(status: string) { return ({ queued: "default", running: "processing", succeeded: "success", failed: "error", cancelled: "default", uncertain: "warning" } as Record<string, string>)[status] || "default"; }
function ExecutionSection({ title, icon, children }: { title: string; icon: React.ReactNode; children: React.ReactNode }) { return <section className="mb-5"><h3 className="mb-2 flex items-center gap-1.5 text-[var(--fs-tiny)] font-semibold uppercase tracking-widest text-foreground/38">{icon}{title}</h3><div className="space-y-2">{children}</div></section>; }
function ExecutionRow({ label, value }: { label: string; value: string }) { return <div className="grid grid-cols-2 gap-2 text-[var(--fs-caption)]"><span className="text-foreground/38">{label}</span><span className="break-all text-foreground/68">{value}</span></div>; }
function ExecutionHint({ text }: { text: string }) { return <div className="rounded-md border border-dashed border-border/80 p-2.5 text-[var(--fs-caption)] leading-5 text-foreground/45">{text}</div>; }
