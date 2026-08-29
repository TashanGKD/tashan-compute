# Tashan Compute

独立的多人远程计算平台：用户从公开 Skill 安装 `tcompute`，用管理员创建的账号登录，在 AUP 上创建个人空间或共享空间、同步文件、获得完整隔离 Linux Shell、运行语言工具链/数据库/daemon，并按需开放 HTTPS 服务。

## 已部署入口

- 控制面与登录：<https://compute.tashan.chat>
- Workspace HTTPS：`*.workspaces.compute.tashan.chat`
- Coder OSS：v2.35.6
- Incus：6.0 LTS，非特权 user namespace

普通用户不需要也不会获得 Tailscale、AUP SSH、Incus socket、宿主 Docker socket或宿主路径。公开仓库本身不授予账号或管理员身份。

## 从 Skill 安装

把仓库中的 `skill/tashan-compute` 安装为 Codex Skill，然后由 Skill 执行：

```bash
bash scripts/install-cli.sh --check
bash scripts/install-cli.sh --install
tcompute login --email you@tashan.chat
```

安装器默认不做任何修改；`--install` 下载带 SHA256 的 v0.2.0 包。每个平台包同时含 `tcompute` 与固定 Coder CLI，不要求用户安装 Go、Node、Docker 或 Tailscale。账号只能由平台管理员创建。

## CLI 闭环

```bash
# 个人空间
tcompute personal create my-space
tcompute workspace list
tcompute shell my-space

# 组织共享空间
tcompute org create shared-lab --admin alice   # 仅平台管理员
tcompute org member add shared-lab bob
tcompute org list
tcompute shell alice/shared-lab

# 文件：默认 dry-run，确认后才传输
tcompute sync push shared-lab ./code project
tcompute sync push shared-lab ./code project --apply
tcompute sync pull shared-lab ./results project/results --apply

# 计算容器内
tcompute shell shared-lab -- python3 script.py
tcompute shell shared-lab -- docker build -t experiment .

# HTTPS 服务，默认需要 owner 登录
tcompute service private shared-lab
tcompute service authenticated shared-lab
tcompute service public shared-lab   # 明确开放匿名互联网访问
```

持久文件位于 `/home/coder`：个人空间默认 50 GiB，组织空间 500 GiB。只有平台管理员可创建组织空间并指定首位组织管理员；计算容器停止时可销毁并快速重建，但 home volume 保留。预装 Python、Node.js、Go、Rust、C/C++、PostgreSQL、Redis、Podman/Buildah，并提供 Docker-compatible `docker build`。

把可执行启动器放在 `/home/coder/.tcompute/service`，常驻进程会随工作空间启动恢复；端口 8000 自动获得 HTTPS workspace hostname。成员移除后 CLI 自动重启空间，使撤权立即生效。

## 安全与资源边界

- Workspace root 映射为宿主非 root UID。
- 标准限制：4 CPU、8 GiB RAM、2048 pids、8 GiB ephemeral root；个人 persistent home 50 GiB、组织 persistent home 500 GiB。
- 用户总池上限：30 CPU、48 GiB RAM、700 GiB；宿主保留平台与应急资源。
- 拒绝宿主、RFC1918/CGNAT、链路本地和云元数据访问；允许公网出站。
- HTTPS 服务默认 owner 登录；`service public` 才允许匿名访问。
- `sync` 默认 dry-run且无 `--delete`；`workspace delete` 必须显式 `--yes`。

## 开发与验证

```bash
go test ./...
bash scripts/check-capability-coverage.self-test.sh
bash scripts/check-release-contract.self-test.sh
bash deploy/check-coder-stack.self-test.sh
bash deploy/configure-incus.self-test.sh
bash deploy/verify-incus-isolation.self-test.sh
bash deploy/check-public-network.self-test.sh
bash tests/distribution/build-cli-release.sh
bash tests/distribution/install-cli.sh
```

架构与部署计划见 [`docs/superpowers/specs/2026-08-29-coder-incus-design.md`](docs/superpowers/specs/2026-08-29-coder-incus-design.md) 和 [`docs/superpowers/plans/2026-08-29-coder-incus-deployment.md`](docs/superpowers/plans/2026-08-29-coder-incus-deployment.md)。安全报告方式见 [`SECURITY.md`](SECURITY.md)。
