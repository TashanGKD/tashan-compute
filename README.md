# Tashan Compute

> 通过 Skill 与 CLI 使用个人空间、组织共享空间和隔离计算资源的独立远程计算平台。

## 当前状态

项目处于设计完成、等待实施计划的阶段。仓库尚无可运行服务或可发布 CLI；任何安装、运行或部署命令都必须在对应实现和验证完成后才加入本 README。

## 公开仓库与访问权限

本仓库公开是为了让任何用户和 AI Agent 都能审查源码、安装 `tashan-compute` Skill，并由 Skill 安装 `tcompute` CLI。公开源码不授予任何平台账号、管理员角色或 AUP 访问权。

- 没有平台管理员创建的用户名和初始密码，CLI 不能登录或调用受保护能力。
- 管理员角色只由服务器数据库中的授权记录决定；本地参数、环境变量、修改 CLI 源码或自行签发 Token 都不能产生管理员身份。
- 仓库不保存账号、初始密码、Token、签名密钥、数据库/对象存储凭据、AUP SSH 信息或生产部署密钥。
- AUP 不向最终用户开放 SSH。用户只能通过公开 HTTPS API 和服务端验证的设备 Token 操作被授权空间。

安全边界与漏洞报告方式见 [`SECURITY.md`](SECURITY.md)。

## 快速开始

当前尚未发布可安装版本。首个 Release 完成后，本节只提供两步：从本公开仓库安装 Skill，再由 Skill 安装固定版本且经过校验和验证的 CLI。安装本身不会登录服务器，也不会创建账号。

## 常用命令

当前没有已发布命令。设计中的命令面以 [`docs/superpowers/specs/2026-08-29-tashan-compute-design.md`](docs/superpowers/specs/2026-08-29-tashan-compute-design.md) 为准，只有实际实现并通过测试的命令才会进入本表。

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

## 环境变量

阅读仓库和未来安装客户端不需要环境变量。服务器开发配置将在 `.env.example` 中只列变量名和安全说明；真实值不得提交。CLI 登录凭据保存在操作系统安全存储中，不通过仓库或 `.env` 分发。

## 部署边界

生产部署必须使用独立目录、端口、数据库、对象存储命名空间、容器、域名入口、Secret 和回滚记录，不影响任何既有他山项目。
