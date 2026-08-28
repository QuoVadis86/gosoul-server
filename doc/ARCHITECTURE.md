# 架构约定 (ARCHITECTURE)

> 权威分层规则。新增/重构代码必须先对齐本文；与既有实现冲突时以本文为准并修正实现。

## 分层总览

```
transport ──► router ──► feature handler ──► feature service ──► domain service ──► storage(ports)
(WS 帧)            (分发)      (RPC 适配)         (面服务)            (领域)              (仓储接口)
```

单向依赖，禁止反向与跨层跳用。

## Feature 包分两型

统一按 feature 分包（lobby / user / admin / game / deal / gateway …）。
每包内按职责拆文件，**文件名即约定**：

| 文件 | 职责 | 是否必有 |
|---|---|---|
| `dto.go` | 对外协议/API 传输对象（json tag 对齐 proto 字段名） | **仅 surface 包** |
| `handler.go` | 协议适配：解析请求 → 调域 → 组装 DTO 响应 | **仅 surface 包** |
| `entity.go` / `ports.go` / `service.go` | 领域实体 / 仓储端口 / 领域服务：纯 Go、协议无关 | 域包 |
| `domain.go`（如 engine/墙） | 纯规则类型与接口 | 按需 |

### Surface 包（直接暴露 RPC / HTTP 面）
- `lobby`（Majsoul RPC 面）、`admin`（GM HTTP 面）、`game`（对局 RPC 面）
- 只做协议适配：DTO 定义 + handler 映射，**业务逻辑归域**，handler 不写业务。

### Domain 包（被 Surface 调用）
- `user`（账号/角色/货币域）：`entity.go`(实体) + `ports.go`(仓储接口) + `service.go`(领域服务 + 读模型 Home)
- `game/engine`（纯规则）、`deal`（牌山）、`storage`（持久化**实现**，依赖域端口）
- 域包**没有**协议 DTO/handler——协议适配由调用它的 surface 包持有。
  不为凑三层给无协议面的包造假层（抽象滥用）。
- 实体与错误契约归域：`user.Account/Character/Wallet`、`user.ErrNotFound`；
  storage 只做实现并返回域类型 → **持久化包类型不泄漏到面层**。

## 依赖规则

1. 面层 handler → 面 service → 域 service → 仓储接口；禁止跳过中间层。
2. 域层不得 import transport/router/protocol（协议无关）。
3. 仓储接口定义在 `storage`（ports），实现（SQL）同包；调用方只依赖接口。
4. `protocol/names.go` 是协议名字符串的**唯一来源**：方法名、消息类型名常量。
5. 新增对外 RPC 的步骤：
   1. 检查 `protocol/names.go` 补方法/类型常量
   2. `dto.go` 加请求/响应 DTO
   3. `handler.go` 注册 `r.Handle(MethodX, h.xxx)`
   4. service 层补能力；若不需要业务，交由 surface 层空响应机制兜底

## 交互时序（对局面）

```
transport(session) ── Dispatch ──► lobby.handler ──► lobby.service ──► user.service ──► storage
                                     │ (room/match)                       ▲
                                     └────────► game.service(会话) ───────┘
                                                     │
                                                     ▼
                                              engine(纯规则) + ai.Player + deal.WallFactory
```

对局会话（game.service）持 engine 实例；engine 无 I/O，经 WallFactory/Player 注入随机与决策。

## 现状对照（对齐中）

- lobby: ✅ 纯 surface（dto + handler + surface 空响应）；业务经 user 域
- admin: ✅ 薄 HTTP 适配，域逻辑在 user
- user: ✅ 域完整（entity/ports/service + Home 读模型），storage 依赖域
- game: ⚠️ 传输/会话层未落地，engine/ai 骨架已建
- protocol: ✅ registry + names.go 常量中心；surface 骨架消息自举