# 编码规范 (CODING-STYLE)

## 常量优先
- 魔法字符串/数字一律常量；**协议方法/类型/notify/action 名只在 `internal/protocol/names.go`**。
- 面层/域层不得散落 `".lq.xxx"`、`"lq.ResXxx"` 字面量（测试除外）。
- 默认值、端口、ID（角色/道具/avatar）用包级常量；允许全局配置覆盖的不作为常量。

## 分层纪律（见 ARCHITECTURE.md）
- 依赖单向：surface → domain → storage(ports)；域层禁止 import transport/router/protocol。
- surface 包：dto/handler 只做适配；业务一律进域。
- 域包：实体/错误契约/端口归域；持久化实现返回域类型，禁止存储类型泄漏。
- 新增 RPC 步骤：names.go 常量 → dto → handler 注册 → 域能力；无业务先由 surface 兜底。

## 命名
- 包名全小写单数；handler 方法以动词开头（login/fetchInfo/...）。
- 常量名表达意图：MethodXxx/TypeXxx/NotifyXxx/ActionXxx/DefaultXxx。

## 质量门
- `gofmt`/`go vet` 零告警；`go test ./...` 全绿才提交。
- 错误用领域级契约（如 `user.ErrNotFound`），不裸吞；`errors.Is` 判断。
- 提交前自查：无调试残留、无空 catch、无日志噪音。
