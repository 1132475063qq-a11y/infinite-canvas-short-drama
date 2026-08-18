# AI Creative Studio V2：Ecommerce Creative Studio Prototype 附录

更新时间：2026-08-13
状态：`READY FOR PROTOTYPE IMPLEMENTATION`
适用范围：当前仓库的 Ecommerce Domain。

本次完整聊天记录同步后的两个正式执行文件：

- [`ecommerce-agent-team-v1.2.1-codex-ready.md`](./ecommerce-agent-team-v1.2.1-codex-ready.md)：架构基准与 P0 执行协议。
- [`ecommerce-prototype-implementation-spec-v1.0.md`](./ecommerce-prototype-implementation-spec-v1.0.md)：直接给 Codex 的 Prototype 任务书。

本附录负责把它们接入 Film V2 总规划；后续实现细节以这两份拆分文件为准。

## 1. 这份附录解决什么问题

原有《短剧AI无限画布 Production Canvas v2》仍然是 Film Domain 的主规划。本附录是一次**增量同步**，把 Ecommerce Creative Studio 合并进同一个产品，但不覆盖、不改写已经完成的 Film 事实，也不把未来目标当成已实现能力。

总产品现在称为 **AI Creative Studio**：

```text
Shared Canvas Core
├── Film Domain：短剧生产
└── Ecommerce Domain：电商创意生产
```

两条 Domain 共享画布基础设施，但业务语义、Skill、Artifact、Evaluator 和权限边界保持隔离。

## 2. 源优先级与当前事实

实现和验收按以下优先级解释冲突：

1. 当前仓库代码、数据库模型、自动测试和真实浏览器证据。
2. 本附录（仓库内冻结的合并实施边界）。
3. 仓库外的原有 Film 规划文档（仅作为设计来源）。
4. Ecommerce Creative Agent Team 合并接管提示词及其审查记录（作为设计来源，不是实现证明）。

当前已验证的 Film 能力、未完成的 Film 能力和 `FilmArtifact` 继续以 `CONTINUE-HERE.md` 为准。Ecommerce 已有 Provider-free Prototype A 的独立 TypeScript 合同、Provider Evaluation 记录合同、Skill 文件种子、独立 `EcommerceArtifact` 持久化 API、项目类型入口和开发面板，覆盖 ProductDNA、CreativeShotPlan、Artifact 版本、ExpectedRelationGraph、Executor、Result Grid 与基础 QA；仍没有 Ecommerce Agent Runtime 或真实生成/QA 闭环。

因此本附录中的“锁定”表示**合同和范围锁定**；Provider-free 合同骨架与 Provider Evaluation 记录合同已开始实现，但不表示完整产品、真实 Provider、真实媒体或质量门禁已经完成。

## 3. 共享核心与 Domain 隔离

### 3.1 允许复用的 Shared Core

- `CanvasDocumentV2`、节点/连接/端口、历史、Undo/Redo、缩放/平移。
- 8px Grid、Snap、Alignment、Distribution、Collision 和既有布局引擎。
- Project、Canvas Projection、Resource/Asset/AssetVersion、Task/Result、Provider Gateway 边界。
- 后端密钥与渠道配置、任务持久化、轮询、取消、重试基础能力。
- 统一的 Artifact 版本化原则：不可变版本、当前版本指针、可追溯来源。

### 3.2 必须保持隔离的 Domain

```text
web/src/film/**                 # Film 节点、端口、Inspector、Artifact 解析
backend ... FilmArtifact        # 影视 Shot/Acting/Prompt Pack 事实源

web/src/ecommerce/**            # 新增，不能塞入 film/domain
backend ... Ecommerce*          # 新增，不能向 FilmArtifact 加电商字段
```

不得通过“通用 CreativeDirector”“通用 Skill”或把所有字段塞进 `FilmArtifact` 的方式共享业务模型。共享只发生在明确稳定的 Core 接口层。

## 4. Ecommerce 最小领域合同

### 4.1 项目类型

在现有 `Project.Type` 上增加明确的领域值：

```text
short-drama   # 既有 Film 项目
ecommerce     # Ecommerce Creative Studio 项目
```

旧项目默认保持 `short-drama`，不得通过迁移把旧项目变成电商项目。创建页和项目导航后续只负责选择领域，不在 Film 页面混入电商字段。

### 4.2 两条模式和两项 Prototype Skill

```text
STILL_LIFE
└── still-life.lifestyle-tabletop       # Prototype A

MODEL_INTERACTION
└── fashion.natural-walk                # Prototype B
```

V1 规划中的 `hero-pedestal`、`clean-commercial`、`wind-motion` 暂不实现。Detail Page、卖点图、视频、UGC、Copywriting、Marketplace、ERP/CRM 等也不属于 Prototype。

### 4.3 三个 Agent 与 Services

只锁定三个 Agent：

1. `ProductIntelligenceAgent`：商品图片 → `ProductDNA`。
2. `CreativeDirectorAgent`：`ProductDNA` + 展示模式 + 用户目标 → `CreativeDirection`。
3. `SceneDirectorAgent`：`ProductDNA` + `CreativeDirection` + Skill 要求 → `ScenePlan`。

Generation、QA、Still-Life 和 Interaction 不拆成额外 Agent；它们由服务和可追溯任务完成：

```text
ModelMatchingService
SkillRecommendationService
CreativeShotPlanner
ConstraintResolver
GenerationCoordinator
QAPipeline
```

`ConsumerInsightService` 不是 Prototype 主链。

### 4.4 Artifacts 与生命周期

Prototype 使用以下名称，不再引入 `CampaignDNA`、`InteractionPlan`、`SceneInteractionGraph`：

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

所有 Ecommerce Artifact 必须包含 `schema_version` 与 `revision`，两者含义不同。生命周期为：

```text
Draft → Review → Finalize → Immutable Artifact
```

Finalized Artifact 不覆盖；修改必须创建新 revision，并保留 source artifact、skill、provider、task 和输入资产引用。

### 4.5 ProductDNA 真实性边界

ProductDNA 必须分别保存：

- `confirmed_facts`
- `inferred_facts`
- `unknown_facts`
- `forbidden_claims`
- `must_preserve`
- `allowed_variations`
- `forbidden_changes`
- `confidence`
- `source_assets`

商品真实性优先于创意。未知信息不能被 Agent 当成已确认商品属性；生成链路不得把推断内容写回确认事实。

### 4.6 CreativeShotPlan 与 ExpectedRelationGraph

`CreativeShotPlan` 的公共字段至少包括：

```text
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

Still-Life 额外包含商品摆放、台面、道具、支撑、朝向、Logo 可见性和文字安全区；Model Interaction 额外包含模特、姿势、身体力学、产品互动、接触点和服装响应。

`ExpectedRelationGraph` 只用于 Prompt Guidance 和 QA Specification，不是生成模型的 Scene Graph 控制协议，也不能让 LLM 自由发明整张图：

```text
relations.json
→ CreativeShotPlanner
→ 变量绑定
→ ExpectedRelationGraph
→ Prompt Guidance + QA Specification
```

示例关系：`PRODUCT rests_on SURFACE`、`PROPS does_not_occlude PRODUCT`、`LOGO visible_from CAMERA`、`MODEL wears PRODUCT`、`FOOT contacts GROUND`。

## 5. Skill Executor 合同

Prototype 只使用 Template-Based Executor，不建立动态插件 Marketplace 或复杂自动运行时。

```text
CreativeSkillExecutor
├── StillLifeSkillExecutor
└── HumanInteractionSkillExecutor
```

统一执行输入/输出：

```text
Input:
  skill_ref
  ProductDNA ref
  CreativeDirection ref
  ScenePlan ref
  source_asset refs

Output:
  immutable CreativeShotPlan revision
  ExpectedRelationGraph
  structured variations
  evidence (recorded | inferred | unknown)
  warnings / needs_you reasons
```

Executor 不直接调用模型或保存 API Key。它只编译可审计的 `CreativeShotPlan`；真正生成必须经过 `GenerationRequest → Provider Gateway → Task/Result`。

## 6. Prototype 实施顺序

### Prototype A：Lifestyle Tabletop

```text
Product Upload
→ ProductDNA
→ Presentation Mode (STILL_LIFE)
→ CreativeDirection
→ still-life.lifestyle-tabletop
→ ScenePlan
→ CreativeShotPlan
→ 4 Structured Variations
→ Generation
→ Basic QA
→ Result Grid
```

### Prototype B：Natural Walk

```text
Fashion Product
→ ProductDNA
→ Presentation Mode (MODEL_INTERACTION)
→ CreativeDirection
→ fashion.natural-walk
→ ScenePlan
→ CreativeShotPlan
→ 4 Structured Variations
→ Generation
→ Basic QA
→ Result Grid
```

Prototype 阶段优先验证 Creative Engine，可以使用现有 Canvas 的最小节点投影或开发面板；不为了证明 Skill 而先重做完整 Ecommerce Canvas UI。

## 7. Provider、失败、成本和数据边界

### 7.1 Provider 选择

正式 Prototype 只使用一个图片 Provider。不能根据模型名称主观指定。先用约 5 个真实商品做小型 Bake-off，使用相同商品、尺寸、质量、参考素材和提示词条件，比较：

- Product identity
- Reference fidelity
- Instruction following
- Commercial quality
- Generation stability
- Latency
- Cost
- API reliability

Bake-off 结果必须记录 Provider、Model、请求摘要、原始任务 ID、延迟、成本和失败原因，并明确当前结论是 `recorded` 还是 `inferred`。

### 7.2 失败处理

失败不能覆盖原 Artifact 或伪装成成功：

```text
GenerationRequest
→ ProviderJob / Task
→ retryable | terminal_failure | needs_you
→ attempt history
→ Result or structured failure
```

重试必须新建 attempt，保持幂等键、原始输入、Provider 响应摘要和费用记录。自动重试不改变 ProductDNA、CreativeDirection 或已 Finalize 的 CreativeShotPlan。

### 7.3 成本与数据

- API Key、Secret、渠道配置只在后端保存；前端节点只持有引用和脱敏状态。
- 每次生成记录 provider、model、attempt、latency、usage/cost（若上游提供）和计费状态。
- 商品图、生成结果和 Artifact 只按当前 Project 权限读取；日志不得泄露密钥、Cookie 或完整上游凭证。
- 本地测试默认使用 provider-free fixture；真实 Provider 验收必须明确标注为真实调用，不能用静态测试替代。

## 8. QA 与 Benchmark Gate

### 8.1 三层 QA

1. **LEVEL 1 Deterministic**：尺寸、商品检测、商品占比、遮挡、OCR、文字安全区、基础分割。
2. **LEVEL 2 Vision Assisted**：商品是否一致、Logo 是否大致正确、漂浮、手部/接触/人体明显异常；输出 `PASS | UNCERTAIN | FAIL`，`UNCERTAIN` 进入 `NEEDS_YOU`。
3. **LEVEL 3 Human Judgment**：高级感、商业摄影感、创意质量等不提前自动 Gate，只记录人工通过/拒绝。

Prototype 初期不写死 0.85、0.90 或“Skill 必须赢 60%”之类阈值；先记录真实数据，再决定阈值。

### 8.2 Mandatory Benchmark

同一个商品必须尽量固定 Provider、尺寸、数量、质量和参考素材，比较：

```text
A Basic Prompt
B Expert Prompt Template
C Ecommerce Creative Skill
```

Per image 记录 Product Identity、Commercial Usability、Physical Plausibility、AI Artifact Severity；Per batch 记录 Variation Diversity、Style Consistency；另做不知道 A/B/C 身份的 Blind Human Preference。

### 8.3 Gate

- `GO`：Skill 稳定优于 Expert Prompt，进入 Full Ecommerce V1。
- `MODIFY`：有提升但不稳定，调整 Skill、Executor、Provider、ScenePlan、QA 或 ExpectedRelationGraph 后重测。
- `STOP`：Skill 与 Expert Prompt 相当或更差，不扩大 Agent/Skill/Canvas，重新评估核心假设。

## 9. 本轮允许、禁止与下一步

本轮允许：

- 新增 Ecommerce Domain 合同、目录和 provider-free fixture。
- 复用 Shared Canvas Core，做最小项目类型选择和 Prototype A 数据链路。
- 为 ProductDNA、CreativeDirection、ScenePlan、CreativeShotPlan、QA 和 Benchmark 建立可测试的版本化边界。

本轮禁止：

- 完整 Ecommerce V1 或完整五项 Skill。
- Detail Page、卖点图、视频、UGC、Copywriting、Marketplace、用户 Skill Builder、训练模型、品牌系统、ERP/CRM、投放和自动发布。
- 多 Provider 同时接入、绕过 Provider Gateway、前端保存 API Key。
- 把 Ecommerce 字段塞进 `FilmArtifact`，或修改 Film Domain 语义以迁就电商。

下一步唯一编码目标：在当前合同之上，接入用户明确提供且实际可调用的候选渠道，执行约 5 个商品的真实 Bake-off，记录统一设置指纹、延迟、失败、成本、评分和盲测；未提供渠道或素材前不得伪造真实结果，也不扩展完整 Ecommerce Canvas。

完整聊天记录进一步锁定了 P0 顺序：`Schema → Skill 文件协议 → Executor → ExpectedRelationGraph → Benchmark Evaluation → Provider Evaluation/Bake-off`。Provider 不能按品牌预设；Prototype UI 可以是 CLI、简单表单、Developer panel 或最小 Canvas 投影。Benchmark 采用 1–5 分和 Blind Preference，统计显著性、固定胜率、固定成本与固定保存天数都不是硬编码门禁。

## 10. 文档同步关系

- 本附录：当前合并项目的 Ecommerce 实施边界和下一步合同。
- `CONTINUE-HERE.md`：跨账号、跨会话的总状态和当前门禁。
- `NEW-ACCOUNT-HANDOFF.md`：新账号接管顺序。
- `docs/production-canvas/phase-0-baseline.md`：共享基线与事实源边界。
- `CHANGELOG.md`：本次文档同步记录。
- `docs/production-canvas/ecommerce-agent-team-v1.2.1-codex-ready.md`：完整架构基准。
- `docs/production-canvas/ecommerce-prototype-implementation-spec-v1.0.md`：完整 Prototype 任务书。
- Downloads 中的原有 Film 规划和 Ecommerce 合并提示词保留为来源证据；本次只做增量附录，不覆盖原文。
