# Tashan Compute 产品与技术设计

日期：2026-08-29

状态：用户已确认产品边界、账号模式、组织权限、技术架构、文件协作模型、Secret 和完整计算范围。

## 1. 一句话定义

Tashan Compute 是一个由 Skill 驱动、以 CLI 为主要入口、提供个人空间和组织共享空间的独立远程计算平台，使多名用户无需 Tailscale，即可安全地在 AUP 上协作管理文件、同步代码、运行任务并使用长期服务。

## 2. 独立项目边界

本项目使用独立本地目录 `/Users/boyuan/aiwork/tashan-compute` 和独立 GitHub 仓库 `TashanGKD/tashan-compute`。产品名为 `Tashan Compute`，CLI 命令为 `tcompute`，Codex Skill 名为 `tashan-compute`。

本项目必须具备独立的：

- 账号与设备会话；
- API、CLI、Skill、Executor 和 Gateway；
- PostgreSQL 数据库、Redis 命名空间和 MinIO/S3 对象存储命名空间；
- AUP 部署目录、Compose project、端口、服务账号、Secret、日志、备份和回滚记录；
- ECS HTTPS 入口与反向隧道。

旧 OrgSpace、TopicLab、实训营和其他他山项目只能作为研究资料。Tashan Compute 不导入它们的源码包，不共享数据库表、账号、Token、对象存储凭据、运行目录或部署生命周期。复制经过重新审查的通用思想是允许的，运行时耦合和跨项目依赖是不允许的。

## 3. v1 范围

### 3.1 必须完成

- 管理员创建账号，用户首次登录修改密码，多设备 Token 与设备撤销。
- 平台管理员创建组织，组织管理员管理已有用户的成员身份和角色。
- 每个用户一个个人空间，每个组织一个组织共享空间。
- 文件上传、下载、读取、列表、版本、搜索、同步、回收站和恢复。
- 文件版本冲突检测与不可变运行快照。
- 加密 Secret、授权、撤销和短期运行注入。
- Python、Node.js、常见编译型语言和 Docker/OCI 构建。
- `job`、`build`、`service`、`database`、`daemon` 五类运行对象。
- 日志流、状态、取消、重启、结果文件、构建产物和审计。
- 用户服务自动 HTTPS 域名、默认登录保护和显式匿名公开。
- 公网出站访问；私网、元数据、宿主机和平台控制面隔离。
- API、CLI、Skill 的全能力覆盖和 CI 强制一致性门禁。

### 3.2 不属于本项目

- OKR、待办、审批、报销、短信、聊天、会议和 AI 员工。
- 通用 Git 托管、分支审查和合并平台。用户可以在本地使用 Git，但 v1 服务端协作基于文件版本和运行快照。
- 向用户提供 AUP SSH、宿主机 Shell、Docker socket 或任意宿主机端口映射。

## 4. 用户入口

```text
用户自己的 Codex
  -> 安装 tashan-compute Skill
  -> Skill 检查或安装 tcompute 单文件 CLI
  -> 用户使用用户名和密码登录
  -> CLI 安全保存该设备的用户 Token
  -> 通过公网 HTTPS API 访问平台
  -> 选择个人空间或组织空间
  -> 管理文件、Secret、快照和运行对象
```

最终用户不安装 Tailscale，也不使用 SSH。Tailscale 或其他管理隧道只属于平台运维链路，不能出现在普通用户的使用说明中。

CLI 无参数只显示帮助，不读取凭据、不访问网络、不安装依赖、不执行动作。源码开发默认连接 loopback；正式 Release 明确连接生产 HTTPS 地址。

## 5. 身份与组织

### 5.1 账号

- 不开放自主注册。
- 只有平台管理员可以创建用户和初始密码。
- 初始密码首次登录后必须修改；修改完成前只能访问改密与退出接口。
- 忘记密码由平台管理员重置。重置成功后，该用户全部旧设备会话立即失效。
- 同一真实用户只有一个账号；多台机器通过不同设备 Token 区分。
- 密码只保存抗暴力破解的摘要；Token、密码和 Secret 不进入日志、审计正文或 CLI 输出。

### 5.2 组织

- 只有平台管理员可以创建组织。
- 组织管理员可以添加、移除已经存在的平台用户并设置角色。
- 一个用户可以属于多个组织，但每个请求必须明确空间，不能根据最近使用状态隐式选择组织。

### 5.3 角色

- `platform_admin`：创建/停用账号、重置密码、创建组织、授予平台资源预算和执行 break-glass 运维。
- `org_admin`：管理本组织成员、空间配额范围内的策略、目录 ACL、Secret 授权和全部组织运行对象。
- `developer`：默认可读写组织空间、上传代码、创建快照、提交计算并查看组织任务与结果。
- `viewer`：只读文件、日志和结果，不能修改空间或提交运行。

组织管理员可以建立受限目录，只授权指定成员或角色。管理员读取受限内容和使用 break-glass 都必须产生高风险审计。

## 6. 空间、配额和文件

### 6.1 空间

- 每个用户自动拥有一个个人空间，仅本人可访问。
- 每个组织自动拥有一个组织空间，按成员角色和目录 ACL 授权。
- 运行对象、数据库卷、日志、结果和产物都必须归属一个空间。
- 个人空间和组织空间在元数据、对象前缀、加密域、运行挂载和配额账本上隔离。

初始配额采用已确认的产品默认值：个人空间 50 GiB，可由平台管理员提高但不超过 500 GiB；组织空间 500 GiB。配额变更不自动删除数据；用量高于新上限时空间转为只读，直至用量降至上限内。

### 6.2 存储模型

PostgreSQL 保存 `Space`、`FileEntry`、`FileVersion`、`UploadSession`、`StorageReservation`、`TrashEntry` 和授权元数据。文件内容、代码快照、构建产物和计算结果进入独立 MinIO/S3 命名空间。

路径是用户界面，不是授权边界。服务端基于不可猜测对象 ID、`space_id` 和调用者权限授权；用户字符串不得直接拼成宿主机路径或对象键。

上传采用分片、断点续传、客户端与服务端校验和。上传开始前事务性预留配额，完成后转为实际占用；中断、超时、校验失败或事务失败必须释放预留和临时分片。回收站内容在永久清理前继续占用配额。

### 6.3 多人同步与版本冲突

`tcompute file sync` 默认只输出 dry-run 计划。实际同步时，客户端提交本地 SHA-256、远端基线版本和操作清单。若远端文件已被其他成员修改，服务端拒绝覆盖并返回机器可读冲突清单。

同名文件不得静默覆盖。用户必须显式选择创建新版本、解决冲突或覆盖；覆盖仍保留可恢复版本。删除默认进入回收站，永久删除需要显式确认和审计。

## 7. 不可变运行快照

每次运行以一个不可变 `WorkspaceSnapshot` 为输入，记录：

- 空间 ID、文件版本清单和根摘要；
- 创建用户、设备、时间和来源同步操作；
- 基础镜像或 Dockerfile 内容摘要；
- Secret 引用名称和版本，不记录明文；
- 资源策略和运行命令的结构化参数。

任务开始后，空间中的后续修改不能改变该任务输入。重跑默认复用原快照和配置；若使用当前文件，必须创建新快照和新任务。日志、结果和产物反向关联任务与快照，以支持复现。

## 8. Secret

- `tcompute secret set` 通过隐藏输入或 stdin 接收 Secret，禁止命令行参数明文和终端回显。
- Secret 使用独立密钥域加密保存，创建后不提供读取明文接口。
- 个人 Secret 只能由本人授权；组织 Secret 可授权给成员、角色或特定工作负载。
- Executor 只在任务启动时按短期租约注入，任务结束、失败或取消后撤销并清理。
- Secret 不写入空间、快照正文、镜像、构建缓存、日志、错误信息或审计正文。
- 创建、更新、授权、使用、撤销和失败清理均记录掩码审计。

## 9. 计算模型

统一对象 `RuntimeWorkload` 支持：

- `job`：短任务、批处理和可取消的计算；
- `build`：使用 rootless BuildKit 构建 OCI 镜像；
- `service`：长期 Web 服务；
- `database`：带持久数据卷的数据库；
- `daemon`：Agent 或其他后台常驻进程。

每个对象包含空间、快照、结构化 argv、工作目录、镜像/构建来源、Secret 引用、资源策略、期望状态、实际状态、重启策略、日志游标和产物清单。

API 不执行用户代码。API 完成认证、授权、配额预留和任务持久化后写入可靠队列；AUP 上独立的 Rootless Executor 领取任务，创建隔离环境并回报状态。外部服务超时后，控制面必须先查询真实运行状态，再决定重试或补偿，避免重复任务。

## 10. 隔离与资源

工作负载启动时必须显式指定一个空间，只挂载该空间的授权快照、输出目录和受控持久卷。禁止挂载其他空间、宿主机用户目录、平台目录、Docker socket、容器运行时 API 或 Kubernetes 控制面。

网络允许 DNS、HTTP(S) 和用户明确需要的公网出站；在网络层拒绝：

- IPv4/IPv6 loopback、RFC1918、链路本地和 IPv6 ULA；
- AUP 管理网、容器桥接网、其他租户网段和平台控制面；
- 云元数据、Docker API 和容器编排控制 API；
- DNS 首次解析为公网但连接时重绑定到私网的目标。

默认资源模式为 `capped`，限制 CPU、内存、进程数、临时盘、并发和运行时长。`elastic` 只能由平台管理员授予，可使用用户池空闲资源，但不能侵占平台保留和应急保留。资源隔离必须由 cgroup v2、cpuset、pids、磁盘额度和 rootless runtime 机器执行，不能只保存数据库字段。

在 rootless 容器、BuildKit、网络拒绝和资源保留的实机负例验证通过前，生产环境不得接受用户计算。

## 11. 服务、数据库与 daemon

用户服务通过平台 Gateway 暴露，不允许直接映射宿主机公网端口。服务获得受控的 `*.tashan.chat` HTTPS 域名，默认 `private`：域名公网可达，但访问者必须登录且拥有该空间权限。

`tcompute service public` 必须完整提示“互联网上任何匿名用户都将可以访问此服务”，要求确认并产生高风险审计。`service private` 可恢复登录保护。

数据库默认只接受同一空间授权工作负载的内部连接。用户交互访问通过 `tcompute db connect` 建立带过期时间、账号授权和审计的临时代理，不直接开放数据库公网端口。

`service`、`database` 和 `daemon` 使用期望状态调和。API、Executor、Gateway 或 AUP 重启后，平台根据持久状态恢复应该运行的对象，同时以幂等键避免重复创建实例和数据卷。

## 12. 技术架构

主要实现语言为 Go：

- `cmd/tcompute`：单文件跨平台 CLI；
- `cmd/api`：认证、组织、空间、文件、Secret、任务和审计控制面；
- `cmd/executor`：AUP rootless 运行时与资源执行；
- `cmd/gateway`：动态服务入口、登录保护与代理；
- `internal/...`：按领域拆分的业务和基础设施实现；
- `skill/tashan-compute`：Codex Skill 与固定版本安装器。

PostgreSQL 是账号、权限、配额、任务和审计真相；MinIO/S3 保存文件与产物；Redis 用于可靠队列协调、短期租约和实时日志通知，但不能成为不可恢复的唯一真相。

生产组件部署在 AUP 的独立目录和隔离运行环境。公网用户经 ECS TLS 入口和平台专用持久反向隧道访问 AUP API/Gateway。用户数据链路不依赖 Tailscale。

## 13. CLI 命令面

```text
tcompute auth ...
tcompute device ...
tcompute admin user create|reset-password|disable ...
tcompute org create|list|member ...
tcompute space list|show|quota ...
tcompute file ls|read|upload|download|sync|remove|restore|versions ...
tcompute snapshot create|list|show ...
tcompute secret set|list|grant|revoke ...
tcompute run submit|list|status|logs|cancel|rerun ...
tcompute build submit|status|logs ...
tcompute service deploy|status|logs|restart|private|public ...
tcompute db create|status|exec|connect|backup|restore ...
tcompute daemon deploy|status|logs|restart ...
tcompute audit list ...
tcompute capability list|describe --json
```

所有空间命令必须显式提供 `--space`。组织管理命令必须显式提供 `--org`。机器调用提供稳定 JSON、标准错误码、stdout/stderr 分离和日志流游标。写操作支持幂等键；非交互高风险操作同时要求明确确认参数和幂等键，确认参数不能绕过权限。

## 14. Capability 与 Skill

服务端每个用户可调用能力必须注册唯一 capability，并声明版本、输入/输出 schema、权限、对象作用域、副作用、幂等性、确认级别、CLI 绑定、JSON/流式输出、错误码和审计动作。

CI 必须机器比较服务端 capability、CLI command binding 和 Skill 引用。任一服务端能力缺少 CLI 绑定或 Skill 引用了不存在的能力时，构建失败。每个一致性 gate 必须有同名负向 self-test，实际构造缺失绑定、错误 schema 和错误 capability ID，证明门禁能够阻断漂移。

Skill 负责检查和安装固定版本的单文件 CLI、读取 CLI 帮助和 capability、引导隐藏输入及执行命令。Skill 不复制业务逻辑，不读取或转述 Token、密码和 Secret，也不成为第二份 API 真源。

## 15. 必须先编码的拒绝测试

在实现正常路径前，至少覆盖：

- `../`、绝对路径、编码穿越、Unicode 混淆、符号链接、硬链接、前缀碰撞和挂载逃逸；
- 猜测对象 ID、跨空间文件版本、跨组织快照、被移除成员继续读取和 viewer 提交运行；
- 上传并发超卖、分片中断、校验失败、完成事务失败后的配额与临时对象清理；
- argv/环境变量注入、恶意 Dockerfile、Secret 日志泄漏和构建缓存泄漏；
- 访问 loopback、私网、云元数据、Docker socket、平台 API 和其他空间服务；
- 伪造 Host、保留 slug、跨服务 upstream、数据库公网端口和 DNS 重绑定；
- 任务取消、Executor 崩溃、AUP 重启、队列重复投递和状态回报丢失后的幂等恢复；
- 密码重置后的旧设备 Token、被撤销设备和首次登录未改密账号调用业务接口。

任何失败路径中创建的配额预留、临时分片、运行租约、Secret 注入、临时数据库代理和动态路由都必须有清理或补偿测试。

## 16. 交付顺序

虽然目标是完整闭环，实施仍按可验证的纵向切片推进：

1. 独立仓库、契约、CI、安全账号、设备、组织和公网控制面。
2. 个人/组织空间、配额、MinIO、文件版本、同步冲突和回收站。
3. 不可变快照、Secret、可靠任务队列和本地假 Executor。
4. AUP Rootless Executor、Python/Node/编译任务、日志与结果。
5. Rootless BuildKit、镜像产物与 Docker 构建。
6. 长期 service/database/daemon、调和恢复和临时数据库代理。
7. 动态 HTTPS Gateway、默认私密和显式 public。
8. Skill/CLI Release、新用户安装和双用户生产验收。

每个切片必须同时交付 API、CLI、Skill 引用、审计、拒绝测试、故障恢复测试和 capability gate；不允许先实现只有后端可用的隐藏能力。

## 17. v1 验收

至少由两名真实用户、两台未安装 Node.js、Go、pnpm 或 Tailscale 的电脑完成：

1. 平台管理员创建用户和组织；用户首次登录修改密码并获得独立设备 Token。
2. 两名用户进入同一组织空间，同步同一项目；并发修改产生明确冲突而非静默覆盖。
3. 从固定快照分别运行 Python、Node.js、编译型项目和 Docker 构建，并获得可复现的日志、退出状态与结果文件。
4. 部署默认私密的 Web 服务、数据库和 daemon；验证登录保护、临时数据库代理和重启恢复。
5. 显式执行 `service public` 后匿名访问成功，再恢复 private 并立即阻断匿名访问。
6. 容器公网出站成功，但访问个人空间、另一个组织、AUP 控制面、私网、元数据和宿主机服务均失败。
7. 管理员重置一个用户密码后，该用户所有旧设备 Token 立即失效。
8. 从公开或经授权可访问的 GitHub Skill 路径一键安装 Skill，再由 Skill 安装 CLI 并完成上述操作。
9. 删除任一 capability 的 CLI 绑定后，CI 负向自测稳定失败。
10. API、Redis、Executor、Gateway 和 AUP 依次重启后，任务和长期对象正确调和，且不重复执行不可重入副作用。

只有全部验收通过，才可以声称“个人空间、组织空间和计算已形成多人协作闭环”。
