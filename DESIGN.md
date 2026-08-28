# gosoul 架构设计

gosoul 是一个 Majsoul(雀魂) 协议兼容的日麻私服。玩家使用**未被修改的官方客户端**，通过本地 MITM
代理接入；运营方通过 **GM 管理台**管理账号、AI 水平、匹配配置和自定义牌山。

## 总览

```
         玩家                         运营方
    ┌───────────┐               ┌───────────────┐
    │ 官方客户端  │               │ GM 管理台(Web) │
    │ (零改动)    │               │ Vue3+Element  │
    └─────┬─────┘               └───────┬───────┘
          │ 设为 HTTP 代理               │ /api/admin/*
          ▼                             ▼
    ┌─────────────────────────────────────────────┐
    │               gosoul gateway (MITM)          │
    │   TLS 终结(本机CA) · CONNECT · 域名路由      │
    │   *.maj-soul.com → 本地服务                  │
    │   其余流量 → 原样透传                        │
    └──────────────┬──────────────────────────────┘
                   │ liqi protobuf over WS
    ┌──────────────▼──────────────────────────────┐
    │                核心服务                      │
    │  lobby: 登录/房间/匹配    resource: 资源/CDN │
    │  game:  对局(引擎+AI)                       │
    └─────────────────────────────────────────────┘
```

## 分层与包

> 方向：技术分享向后端还原。GM 管理台/admin/web 已移除（见 doc/VISION.md）。

```
cmd/server          组合根(仅启动 gateway + 核心服务)
internal/
├── config           配置(YAML+env, GOSOUL_ 前缀)
├── protocol         协议层
│   ├── wire.go      5 层封包编解码(Frame)
│   ├── wrapper.go   wrapper protobuf + varint
│   ├── xor.go       Majsoul action XOR 编解码
│   ├── message.go   动态消息门面(反射字段访问)
│   └── registry.go  liqi.proto 运行时编译 + liqi.json 路由表
├── transport        核心服务 WS 传输与连接会话(尚未落地)
├── router           消息分发: 方法名 → handler(尚未落地)
├── gateway          MITM 代理（基于 elazarl/goproxy）
│   ├── ca.go        根 CA 生成/加载 + 注入 goproxy 全局签名
│   └── server.go    域名劫持路由 + clientgate 本地应答
├── lobby            大厅服务(登录/房间/匹配)  ← 移植自参考实现
├── game
│   ├── engine       纯领域引擎: 墙/摸打/役/符翻(无 I/O)
│   ├── yaku         役种检测(含数え役满/双立直边界)
│   ├── ai           Player 接口 + 内置档位 + mjai 适配器
│   └──              game server: 对局会话管理
├── match            匹配与对局生命周期(尚未落地)
├── deal             牌山构造工具集(引擎测试用, 无 UI)
└── storage          持久化接口 + SQLite(轻量, 牌谱为主)
```

## 核心领域设计

### 引擎纯净性

`engine` **不依赖** deal/ai/network。随机性与决策全部通过接口注入：

- 牌山: `engine.WallFactory.BuildWall(ctx, RoundMeta) → *Wall`
- AI: `ai.Player` 纯决策接口(ChooseDiscard/ChooseCall/ChooseSelfAction)
- 引擎只消费 `engine.Wall` 与 `ai.Player`，单测零依赖

### 牌山构造（引擎测试工具）

`deal` 提供随机墙与具名预设（指定手牌/指示牌/摸牌前缀），注入点仍是 `engine.WallFactory`。
用途限定为**引擎正确性验证**（如构造特定手牌测役种），不提供任何 UI 或运营能力。

### AI 扩展点（对齐 mjai 生态）

```
ai.Player (纯接口)
├── builtin.novice / normal / expert   内置强度档
├── mjai.Player                        子进程适配器(Mortal/Akochan/自研模型)
│   └── JSONL stdin/stdout, 15 种事件(tsumo/dahai/chi/pon/kakan...)
└── Register(name, Factory)            自定义模型在 init 时注册
GM 按座位选档: AIConfig.Seats = ["expert","mjai","normal","novice"]
```

### 协议热更新

`data/liqi.proto` + `data/liqi.json` 内嵌；更新后重新 `go build` 即换协议，无需改代码。

## 数据流（对局）

```
客户端 authGame → gateway(路由) → game server
game server: engine.StartRound(WallFactory) → ActionNewRound(24手牌? no 13)
            → 轮转: DealTile/DiscardTile/ChiPengGang/AnGangAddGang
            → 胡牌: 引擎役/符翻计算 → ActionHule
            → 分支: 流局/连庄/血战
AI 座位: engine 调 ai.Player 决策, 事件同步经 ActionPrototype 推送
```

## 落地顺序

1. ✅ 协议层(wire/xor/message/registry) + 单测(与 JS 参考逐字节对齐)
2. ✅ engine 墙抽象 + deal(Random/Preset) + 单测
3. ✅ ai 接口 + 内置三档 + mjai 草案
4. ✅ gateway MITM(CA/CONNECT/路由/透传)
5. ✅ admin GM API + ADR
6. ⏳ transport/router + lobby 移植（登录/房间/匹配）
7. ⏳ engine 完整规则移植（打牌/役/符翻/数え役满/双立直）
8. ⏳ game server 对局会话 + AI 接入
9. ⏳ storage(SQLite) 落库账号/牌谱/预设
10. ⏳ 前端 Vue3 控制台 + WS 实时
11. ⏳ 60 番场景预设 + 自定义牌山下发演示
12. ⏳ Tauri 桌面壳（代理+管理台一键启动）

## 关键参考

- 协议参考: Moli13337/MahjongSoul-4.0.26 (GPL-3.0, 仅作协议与行为参考)
- AI 标准: gimite/mjai, Cryolite/mjai, Equim-chan/Mortal
- 引擎移植源: 参考实现 mahjong-engine.ts