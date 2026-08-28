# gosoul

Majsoul(雀魂) 服务端的完整平替实现，Go 从零还原。

- **用户面**：登录/注册、角色/装扮、货币、抽奖、成就、房间/匹配、对局、牌谱
- **GM 面**：账号管理、资源发放（API 先行，页面后置）
- **接入**：MITM 代理软件，官方客户端零改动即可连入

> 方向与边界见 [doc/VISION.md](doc/VISION.md)；开发方向与里程碑见 [doc/DEVELOPMENT.md](doc/DEVELOPMENT.md)；
> 技术决策见 [doc/ADR-0001.md](doc/ADR-0001.md)；架构全文见 [DESIGN.md](DESIGN.md)。

## 架构

```
玩家(官方客户端) ─► gosoul gateway ─► Lobby / Game / Resource
                     (MITM 代理)          │
                        │                 └► engine(纯规则) + ai + deal
                        └► 非游戏域透传
GM API ────────────────────────────────────► user 域(storage/SQLite)
```

| 层 | 包 | 说明 |
|---|---|---|
| 协议 | `internal/protocol` | liqi 五层封包 / XOR 动作层 / 动态消息 / 路由表，与参考实现逐字节对齐（单测） |
| 接入 | `internal/gateway` | 基于 `elazarl/goproxy` 的 MITM 代理，自持 CA，域名劫持 + 盲隧道 |
| 大厅 | `internal/lobby` | 登录/注册/建房/匹配（域服务，传输层映射中） |
| 对局 | `internal/game` | engine(纯领域) + yaku(役) + ai(可插拔/mjai) + deal(测试牌山) |
| 用户域 | `internal/user` | 账号/角色/货币域服务 |
| 存储 | `internal/storage` | SQLite + 版本化迁移，仓储接口隔离 |
| 管理面 | `internal/admin` | GM API（无 UI） |

## 快速开始

```bash
go build -o gosoul ./cmd/server

# 运行（YAML 配置，环境变量：GOSOUL_CONFIG / GOSOUL_DB ...）
./gosoul

# GM 管理面（建号/发资源/查号）
curl -X POST http://127.0.0.1:9090/api/admin/accounts -d '{"username":"u","password":"p"}' 

# 代理：客户端把 HTTP 代理/ PAC 指向 gateway 监听端口，信任 CA 后
# 游戏域名流量即被本地接管（其余流量不受影响）
```

## 当前状态

已落地：协议层（参考实现字节级对齐测试）、MITM 网关（劫持应答/盲隧道/反证单测）、
SQLite 存储（迁移/账号/钱包/角色）、用户域（注册登录/发资源）、GM API、**Lobby 登录注册闭环**
（WS transport + 路由分发 + 登录/自动注册/fetchInfo，端到端单测覆盖）。

推进中（按 [VISION.md](doc/VISION.md) 优先序）：
P0 核心闭环 → P1 成长系统 → P2 运营系统 → P3 外围。

## 技术栈

Go 1.26 · SQLite (modernc, 零 cgo) · protobuf dynamicpb · elazarl/goproxy · slog。
无 ORM，不手写第三方能力（协议 wire 部分除外，属必要自实现）。