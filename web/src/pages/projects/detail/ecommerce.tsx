import { useMemo, useState } from "react";
import { App, Button, Input, Select, Tag } from "antd";
import { CheckCircle2, CircleAlert, Database, FlaskConical, LockKeyhole, RefreshCw, Sparkles } from "lucide-react";

import { runProviderFreePrototypeA, type PrototypeAChain } from "@/ecommerce/prototype/prototype-a";
import { hydratePrototypeAFromArtifacts, prototypeChainToArtifactInputs, stableArtifactKeyPart } from "@/ecommerce/prototype/persistence";
import type { BasicQAReport } from "@/ecommerce/prototype/basic-qa";
import type { EcommerceResultGrid } from "@/ecommerce/prototype/result-grid";
import { saveProjectEcommerceArtifact, type EcommerceArtifact } from "@/services/api/projects";

import { formatTime, type ProjectDetailViewProps } from "./shared";

const artifactLabels: Record<string, string> = {
    product_dna: "ProductDNA",
    creative_direction: "CreativeDirection",
    scene_plan: "ScenePlan",
    creative_shot_plan: "CreativeShotPlan",
    qa_report: "Basic QA",
};

export default function EcommercePrototypeView({ detail, refreshProject }: ProjectDetailViewProps) {
    const { message } = App.useApp();
    const defaultAssetId = detail.assets[0]?.id || "";
    const [productAssetId, setProductAssetId] = useState(defaultAssetId);
    const [userGoal, setUserGoal] = useState("生成一组可用于商品详情页的生活方式买点图");
    const [chain, setChain] = useState<PrototypeAChain | null>(null);
    const [running, setRunning] = useState(false);
    const [runError, setRunError] = useState("");
    const persisted = useMemo(() => hydratePrototypeAFromArtifacts(detail.ecommerceArtifacts || [], productAssetId), [detail.ecommerceArtifacts, productAssetId]);
    const resultGrid = chain?.resultGrid || persisted.resultGrid;
    const basicQA = chain?.basicQA || persisted.basicQA;
    const latestArtifacts = useMemo(() => latestArtifactsByType(detail.ecommerceArtifacts || [], productAssetId), [detail.ecommerceArtifacts, productAssetId]);
    const assetOptions = detail.assets.map((asset) => ({ label: `${asset.title || "未命名商品"} · ${asset.id}`, value: asset.id }));

    async function runPrototype() {
        const assetId = productAssetId.trim();
        if (!assetId) {
            message.warning("先选择或输入一个商品资产 ID");
            return;
        }
        setRunning(true);
        setRunError("");
        try {
            const nextChain = runProviderFreePrototypeA({
                projectId: detail.project.id,
                productAssetId: assetId,
                productDna: { confidence: 0 },
                userGoal: userGoal.trim() || undefined,
                now: new Date().toISOString(),
            });
            const inputs = prototypeChainToArtifactInputs(nextChain);
            for (const input of inputs) {
                await saveProjectEcommerceArtifact(detail.project.id, input);
            }
            setChain(nextChain);
            refreshProject();
            message.success("Prototype A 合同链已保存；当前仍未调用真实 Provider");
        } catch (error) {
            const detailMessage = error instanceof Error ? error.message : "Prototype A 运行失败";
            setRunError(detailMessage);
            message.error(detailMessage);
        } finally {
            setRunning(false);
        }
    }

    return (
        <div className="space-y-6">
            <section className="rounded-xl border border-border/70 bg-background/70 p-4 shadow-sm sm:p-5">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                    <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-2 text-xs text-foreground/48"><span className="inline-flex items-center gap-1.5 font-semibold text-[var(--workspace-accent)]"><FlaskConical className="size-3.5" />Ecommerce Prototype A</span><Tag color="blue" className="m-0 !rounded-full">Provider-free</Tag><Tag className="m-0 !rounded-full">开发面板</Tag></div>
                        <h2 className="mt-2 text-xl font-semibold tracking-normal">商品图创意链路</h2>
                        <p className="mt-1 max-w-2xl text-sm leading-6 text-foreground/55">先验证 ProductDNA → CreativeDirection → ScenePlan → CreativeShotPlan → Result Grid 的可追溯合同。这里不会发送图片请求，也不会伪造生成结果。</p>
                    </div>
                    <div className="flex shrink-0 items-center gap-2 text-xs text-foreground/45"><LockKeyhole className="size-3.5" />事实源：后端 EcommerceArtifact</div>
                </div>
                <div className="mt-5 grid gap-3 border-t border-border/60 pt-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] lg:items-end">
                    <label className="grid gap-1.5 text-xs"><span className="font-medium text-foreground/65">商品资产</span><Select showSearch allowClear value={productAssetId || undefined} options={assetOptions} placeholder={assetOptions.length ? "选择已关联商品资产" : "暂无已关联资产，可直接输入 ID"} onChange={(value) => setProductAssetId(value || "")} optionFilterProp="label" /></label>
                    <label className="grid gap-1.5 text-xs"><span className="font-medium text-foreground/65">商品资产 ID / 测试引用</span><Input value={productAssetId} onChange={(event) => setProductAssetId(event.target.value)} placeholder="例如：product-image-01" /></label>
                    <Button type="primary" icon={running ? <RefreshCw className="size-4 animate-spin" /> : <Sparkles className="size-4" />} loading={running} onClick={() => void runPrototype()} disabled={detail.project.status === "archived"}>{running ? "运行合同链" : "运行 Prototype A"}</Button>
                </div>
                <label className="mt-3 grid gap-1.5 text-xs"><span className="font-medium text-foreground/65">本次商业目标</span><Input value={userGoal} onChange={(event) => setUserGoal(event.target.value)} placeholder="例如：强调材质、容量或使用场景" /></label>
                {!detail.assets.length ? <p className="mt-3 text-xs leading-5 text-amber-600 dark:text-amber-400">当前项目还没有已关联资产。可以先输入一个测试引用验证合同；真实商品事实仍需后续从资产上传/绑定流程进入。</p> : null}
                {runError ? <p className="mt-3 flex items-center gap-1.5 text-xs text-red-500"><CircleAlert className="size-3.5" />{runError}</p> : null}
            </section>

            <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-5">
                {(["product_dna", "creative_direction", "scene_plan", "creative_shot_plan", "qa_report"] as const).map((type) => {
                    const artifact = latestArtifacts[type];
                    return <ArtifactStatusCard key={type} type={type} artifact={artifact} />;
                })}
            </section>

            <section className="rounded-xl border border-border/70 bg-background/70 p-4 shadow-sm sm:p-5">
                <div className="flex flex-col gap-2 border-b border-border/60 pb-3 sm:flex-row sm:items-end sm:justify-between"><div><div className="flex items-center gap-2 text-xs font-semibold text-[var(--workspace-accent)]"><Database className="size-3.5" />Result Grid Projection</div><h3 className="mt-1 text-lg font-semibold">四个创意变体</h3></div><span className="text-xs text-foreground/45">{resultGrid ? `${resultGrid.slots.length} 个槽位 · ${chain ? "本次运行" : "已从后端恢复"}` : "尚未运行"}</span></div>
                {resultGrid ? <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">{resultGrid.slots.map((slot) => <ResultSlotCard key={slot.id} slot={slot} />)}</div> : <div className="mt-4 rounded-lg border border-dashed border-border/80 px-4 py-8 text-center text-sm text-foreground/45">运行 Prototype A 后，这里会显示四个等待 Provider 的结果槽位。</div>}
            </section>

            <section className="rounded-xl border border-border/70 bg-background/70 p-4 shadow-sm sm:p-5">
                <div className="flex items-center justify-between gap-3 border-b border-border/60 pb-3"><div><h3 className="text-lg font-semibold">结构化 QA</h3><p className="mt-1 text-xs text-foreground/48">QA 只检查合同与引用完整性，不代表媒体质量通过。</p></div>{basicQA ? <QAOutcome outcome={basicQA.outcome} /> : <Tag className="m-0 !rounded-full">待运行</Tag>}</div>
                {basicQA ? <div className="mt-4 grid gap-2 sm:grid-cols-3">{basicQA.checks.map((check) => <div key={check.key} className="rounded-lg border border-border/65 px-3 py-3"><div className="flex items-center gap-2 text-xs font-medium"><QAOutcome outcome={check.outcome} /><span>{check.key}</span></div><p className="mt-2 text-xs leading-5 text-foreground/52">{check.detail}</p></div>)}</div> : null}
            </section>
        </div>
    );
}

function ArtifactStatusCard({ type, artifact }: { type: string; artifact?: EcommerceArtifact }) {
    return <div className="rounded-xl border border-border/70 bg-background/70 p-3.5 shadow-sm"><div className="flex items-center justify-between gap-2"><span className="truncate text-sm font-semibold">{artifactLabels[type] || type}</span>{artifact ? <CheckCircle2 className="size-4 text-emerald-500" /> : <span className="size-2 rounded-full bg-foreground/20" />}</div><div className="mt-2 text-xs text-foreground/45">{artifact ? `v${artifact.revision} · ${artifact.lifecycle}` : "未生成"}</div>{artifact ? <div className="mt-1 truncate text-[var(--fs-tiny)] text-foreground/35" title={artifact.artifactKey}>{artifact.artifactKey}</div> : null}{artifact ? <div className="mt-2 text-[var(--fs-tiny)] text-foreground/35">更新于 {formatTime(artifact.updatedAt)}</div> : null}</div>;
}

function ResultSlotCard({ slot }: { slot: EcommerceResultGrid["slots"][number] }) {
    const status = slot.status === "waiting" ? "等待 Provider" : slot.status;
    return <div className="min-h-32 rounded-lg border border-border/70 bg-foreground/[.025] p-3"><div className="flex items-center justify-between gap-2"><span className="text-sm font-medium">{slot.variationId}</span><Tag className="m-0 !rounded-full !text-[var(--fs-tiny)]">{status}</Tag></div><p className="mt-4 text-xs leading-5 text-foreground/45">{slot.needsYouReason || "Provider 结果将在后续阶段写入。"}</p></div>;
}

function QAOutcome({ outcome }: { outcome: BasicQAReport["outcome"] | "PASS" | "FAIL" }) {
    const label = outcome === "PASS" ? "通过" : outcome === "FAIL" ? "失败" : "待核对";
    const color = outcome === "PASS" ? "success" : outcome === "FAIL" ? "error" : "warning";
    return <Tag color={color} className="m-0 !rounded-full">{label}</Tag>;
}

function latestArtifactsByType(artifacts: EcommerceArtifact[], productAssetId: string) {
    const scope = productAssetId.trim() ? `:${stableArtifactKeyPart(productAssetId)}` : "";
    return artifacts.reduce<Record<string, EcommerceArtifact>>((result, artifact) => {
        if (scope && !artifact.artifactKey.endsWith(scope)) return result;
        const current = result[artifact.artifactType];
        if (!current || artifact.revision > current.revision) result[artifact.artifactType] = artifact;
        return result;
    }, {});
}
