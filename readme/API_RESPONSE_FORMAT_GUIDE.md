# 外卖系统API响应格式标准化指南

## 🎯 项目背景

当前外卖系统存在HTTP接口响应格式混乱问题，表现为：
- 响应结构不统一（裸数据、map格式、字符串混合）
- 错误处理不规范（简单字符串错误信息）
- 状态码使用混乱（200/201混用）

## 📋 统一响应格式设计

### 1. 标准响应结构

```json
{
  "code": 200,
  "message": "操作成功",
  "data": {...},
  "error": {"type": "ERROR_TYPE", "details": "...", "path": "..."},
  "timestamp": 1699123456789,
  "requestId": "req-uuid-xxx"
}
```

### 2. 状态码规范

| Code | 状态类型 | 使用场景 |
|------|----------|----------|
| 200  | 成功     | GET/PUT/POST请求成功 |
| 201  | 已创建   | 资源创建成功 |
| 400  | 请求错误 | 参数验证失败 |
| 401  | 未授权   | token无效或过期 |
| 403  | 禁止访问 | 权限不足 |
| 404  | 未找到   | 资源不存在 |
| 422  | 验证错误 | 业务逻辑验证失败 |
| 500  | 服务器错误 | 内部异常 |

### 3. 数据结构要求

**统一Data字段意义：**
- **技术标准化**：消除解析不确定性，实现零配置解析
- **开发高效化**：客户端无需为每个接口写不同解析逻辑
- **生态集成**：支持自动化文档、SDK、测试工具生成

**数据类型示例：**
```json
// 单个对象
{"data": {"id": 1, "name": "测试店铺"}}

// 列表数据
{"data": [{"id": 1, "name": "店铺A"}, {"id": 2, "name": "店铺B"}]}

// 分页数据
{
  "data": {
    "list": [],
    "total": 100,
    "page": 1,
    "size": 10
  }
}
```

## 🛠️ 实现方案

### 1. 服务端工具库（Go）

```go
// response/response.go
package response

type APIResponse struct {
    Code      int         `json:"code"`
    Message   string      `json:"message"`
    Data      interface{} `json:"data,omitempty"`
    Error     *APIError   `json:"error,omitempty"`
    Timestamp int64       `json:"timestamp"`
    RequestID string      `json:"requestId"`
}

type APIError struct {
    Type    string      `json:"type"`
    Details interface{} `json:"details"`
    Path    string      `json:"path,omitempty"`
}

// 工具函数
func Success(w http.ResponseWriter, data interface{}, message string)
func SuccessWithCode(w http.ResponseWriter, data interface{}, message string, code int) 
func Error(w http.ResponseWriter, message string, code int)
func ValidationError(w http.ResponseWriter, details string, field string)
func ServerError(w http.ResponseWriter, err error)
```

### 2. 客户端SDK示例

**Go SDK：**
```go
type APIClient struct {
    baseURL string
    client  *http.Client
}

func (c *APIClient) Request(method, path string, body, target interface{}) error {
    var apiResp struct {
        Code int             `json:"code"`
        Data json.RawMessage `json:"data"`
        Message string       `json:"message"`
    }
    
    if apiResp.Code != 200 {
        return fmt.Errorf(apiResp.Message)
    }
    
    return json.Unmarshal(apiResp.Data, target)
}
```

**TypeScript SDK：**
```typescript
interface APIResponse<T> {
  code: number;
  message: string;
  data: T;
  timestamp: number;
  requestId: string;
}

class APIClient {
  async get<T>(url: string): Promise<T> {
    const response = await axios.get(url);
    const { code, data, message } = response.data;
    if (code !== 200) throw new Error(message);
    return data as T;
  }
}
```

## 📅 实施计划

### 阶段1：工具库开发
- [ ] 创建response包和核心结构体
- [ ] 实现通用响应工具函数
- [ ] 添加中间件支持（请求ID、统一日志）

### 阶段2：接口迁移
- [ ] auth.go（注册、登录、刷新token）
- [ ] order.go（订单相关接口）  
- [ ] shop.go（商家相关接口）
- [ ] user.go（用户相关接口）
- [ ] rider.go（骑手相关接口）
- [ ] review.go（评价相关接口）
- [ ] product.go（商品相关接口）
- [ ] message.go（消息相关接口）

### 阶段3：测试验证
- [ ] 接口响应格式验证测试
- [ ] 错误处理场景测试
- [ ] 性能测试（确保额外开销<1ms）

### 阶段4：文档更新
- [ ] API文档标准化
- [ ] 前端SDK使用指南
- [ ] 测试用例模板

## 🏗️ 目录结构建议

```
take-out/
├── api/
│   ├── response/          # 响应格式工具库
│   ├── middleware/        # 统一中间件
│   └── docs/             # API文档
├── handlers/
│   ├── auth.go           # 已迁移示例
│   ├── order.go          # 待迁移...
│   └── ...
├── web/
│   ├── client-ts/        # TypeScript SDK
│   └── client-go/        # Go SDK
└── tests/
    └── api/              # 接口测试
```

## 📱 兼容性策略

### 向后兼容方案
- 新接口使用新格式
- 老接口增加`API-Version`头识别（向下兼容版本）
- 提供3个月并行支持期

### 前端迁移策略
- 优先处理核心接口（登录、订单查询）
- 分批灰度发布，按用户百分比切换
- 设置回滚机制确保稳定性