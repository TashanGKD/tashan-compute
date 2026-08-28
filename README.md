# Tashan Compute

> 通过 Skill 与 CLI 使用个人空间、组织共享空间和隔离计算资源的独立远程计算平台。

## 当前状态

安全地基已经实现：管理员托管账号、首次改密、多设备会话、组织成员角色、审计、API、CLI 命令树、公开 Skill、固定版本安装器和 CI 门禁均已有可运行代码与测试。个人/组织文件空间和计算执行器尚未实现；当前版本不得宣称完整产品可用。

## 公开仓库与访问权限

本仓库公开是为了让任何用户和 AI Agent 都能审查源码、安装 `tashan-compute` Skill，并由 Skill 安装 `tcompute` CLI。公开源码不授予任何平台账号、管理员角色或 AUP 访问权。

- 没有平台管理员创建的用户名和初始密码，CLI 不能登录或调用受保护能力。
- 管理员角色只由服务器数据库中的授权记录决定；本地参数、环境变量、修改 CLI 源码或自行签发 Token 都不能产生管理员身份。
- 仓库不保存账号、初始密码、Token、签名密钥、数据库/对象存储凭据、AUP SSH 信息或生产部署密钥。
- AUP 不向最终用户开放 SSH。用户只能通过公开 HTTPS API 和服务端验证的设备 Token 操作被授权空间。

安全边界与漏洞报告方式见 [`SECURITY.md`](SECURITY.md)。

## 快速开始

### 前置条件

- Go 1.26
- Docker 29+ 与 Docker Compose 5+

### 本地验证

```bash
docker compose -f compose.test.yml up -d postgres redis
bash scripts/verify-foundation.sh
docker compose -f compose.test.yml down --volumes
```

当前尚未发布 GitHub CLI Release。仓库中的安装器只用于本地分发测试；在 Release 资产和生产健康门禁完成前，不应引导外部用户执行安装。

## 常用命令

| 命令 | 用途 |
| --- | --- |
| `bash scripts/verify-foundation.sh` | 运行单元、集成、E2E、分发和安全门禁 |
| `go run ./cmd/tcompute` | 仅显示本地 CLI 帮助，不联网 |
| `go run ./cmd/tcompute-api` | 仅显示 API 服务帮助，不加载配置 |
| `go run ./cmd/tcompute-api serve` | 显式启动已配置的本地 API |
| `go run ./cmd/tcompute-admin bootstrap --username <name>` | 在服务器本地直连数据库创建首位管理员 |
| `bash tests/distribution/build-cli-release.sh` | 验证三平台 CLI Release 构建 |
| `bash tests/distribution/install-cli.sh` | 验证空用户安装、升级和失败回滚 |

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
    plans/       分阶段实施计划
cmd/             CLI、API 和服务器本地管理员入口
internal/        认证、授权、存储、HTTP、CLI 与应用服务
migrations/      带 SHA-256 防漂移的 PostgreSQL 迁移
capabilities/    API/CLI/Skill 能力真源
skill/           可公开安装的 Codex Skill
scripts/         验证、构建与一致性门禁
tests/           双用户 E2E 与分发测试
```

代码目录、常用命令、本地启动方式和环境变量会由实施计划确定，并与首个可运行纵向切片同时加入，避免在实现前声明不存在的接口或命令。

## 关键文档

| 文档 | 用途 |
| --- | --- |
| [`docs/superpowers/specs/2026-08-29-tashan-compute-design.md`](docs/superpowers/specs/2026-08-29-tashan-compute-design.md) | 已确认的独立产品与技术设计 |
| [`docs/superpowers/plans/2026-08-29-foundation-auth-distribution.md`](docs/superpowers/plans/2026-08-29-foundation-auth-distribution.md) | 安全地基、认证和公开分发实施计划 |
| [`docs/verification/2026-08-29-foundation.md`](docs/verification/2026-08-29-foundation.md) | 当前验证证据与未完成边界 |

## 环境变量

阅读仓库和未来安装客户端不需要环境变量。`tcompute-api serve` 使用 `.env.example` 中列出的变量，包括 PostgreSQL、Redis、Ed25519 密钥文件、Refresh Pepper、可信代理和 CORS 来源。真实值不得提交；三个 Secret 文件必须是非符号链接、权限不超过 `0600`，内容为固定长度随机字节的 Base64。

## 部署边界

生产部署必须使用独立目录、端口、数据库、对象存储命名空间、容器、域名入口、Secret 和回滚记录，不影响任何既有他山项目。

当前尚未部署到 AUP，也未配置生产域名、GitHub Release 或真实用户账号。
