# Ecommerce Creative Studio — Prototype Implementation Spec v1.0

更新时间：2026-08-13
执行仓库：`/Users/xiangyuqin/Downloads/infinite-canvas-short-drama`
前置架构：[`ecommerce-agent-team-v1.2.1-codex-ready.md`](./ecommerce-agent-team-v1.2.1-codex-ready.md)

## 1. 给 Codex 的任务边界

你接手的是现有 AI Creative Studio / Infinite Canvas 项目，不是新建 Ecommerce 项目。先读：

1. `CONTINUE-HERE.md`
2. `NEW-ACCOUNT-HANDOFF.md`
3. `docs/production-canvas/ecommerce-agent-team-v1.2.1-codex-ready.md`
4. 本文件
5. `/Users/xiangyuqin/Downloads/短剧AI无限画布_Production_Canvas_v2_超详细发展规划.md`
6. 仓库内 `AGENTS.md`

执行前检查当前分支和 Git 状态，保留用户改动，不重做 Film Phase 1–4 已完成内容。

### 当前 Gate

```text
READY FOR PROTOTYPE IMPLEMENTATION
```

### 当前目标

只证明 Prototype A 的 Creative Engine 数据链路可追溯，并为 Provider Bake-off 和后续真实生成留下稳定合同：

```text
Product Upload
→ ProductDNA
→ Presentation Mode
→ CreativeDirection
→ still-life.lifestyle-tabletop
→ ScenePlan
→ CreativeShotPlan
→ 4 Structured Variations
→ Basic QA
→ Result Grid
```

本阶段不实现 Full Ecommerce V1，不把静态 Fixture、Mock 结果或自动测试说成真实模型质量证明。

## 2. P0 先做准备，不直接接模型

### P0.1 Schema Contract

在独立 Ecommerce 目录建立 TypeScript 合同，至少包含：

```text
EcommerceProjectMode
EcommerceNodeKind
EcommerceDomainRef
EcommerceArtifactLifecycle
EcommerceEvidence
ProductDNA
CreativeDirection
ScenePlan
CreativeShotPlan
ExpectedRelationGraph
GenerationJob
GeneratedAsset
QAReport
```

要求：

- `short-drama` 和 `ecommerce` 是明确的 Project Type。
- 所有 Artifact 有 `schema_version`、`revision`、lifecycle、source refs 和 evidence。
- ProductDNA 分离 confirmed / inferred / unknown / forbidden claims。
- Finalized Artifact 不可覆盖。
- 不导入 `FilmArtifact`，不修改 FilmNodeKind 以承载电商。

推荐目录：

```text
web/src/ecommerce/
├── domain/
│   ├── types.ts
│   ├── artifact.ts
│   ├── product-dna.ts
│   ├── creative-direction.ts
│   ├── shot-plan.ts
│   ├── relation-graph.ts
│   └── state-machine.ts
├── skills/
│   ├── types.ts
│   ├── loader.ts
│   ├── still-life-lifestyle-tabletop/
│   └── human-interaction-natural-walk/
├── executor/
│   ├── skill-executor.ts
│   ├── still-life-executor.ts
│   └── human-interaction-executor.ts
├── prototype/
│   ├── prototype-a.ts
│   ├── result-grid.ts
│   └── basic-qa.ts
└── index.ts
```

路径可以按现有工程习惯调整，但 Domain 不得落入 `web/src/film/**`。

### P0.2 Skill 文件协议

只创建 Prototype A 的 Skill 文件，严格使用：

```text
skill/still-life.lifestyle-tabletop/
├── SKILL.md
├── manifest.json
├── placement.json
├── constraints.json
├── variations.json
├── relations.json
├── prompt-template.txt
└── tests/
```

`relations.json` 必须是关系模板，不是任意 LLM 输出。Prototype B 的目录和 executor 类型可以先定义合同，但不实现自然行走逻辑，直到 A 通过门禁。

### P0.3 Skill Executor

建立可替换的 Template-Based `CreativeSkillExecutor`：

```text
Skill + ProductDNA + CreativeDirection + ScenePlan
→ SkillLoader
→ applicability / constraint validation
→ variable binding
→ CreativeShotPlan revision
→ ExpectedRelationGraph + variations + evidence
```

Executor 不访问 Provider、不接收 Secret、不保存媒体、不直接创建伪造 Result。输出必须能追溯到 skill id/version、输入 Artifact revision 和 source asset ids。

### P0.4 ExpectedRelationGraph

由 `relations.json` + `CreativeShotPlanner` 生成：

```text
PRODUCT rests_on SURFACE
PROPS does_not_occlude PRODUCT
LOGO visible_from CAMERA
```

图同时服务 Prompt Guidance 和 Basic QA checklist。它不是图像模型 Scene Graph 控制协议，也不允许 LLM 自由生成整个 graph。

### P0.5 Benchmark Evaluation Protocol

先实现记录合同，不先写胜率阈值：

```text
BenchmarkCase
BenchmarkVariant (basic | expert | skill)
PerImageScore (1–5)
BatchScore (1–5)
BlindPreference
FailureMode
Cost
Latency
Evidence
```

Per-image：Product Identity、Commercial Usability、Physical Plausibility、AI Artifact Severity（低分更好）。Per-batch：Variation Diversity、Style Consistency。样本小的时候以效果量、盲测、First-Pass Acceptance、失败模式和跨商品趋势为主，不把统计显著性作为硬门禁。

### P0.6 Provider Evaluation / Bake-off

正式 Prototype A 生成前做小型 Bake-off：约 5 个真实商品 × 当前实际可稳定调用且满足参考图/编辑要求的候选 Provider。候选不得按品牌名预设；统一比较：

```text
Reference Fidelity
Product Identity
Commercial Quality
Instruction Following
Variation Ability
Latency
Failure Rate
Cost
API Stability
```

选定后，A/B/C 三组和两个 Prototype 使用同一个 Provider，避免 Provider 差异污染 Skill 结论。成本和保存期限不写示例数值。

当前已加入 Provider Evaluation 的纯 TypeScript 合同：候选能力、统一设置指纹、测试 Case、每次尝试、任务/结果引用、延迟、usage/cost、结构化失败、1–5 分维度（含 API Stability）、Blind Preference 和显式 `GO | MODIFY | STOP` 决策。合同不会调用 Provider、生成媒体或自动推断 `GO`；真实 Bake-off 仍必须由实际可用渠道和人工/媒体证据完成。

## 3. Prototype A 数据合同

### 3.1 输入

```text
PrototypeAInput {
  projectId: string
  sourceAssetIds: string[]
  productImageId: string
  userGoal?: string
  requestedCount: 4
}
```

原始商品图必须保留为 source asset。分析失败不能用猜测补齐 confirmed facts。

### 3.2 ProductDNA

至少包含：类别、颜色、形状、可见材质、Logo、包装、关键特征、must_preserve、allowed_variations、forbidden_changes、confidence、source_assets，以及 confirmed/inferred/unknown/forbidden 分层。

### 3.3 CreativeDirection 和 ScenePlan

Prototype A 默认 `mode: STILL_LIFE`。CreativeDirection 记录用户目标、视觉方向、商业使用上下文和约束；ScenePlan 记录台面/表面、背景、灯光、构图、道具策略、Logo 可见性和文字安全区。

### 3.4 CreativeShotPlan

必须生成 4 个结构化 variations，而不是四个无关联 Prompt。每个 variation 记录：

```text
variation_id
camera
lighting
composition
product_placement
surface
props
support
orientation
logo_visibility
text_safe_zones
constraints
expected_relations
prompt_preview
source_artifact_refs
```

4 个 variation 应该有明确变化轴，同时保持 `must_preserve` 和商品事实不变。

## 4. Basic QA 和 Result Grid

### 4.1 Basic QA

Provider-free 阶段只实现可确定、可测试的基础检查合同：

- 结果数量是否与请求一致。
- 状态、mime type、尺寸和 source task 引用是否完整。
- 生成结果是否可追溯到 variation、CreativeShotPlan revision 和 ProductDNA revision。
- 明显缺失字段进入 `UNCERTAIN` 或 `NEEDS_YOU`，不得自动判定为商业质量通过。

真实媒体阶段再接 LEVEL 1 deterministic 和 LEVEL 2 vision-assisted 检查；LEVEL 3 高级感/商业摄影感保留人工判断。

### 4.2 Result Grid

Result Grid 是结果 Projection，不是事实源。每张卡片显示：

```text
variation id
generation status
provider/model（如已真实调用）
task/attempt
QA status
evidence
needs_you reason
```

Prototype UI 可以是简单 Web 表单、Developer panel 或现有 Canvas 最小节点投影，不要求完整交互画布。

## 5. 失败、重试、成本和隐私

- Product Intelligence 失败：保留源图，生成结构化 failure，进入 `NEEDS_YOU`。
- 四个任务部分失败：保留成功项和每个失败 attempt；不整体伪装成成功。
- 全部失败或 Provider 不可用：停止新提交，保留可重试状态、原始输入和失败原因。
- Retry 新建 attempt，不覆盖原 Artifact/Result。
- API Key 只在后端；前端只显示脱敏 Provider 状态。
- 记录实际 provider/model/usage/cost/latency/计费状态，不写死 `$0.29` 等示例。
- Prototype 默认使用本地/测试商品并最小化持久化；不把“保存 30 天”写成事实。

## 6. 允许修改与明确禁止

### 允许

- `web/src/ecommerce/**` 独立合同、Skill loader、Prototype A executor、fixture 和纯函数测试。
- 最小 Project Type 解析，使 `ecommerce` 项目不落入 Film 投影。
- Shared Core 的稳定扩展点，但不改变旧 Film 文档迁移和 Film Artifact 语义。

### 禁止

- 完整 Ecommerce V1、完整五项 Skill、Detail Page、卖点图、视频、UGC、Copywriting。
- 多 Provider 同时接入、提前决定具体模型、绕过 Provider Gateway。
- Marketplace、User Skill Builder、训练模型、Full Brand System、ERP、CRM、投放、自动发布。
- 把电商 Artifact 写入 `FilmArtifact`，或把 Prototype Mock 说成真实媒体质量证明。

## 7. 验收顺序和交付物

### A. Provider-free 合同验收

- `Product Upload → ProductDNA → CreativeDirection → ScenePlan → Skill → CreativeShotPlan` 全链路可运行。
- 4 个 variations 有稳定 ID、版本、关系图和 source refs。
- Basic QA 能识别结构缺失和 `NEEDS_YOU`。
- Result Grid 能展示四个结果槽位，即使结果还是 provider-free 状态。
- 自动测试覆盖生命周期、不可变 revision、关系绑定、4 variations、失败状态和跨项目隔离。

### B. Bake-off 验收

- 约 5 个商品、候选 Provider、统一条件和实际成本/延迟记录齐全。
- Provider 选择有证据，不按品牌名预设。

### C. 真实 Prototype A 验收

同一商品跑：

```text
A Basic Prompt
B Expert Prompt Template
C still-life.lifestyle-tabletop
```

同一 Provider、数量、尺寸、质量和参考素材尽量一致；完成 1–5 分评估、Blind Preference、First-Pass Acceptance、失败模式、成本、延迟和证据等级记录。

### D. Prototype B

仅在 A 完成并经过复盘后实现 `fashion.natural-walk`，再跑同样的 A/B/C Benchmark。

### E. 最终门禁

```text
GO / MODIFY / STOP
```

没有真实 Provider 和人工/媒体证据时，只能报告“合同或结构测试通过”，不能报告 GO。

## 8. 本阶段完成定义

本 Spec 的完成不是把完整电商产品做完，而是：

1. P0 合同和 Provider Evaluation 规则固化，并有可复用的记录合同。
2. Prototype A 无 Provider 链路和测试可重复运行。
3. 真实生成前，所有 Provider/成本/隐私/失败边界可审计。
4. 只有完成 Benchmark 后，才允许做 Prototype B 和 GO/MODIFY/STOP 决策。
