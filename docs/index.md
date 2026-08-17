# 文档索引

本目录是当前仓库面向 AI 协作者和开发者的短索引。代码、数据库迁移和已提交的测试才是实现事实；规划文档、测试合同和历史验收记录不能互相替代。

## 接手顺序

1. [AGENTS.md](../AGENTS.md)：仓库边界、接口、安全、UI、验证和提交规则。
2. [CONTINUE-HERE.md](../CONTINUE-HERE.md)：当前分支的 Film / Ecommerce 状态与下一步顺序。
3. [功能说明](content/docs/features.mdx)：面向产品的当前能力边界。
4. [代码地图](content/docs/code-map.mdx)：前后端、画布、Film 与 Ecommerce 的模块职责。
5. [待办](content/docs/todo.mdx) 和 [验收记录](content/docs/pending-test.mdx)：未完成事项与尚待复核的实现。

## 领域合同

- [Film 生成请求与执行事实链](production-canvas/film-generation-request-contract.md)
- [Film 场景空间连续性合同](production-canvas/scene-spatial-contract.md)
- [Ecommerce Prototype A 附录](production-canvas/ecommerce-prototype-addendum.md)
- [Ecommerce Prototype 实施规格](production-canvas/ecommerce-prototype-implementation-spec-v1.0.md)
- [Ecommerce Agent Team 架构基线](production-canvas/ecommerce-agent-team-v1.2.1-codex-ready.md)
- [后端数据库说明](content/docs/backend/backend-database.mdx)
- [UI 设计系统规范](ui-design-system.md)

## 事实边界

- Film 的图片生成事实链已有历史真实验收记录；当前工作树的空间系统、视频结果投影和渠道目录同步仍须按验收记录复核。视频 Provider 尚未验收。
- Ecommerce 已实现 Provider-free Prototype A 的数据合同、持久化和开发面板；它不是完整 Ecommerce V1，也没有真实 Provider、媒体生成、Agent Runtime 或自动视觉 QA。
- API Key、Cookie、测试账号、真实任务标识、媒体链接和本机绝对路径不得写入文档或提交记录。
