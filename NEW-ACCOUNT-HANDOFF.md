# 新账号续接说明：AI Creative Studio（短剧 + 电商）无限画布

更新时间：2026-08-13

## 1. 当前项目位置

本地正式仓库：

```text
/Users/xiangyuqin/Downloads/infinite-canvas-short-drama
```

GitHub 仓库：

```text
https://github.com/1132475063qq-a11y/infinite-canvas-short-drama
```

当前开发分支：

```text
codex/phase4-film-nodes
```

最近阶段提交：

```text
f4d7cf4 feat(film): add versioned production artifacts
```

## 2. 换 Codex 账号时

只更换 Codex / ChatGPT 账号，不会修改本地 Git 仓库、GitHub 登录状态、分支或 Docker 数据。

新账号进入 Codex 后：

1. 打开本地目录 `/Users/xiangyuqin/Downloads/infinite-canvas-short-drama`。
2. 把下面“新会话第一条指令”发给新会话。
3. 不要重新克隆项目，不要重新初始化 Git，不要删除 Docker volume。
4. 新会话必须先检查当前分支、Git 状态、`CONTINUE-HERE.md` 和 Ecommerce 附录，再开始下一阶段。

## 3. 新会话第一条指令

```text
请接手本地项目：
/Users/xiangyuqin/Downloads/infinite-canvas-short-drama

先完整读取：
1. /Users/xiangyuqin/Downloads/infinite-canvas-short-drama/NEW-ACCOUNT-HANDOFF.md
2. /Users/xiangyuqin/Downloads/infinite-canvas-short-drama/CONTINUE-HERE.md
3. /Users/xiangyuqin/Downloads/infinite-canvas-short-drama/docs/production-canvas/ecommerce-prototype-addendum.md
4. /Users/xiangyuqin/Downloads/infinite-canvas-short-drama/docs/production-canvas/ecommerce-agent-team-v1.2.1-codex-ready.md
5. /Users/xiangyuqin/Downloads/infinite-canvas-short-drama/docs/production-canvas/ecommerce-prototype-implementation-spec-v1.0.md
6. /Users/xiangyuqin/Downloads/infinite-canvas-short-drama/docs/production-canvas/film-generation-request-contract.md
7. /Users/xiangyuqin/Downloads/电商创意工作台设计.md
8. /Users/xiangyuqin/Downloads/短剧AI无限画布_Production_Canvas_v2_超详细发展规划.md
9. 仓库内 AGENTS.md

然后执行 git status -sb、git branch --show-current、git log -8 --oneline，确认当前分支为 codex/phase4-film-nodes，且 Film 阶段提交 f4d7cf4 已存在。

不要重做 Phase 1、Phase 1.5、Phase 2、Phase 3。Film 的 Generation Request 版本事实、Canvas Projection、安全边界、只读 `GenerationRequest → Task Draft` 合同、只读 Provider Route Catalog 和 revision-safe Canvas Projection Patch 已经完成；下一步实现 Provider Gateway 的原子 Task 创建。Ecommerce 的 P0 合同和 Provider Evaluation 记录合同已经完成，下一步只是在用户提供实际可调用渠道和素材后执行真实 Bake-off，再从 Prototype A 真实生成/QA 开始。先保持 Film/Ecommerce Domain 隔离，并按 `AGENTS.md` 的当前验证纪律执行；不要提前接入多家 Provider、Full Ecommerce V1 或复杂 Agent Runtime。
```

## 4. 当前已经完成

- Phase 1：Film Semantic Layer 基础。
- Phase 1.5：26 种 FilmNodeKind、Typed Port、CanvasDocumentV2 契约冻结。
- Phase 2：Grid、Snap、Alignment、Distribution、Scene Lane、Collision 核心能力。
- Phase 3：完整 Project Navigator、顶部生产状态、Inspector Shell、Bottom Shot Strip。
- Phase 4 第一批：Scene、Shot、Character、Location、Prop、Acting、Prompt Pack 的事实源投影和不可变 FilmArtifact 版本。
- Phase 4 Generation Request：Prompt Pack 来源冻结、`generation_request` 版本 Artifact、SourceRefs、安全拒绝规则、画布节点/Inspector/自动布局，以及按精确 Artifact 版本读取的只读 Task Draft 合同、只读 Provider Route Catalog 和 revision-safe Canvas Projection Patch；尚未创建真实 Provider Task。
- Phase 3 自动测试：78/78 通过。
- Phase 4 Acting / Prompt Pack 自动测试：88/88 通过；typecheck、production build 和 `git diff --check` 通过。
- 本次新增的 Generation Request、Provider Route Catalog 与 Canvas Projection Patch 合同测试已写入工作区，但按当前 `AGENTS.md` 的默认验证纪律未运行；上述 88/88 与构建结果仅代表此前 Acting / Prompt Pack 验收，不能外推为本次验收。
- Ecommerce 文档合同：已同步完整 838 行架构审查记录；Provider-free Prototype A 合同、Provider Evaluation 记录合同、后端 `EcommerceArtifact` 持久化 API、项目类型入口、独立商品资产页和开发面板已完成当前原型边界，真实 Provider Bake-off、Agent Runtime 和完整 Ecommerce Canvas 仍未开始。
- 历史 TypeScript typecheck、production build 和三档真实浏览器验收结果见 `CONTINUE-HERE.md`；它们不替代本次 Generation Request 的待验证项。

详细状态和未完成项以 `CONTINUE-HERE.md` 为准。

## 5. 下一步范围（两条 Domain 隔离）

```text
Film Phase 4/5：Provider Gateway 路由解析与原子 Task 创建边界

Ecommerce Prototype：Product Upload → ProductDNA → CreativeDirection → Lifestyle Tabletop → CreativeShotPlan → Result Grid
```

Film 只做：

- 保持已完成的 Generation Request / Task Draft 合同、只读 Provider Route Catalog 和 revision-safe Canvas Projection Patch；下一步由 Provider Gateway 在同一事务内重新解析路由、校验能力/计费、创建 Task 并写入对应 Patch，成功后读取 Canvas 才会投影 `DomainRef.taskId`。
- 后端 Project / Scene / Shot / Asset / AssetVersion / FilmArtifact 是事实源；Canvas 仅做 Projection。

Ecommerce 只做：

- 在现有独立合同、后端持久化 API 和开发面板基础上，先使用 `web/src/ecommerce/evaluation/provider-evaluation.ts` 记录合同，接入用户明确提供且实际可调用的候选渠道，完成 P0.6 Provider Evaluation / Bake-off。
- 统一记录候选 Provider 的输入、成本、失败原因、结果引用和盲测数据；真实生成仍不得绕过 Provider Gateway。
- 只使用 `ProductIntelligenceAgent`、`CreativeDirectorAgent`、`SceneDirectorAgent` 这三个 Agent 合同。
- 只实现 `still-life.lifestyle-tabletop`；`fashion.natural-walk` 在 A 通过门禁后再做。
- 保持 ProductDNA、CreativeShotPlan、QAReport、Benchmark 的版本和证据边界。

当前禁止：

- Ecommerce Full V1、完整五项 Skill、Detail Page、卖点图、视频、UGC、Copywriting。
- 多 Provider 同时接入、绕过 Provider Gateway 或前端保存 API Key。
- 多人协作。
- 把规划文档中的未来目标误报为已经实现。

## 6. 本地运行说明

标准开发启动方式：

```bash
cd /Users/xiangyuqin/Downloads/infinite-canvas-short-drama
LOCAL_UID=$(id -u) LOCAL_GID=$(id -g) docker compose -f docker-compose.dev.yml up --build
```

标准地址：

```text
http://localhost:3000
```

注意：当前机器上 `/Users/xiangyuqin/Desktop/Infinite-Canvas-main` 曾占用 `3000` 端口。Phase 3 验收时，本仓库前端临时使用 `http://localhost:3001`。如果看到旧版侧栏或旧 Inspector，先检查端口对应的进程和工作目录，不要误判代码没有更新。

## 7. 如果还要更换 GitHub 账号

换 Codex 账号不要求换 GitHub 账号。只有明确要把 GitHub 登录也切换时才执行：

```bash
gh auth status
gh auth logout -h github.com
gh auth login -h github.com -p https -w
gh auth setup-git
gh auth status
```

如果新 GitHub 账号没有原仓库写权限，可选择：

1. 继续使用原 GitHub 账号推送；或
2. 给新账号添加仓库协作者权限；或
3. Fork 后把新仓库设置为新的 remote。

不要把 GitHub Token、API Key、Cookie 或登录码写入本文件、项目文件或 Git 提交。

## 8. 恢复检查

新会话开始时运行：

```bash
cd /Users/xiangyuqin/Downloads/infinite-canvas-short-drama
git status -sb
git branch --show-current
git log -8 --oneline
git remote -v
git rev-list --left-right --count HEAD...@{upstream}
```

正确状态应包括：

```text
codex/phase4-film-nodes
f4d7cf4 feat(film): add versioned production artifacts
0 0
```

最后一行 `0 0` 表示本地与 GitHub 对应分支完全同步。
