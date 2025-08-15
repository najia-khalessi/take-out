# Go-Take-Out 外卖平台后端

这是一个基于 Go 语言构建的高性能外卖平台后端系统。

## 系统特点

- **后端框架**: 标准 Go `net/http`
- **数据库**: PostgreSQL + Redis
- **容器化**: 基于 Docker 和 Docker Compose
- **监控**: Prometheus 指标端点 (`/metrics`)
- **认证**: JWT (用户、商家、骑手)
- **AI 功能**: 集成 AI 对用户评价进行情感分析

---

## 外部扩展项目: pgsql-openai

本项目的一个核心特色是利用 AI 分析用户评价。为了实现这一点，我们集成了一个外部的 PostgreSQL 扩展。

- **项目名称**: `pgsql-openai`
- **项目来源**: [https://github.com/pramsey/pgsql-openai](https://github.com/pramsey/pgsql-openai)
- **用途**: 此扩展使得 PostgreSQL 数据库可以直接调用 OpenAI API (或兼容的本地模型如 Ollama)，从而允许我们通过 SQL 查询对新产生的用户评价进行自动化的情感分析和分类。本项目中的 `pgsql-openai` 目录是该项目的本地克隆，用于编译和安装。

---

## 本地部署指南

本指南将引导在本地机器上运行此项目，其中 Go 应用将作为 Docker 容器运行，并连接到**本机安装的 PostgreSQL** 和 Docker 化的 Redis。

### 依赖准备

请确保本地环境已安装以下软件：
1.  **Git**: 用于克隆代码仓库。
2.  **Docker** 和 **Docker Compose**: 用于运行应用服务和 Redis。
3.  **PostgreSQL**: 在主机上直接安装和运行。
4.  **Go** 和 **Make**: 用于编译 `pgsql-openai` 扩展。

### 部署步骤

#### 1. 克隆代码仓库
```bash
git clone <repository_url>
cd take-out
```

#### 2. 设置 PostgreSQL 数据库
连接到本地的 PostgreSQL 服务，并创建一个专用的数据库和用户。

```sql
-- 使用 psql 或喜欢的数据库客户端执行
CREATE DATABASE takeoutdb;
CREATE USER youruser WITH ENCRYPTED PASSWORD 'yourpassword';
GRANT ALL PRIVILEGES ON DATABASE takeoutdb TO youruser;
```

#### 3. 编译并安装 pgsql-openai 扩展
此步骤将编译 `pgsql-openai` 扩展并将其安装到本地的 PostgreSQL 中。

```bash
# 进入扩展目录
cd pgsql-openai

# 编译
make

# 安装 (需要管理员权限)
sudo make install

# 返回项目根目录
cd ..
```

安装成功后，连接到刚刚创建的 `takeoutdb` 数据库并启用扩展。

```sql
-- 依次执行以下命令
\c takeoutdb;
CREATE EXTENSION IF NOT EXISTS http;      -- pgsql-openai 的依赖
CREATE EXTENSION IF NOT EXISTS openai;
```

#### 4. 初始化数据库表结构
项目所需的全部数据表结构都定义在 `database/init.sql` 文件中。需要将此文件的内容导入到 `takeoutdb` 数据库中。

```bash
# 在项目根目录运行此命令
psql -U youruser -d takeoutdb -f database/init.sql
```

#### 5. 配置环境变量
在项目根目录创建一个名为 `config.env` 的文件。这是应用读取所有配置的地方。

```bash
cp config.env.example config.env # 如果有示例文件的话，或者手动创建
```

编辑 `config.env` 文件，并填入以下内容：

```env
# 数据库连接字符串
DATABASE_URL=postgres://youruser:yourpassword@host.docker.internal:5432/takeoutdb?sslmode=disable

# Redis 连接地址 (因为 Redis 在 Docker 网络中，所以可以直接用服务名)
REDIS_URL=redis:6379
```

#### 6. 理解并解决 Docker 与主机的网络通信问题

**关键配置**：`DATABASE_URL` 中的主机名为什么必须是 `host.docker.internal`？

- **问题的本质：网络命名空间隔离 (Network Namespace Isolation)**
  这个问题的核心在于 Docker 容器和主机默认不共享网络命名空间。当 Docker 启动一个容器时，它会为该容器创建一个独立的网络环境，这个环境拥有自己的一套完整且隔离的网络资源，包括：
    - 自己的网络接口 (如 `eth0`)
    - 自己的 IP 地址和路由表
    - **以及最重要的，自己的 `localhost` (`127.0.0.1`) 回环地址。**

- **`localhost` 的误区**
  - 在**主机**上，`localhost` 指向主机本身。
  - 在**容器内部**，`localhost` 指向的是容器自己，而不是主机。
  
  因此，当容器内的 Go 应用尝试连接 `postgres://...@localhost:5432` 时，它实际上是在寻找容器**内部**的 5432 端口，但 PostgreSQL 服务是运行在主机上的。由于网络命名空间的隔离，这个连接请求无法“跨越”到主机，从而导致连接失败。

- **解决方案：`host.docker.internal`**
  为了解决这个问题，Docker 提供了一个特殊的 DNS 名称：`host.docker.internal`。这个地址会从容器的网络命名空间内部，被 Docker 自动解析为主机的内部 IP 地址。通过在连接字符串中使用它，Go 应用容器就能成功地“跨越”这层网络隔离，找到并连接到在主机上运行的 PostgreSQL 服务。

#### 7. 启动服务
现在，所有配置都已完成。在项目根目录运行 `docker-compose` 来启动应用和 Redis。

```bash
docker-compose up --build
```
`--build` 标志会确保 Go 应用被重新编译。服务启动后，应该能看到应用成功连接到数据库和 Redis 的日志。


