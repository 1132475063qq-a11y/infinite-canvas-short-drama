# 新账号续接说明：AI Creative Studio（短剧 + 电商）无限画布

本文件记录 `codex/phase4-film-nodes` 的续接边界。每次接手前都必须检查本地 `git status`、当前分支和上游同步状态；GitHub 提交只代表代码已同步，不代表浏览器、真实 Provider 或媒体质量已经验收。

## 1. 项目与交接边界

| 项目项 | 当前值 |
| --- | --- |
| 正式仓库 | 当前 checkout 的仓库根目录 |
| GitHub 仓库 | `https://github.com/1132475063qq-a11y/infinite-canvas-short-drama` |
| 当前分支 | `codex/phase4-film-nodes` |
| 开发方向 | Film Production Canvas 为主；Ecommerce Creative Studio 保持独立 Prototype A 边界 |

切换 Codex / ChatGPT 账号时，本地仓库、Git 历史、Docker 数据和 GitHub 远端代码仍留在电脑上；当前 Codex 任务、侧边栏历史、账号额度、订阅和记忆不会自动迁移。新账号只需打开同一个本地目录，或从 GitHub 拉取上述分支。

不要把 API Key、GitHub Token、Cookie、登录码或本地环境文件提交到仓库。

## 2. 本轮已落地：Film Provider Gateway 执行事实链

Provider Gateway 的图片链路已经完成真实浏览器验收。空间连续性、视频结果投影和渠道目录同步已写入代码与测试合同，但当前分支仍需要针对性后端测试、前端构建与登录态浏览器回归；视频 Provider 没有验收。

### 已实现的后端事实源与事务边界

- `FilmArtifact(generation_request)` 是不可变的生成请求快照，精确冻结 Prompt Pack、Shot Contract 与资产版本来源。
- 原子提交路由会在同一事务中创建 queued `Task`、初始 queued `GenerationAttempt`、可选计费预留，以及服务端 `CanvasProjectionPatch`；失败时全部回滚。
- `GenerationAttempt` 保存每次领取、租约恢复、取消和 Retry 的执行编号；`ProviderJob` 按 Attempt 归属上游任务号；成功时 `Task`、Attempt 与 `Result(kind = film_generation_result)` 同事务落库。
- 失效 Worker 的进度、终态与完成写入必须匹配原租约，不能覆盖后继 Attempt；迟到的上游观察会保留在原 Attempt，不会改写新 Attempt 的 Task 指针。
- 已知上游 ID、但没有确认终态证据的失败保留为 `uncertain`，避免错误标成明确失败；明确 4xx 拒绝仍可记录为失败。
- 新增只读执行历史接口：`GET /projects/:projectId/film-generation-requests/:artifactId/executions`。
- Canvas 读取时由服务端临时投影 `taskId`、`generationAttemptId`、`providerJobId`、`resultId`；浏览器保存时会剥离这些运行时 ID，Canvas 不是事实源。

### 测试合同与验证边界

- 原子创建与幂等重放；损坏事实链重建保护；事务回滚；路由漂移拒绝。
- 成功链、上游状态不明、排队后取消再 Retry、租约恢复、迟到 Provider 观察和过期 Worker 完成写入隔离。
- Canvas Projection Patch 只投影服务端真实绑定，客户端不能伪造或擦除运行时 ID。

完整待验证清单见 [docs/content/docs/pending-test.mdx](docs/content/docs/pending-test.mdx)，完整合同见 [docs/production-canvas/film-generation-request-contract.md](docs/production-canvas/film-generation-request-contract.md)。

## 3. 已完成的前置阶段

### Film Production Canvas

- Phase 1：Film Semantic Layer 基础。
- Phase 1.5：FilmNodeKind、Typed Port、CanvasDocumentV2 契约冻结。
- Phase 2：Grid、Snap、Alignment、Distribution、Scene Lane、Collision 核心布局能力。
- Phase 3：Project Navigator、顶部生产状态、Inspector Shell、Bottom Shot Strip；此前已有独立的自动测试和浏览器验收记录。
- Phase 4：Scene、Shot、Character、Location、Prop、Acting、Prompt Pack、Generation Request、只读 Task Draft、只读 Provider Route Catalog、revision-safe Canvas Projection Patch。
- 当前 Phase 4/5 工作树的完整回归仍以 `pending-test.mdx` 记录为准，不能把以前的测试通过结果外推到当前全部改动。

### Ecommerce Creative Studio（保持 Prototype A）

- 已有 ProductDNA、CreativeDirection、ScenePlan、CreativeShotPlan、QAReport 和 Result Grid 的独立合同与后端持久化原型。
- 已有 Provider Evaluation 的纯 TypeScript 记录合同，可表达候选渠道、输入指纹、结果、成本、失败证据、盲测与 GO/MODIFY/STOP 决策；它尚未持久化，也不会调用 Provider。
- 尚未做真实 Provider Bake-off、真实媒体生成、Agent Runtime、完整 Ecommerce Canvas 或 Full V1；不要把文档规划误报为已实现。

## 4. 仍未完成：严格的下一步顺序

### Film：先完成一条可验收的生成链

1. **复跑本轮架构修复**：验证空间 Pack 双页面并发冲突、活动任务删除门禁、Task 双 ID 迁移与全部现有回归。
2. **浏览器验收空间编辑器**：新建、保存、ready/locked、冲突刷新、Gate FAIL/UNCERTAIN/PASS 和 Generation Request 门禁。
3. **补跨 Scene 的 Location/开口身份**：只有确有多场景连续性需求时再增加项目级 Location/Opening ID，不把 SceneHierarchy 误报为已实现。
4. **再做端用户 Retry/QC UI 与数据库 Golden Project**。视频保持禁测，除非用户另行明确授权。

当前禁止：一开始接多家 Provider、前端保存 API Key、提前实现复杂 Agent Runtime、多人协作或 Full Ecommerce V1。

### Ecommerce：等真实输入与渠道明确后再推进

1. 用户提供实际可调用的候选 Provider、产品图和允许测试的素材。
2. 先执行 P0.6 Provider Evaluation / Bake-off，记录成本、失败、质量和盲测证据。
3. 只在 Bake-off 达到门槛后，继续 `Product Upload → ProductDNA → CreativeDirection → Lifestyle Tabletop → CreativeShotPlan → Result Grid` 的真实闭环。

仍只做 `still-life.lifestyle-tabletop`；卖点图、详情页、UGC、视频、文案和更多 Skill 都不在当前 Prototype A 范围内。

## 5. 新账号的第一条指令

将下面文字直接发给新账号的 Codex：

```text
请接手当前仓库根目录中的项目。

先完整读取：
1. NEW-ACCOUNT-HANDOFF.md
2. CONTINUE-HERE.md
3. AGENTS.md
4. docs/content/docs/pending-test.mdx
5. docs/production-canvas/film-generation-request-contract.md
6. docs/production-canvas/ecommerce-prototype-addendum.md
7. docs/production-canvas/ecommerce-agent-team-v1.2.1-codex-ready.md
8. docs/production-canvas/ecommerce-prototype-implementation-spec-v1.0.md
9. 仓库外的 Film 主规划（如当前环境提供，仅作为设计来源）
10. 仓库外的 Ecommerce 架构审查记录（如当前环境提供，仅作为设计来源）

然后只做只读检查：git status -sb、git branch --show-current、git log -8 --oneline、git remote -v、git rev-list --left-right --count HEAD...@{upstream}。

先以当前本地文件和 `git diff` 为事实，不要执行会覆盖本地改动的 pull/reset/checkout。Provider Gateway 图片真实链路已经验收，视频未测试。

下一步先复跑空间 Pack 乐观锁、项目删除门禁和 Task 双 ID 迁移测试，再做登录态空间编辑器验收。Film 与 Ecommerce Domain 必须隔离；不要提前接视频、Full Ecommerce V1 或复杂 Agent Runtime。
```

## 6. 新账号恢复步骤

### 使用现有本地目录

```bash
cd <repository-root>
git status -sb
git pull --ff-only
git branch --show-current
git log -1 --oneline
git rev-list --left-right --count HEAD...@{upstream}
```

最后一行是 `0 0` 时，表示本地与 GitHub 对应分支同步。

### 在另一台电脑重新获取代码

```bash
git clone https://github.com/1132475063qq-a11y/infinite-canvas-short-drama.git
cd infinite-canvas-short-drama
git switch --track origin/codex/phase4-film-nodes
```

如果新 GitHub 账号没有仓库写权限，继续使用原 GitHub 登录推送，或由仓库所有者添加新账号为协作者；不要把访问令牌写进代码或交接文档。

## 7. 启动与验收提醒

项目的本地启动、端口与历史验收记录见 [CONTINUE-HERE.md](CONTINUE-HERE.md)。不要因为页面能打开或代码已推送就把当前后端事实链视为验收通过；本轮尚缺针对性回归、视频 Provider 和真实媒体质量证据。

Git 推送只保存代码和交接信息，不会启动服务、调用模型或改变任何 Provider 配置。
