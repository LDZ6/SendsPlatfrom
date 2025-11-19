# TCC架构优化实现

本项目基于GoTcc框架实现了TCC（Try-Confirm-Cancel）分布式事务架构，用于优化项目中的复杂业务场景。

## 架构概述

TCC架构通过三个阶段来保证分布式事务的一致性：

1. **Try阶段**：尝试执行业务操作，但不提交，只是预留资源
2. **Confirm阶段**：确认提交，正式执行业务操作
3. **Cancel阶段**：取消操作，释放预留的资源

## 项目中的TCC优化场景

### 1. YearBill数据初始化

**优化前的问题：**
- 支付数据初始化和学习数据初始化是并发执行的
- 如果其中一个失败，另一个已经成功的数据无法回滚
- 数据一致性无法保证

**TCC优化后：**
- 将支付数据初始化、学习数据初始化、排名更新分别封装为TCC组件
- 通过TCC事务管理器协调三个组件的执行
- 保证数据的一致性和原子性

**相关文件：**
- `app/yearBill/service/tcc_components.go` - YearBill TCC组件实现
- `app/yearBill/service/tcc_integration.go` - YearBill TCC服务集成

### 2. BoBing投掷操作

**优化前的问题：**
- 投掷操作涉及排名更新、记录保存、缓存更新等多个步骤
- 如果中间步骤失败，已执行的步骤无法回滚
- 投掷结果的原子性无法保证

**TCC优化后：**
- 将排名更新、投掷记录、记录保存分别封装为TCC组件
- 通过TCC事务管理器协调组件的执行
- 保证投掷操作的原子性

**相关文件：**
- `app/boBing/service/tcc_components.go` - BoBing TCC组件实现
- `app/boBing/service/tcc_integration.go` - BoBing TCC服务集成

### 3. 用户登录流程

**优化前的问题：**
- 用户登录涉及微信登录、学号获取、用户信息获取等多个步骤
- 如果中间步骤失败，已获取的信息无法回滚
- 用户数据的完整性无法保证

**TCC优化后：**
- 将用户登录、学校用户登录、群众登录分别封装为TCC组件
- 通过TCC事务管理器协调组件的执行
- 保证用户数据的完整性

**相关文件：**
- `app/user/service/tcc_components.go` - User TCC组件实现
- `app/user/service/tcc_integration.go` - User TCC服务集成

## 核心组件

### 1. TXStore - 事务日志存储

**功能：**
- 记录事务的创建、更新、提交状态
- 支持MySQL和Redis双重存储
- 提供分布式锁机制

**实现文件：**
- `app/common/tcc/txstore.go`

### 2. TXManager - 事务管理器

**功能：**
- 协调TCC组件的执行
- 管理事务的生命周期
- 提供监控和恢复机制

**实现文件：**
- `app/common/tcc/txmanager.go`

### 3. Registry - 组件注册中心

**功能：**
- 管理TCC组件的注册和发现
- 提供组件查找功能
- 支持组件的动态注册和注销

**实现文件：**
- `app/common/tcc/registry.go`

## 使用方法

### 1. 初始化TCC服务

```go
// 创建YearBill TCC服务
yearBillTCCService := NewYearBillTCCService(db, redisClient)
defer yearBillTCCService.Stop()

// 创建BoBing TCC服务
boBingTCCService := NewBoBingTCCService(db, redisClient)
defer boBingTCCService.Stop()

// 创建User TCC服务
userTCCService := NewUserTCCService(db, redisClient)
defer userTCCService.Stop()
```

### 2. 执行TCC事务

```go
// YearBill数据初始化
task := model.DadaInitTask{
    JsSessionId:  "test_js_session",
    HallTicket:   "test_hall_ticket",
    GsSession:    "test_gs_session",
    Emaphome_WEU: "test_emaphome_weu",
    StuNum:       "2125102013",
}

err := yearBillTCCService.DataInitWithTCC(ctx, task)

// BoBing投掷操作
req := &BoBingPb.BoBingPublishRequest{
    OpenId:   "test_open_id",
    StuNum:   "2125102013",
    NickName: "test_user",
    Flag:     "10",
    Check:    "test_check",
}

err = boBingTCCService.BoBingPublishWithTCC(ctx, req)

// 用户登录
loginReq := &userPb.UserLoginRequest{
    Code: "test_code",
}

err = userTCCService.UserLoginWithTCC(ctx, loginReq)
```

### 3. 查询事务状态

```go
// 获取事务状态
tx, err := tccService.GetTransactionStatus(ctx, txID)
if err != nil {
    // 处理错误
}

// 检查事务状态
switch tx.Status {
case "hanging":
    // 事务执行中
case "successful":
    // 事务成功
case "failure":
    // 事务失败
}
```

## 配置选项

### TXManager配置

```go
txManager := tcc.NewTXManager(txStore, 
    tcc.WithTimeout(30*time.Second),      // 事务超时时间
    tcc.WithMonitorTick(5*time.Second),   // 监控间隔
)
```

### 数据库表结构

需要创建事务记录表：

```sql
CREATE TABLE transactions (
    tx_id VARCHAR(255) PRIMARY KEY,
    components JSON NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

## 监控和恢复

### 1. 自动监控

TXManager会自动监控未完成的事务，定期检查并推进事务状态。

### 2. 超时处理

超过指定时间的事务会被自动标记为失败，并执行Cancel操作。

### 3. 故障恢复

系统重启后，TXManager会自动恢复未完成的事务，继续执行或回滚。

## 性能优化

### 1. 并发执行

TCC组件在Try阶段可以并发执行，提高性能。

### 2. 缓存优化

使用Redis缓存事务状态，减少数据库查询。

### 3. 批量处理

支持批量处理多个事务，提高吞吐量。

## 注意事项

1. **幂等性**：TCC组件必须支持幂等操作，避免重复执行
2. **超时设置**：合理设置事务超时时间，避免长时间占用资源
3. **监控告警**：建议配置监控告警，及时发现和处理异常事务
4. **数据一致性**：确保TCC组件的Cancel操作能够正确回滚数据

## 扩展性

### 1. 新增TCC组件

```go
// 实现TCCComponent接口
type NewTCCComponent struct {
    // 组件字段
}

func (c *NewTCCComponent) ID() string {
    return "new_component"
}

func (c *NewTCCComponent) Try(ctx context.Context, req *TCCReq) (*TCCResp, error) {
    // Try阶段实现
}

func (c *NewTCCComponent) Confirm(ctx context.Context, txID string) (*TCCResp, error) {
    // Confirm阶段实现
}

func (c *NewTCCComponent) Cancel(ctx context.Context, txID string) (*TCCResp, error) {
    // Cancel阶段实现
}

// 注册组件
txManager.Register(&NewTCCComponent{})
```

### 2. 自定义TXStore

可以实现自定义的TXStore来支持其他存储后端。

### 3. 监控集成

可以集成Prometheus、Grafana等监控系统来监控TCC事务的执行情况。
