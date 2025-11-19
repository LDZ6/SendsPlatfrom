package performance

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// DatabaseRouter 数据库路由器
type DatabaseRouter struct {
	masterDB *sql.DB
	slaveDBs []*sql.DB
	strategy RoutingStrategy
	mutex    sync.RWMutex
	stats    *RouterStats
}

// RouterStats 路由器统计
type RouterStats struct {
	MasterQueries int64
	SlaveQueries  int64
	TotalQueries  int64
	ErrorCount    int64
	LastUpdate    time.Time
}

// RoutingStrategy 路由策略接口
type RoutingStrategy interface {
	Route(operation string, query string) (bool, error) // 返回是否使用主库
	GetSlaveIndex() int
	SetSlaveHealth(index int, healthy bool)
	GetSlaveHealth(index int) bool
}

// NewDatabaseRouter 创建数据库路由器
func NewDatabaseRouter(masterDB *sql.DB, slaveDBs []*sql.DB, strategy RoutingStrategy) *DatabaseRouter {
	return &DatabaseRouter{
		masterDB: masterDB,
		slaveDBs: slaveDBs,
		strategy: strategy,
		stats:    &RouterStats{},
	}
}

// Query 执行查询
func (dr *DatabaseRouter) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	// 判断是否使用主库
	useMaster, err := dr.strategy.Route("SELECT", query)
	if err != nil {
		return nil, err
	}

	if useMaster {
		return dr.queryMaster(ctx, query, args...)
	}

	return dr.querySlave(ctx, query, args...)
}

// QueryRow 执行单行查询
func (dr *DatabaseRouter) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	// 判断是否使用主库
	useMaster, err := dr.strategy.Route("SELECT", query)
	if err != nil {
		// 出错时使用主库
		return dr.queryRowMaster(ctx, query, args...)
	}

	if useMaster {
		return dr.queryRowMaster(ctx, query, args...)
	}

	return dr.queryRowSlave(ctx, query, args...)
}

// Exec 执行更新操作
func (dr *DatabaseRouter) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	// 更新操作总是使用主库
	return dr.execMaster(ctx, query, args...)
}

// queryMaster 查询主库
func (dr *DatabaseRouter) queryMaster(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	dr.mutex.Lock()
	dr.stats.MasterQueries++
	dr.stats.TotalQueries++
	dr.mutex.Unlock()

	return dr.masterDB.QueryContext(ctx, query, args...)
}

// querySlave 查询从库
func (dr *DatabaseRouter) querySlave(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	slaveIndex := dr.strategy.GetSlaveIndex()
	if slaveIndex >= len(dr.slaveDBs) {
		// 从库索引超出范围，使用主库
		return dr.queryMaster(ctx, query, args...)
	}

	slaveDB := dr.slaveDBs[slaveIndex]
	if !dr.strategy.GetSlaveHealth(slaveIndex) {
		// 从库不健康，使用主库
		return dr.queryMaster(ctx, query, args...)
	}

	dr.mutex.Lock()
	dr.stats.SlaveQueries++
	dr.stats.TotalQueries++
	dr.mutex.Unlock()

	rows, err := slaveDB.QueryContext(ctx, query, args...)
	if err != nil {
		// 查询失败，标记从库为不健康
		dr.strategy.SetSlaveHealth(slaveIndex, false)
		dr.mutex.Lock()
		dr.stats.ErrorCount++
		dr.mutex.Unlock()
	}

	return rows, err
}

// queryRowMaster 查询主库单行
func (dr *DatabaseRouter) queryRowMaster(ctx context.Context, query string, args ...interface{}) *sql.Row {
	dr.mutex.Lock()
	dr.stats.MasterQueries++
	dr.stats.TotalQueries++
	dr.mutex.Unlock()

	return dr.masterDB.QueryRowContext(ctx, query, args...)
}

// queryRowSlave 查询从库单行
func (dr *DatabaseRouter) queryRowSlave(ctx context.Context, query string, args ...interface{}) *sql.Row {
	slaveIndex := dr.strategy.GetSlaveIndex()
	if slaveIndex >= len(dr.slaveDBs) {
		// 从库索引超出范围，使用主库
		return dr.queryRowMaster(ctx, query, args...)
	}

	slaveDB := dr.slaveDBs[slaveIndex]
	if !dr.strategy.GetSlaveHealth(slaveIndex) {
		// 从库不健康，使用主库
		return dr.queryRowMaster(ctx, query, args...)
	}

	dr.mutex.Lock()
	dr.stats.SlaveQueries++
	dr.stats.TotalQueries++
	dr.mutex.Unlock()

	return slaveDB.QueryRowContext(ctx, query, args...)
}

// execMaster 执行主库更新
func (dr *DatabaseRouter) execMaster(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	dr.mutex.Lock()
	dr.stats.MasterQueries++
	dr.stats.TotalQueries++
	dr.mutex.Unlock()

	return dr.masterDB.ExecContext(ctx, query, args...)
}

// GetStats 获取统计信息
func (dr *DatabaseRouter) GetStats() *RouterStats {
	dr.mutex.RLock()
	defer dr.mutex.RUnlock()

	return &RouterStats{
		MasterQueries: dr.stats.MasterQueries,
		SlaveQueries:  dr.stats.SlaveQueries,
		TotalQueries:  dr.stats.TotalQueries,
		ErrorCount:    dr.stats.ErrorCount,
		LastUpdate:    dr.stats.LastUpdate,
	}
}

// RoundRobinStrategy 轮询策略
type RoundRobinStrategy struct {
	currentIndex int
	slaveHealth  []bool
	mutex        sync.RWMutex
}

// NewRoundRobinStrategy 创建轮询策略
func NewRoundRobinStrategy(slaveCount int) *RoundRobinStrategy {
	return &RoundRobinStrategy{
		slaveHealth: make([]bool, slaveCount),
	}
}

// Route 路由决策
func (rrs *RoundRobinStrategy) Route(operation string, query string) (bool, error) {
	// 写操作使用主库
	if operation == "INSERT" || operation == "UPDATE" || operation == "DELETE" {
		return true, nil
	}

	// 读操作使用从库
	return false, nil
}

// GetSlaveIndex 获取从库索引
func (rrs *RoundRobinStrategy) GetSlaveIndex() int {
	rrs.mutex.Lock()
	defer rrs.mutex.Unlock()

	// 找到下一个健康的从库
	for i := 0; i < len(rrs.slaveHealth); i++ {
		rrs.currentIndex = (rrs.currentIndex + 1) % len(rrs.slaveHealth)
		if rrs.slaveHealth[rrs.currentIndex] {
			return rrs.currentIndex
		}
	}

	// 没有健康的从库，返回第一个
	return 0
}

// SetSlaveHealth 设置从库健康状态
func (rrs *RoundRobinStrategy) SetSlaveHealth(index int, healthy bool) {
	rrs.mutex.Lock()
	defer rrs.mutex.Unlock()

	if index >= 0 && index < len(rrs.slaveHealth) {
		rrs.slaveHealth[index] = healthy
	}
}

// GetSlaveHealth 获取从库健康状态
func (rrs *RoundRobinStrategy) GetSlaveHealth(index int) bool {
	rrs.mutex.RLock()
	defer rrs.mutex.RUnlock()

	if index >= 0 && index < len(rrs.slaveHealth) {
		return rrs.slaveHealth[index]
	}

	return false
}

// WeightedRoundRobinStrategy 加权轮询策略
type WeightedRoundRobinStrategy struct {
	weights      []int
	currentIndex int
	slaveHealth  []bool
	mutex        sync.RWMutex
}

// NewWeightedRoundRobinStrategy 创建加权轮询策略
func NewWeightedRoundRobinStrategy(weights []int) *WeightedRoundRobinStrategy {
	return &WeightedRoundRobinStrategy{
		weights:     weights,
		slaveHealth: make([]bool, len(weights)),
	}
}

// Route 路由决策
func (wrrs *WeightedRoundRobinStrategy) Route(operation string, query string) (bool, error) {
	// 写操作使用主库
	if operation == "INSERT" || operation == "UPDATE" || operation == "DELETE" {
		return true, nil
	}

	// 读操作使用从库
	return false, nil
}

// GetSlaveIndex 获取从库索引
func (wrrs *WeightedRoundRobinStrategy) GetSlaveIndex() int {
	wrrs.mutex.Lock()
	defer wrrs.mutex.Unlock()

	// 找到下一个健康的从库
	for i := 0; i < len(wrrs.slaveHealth); i++ {
		wrrs.currentIndex = (wrrs.currentIndex + 1) % len(wrrs.slaveHealth)
		if wrrs.slaveHealth[wrrs.currentIndex] {
			return wrrs.currentIndex
		}
	}

	// 没有健康的从库，返回第一个
	return 0
}

// SetSlaveHealth 设置从库健康状态
func (wrrs *WeightedRoundRobinStrategy) SetSlaveHealth(index int, healthy bool) {
	wrrs.mutex.Lock()
	defer wrrs.mutex.Unlock()

	if index >= 0 && index < len(wrrs.slaveHealth) {
		wrrs.slaveHealth[index] = healthy
	}
}

// GetSlaveHealth 获取从库健康状态
func (wrrs *WeightedRoundRobinStrategy) GetSlaveHealth(index int) bool {
	wrrs.mutex.RLock()
	defer wrrs.mutex.RUnlock()

	if index >= 0 && index < len(wrrs.slaveHealth) {
		return wrrs.slaveHealth[index]
	}

	return false
}

// ShardingStrategy 分片策略接口
type ShardingStrategy interface {
	GetShard(key string) int
	GetTable(key string) string
	GetShardCount() int
}

// HashShardingStrategy 哈希分片策略
type HashShardingStrategy struct {
	shardCount  int
	tablePrefix string
}

// NewHashShardingStrategy 创建哈希分片策略
func NewHashShardingStrategy(shardCount int, tablePrefix string) *HashShardingStrategy {
	return &HashShardingStrategy{
		shardCount:  shardCount,
		tablePrefix: tablePrefix,
	}
}

// GetShard 获取分片索引
func (hss *HashShardingStrategy) GetShard(key string) int {
	hash := 0
	for _, c := range key {
		hash = hash*31 + int(c)
	}
	return hash % hss.shardCount
}

// GetTable 获取表名
func (hss *HashShardingStrategy) GetTable(key string) int {
	shard := hss.GetShard(key)
	return fmt.Sprintf("%s_%d", hss.tablePrefix, shard)
}

// GetShardCount 获取分片数量
func (hss *HashShardingStrategy) GetShardCount() int {
	return hss.shardCount
}

// RangeShardingStrategy 范围分片策略
type RangeShardingStrategy struct {
	ranges      []int64
	tablePrefix string
}

// NewRangeShardingStrategy 创建范围分片策略
func NewRangeShardingStrategy(ranges []int64, tablePrefix string) *RangeShardingStrategy {
	return &RangeShardingStrategy{
		ranges:      ranges,
		tablePrefix: tablePrefix,
	}
}

// GetShard 获取分片索引
func (rss *RangeShardingStrategy) GetShard(key string) int {
	// 这里需要根据key解析出数值
	// 简化实现，使用哈希
	hash := 0
	for _, c := range key {
		hash = hash*31 + int(c)
	}

	value := int64(hash)
	for i, rangeValue := range rss.ranges {
		if value < rangeValue {
			return i
		}
	}

	return len(rss.ranges) - 1
}

// GetTable 获取表名
func (rss *RangeShardingStrategy) GetTable(key string) int {
	shard := rss.GetShard(key)
	return fmt.Sprintf("%s_%d", rss.tablePrefix, shard)
}

// GetShardCount 获取分片数量
func (rss *RangeShardingStrategy) GetShardCount() int {
	return len(rss.ranges)
}

// DatabaseSharding 数据库分片
type DatabaseSharding struct {
	databases []*sql.DB
	strategy  ShardingStrategy
	stats     *ShardingStats
	mutex     sync.RWMutex
}

// ShardingStats 分片统计
type ShardingStats struct {
	ShardQueries []int64
	TotalQueries int64
	ErrorCount   int64
	LastUpdate   time.Time
}

// NewDatabaseSharding 创建数据库分片
func NewDatabaseSharding(databases []*sql.DB, strategy ShardingStrategy) *DatabaseSharding {
	return &DatabaseSharding{
		databases: databases,
		strategy:  strategy,
		stats:     &ShardingStats{ShardQueries: make([]int64, len(databases))},
	}
}

// Query 执行查询
func (ds *DatabaseSharding) Query(ctx context.Context, key string, query string, args ...interface{}) (*sql.Rows, error) {
	shard := ds.strategy.GetShard(key)
	if shard >= len(ds.databases) {
		return nil, fmt.Errorf("shard index out of range: %d", shard)
	}

	db := ds.databases[shard]
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		ds.mutex.Lock()
		ds.stats.ErrorCount++
		ds.mutex.Unlock()
		return nil, err
	}

	ds.mutex.Lock()
	ds.stats.ShardQueries[shard]++
	ds.stats.TotalQueries++
	ds.mutex.Unlock()

	return rows, nil
}

// QueryRow 执行单行查询
func (ds *DatabaseSharding) QueryRow(ctx context.Context, key string, query string, args ...interface{}) *sql.Row {
	shard := ds.strategy.GetShard(key)
	if shard >= len(ds.databases) {
		// 返回错误行
		return &sql.Row{}
	}

	db := ds.databases[shard]
	row := db.QueryRowContext(ctx, query, args...)

	ds.mutex.Lock()
	ds.stats.ShardQueries[shard]++
	ds.stats.TotalQueries++
	ds.mutex.Unlock()

	return row
}

// Exec 执行更新操作
func (ds *DatabaseSharding) Exec(ctx context.Context, key string, query string, args ...interface{}) (sql.Result, error) {
	shard := ds.strategy.GetShard(key)
	if shard >= len(ds.databases) {
		return nil, fmt.Errorf("shard index out of range: %d", shard)
	}

	db := ds.databases[shard]
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		ds.mutex.Lock()
		ds.stats.ErrorCount++
		ds.mutex.Unlock()
		return nil, err
	}

	ds.mutex.Lock()
	ds.stats.ShardQueries[shard]++
	ds.stats.TotalQueries++
	ds.mutex.Unlock()

	return result, nil
}

// GetStats 获取统计信息
func (ds *DatabaseSharding) GetStats() *ShardingStats {
	ds.mutex.RLock()
	defer ds.mutex.RUnlock()

	shardQueries := make([]int64, len(ds.stats.ShardQueries))
	copy(shardQueries, ds.stats.ShardQueries)

	return &ShardingStats{
		ShardQueries: shardQueries,
		TotalQueries: ds.stats.TotalQueries,
		ErrorCount:   ds.stats.ErrorCount,
		LastUpdate:   ds.stats.LastUpdate,
	}
}
