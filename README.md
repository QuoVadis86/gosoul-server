# gosoul

**A self-hosted Mahjong Soul (雀魂) private server written in Go.**

实现 Majsoul 客户端（Web/Steam）可直接接入的协议兼容后端：登录注册、角色/货币、
房间匹配、日麻对局引擎、GM 管理面，全程 Go 实现、零 cgo、单二进制。

```
官方客户端 ──► gosoul gateway(MITM 代理) ──► lobby / game / admin
                  (域名劫持 + TLS 终结)         │
                                               └► engine(纯规则) + ai + storage(SQLite)
```

## Features

- **用户面**：登录/注册（自动注册 + bcrypt）、钱包、角色、成就/抽奖/商店（API 面已就绪）
- **对局面**：日麻完整规则引擎（全役种 + 双立直 + 数え役满，规划中）、AI 可插拔（内置档位 + mjai 协议接入）
- **接入**：MITM 代理（官方客户端零改动）、牌山可注入（引擎测试）
- **GM 面**：账号/资源/角色发放（API 先行）
- **工程**：纯领域引擎 + 接口注入、SQLite 零 cgo、协议类型自举热更新

## Quick start

```bash
go build -o gosoul ./cmd/server
./gosoul            # gateway:8080 / lobby:8441 / admin:9090，DB 自动迁移

# 接入客户端：系统代理指向 gateway，信任其签发的 CA 后，游戏域名流量即被接管
```

环境变量：`GOSOUL_CONFIG`（YAML 配置路径）、`GOSOUL_DB`（数据库文件路径）。

## 目录结构

```
cmd/server          组合根
internal/
├── protocol        五层封包 / XOR / 动态消息 / 方法·类型常量中心 / 路由表自举
├── gateway         基于 elazarl/goproxy 的 MITM 代理（自持 CA）
├── transport       WebSocket 会话 + 帧分发
├── router          方法 → handler 分发
├── lobby           客户端 RPC 面（dto/handler/surface 空响应）
├── user            账号/钱包/角色域（entity + ports + service + Home 读模型）
├── admin           GM HTTP 面
├── game            engine(纯规则) / ai(Player 接口) / deal(牌山)
└── storage         SQLite 持久化（实现域端口）
```

架构与分层约定：[ARCHITECTURE](doc/ARCHITECTURE.md) · 编码规范：[CODING-STYLE](doc/CODING-STYLE.md)
方向与里程碑：[VISION](doc/VISION.md)

## Status

- ✅ 协议层（参考字节级对齐单测）、MITM 网关（端到端单测）
- ✅ 登录注册闭环、全 API 面 439 个 RPC 正确类型空响应、GM 账号/发资源
- ⏳ 房间/匹配 → 对局引擎移植（最大块）→ 成长/运营系统 → 牌谱

## 必要申明

- 本项目是**非官方**实现，与 Majsoul 官方及其开发/发行方没有任何关联。
- 项目**不包含任何官方素材、资源文件或客户端数据**；仅实现网络协议与后端逻辑。
- 游戏名称、商标与美术内容版权归其各自权利人所有。
- 本项目**仅供学习与研究**用途。请勿用于商业运营或牟利；使用者对部署与合规自行负责。
- License: [GPL-3.0](LICENSE)。
