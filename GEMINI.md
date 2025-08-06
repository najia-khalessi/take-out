# GEMINI.md

该文件为 gemini-cli 在此代码库中工作时提供指导。
后续所有信息用中文描述
需要完成的工作在TODO.md里
每次执行用户命令之前重新阅读TODO.md
每次修改完代码提交一个git commit

## 项目概述
一个用 Go 编写的外卖食品配送平台后端。一个处理用户、餐厅、骑手、订单、支付和实时消息的综合系统。

## 关键技术栈
- **语言**: Go (Golang)
- **数据库**: MySQL + Redis
- **监控**: Prometheus 指标
- **日志**: Logrus 结构化日志
- **架构**: 基于令牌认证的 RESTful API

## 核心领域模型
- **User**: 下订单的客户
- **Shop**: 出售食品的餐厅
- **Rider**: 配送人员
- **Product**: 餐厅的食品项目
- **Order**: 包含配送详情的客户订单
- **Message**: 用于订单协调的群聊
- **Group**: 订单分组功能

## 项目结构
```
take-out/
├── database/          # 数据库层 (MySQL + Redis)
├── handlers/          # HTTP 处理程序和 API 端点
├── models/           # 数据结构 (struct.go)
├── logging/          # 日志配置
├── monitoring/       # Prometheus 指标
├── main.go          # 应用程序入口点
├── config.env       # 环境配置
└── database/init.sql # 数据库模式
```

## 入门命令

### 设置与安装
```bash
# 安装依赖
go mod download

# 设置数据库
git clone <repo-url>
vim config.env  # 配置数据库凭据
```

### 数据库设置
```bash
# 运行数据库初始化
mysql -u root -p < database/init.sql

# 在 config.env 中配置环境变量
```

### 开发命令
```bash
# 运行应用程序
export $(cat config.env | xargs)
go run main.go

# 构建应用程序
go build -o takeout-backend

# 运行测试 (如果存在任何测试文件)
go test ./...

# 检查类型/lint 问题
go vet ./...
```

## 主要功能
- **认证**: 基于 JWT 令牌的用户、商家和骑手认证
- **实时更新**: 用于订单状态更新的 Redis pub/sub
- **地理定位**: 为用户提供基于距离的商店查找
- **订单管理**: 完整的订单生命周期 (创建、接受、取货、交付)
- **实时消息**: 用于订单协调的群聊
- **性能监控**: 在 /metrics 处公开的 Prometheus 指标
- **定期任务**: 每周清理、数据库监控、订单消费者

## API 端点概述

### 公共端点 (无需身份验证)
- `POST /user/register` - 用户注册
- `POST /user/login` - 用户登录
- `POST /shop/register` - 商家注册
- `POST /shop/login` - 商家登录
- `POST /rider/apply` - 骑手申请
- `POST /refresh` - 令牌刷新

### 用户认证端点
- `GET /user/shops` - 获取所有商店
- `GET /user/products` - 获取商店产品
- `POST /user/order` - 创建订单
- `GET /user/order/status` - 检查订单状态
- `GET /user/nearby-shops` - 查找附近的商店
- `POST /user/im/send` - 发送聊天消息
- `GET /user/im/messages` - 获取聊天消息

### 商家认证端点
- `POST /shop/add_product` - 将产品添加到菜单
- `POST /shop/update_stock` - 更新产品库存
- `POST /shop/accept_order` - 接受传入订单
- `POST /shop/publish_order` - 将订单发布给骑手

### 骑手认证端点
- `POST /rider/grab` - 抢配送订单
- `POST /rider/complete` - 将订单标记为已送达

## 数据库架构
- **主要**: 用于持久化数据的 MySQL
- **缓存**: 用于会话管理、pub/sub 和缓存的 Redis
- **连接池**: 配置的 Redis 连接池
- **监控**: 用于数据库健康检查的后台 goroutine

## 配置文件
- **config.env**: 数据库和 Redis 的环境变量
- **database/init.sql**: 完整的数据库模式
- **main.go**: 集中的路由定义和中间件

## 后台服务
- **OrderConsumer**: 基于 Redis 的订单处理
- **DBMonitor**: 数据库健康监控
- **CleanupScheduler**: 每周数据库清理
