此文件为 Gemini-cli 提供在此仓库中工作的指导。
优先阅读GEMINI.md,再阅读TODO.md
对于常用的或需要容器化部署的组件，优先从网上寻找开源镜像，而不是自己构建
**在分析代码流程过程中，任何的结论，都需要代码的原文位置引用**
**后续所有信息用中文输出**
**完成一次修改后,提交git commit,commit写修改的内容,遵守commit通用风格**

## 项目概览

这是一个**基于Go的外卖平台后端**，具有以下架构特点：
- **后端框架**：标准 Go `net/http` 与自定义路由 (`http.NewServeMux`)
- **数据库**：PostgreSQL + Redis 缓存/会话管理/消息队列
- **容器化**：基于Docker的部署方案 (`docker-compose.yml`)
- **监控**：Prometheus 指标端点
- **认证**：JWT 用户认证（骑手、顾客、商家），包含访问令牌和刷新令牌机制
- **响应格式**：统一的JSON API响应结构 (定义于 `response/` 目录)

## 关键命令

### 开发环境
```bash
# 本地运行应用 (需先配置好config.env)
go run main.go

# 构建应用程序
go build -o takeout-backend main.go

# 通过 Docker Compose 运行所有服务 (推荐)
docker-compose up

# 单独运行 Redis 服务
docker-compose up redis
```

### 测试
项目采用一套基于Shell脚本的端到端集成测试框架。

```bash
# 运行所有测试 (会自动检测服务状态)
./Test/run_all_tests.sh

# 运行特定模块的测试
./Test/run_all_tests.sh auth     # 仅测试注册登录
./Test/run_all_tests.sh order    # 仅测试订单流程

# 查看详细测试指南
cat ./Test/README_TEST_GUIDE.md
```
*注：传统的 `go test ./...` 单元测试框架未被主要使用。*

### 数据库
- **主数据库**: PostgreSQL，通过 `database/init.sql` 初始化表结构。
- **关键功能**:
  - 用户管理（顾客、骑手、商家）
  - 商家/商品目录
  - 订单管理与跟踪
  - 基于订单的实时消息群组
  - 用户评价与AI分析系统

### 配置
- **主配置**：`config.env`（环境变量）
- **数据库连接**：通过 `config.env` 中的 `DATABASE_URL` 配置
- **Redis连接**：通过 `config.env` 中的 `REDIS_URL` 配置

## 架构详情

### 目录结构
```
take-out/
├── main.go              # 应用入口点, 注册所有HTTP路由
├── handlers/            # HTTP路由处理器, 核心业务逻辑
├── database/            # 数据库模型、查询和连接管理
├── models/              # Go数据结构定义
├── response/            # 统一的API响应和中间件
├── monitoring/          # Prometheus指标收集
├── logging/             # 基于logrus的集中日志
├── Test/                # 端到端集成测试脚本和数据
├── pgsql-openai/        # (子模块) PostgreSQL AI集成扩展
├── docker-compose.yml   # 容器编排配置
└── Dockerfile           # 应用容器构建
```

### 关键处理器（REST API）
所有API路径均以 `/api` 为前缀。

- **认证 (路径: `/api/auth/...`)**:
  - `POST /user/register`, `POST /user/login`
  - `POST /shop/register`, `POST /shop/login`
  - `POST /rider/register`, `POST /rider/login`
  - `POST /refresh` (刷新Access Token)
  - `POST /logout` (实现于 `handlers/auth.go`，但未在 `main.go` 中注册)

- **用户 (路径: `/api/user/...`, 需要用户Token)**:
  - `GET /shops` (获取所有商家)
  - `GET /products` (获取指定商家的商品)
  - `POST /order` (下单)
  - `GET /order/status` (查询订单状态)
  - `GET /nearby-shops` (获取附近商家)
  - `POST /im/send` (发送消息)
  - `GET /im/messages` (获取群聊消息)
  - `POST /review/create` (创建评价)
  - `PUT /review/update` (更新评价)

- **商家 (路径: `/api/shop/...`, 需要商家Token)**:
  - `POST /add_product` (添加商品)
  - `POST /update_stock` (更新库存)
  - `POST /accept_order` (接单)
  - `POST /publish_order` (发布订单到配送队列)
  - `GET /reviews` (获取本店评价)
  - `POST /review/reply` (回复评价)
  - `GET /review/analytics` (获取评价分析)

- **骑手 (路径: `/api/rider/...`, 需要骑手Token)**:
  - `POST /grab` (抢单)
  - `POST /complete` (完成订单)
  - `POST /confirm_delivery` (确认送达)

- **监控**:
  - `GET /metrics`

### 数据库架构 (源自 `database/init.sql`)
- **users**: 顾客信息
- **shops**: 商家信息
- **riders**: 骑手信息
- **products**: 商品条目
- **orders**: 订单信息 (核心表)
- **groups**: 关联订单、用户、商家、骑手的聊天群组
- **messages**: 实时通信消息
- **reviews**: 用户评价信息
- **ai_analysis**: 对评价的AI分析结果
- **token_blacklist**: JWT黑名单，用于实现登出功能

### 监控
项目集成了 Prometheus 监控，通过 `/metrics` 端点暴露以下核心指标 (源自 `monitoring/metrics.go`):
- `http_request_duration_seconds`: 按路径划分的 HTTP 请求耗时分布。
- `http_requests_total`: 按路径、方法和状态码统计的 HTTP 请求总数。
- `db_call_duration_seconds`: 按操作类型划分的数据库（PostgreSQL）调用耗时分布。
- `redis_call_duration_seconds`: 按操作类型划分的 Redis 调用耗时分布。
- `log_queue_size`: 日志队列当前大小。
- `logs_dropped_total`: 因队列满而丢弃的日志总数。

### 外部依赖 (`go.mod`精选)
- `github.com/golang-jwt/jwt`: JWT认证
- `github.com/go-redis/redis/v8` & `github.com/redis/go-redis/v9`: Redis客户端
- `github.com/lib/pq`: PostgreSQL驱动
- `github.com/sirupsen/logrus`: 日志
- `github.com/prometheus/client_golang`: Prometheus指标
- `golang.org/x/crypto`: 用于密码哈希 (`bcrypt`)
- `github.com/joho/godotenv`: 用于加载 `.env` 文件

## 常见操作

### 添加新REST端点
1. 在 `handlers/` 目录下对应的文件中添加处理器函数 (例如 `handlers/user.go`)。
2. 处理器函数应使用 `response/` 包中的函数 (`response.Success`, `response.Error` 等) 来返回统一格式的JSON。
3. 如果需要，在 `database/` 中添加新的数据库操作函数。
4. 在 `main.go` 中找到对应的路由组 (如 `userRoutes`)，注册新的路由。
5. 确保新的数据库和Redis调用被 `monitoring.RecordDBTime` 和 `monitoring.RecordRedisTime` 包裹，以进行性能监控。
6. 在 `Test/` 目录下添加或修改测试脚本以覆盖新功能。