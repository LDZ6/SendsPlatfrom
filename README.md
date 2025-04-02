# 华侨大学桑梓微助手部分服务

## 项目概述
**项目名称**: 桑梓微助手部分后端服务

**项目周期**: 2024年07月 - 至今

**项目地址**: [GitHub](https://github.com/LDZ6/SendsPlatfrom)

**项目简介**:
本项目是华侨大学桑梓微助手微信公众号的部分后端服务，提供四个核心微服务：
1. **教务处微服务**：提供学生的课表、成绩、学分等信息查询功能。
2. **博饼游戏微服务**：举办福建中秋节博饼活动，包含排行榜和实时播报功能。
3. **年度总结微服务**：爬取用户消费记录，并生成年度账单报告。
4. **用户微服务**：负责登录身份校验及部分微信公众号官方功能。

## 技术栈
- **后端框架**: Gin
- **数据库**: MySQL、Redis
- **ORM框架**: Gorm
- **API文档**: Swagger
- **消息队列**: RabbitMQ
- **服务发现与配置**: Etcd
- **微服务通信**: Grpc
- **容器化部署**: Docker Compose
- **微信公众号开发**: 微信公众号 API
- **其他**: 多线程爬虫、数据加密、并发控制

## 关键技术实现
### 1. 博饼游戏微服务
- **使用 Zset 存储与排序用户分数**，提高查询效率与准确性。
- **AES 加密用户敏感信息**，保障数据安全和隐私。

### 2. 年度总结微服务
- **RabbitMQ 作为消息中间件**，实现异步处理、削峰填谷、解耦。
- **多线程爬虫高效爬取学校 API**，提高数据抓取速度。
- **WebSocket 实时显示数据加载进度**，优化用户体验。

### 3. 用户微服务
- **面向不同人群（公众、在校生、校友）** 进行微服务拆分，满足不同需求。

### 4. 教务处微服务
- **使用 goroutine 进行多线程爬取**，提升数据获取与处理效率。

### 5. 网关部分
- **自定义令牌桶算法限流**，保证系统稳定，防止流量过载。

## 服务架构
本项目采用 **微服务架构**，通过 gRPC 进行服务间通信，Etcd 进行服务发现，并结合 RabbitMQ 进行异步消息处理。

### **Docker Compose 服务配置**
```yaml
version: "3"
services:
  user:
    restart: unless-stopped
    build:
      dockerfile: ./app/user/cmd/Dockerfile
      context: .
    depends_on:
      - db
      - redis
  bobing:
    restart: unless-stopped
    build:
      dockerfile: ./app/boBing/cmd/Dockerfile
      context: .
    depends_on:
      - db
      - redis
  school:
    restart: unless-stopped
    build:
      dockerfile: ./app/school/cmd/Dockerfile
      context: .
    depends_on:
      - redis
  year_bill:
    restart: unless-stopped
    build:
      dockerfile: ./app/yearBill/cmd/Dockerfile
      context: .
    depends_on:
      - db
      - redis
  gateway:
    restart: unless-stopped
    build:
      dockerfile: ./app/gateway/cmd/Dockerfile
      context: .
    ports:
      - "10811:8889"
    depends_on:
      - bobing
      - user
      - school
      - year_bill
  db:
    image: mysql:5.7.18
    restart: unless-stopped
    environment:
      - MYSQL_ROOT_PASSWORD=yourpassword
      - MYSQL_PASSWORD=yourpassword
    ports:
      - "10821:3306"
  redis:
    image: redis:7.0.12-alpine
    restart: unless-stopped
    ports:
      - "10916:6379"
  etcd:
    image: bitnami/etcd:3.4.15
    restart: unless-stopped
    ports:
      - "62379:2379"
  rabbitmq:
    image: rabbitmq:3-management
    restart: unless-stopped
    ports:
      - "5672:5672"
      - "15672:15672"
```

## 部署与运行
### 1. **克隆项目**
```sh
git clone https://github.com/LDZ6/SendsPlatfrom.git
cd SendsPlatfrom
```

### 2. **配置环境变量**
修改 `.env` 文件，填入 MySQL、Redis、RabbitMQ 相关配置。

### 3. **使用 Docker Compose 启动服务**
```sh
docker-compose up -d
```

### 4. **访问 API 文档**
Swagger API 文档默认开放在 `http://localhost:8080/swagger/index.html`

## 代码结构
```
SendsPlatfrom/
│── app/
│   ├── user/        # 用户微服务
│   ├── boBing/      # 博饼游戏微服务
│   ├── school/      # 教务处微服务
│   ├── yearBill/    # 年度总结微服务
│   ├── gateway/     # API 网关
│── config/          # 配置文件
│── docs/            # Swagger API 文档
│── scripts/         # 部署脚本
│── docker-compose.yml # 容器化配置
│── main.go          # 入口文件
```

## 项目收获
- 深入掌握 **微服务架构设计与实现**。
- 提升 **关系型数据库与非关系型数据库优化能力**。
- 练习 **高并发编程与异步处理技术**。
- 积累 **微信公众号开发经验**。
- 通过实际应用 **提高用户体验**，获得广泛好评。

## 贡献指南
如果你有兴趣为该项目贡献代码或提出改进意见，请参考 [贡献指南](CONTRIBUTING.md)。

## 许可证
本项目遵循 [MIT 许可证](LICENSE)。

