# AGENTS.md — scene 后端（题目01 电商订单交易系统）

> 本文件是后端项目的最高约束。**任何 AI Agent（开发/测试/验收）在生成或修改代码前，必须先读取并遵守本文件；每次生成代码都必须走一遍本文件的约束与 §6 的两个 Skill。** 学生未明确同意，任何 Agent 不得修改本文件。

## 1. 项目定位

- 仓库：https://github.com/ovo133838-lang/scene （分支 `main` 仅初始化，开发在 `dev/01`）
- 技术栈：Go + Gin；MySQL 8.x；Redis 7.x；鉴权 JWT。
- 用户端：Vue3 + Element Plus，**并入本仓库 `web/` 目录，构建产物经 `go:embed` 打进单二进制**（同源提供页面，不单独建仓库）。管理端独立于 scene-manage 仓库。
- 本题为题目01：小型电商订单交易、发货与售前退款。业务规则以原始题面与《需求规格说明书》为准。

## 2. 整体框架结构（不得擅自修改）

目录结构已由学生规定，**任何 Agent 不得擅自增删层级、改名或引入未约定的框架**：

```text
gin/
├── cmd/server/main.go        # 入口：装配路由、中间件、依赖注入
├── internal/
│   ├── auth/                 # JWT 签发/校验、登录、RBAC 守卫
│   ├── product/              # 商品域（上下架、改价、库存）
│   ├── order/                # 订单域（下单占用库存、状态机、取消/超时）
│   ├── payment/              # 支付域（沙箱发起、验签、回调、幂等）
│   ├── refund/               # 退款域（申请、审核、渠道退款、重试）
│   ├── shipment/             # 发货域（发货登记、一次性约束）
│   ├── report/               # 运营汇总
│   ├── mockpay/              # 可控回调网关（异常注入，仅教学演示）
│   ├── store/                # 数据访问层（MySQL/Redis 全部经此处）
│   └── clock/                # 可配置教学时钟
├── web/                      # 用户端 Vue3+Element Plus（源码+dist，go:embed 打包）
├── migrations/               # 数据库 DDL 与种子数据
├── configs/config.toml       # 配置（含支付宝沙箱密钥，.gitignore 不提交）
├── docs/                     # 01需求 02数据字典 03接口文档 04任务 05原型 06开发 07测试 08验收 09环境 10Skill 11部署
├── AGENTS.md
├── go.mod
└── .gitignore
```

约定：HTTP 处理器只能调用 `internal/*` 的业务方法；**状态迁移集中实现，禁止在 handler 里散落 `UPDATE status`**；数据库访问统一走 `store` 层。

## 3. 数据库设计（9 张表，不得擅自改表结构）

金额一律 `BIGINT` 存「分」；`channel_txn_no`、`shipment.order_id` 唯一约束；订单状态用字符串枚举。

| 表 | 关键字段 | 说明 |
| --- | --- | --- |
| `user` | id, username(UNIQUE), password_hash, role, created_at | role ∈ buyer/operator/warehouse/service |
| `product` | id, name, price(BIGINT 分), stock(可售), status(on/off), created_at, updated_at | stock 实时扣减/释放 |
| `order` | id(订单号), buyer_id, product_id, product_name(快照), price_snapshot(分), quantity, total_amount(分), status, created_at, expired_at, updated_at | 保存下单时快照 |
| `payment` | id, order_id, channel_txn_no(UNIQUE), amount(分), status, notify_at, created_at | 渠道交易号全局唯一 |
| `refund` | id, order_id, payment_id, amount(分), type(sale/late_payment), status, reason, retry_count, created_at, updated_at | type 区分售前/迟到支付 |
| `shipment` | id, order_id(UNIQUE), carrier, tracking_no, operator_id, shipped_at | 每单一条 |
| `payment_abnormal` | id, order_id, channel_txn_no, amount(分), status, reason, created_at | 迟到/异常支付待处理 |
| `role_permission` | id, role, resource, action | RBAC 权限点 |
| `order_event` | id, order_id, event, detail, operator_id, created_at | 状态变更/操作留痕（BR-08 追溯） |

> `order_event` 是对需求文档 §6 的补充，用于保证“异常记录不能通过直接修改订单状态消失”。

## 4. 接口清单（已根据原型固化，不得增删改路径/语义）

鉴权：除 `POST /api/login` 与 `POST /pay/notify` 外，全部需 JWT；管理端接口另需角色校验；买家接口需资源归属校验。

### 4.1 登录
- `POST /api/login`　登录（买家/运营/仓库/客服统一），返回 JWT

### 4.2 用户端（买家）
- `GET /api/products`　在售商品列表
- `GET /api/products/{id}`　商品详情（可售数量）
- `POST /api/orders`　下单（占用库存）
- `GET /api/orders`　我的订单列表
- `GET /api/orders/{id}`　订单详情（含支付/退款/发货状态）
- `POST /api/orders/{id}/cancel`　取消待付款订单
- `POST /api/orders/{id}/pay`　发起支付，返回沙箱收银台链接
- `POST /api/orders/{id}/refund`　申请售前退款

### 4.3 支付回调
- `POST /pay/notify`　支付宝沙箱异步通知（真实验签，幂等）
- `POST /api/admin/pay/mock-notify`　可控回调网关（注入成功/失败/超时/重复/迟到/验签失败/金额不符/未知订单）

### 4.4 管理端 · 运营
- `GET /api/admin/products`　商品管理列表
- `POST /api/admin/products/{id}/shelf`　上架/下架
- `PUT /api/admin/products/{id}`　改售价/库存
- `GET /api/admin/products/stock`　库存看板
- `GET /api/admin/summary`　交易汇总（收款/退款/发货/异常）
- `GET /api/admin/orders/{id}/records`　订单收款/退款/发货/异常记录

### 4.5 管理端 · 仓库（商家）
- `GET /api/admin/ship/pending`　可发货订单
- `POST /api/admin/ship/{id}`　发货登记（承运方+运单号，一次性）
- `GET /api/admin/ship/shipped`　已发货列表

### 4.6 管理端 · 客服
- `GET /api/admin/refunds/pending`　退款待审核
- `POST /api/admin/refunds/{id}/approve`　同意退款
- `POST /api/admin/refunds/{id}/reject`　拒绝退款（必填原因）
- `POST /api/admin/refunds/{id}/retry`　退款重试
- `GET /api/admin/pay-abnormal`　异常支付列表
- `POST /api/admin/pay-abnormal/{id}/refund`　异常支付退款（不恢复库存）

## 5. Agent 角色职责与边界（独立会话）

### 5.1 开发 Agent
- 职责：按需求/原型/任务清单，在 `dev/01` 上实现 `internal/*`、`cmd`、`migrations`、单元测试。
- 可修改：`cmd/`、`internal/`、`migrations/`、`configs/config.toml`、`docs/04-开发记录.md`、`go.mod`、`go.sum`。
- 需学生确认：新增依赖、业务规则取舍、接口变更。
- 边界：不得修改测试/验收报告；不得改 `AGENTS.md` 与框架结构；不得扩大范围（不做购物车/优惠/部分退款/售后物流/真实生产支付）。

### 5.2 测试 Agent（独立会话）
- 职责：按原始题面制定并执行测试（单元、集成/回调矩阵、并发、越权）。
- 可修改：仅 `docs/05-测试报告.md` 及 `*_test.go` 补充；不得改生产代码。
- 交接：缺陷清单 + 复测记录。

### 5.3 验收 Agent（独立会话）
- 职责：按原始题面逐项核对（业务、权限、数据一致性、错误处理、敏感信息、部署）。
- 可修改：仅 `docs/06-验收报告.md`；不得改任何代码/测试。
- 交接：逐项结论 + 证据位置。

## 6. 每次生成代码必须走一遍的约束与 Skill

### 6.1 开发前走查全部业务流程（硬性）
写任何业务代码前，必须先通读并走查五组场景：正常成交、缺货与弃单（20 并发）、售前退款、迟到支付、操作竞争，理解后再编码，不得只做单一成功路径。

### 6.2 业务约束检查 Skill（t01-order-business-check）
每完成一个模块（order/payment/refund/shipment/product），逐条核对 BR-01~BR-08 并输出检查报告（规则/结果/证据），不通过先修复。

### 6.3 验收检查 Skill（t01-acceptance-check）
验收阶段实测五组场景 + 交叉检查（权限/一致性/错误处理/敏感信息/可维护性），输出逐项结论。

## 7. 提交与分支规则

- 开发只提交 `dev/01`，禁止直推 `main`；`main` 只经 PR 合并。
- 提交格式（依据 commit-convention.md）：`[类型] [版本号] [编号] 中文描述`
  - feat/fix 须先 `meegle workitem get --work-item-id <编号> --project-key r17te9` 核验真实编号并保留输出；核验失败不得提交。
  - 一次 commit 只一种类型；不得用 docs/test/opt 伪装缺编号的功能变更。
- 合并前置门禁：`gofmt`、`go vet`、覆盖率 ≥85%（整体+各核心包）、`go test -race` 通过、测试/验收报告齐备。

## 8. 代码约定与禁止事项

- 金额一律分（int64/BIGINT），禁止浮点。
- 待付款 10 分钟超时用可配置教学时钟（`internal/clock`），测试与演示同规则，演示注明缩短。
- 身份一律取 JWT，不信任请求参数中的用户编号。
- 支付宝沙箱私钥/公钥、MySQL/Redis 密码存 `configs/` 并 `.gitignore`；日志/截图/视频/导出脱敏。
- 禁止编造接口/依赖用法/执行结果/测试报告/部署记录；禁止删除失败测试、弱化断言、跳过关键路径、排除自编代码冲覆盖率；禁止泄露密钥。

## 9. 版本记录

| 版本 | 日期 | 变更 |
| --- | --- | --- |
| v1 | 2026-11-15 | 初版：框架结构、数据库、接口清单、Agent 职责、Skill 约束 |
