# Ecommerce Creative Agent Team v1.2.1 — Codex Ready

更新时间：2026-08-13
状态：`APPROVED FOR PROTOTYPE IMPLEMENTATION`
注意：不是 `READY FOR FULL V1 IMPLEMENTATION`

## 1. 文档定位

这是 Ecommerce Creative Agent Team v1.2 Final Architecture Review 的 Codex-Ready 修订版。它补齐了原审查指出的执行缺口，但不把架构目标、Skill 设计或审查结论当成已经实现的代码。

本项目不是独立 Ecommerce Canvas，而是当前无限画布产品的新增一级 Domain：

```text
AI Creative Studio / Infinite Canvas
├── Film / 短剧
└── Ecommerce / 电商创意
```

共享底层 Canvas Core、Asset/Artifact Store、Task/Result、Model Gateway 和项目权限；Film 与 Ecommerce 的 Agent、Skill、Schema、Evaluator 和业务 Artifact 保持隔离。

## 2. 设计原则和来源优先级

实现冲突时按以下顺序处理：

1. 当前仓库代码、数据库、自动测试和真实浏览器/Provider 证据。
2. 本文档和配套的 [`ecommerce-prototype-implementation-spec-v1.0.md`](./ecommerce-prototype-implementation-spec-v1.0.md)。
3. 仓库内 [`ecommerce-prototype-addendum.md`](./ecommerce-prototype-addendum.md)。
4. 用户提供的完整聊天记录、原有 Film 规划和外部参考项目。

### 2.1 必须遵守

- 不推倒现有 Film 架构，不新建第二套 Canvas。
- Canvas 是 Projection；Project、Asset、Artifact、Task、Result 和版本记录才是事实源。
- API Key、Cookie、Provider Secret 只能在后端 Secret/渠道配置层。
- Prototype 先验证 Creative Engine 假设，再决定是否进入 Full Ecommerce V1。
- 不以未验证的市场百分比、QA 胜率、成本示例或保存天数为产品承诺。

### 2.2 当前门禁

```text
Architecture v1.2
→ v1.2.1 Codex-Ready
→ P0 Architecture/Provider Preparation
→ Prototype A
→ Benchmark
→ Prototype B
→ Benchmark
→ GO / MODIFY / STOP
→ (仅 GO 才能进入 Full Ecommerce V1)
```

## 3. Domain 和项目类型

现有 `Project.Type` 增加明确的 `ecommerce` 值；旧项目保持 `short-drama`，不通过迁移改变旧项目语义。

```text
short-drama  → Film Domain
ecommerce    → Ecommerce Domain
```

Ecommerce 代码新增到独立目录，例如 `web/src/ecommerce/**` 和后端 `Ecommerce*` 合同。不得向 `web/src/film/**`、`FilmArtifact` 或 FilmNodeKind 加电商字段。

## 4. Ecommerce 产品主链

用户提供真实商品图后，不要求用户自己编写复杂 Prompt：

```text
Product Image
→ Product Intelligence
→ ProductDNA
→ Presentation Mode
→ CreativeDirection
→ Creative Skill
→ CreativeShotPlan
→ Structured Variations
→ Batch Generation
→ QA
→ Commercial Assets
```

两条生产模式：

```text
STILL_LIFE
MODEL_INTERACTION
```

## 5. Agent 和 Services

只保留三个 Agent：

### ProductIntelligenceAgent

输入商品图片和来源资产，输出带事实分层的 `ProductDNA`。

### CreativeDirectorAgent

输入 `ProductDNA`、Presentation Mode 和用户视觉目标，输出 `CreativeDirection`。

### SceneDirectorAgent

输入 `ProductDNA`、`CreativeDirection` 和 Skill 要求，输出 `ScenePlan`。

不得为了表现“Agent Team”而新增 EcommerceDirectorAgent、SellingPointAgent、ModelMatchingAgent、GenerationAgent、QAAgent、StillLifeAgent 或 InteractionAgent。

支持 Services：

```text
ModelMatchingService
SkillRecommendationService
CreativeShotPlanner
ConstraintResolver
GenerationCoordinator
QAPipeline
```

`ConsumerInsightService` 可作为后续扩展，但不进入 Prototype 主链。

## 6. Skill Taxonomy 和文件协议

### 6.1 Skill 家族

```text
Creative Skills
├── STILL_LIFE
│   ├── still-life.lifestyle-tabletop   # Prototype A
│   ├── still-life.hero-pedestal        # Full V1 候选
│   └── still-life.clean-commercial     # Full V1 候选
└── HUMAN_INTERACTION
    ├── fashion.natural-walk            # Prototype B
    └── fashion.wind-motion              # Full V1 候选
```

Prototype 只实现 A/B 两项，不提前实现其他三项。

### 6.2 最小 Skill 文件

每个 Prototype Skill 必须具有真实结构，而不是一个 Prompt 别名：

```text
skill/<skill-id>/
├── SKILL.md
├── manifest.json
├── placement.json 或 interaction.json
├── constraints.json
├── variations.json
├── relations.json
├── prompt-template.txt
└── tests/
```

其中必须能表达 Domain Knowledge、Applicability、Execution Rules、Constraints、Structured Variations、Expected Relations 和 Tests。`prompt-template.txt` 只是执行层的一部分，不能代替 Skill。

## 7. Artifact 合同

统一名称：

```text
ProductDNA
CreativeDirection
ScenePlan
ModelProfile
CreativeShotPlan
ExpectedRelationGraph
GenerationJob
GeneratedAsset
QAReport
```

禁止重新引入 `CampaignDNA`、`InteractionPlan`、`SceneInteractionGraph` 作为同义模型。

所有 Ecommerce Artifact 至少包含：

```text
id
project_id
artifact_type
schema_version
revision
lifecycle
payload
source_refs
authority_refs
evidence
created_at
updated_at
```

生命周期：

```text
Draft → Review → Finalize → Immutable Artifact
```

Finalized Artifact 不覆盖；修改创建新 `revision`。`schema_version` 表示结构版本，`revision` 表示同一对象的内容版本，不能混用。

### 7.1 ProductDNA

必须分开记录：

```text
confirmed_facts
inferred_facts
unknown_facts
forbidden_claims
must_preserve
allowed_variations
forbidden_changes
confidence
source_assets
```

商品真实性优先于创意。未知或推断信息不能回写成已确认事实。

### 7.2 CreativeShotPlan

公共字段：

```text
mode
product_ref
creative_direction_ref
skill_ref
scene_ref
camera
lighting
composition
constraints
expected_relations
variations
```

Still-Life 额外记录 `product_placement`、`surface`、`props`、`support`、`orientation`、`logo_visibility`、`text_safe_zones`；Model Interaction 额外记录 `model`、`pose`、`body_mechanics`、`product_interaction`、`contact_points`、`garment_response`。

## 8. Skill Executor

Prototype 使用 Template-Based Executor，不建立 Skill Marketplace、动态插件 Runtime 或 LLM 自由编排：

```text
CreativeSkillExecutor
├── StillLifeSkillExecutor
└── HumanInteractionSkillExecutor
```

执行协议：

```text
SkillLoader
→ 读取 manifest / constraints / variations / relations / template
→ 校验 ProductDNA applicability
→ 绑定 source asset、ProductDNA、CreativeDirection、ScenePlan
→ StillLifeSkillExecutor 或 HumanInteractionSkillExecutor
→ CreativeShotPlan immutable revision
→ ExpectedRelationGraph + structured variations + evidence
```

Executor 输出结构化 Artifact，不直接调用 Provider，不保存 API Key；`GenerationRequest → Provider Gateway → Task/Result` 是唯一生成边界。

## 9. ExpectedRelationGraph

ExpectedRelationGraph 不是图像模型的 Scene Graph 控制协议，只服务于：

1. Prompt Guidance。
2. QA Specification。

Prototype 采用 Skill 文件中的 `relations.json` 模板，由 `CreativeShotPlanner` 绑定实际对象：

```text
relations.json
→ CreativeShotPlanner
→ Product / Scene / Model 变量绑定
→ ExpectedRelationGraph
→ Prompt Guidance + QA Checklist
```

示例：

```text
PRODUCT rests_on SURFACE
PROPS does_not_occlude PRODUCT
LOGO visible_from CAMERA

MODEL wears PRODUCT
LEFT_FOOT contacts GROUND
RIGHT_FOOT contacts GROUND
PRODUCT responds_to BODY_MOTION
```

Prototype 不让 LLM 自由发明整张关系图。

## 10. QA、Benchmark 和统计边界

### 10.1 QA 三级

**LEVEL 1 — Deterministic / Measurable**：尺寸、商品检测、商品占比、明显遮挡、OCR、Text Safe Zone、基础 segmentation。

**LEVEL 2 — Vision Assisted**：商品是否一致、Logo 是否大致正确、是否漂浮、手部/接触/人体是否有明显异常。输出 `PASS | UNCERTAIN | FAIL`；`UNCERTAIN` 转 `NEEDS_YOU`。

**LEVEL 3 — Human Judgment**：高级感、商业摄影感、像不像 AI、布料物理自然度、材质高级程度和整体创意质量，不做自动 Gate，只记录人工通过/拒绝。

Prototype 不写死 `0.85`、`0.90`、`65%` 或 `50%` 等阈值。先记录真实结果，再根据证据定义阈值。

### 10.2 Mandatory Benchmark

同一商品比较：

```text
A — Basic Prompt
B — Expert Prompt Template
C — Ecommerce Creative Skill
```

尽量固定同一 Provider、同一商品、同一数量、尺寸、质量设置和参考素材。

每张图由人工或明确版本的辅助评估记录 1–5 分：

```text
Product Identity
Commercial Usability
Physical Plausibility
AI Artifact Severity
```

其中 AI Artifact Severity 明确记录评分方向（越低越好）。批次记录：

```text
Variation Diversity
Style Consistency
```

另做 Blind Preference：评价者不知道 A/B/C 身份，只判断哪组最像真正的商业摄影。

样本量较小时重点看效果量、人工偏好、First-Pass Acceptance、失败模式和跨商品的一致趋势；统计显著性可作为参考，但不是 Prototype GO/STOP 的硬条件。不得预先写死 Skill 必须高 0.5 分或必须赢 60%。

### 10.3 Prototype Gate

```text
GO     Skill 在多个真实商品上稳定优于 Expert Prompt → 才进入 Full Ecommerce V1
MODIFY 有提升但不稳定 → 调整 Skill / Executor / Provider / ScenePlan / QA / Relation Graph 后重测
STOP   Skill 与 Expert Prompt 相当或更差 → 不扩大 Agent / Skill / Canvas，重评核心假设
```

## 11. P0：架构和 Provider 准备阶段

P0 目标是把执行缺口钉死，不是开始 Full V1。建议一个短周期完成以下六项：

```text
P0.1 锁 Schema
P0.2 锁 Skill 文件协议
P0.3 锁 Executor 协议
P0.4 锁 ExpectedRelationGraph 模板和绑定规则
P0.5 锁 Benchmark Evaluation Protocol
P0.6 Provider Evaluation / Bake-off
```

“一周 P0”或“4–6 周 Prototype”只能作为粗略排程参考，不是交付承诺；真实耗时取决于 Provider 可用性、商品测试集和人工评估，不得为了赶时间跳过 Benchmark。

Provider Evaluation 必须在 Prototype A 正式生成前完成。候选只来自当前实际能够稳定调用、支持参考图/编辑并符合项目约束的 Provider；不能预设 Flux、GPT Image、Seedream、Midjourney 或其他名字。约 5 个真实商品使用统一任务条件比较：

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

选出一个 Provider 后，Prototype A/B 使用同一个 Provider，避免把 Provider 差异误判成 Skill 差异。

## 12. 失败、成本和数据边界

### 12.1 失败和回退

Product Intelligence 失败时保留原商品资产，创建结构化失败记录并进入 `NEEDS_YOU`，不得猜测商品事实。四个生成任务部分失败时保留成功结果和每个 attempt 的失败原因；全部失败时保留 `GenerationJob`、重试入口和人工处理状态，不伪造 Result Grid 完成。Provider 不可用时停止提交新任务，保留可重试状态和原始输入。

每次 Retry 创建新 attempt，记录幂等键、输入 Artifact revision、Provider 响应摘要、延迟、成本和失败原因；不覆盖原 Artifact。

### 12.2 成本

不写死每批 `$0.29` 或任何未验证价格。实际执行记录 Provider、Model、attempt、usage、cost、币种、延迟和计费状态；Prototype 可以设置预算告警，但预算数值必须由真实渠道配置决定。

### 12.3 数据处理

商品图、生成图、Artifact 和日志按 Project 权限隔离；Prototype 默认使用本地或测试商品，最小化持久化和日志内容。不要把固定“保存 30 天”写成架构事实；正式保存期限、删除策略、导出和隐私合规留到产品隐私策略确认后再定。日志不得包含 API Key、Cookie 或完整上游凭证。

## 13. Prototype Canvas Scope

Prototype 首先验证 Creative Engine，不验证完整 Canvas Polish。可以采用：

- CLI/脚本工作流。
- 简单 Web 表单。
- Developer panel。
- 现有 Canvas 的最小节点投影。

最小交互只需要：

```text
上传商品
→ 选择 Mode
→ 选择 Creative Direction
→ 选择 Skill
→ Run
→ Result Grid
→ QA
```

完整 Ecommerce Canvas、资产库、复杂节点视觉和交互等到 Prototype GO 后再做。

## 14. 明确禁止

Prototype 阶段禁止擅自实现：完整五项 Skill、Detail Page、卖点图、视频生成、UGC、Copywriting、多 Provider、Advanced Repair、Skill Marketplace、User Skill Builder、User Model Training、Full Brand System、ERP、CRM、投放、自动发布、完整 Asset Library 和 Ecommerce Full V1。
