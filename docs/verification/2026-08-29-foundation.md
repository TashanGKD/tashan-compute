# Foundation Verification — 2026-08-29

## Outcome

Tashan Compute 的独立安全地基已在本地隔离分支实现并验证。该结论只覆盖账号、设备会话、组织成员、审计、Foundation API/CLI、公开 Skill 与分发门禁；不覆盖文件空间、对象存储、Executor、Docker 构建、长期服务或 AUP 生产部署。

## Verified commands

```text
GOPROXY=https://goproxy.cn,direct bash scripts/verify-foundation.sh
GOPROXY=https://goproxy.cn,direct go test -race ./internal/... ./tests/e2e
```

`verify-foundation.sh` 在真实 PostgreSQL 17.6 与 Redis 8.2.1 测试容器上完成：

- 全仓 Go 单元与集成测试；
- 双用户 HTTP E2E；
- `go vet`；
- 19 项 Capability/CLI/Skill 覆盖；
- Release 元数据一致性及其负向 self-test；
- 公开仓库 Secret 扫描及其负向 self-test；
- Skill 结构校验；
- 空用户安装、幂等升级、校验和/归档攻击和失败回滚；
- macOS arm64/x64 与 Linux x64 CLI 交叉构建。

Race detector 覆盖 `internal/...` 与双用户 E2E，未报告数据竞争。

## Security evidence

- 无参数 `tcompute`、`tcompute-api` 和安装器只显示帮助，不读取凭据、不连接数据库、不访问网络。
- 首位平台管理员只能通过服务器本地 `tcompute-admin bootstrap` 和直接 PostgreSQL 连接创建；该操作只能成功一次。
- Access Token 使用服务器 Ed25519 签名；自签、错误签名、过期、撤销设备、撤销会话和旧密码版本均被拒绝。
- 客户端角色字段不被接受；管理员与组织角色每次从 PostgreSQL 当前状态解析。
- 管理员重置密码和用户主动改密都撤销旧会话，但只有管理员重置会重新设置首次改密状态。
- Keychain/Secret Service 写入通过 stdin，密码与 Token 不进入 CLI argv 或正常输出。
- Secret 文件拒绝符号链接、弱权限、错误长度和超限内容。
- 公开安装失败不会覆盖用户自有命令，也不会破坏已有可用版本。

## Real lifecycle exercised

真实 E2E 执行了以下顺序：

1. 服务器本地创建首位平台管理员。
2. 管理员通过 API 创建 Alice 与 Bob。
3. 两名用户分别首次登录并强制修改密码。
4. 旧 Access Token 立即失效；新密码登录成功。
5. 管理员创建组织并把 Alice 添加为 developer。
6. 管理员重置 Bob 密码，Bob 的旧设备 Token 立即失效。

## Not yet complete

- 未创建 GitHub Release；公开安装器目前只有本地 Release fixture 验证。
- 未部署独立生产 API、PostgreSQL、Redis、MinIO、反向隧道或 HTTPS 域名。
- AUP Rootless Executor 上线前仍必须重新检查 `newuidmap/newgidmap`、cgroup 资源保留和网络拒绝策略。
- 个人空间、组织空间、文件同步、不可变快照、Secret 运行注入与所有计算对象尚未实现。
