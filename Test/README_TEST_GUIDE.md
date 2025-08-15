# take-out外卖系统完整测试指南

## 🚀 快速开始

这套测试脚本为您提供了从注册登录到订单评价的全流程测试能力，使用curl命令行工具验证外卖系统的每个功能模块。

### 📋 测试内容总览

| 测试阶段 | 涵盖功能 | 测试脚本 |
|----------|----------|----------|
| **阶段1: 认证系统** | 用户/商家/骑手注册登录 | `test_auth.sh` |
| **阶段2: 订单流程** | 下单→接单→配送→完成 | `test_order_flow.sh` |
| **阶段3: 消息系统** | 实时群聊、订单通知 | `test_messaging.sh` |
| **阶段4: 评价系统** | 用户评价体系、AI智能分析 | `test_review_system.sh` |

### 🔧 一键测试

```bash
# 1. 确保后端服务已启动
docker-compose up -d

# 2. 运行完整测试
./run_all_tests.sh          # 执行所有系统测试

# 3. 或选择性测试
./run_all_tests.sh auth     # 仅测试注册登录
./run_all_tests.sh order    # 仅测试订单流程
./run_all_tests.sh messaging # 仅测试消息系统
./run_all_tests.sh reviews  # 仅测试评价系统
```

### 📊 测试结果查看

#### 实时查看测试进度

```bash
# 实时监控测试日志
tail -f test_logs/*.log

# 查看生成的测试数据
ls -la test_data/

# 查看最终测试报告
cat test_results/comprehensive_report.md
```

#### 测试文件结构

```
take-out/
├── test_auth.sh           # 认证系统测试
├── test_order_flow.sh     # 订单生命周期测试
├── test_messaging.sh      # 消息系统测试
├── test_review_system.sh  # 评价系统测试
├── run_all_tests.sh       # 完整测试整合器
├── test_data/             # 测试数据和结果
│   ├── config.json        # 测试配置
│   ├── order_id.txt       # 创建的订单ID
│   ├── user_token.txt     # 用户访问Token
│   ├── shop_token.txt     # 商家访问Token
│   ├── rider_token.txt    # 骑手访问Token
│   └── *.json             # 详细测试数据
├── test_logs/            # 详细测试日志
│   ├── auth.log          # 认证测试日志
│   ├── order_flow.log    # 订单测试日志
│   ├── messaging.log     # 消息测试日志
│   └── review_system.log # 评价测试日志
└── test_results/         # 最终结果报告
    ├── comprehensive_report.md
    ├── test_summary.json
    └── test_summary.csv
```

## 🎯 详细使用说明

### 步骤1: 环境准备

```bash
# 确保依赖已安装
sudo apt-get update && sudo apt-get install -y curl jq

# 检查后端服务
./run_all_tests.sh  # 会自动检查服务状态
```

### 步骤2: 运行认证测试

```bash
./test_auth.sh
# 输出示例:
# ✅ 用户注册成功
# ✅ 商家登录成功  
# ✅ Token验证通过
```

### 步骤3: 执行完整业务流程

```bash
./run_all_tests.sh
# 自动执行完整流程:
# 1. 用户注册 → 登录 → 获取Token
# 2. 商家注册 → 登录 → 添加商品
# 3. 用户浏览 → 下单 → 等待商家确认
# 4. 商家接单 → 发布配送 → 通知骑手
# 5. 骑手抢单 → 配送中 → 送达确认
# 6. 实时消息 → 多方沟通 → 完成通知
# 7. 用户评价 → AI分析 → 商家回复
```

## 📈 测试验证点

### 1. 认证系统验证
- [x] 多层次用户注册
- [x] Token生成与验证
- [x] 权限边界控制
- [x] 重复注册处理
- [x] 密码安全存储

### 2. 订单生命周期验证  
- [x] 商品管理功能
- [x] 订单状态流转
- [x] 配送逻辑完整性
- [x] 实时状态更新
- [x] 异常处理机制

### 3. 消息系统验证
- [x] 实时消息传递
- [x] 群组消息历史
- [x] 用户角色区分
- [x] 并发消息处理
- [x] 消息持久化

### 4. 评价系统验证
- [x] 多维度评分机制
- [x] AI智能情感分析
- [x] 商家回复互动
- [x] 数据统计分析
- [x] 筛选查询功能

## 🔍 调试故障排除

### 常见问题解决

```bash
# 1. 服务未启动
curl -I http://localhost:8080/metrics
# 如果返回404，检查服务状态

# 2. Token验证失败
cat test_data/user_token.txt  # 检查Token是否生成

# 3. 测试失败排查
tail -n 50 test_logs/*.log | less

# 4. 重置测试环境
rm -rf test_data/ test_logs/ test_results/
./run_all_tests.sh
```

### 服务状态监控

```bash
# 查看应用日志
docker-compose logs -f app

# 查看数据库连接
docker-compose logs -f postgres

# Redis连接检查
docker-compose exec redis redis-cli ping
```

## 🎇 测试扩展开发

### 添加新测试场景

```bash
# 创建新的测试脚本
./scripts/new_test.sh my_custom_test "自定义测试功能"

# 集成到主测试器
在run_all_tests.sh中新增调用
```

### 压力测试扩展

```bash
# 修改测试脚本添加并发测试
# 可集成Apache Bench或wrk进行性能测试
ab -n 1000 -c 10 http://localhost:8080/api/user/shops
```

## 📋 技术支持

### 获取帮助

1. **查看详细日志**: `tail -f test_logs/*.log`
2. **检查测试报告**: `cat test_results/comprehensive_report.md`
3. **系统架构**: 阅读源代码中的handlers目录
4. **数据库结构**: 查看database/init.sql

### 联系支持
- 报告问题: 创建Issue
- 代码贡献: 提交Pull Request
- 功能请求: 描述测试场景需求

---

**测试创建时间**: 2024年8月14日  
**测试版本**: v1.0完整版  
**适用系统**: Linux/macOS/WSL  
**测试语言**: 中文环境