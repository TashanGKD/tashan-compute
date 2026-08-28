# Tashan Compute

> 通过 Skill 与 CLI 使用个人空间、组织共享空间和隔离计算资源的独立远程计算平台。

## 当前状态

项目处于设计完成、等待实施计划的阶段。仓库尚无可运行服务或可发布 CLI；任何安装、运行或部署命令都必须在对应实现和验证完成后才加入本 README。

## 产品边界

- 用户通过 `tashan-compute` Skill 安装并调用单文件 `tcompute` CLI。
- 用户无需 Tailscale，通过公网 HTTPS API 登录并操作 AUP 上被授权的空间和计算资源。
- 平台支持个人空间、组织空间、版本化文件、不可变运行快照、Secret、批处理、构建、Web 服务、数据库和 daemon。
- 本项目不包含 OKR、待办、审批、短信、聊天、会议或 AI 员工。
- 本项目不依赖旧 OrgSpace 的源码包、数据库、账号、API、部署或运行状态。

## 目录结构

```text
docs/
  superpowers/
    specs/       已确认的产品与技术规格
```

代码目录、常用命令、本地启动方式和环境变量会由实施计划确定，并与首个可运行纵向切片同时加入，避免在实现前声明不存在的接口或命令。

## 关键文档

| 文档 | 用途 |
| --- | --- |
| [`docs/superpowers/specs/2026-08-29-tashan-compute-design.md`](docs/superpowers/specs/2026-08-29-tashan-compute-design.md) | 已确认的独立产品与技术设计 |

## 部署边界

生产部署必须使用独立目录、端口、数据库、对象存储命名空间、容器、域名入口、Secret 和回滚记录，不影响任何既有他山项目。
