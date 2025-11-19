# SendsPlatform

一个基于微服务架构的校园服务平台，提供用户管理、博饼游戏、学校信息查询、年度账单等功能。项目采用Go语言开发，使用gRPC进行服务间通信，支持分布式事务处理。

## 功能特性

### 核心业务功能
- **用户管理**: 微信登录、用户信息管理、多平台账号绑定
- **博饼游戏**: 在线博饼游戏系统、实时排名、游戏记录
- **学校服务**: 课表查询、成绩查询、学分查询、GPA统计
- **年度账单**: 学习数据统计、消费记录分析、个人成长报告

### 技术特性
- **微服务架构**: 基于gRPC的微服务设计，支持独立部署和扩展
- **分布式事务**: 集成TCC模式保证数据一致性
- **服务发现**: 基于Etcd的服务注册与发现机制
- **负载均衡**: 支持多实例负载均衡和故障转移
- **可观测性**: 集成日志、指标、链路追踪系统
- **安全性**: 多层安全防护，支持JWT认证和RBAC权限控制
- **高性能**: 基于Gin框架的高性能HTTP服务，支持连接复用和请求合并

## 软件架构

```
         ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
         │   API Gateway   │    │   User Service  │    │  BoBing Service │
         │   (Port: 8889)  │    │  (Port: 10002)  │    │  (Port: 10003)  │
         └─────────────────┘    └─────────────────┘    └─────────────────┘
                 │                       │                       │
                 └───────────────────────┼───────────────────────┘
                                         │
         ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
         │ School Service  │    │ YearBill Service│    │  Common Library │
         │ (Port: 10004)   │    │ (Port: 10005)   │    │   (Shared)      │
         └─────────────────┘    └─────────────────┘    └─────────────────┘
                                         │
         ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
         │     MySQL       │    │      Redis      │    │     RabbitMQ    │
         │   (Port: 3306)  │    │   (Port: 6379)  │    │   (Port: 5672)  │
         └─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 快速开始

### 依赖检查

确保您的系统已安装以下依赖：

- **Go 1.21+**: 用于编译和运行Go程序
- **Docker & Docker Compose**: 用于容器化部署
- **MySQL 8.0+**: 数据存储
- **Redis 7.2+**: 缓存和会话存储
- **Etcd 3.5+**: 服务发现
- **RabbitMQ**: 消息队列
- **Protobuf**: 用于gRPC接口定义

### 构建

1. **克隆项目**
```bash
git clone https://github.com/your-org/SendsPlatform.git
cd SendsPlatform
```

2. **安装依赖**
```bash
make deps
```

3. **生成代码**
```bash
make proto
make swagger
```

4. **构建所有服务**
```bash
make build
```

### 运行

1. **配置环境变量**
```bash
cp env.example .env
# 编辑 .env 文件，配置必要的环境变量
```

2. **启动开发环境**
```bash
make dev
```

3. **验证服务**
```bash
# 健康检查
make health

# 查看服务状态
make docker-logs
```

## 使用指南

### API接口

#### 网关服务 (端口: 8889)
- **健康检查**: `GET /health`
- **API文档**: `GET /swagger/index.html`

#### 用户服务
- **用户登录**: `POST /user/login`
- **学校登录**: `POST /user/school_login`
- **年度账单登录**: `POST /user/bill_login`

#### 博饼服务
- **获取排名**: `GET /boBing/top`
- **投掷**: `POST /boBing/publish`
- **获取记录**: `GET /boBing/record`

#### 学校服务
- **课表查询**: `POST /school/schedule`
- **学分查询**: `GET /school/xuefen`
- **GPA查询**: `GET /school/gpa`
- **成绩查询**: `GET /school/grade`

#### 年度账单服务
- **学习数据**: `GET /yearBill/learn`
- **消费数据**: `GET /yearBill/pay`
- **排名数据**: `GET /yearBill/rank`
- **评价**: `POST /yearBill/appraise`

### 开发指南

#### 代码规范
```bash
# 格式化代码
make fmt

# 代码检查
make lint

# 运行测试
make test
```

#### 添加新服务
1. 创建服务目录: `mkdir -p app/newservice/{cmd,database,service,types}`
2. 定义gRPC接口: 在 `idl/` 目录下创建 `.proto` 文件
3. 生成代码: `make proto`
4. 实现服务逻辑: 参考现有服务的实现模式
5. 更新配置: 在 `config/config.yml` 和 `docker-compose.yml` 中添加服务定义

#### 数据库迁移
```bash
# 创建迁移文件
make migrate-create

# 运行迁移
make migrate-up

# 回滚迁移
make migrate-down
```

