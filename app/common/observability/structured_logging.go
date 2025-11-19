package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String 日志级别字符串
func (ll LogLevel) String() string {
	switch ll {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel 解析日志级别
func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return LogLevelDebug
	case "INFO":
		return LogLevelInfo
	case "WARN":
		return LogLevelWarn
	case "ERROR":
		return LogLevelError
	case "FATAL":
		return LogLevelFatal
	default:
		return LogLevelInfo
	}
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
	Service   string                 `json:"service,omitempty"`
	Version   string                 `json:"version,omitempty"`
	Hostname  string                 `json:"hostname,omitempty"`
	PID       int                    `json:"pid,omitempty"`
}

// StructuredLogger 结构化日志器
type StructuredLogger struct {
	level      LogLevel
	output     io.Writer
	fields     map[string]interface{}
	formatter  LogFormatter
	aggregator LogAggregator
	mutex      sync.RWMutex
}

// LogFormatter 日志格式化器接口
type LogFormatter interface {
	Format(entry *LogEntry) ([]byte, error)
}

// LogAggregator 日志聚合器接口
type LogAggregator interface {
	Aggregate(entry *LogEntry) error
	Flush() error
	Close() error
}

// JSONFormatter JSON格式化器
type JSONFormatter struct{}

// Format 格式化日志条目
func (jf *JSONFormatter) Format(entry *LogEntry) ([]byte, error) {
	return json.Marshal(entry)
}

// TextFormatter 文本格式化器
type TextFormatter struct {
	TimestampFormat string
	DisableColors   bool
}

// NewTextFormatter 创建文本格式化器
func NewTextFormatter() *TextFormatter {
	return &TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		DisableColors:   false,
	}
}

// Format 格式化日志条目
func (tf *TextFormatter) Format(entry *LogEntry) ([]byte, error) {
	var sb strings.Builder

	// 时间戳
	sb.WriteString(entry.Timestamp.Format(tf.TimestampFormat))
	sb.WriteString(" ")

	// 日志级别
	sb.WriteString(entry.Level.String())
	sb.WriteString(" ")

	// 调用者
	if entry.Caller != "" {
		sb.WriteString(entry.Caller)
		sb.WriteString(" ")
	}

	// 消息
	sb.WriteString(entry.Message)

	// 字段
	if len(entry.Fields) > 0 {
		sb.WriteString(" ")
		for k, v := range entry.Fields {
			sb.WriteString(fmt.Sprintf("%s=%v ", k, v))
		}
	}

	// 追踪信息
	if entry.TraceID != "" {
		sb.WriteString(fmt.Sprintf("trace_id=%s ", entry.TraceID))
	}
	if entry.SpanID != "" {
		sb.WriteString(fmt.Sprintf("span_id=%s ", entry.SpanID))
	}

	sb.WriteString("\n")

	return []byte(sb.String()), nil
}

// MemoryLogAggregator 内存日志聚合器
type MemoryLogAggregator struct {
	logs  []*LogEntry
	mutex sync.RWMutex
}

// NewMemoryLogAggregator 创建内存日志聚合器
func NewMemoryLogAggregator() *MemoryLogAggregator {
	return &MemoryLogAggregator{
		logs: make([]*LogEntry, 0),
	}
}

// Aggregate 聚合日志
func (mla *MemoryLogAggregator) Aggregate(entry *LogEntry) error {
	mla.mutex.Lock()
	defer mla.mutex.Unlock()

	mla.logs = append(mla.logs, entry)

	// 保持最近10000条日志
	if len(mla.logs) > 10000 {
		mla.logs = mla.logs[len(mla.logs)-10000:]
	}

	return nil
}

// Flush 刷新日志
func (mla *MemoryLogAggregator) Flush() error {
	// 内存聚合器不需要刷新
	return nil
}

// Close 关闭聚合器
func (mla *MemoryLogAggregator) Close() error {
	mla.mutex.Lock()
	defer mla.mutex.Unlock()

	mla.logs = nil
	return nil
}

// GetLogs 获取日志
func (mla *MemoryLogAggregator) GetLogs() []*LogEntry {
	mla.mutex.RLock()
	defer mla.mutex.RUnlock()

	logs := make([]*LogEntry, len(mla.logs))
	copy(logs, mla.logs)

	return logs
}

// GetLogsByLevel 根据级别获取日志
func (mla *MemoryLogAggregator) GetLogsByLevel(level LogLevel) []*LogEntry {
	mla.mutex.RLock()
	defer mla.mutex.RUnlock()

	var logs []*LogEntry
	for _, log := range mla.logs {
		if log.Level == level {
			logs = append(logs, log)
		}
	}

	return logs
}

// GetLogsByTimeRange 根据时间范围获取日志
func (mla *MemoryLogAggregator) GetLogsByTimeRange(start, end time.Time) []*LogEntry {
	mla.mutex.RLock()
	defer mla.mutex.RUnlock()

	var logs []*LogEntry
	for _, log := range mla.logs {
		if log.Timestamp.After(start) && log.Timestamp.Before(end) {
			logs = append(logs, log)
		}
	}

	return logs
}

// ElasticsearchLogAggregator Elasticsearch日志聚合器
type ElasticsearchLogAggregator struct {
	url    string
	index  string
	client interface{} // Elasticsearch客户端
	buffer []*LogEntry
	mutex  sync.RWMutex
}

// NewElasticsearchLogAggregator 创建Elasticsearch日志聚合器
func NewElasticsearchLogAggregator(url, index string) *ElasticsearchLogAggregator {
	return &ElasticsearchLogAggregator{
		url:    url,
		index:  index,
		buffer: make([]*LogEntry, 0),
	}
}

// Aggregate 聚合日志
func (ela *ElasticsearchLogAggregator) Aggregate(entry *LogEntry) error {
	ela.mutex.Lock()
	defer ela.mutex.Unlock()

	ela.buffer = append(ela.buffer, entry)

	// 当缓冲区达到1000条时，批量发送
	if len(ela.buffer) >= 1000 {
		return ela.flushBuffer()
	}

	return nil
}

// Flush 刷新日志
func (ela *ElasticsearchLogAggregator) Flush() error {
	ela.mutex.Lock()
	defer ela.mutex.Unlock()

	return ela.flushBuffer()
}

// Close 关闭聚合器
func (ela *ElasticsearchLogAggregator) Close() error {
	return ela.Flush()
}

// flushBuffer 刷新缓冲区
func (ela *ElasticsearchLogAggregator) flushBuffer() error {
	if len(ela.buffer) == 0 {
		return nil
	}

	// 这里应该实现实际的Elasticsearch批量插入逻辑
	// 简化实现，清空缓冲区
	ela.buffer = ela.buffer[:0]

	return nil
}

// NewStructuredLogger 创建结构化日志器
func NewStructuredLogger(level LogLevel, output io.Writer) *StructuredLogger {
	return &StructuredLogger{
		level:      level,
		output:     output,
		fields:     make(map[string]interface{}),
		formatter:  &JSONFormatter{},
		aggregator: NewMemoryLogAggregator(),
	}
}

// SetLevel 设置日志级别
func (sl *StructuredLogger) SetLevel(level LogLevel) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	sl.level = level
}

// SetFormatter 设置格式化器
func (sl *StructuredLogger) SetFormatter(formatter LogFormatter) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	sl.formatter = formatter
}

// SetAggregator 设置聚合器
func (sl *StructuredLogger) SetAggregator(aggregator LogAggregator) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	sl.aggregator = aggregator
}

// WithField 添加字段
func (sl *StructuredLogger) WithField(key string, value interface{}) *StructuredLogger {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	newLogger := &StructuredLogger{
		level:      sl.level,
		output:     sl.output,
		fields:     make(map[string]interface{}),
		formatter:  sl.formatter,
		aggregator: sl.aggregator,
	}

	// 复制现有字段
	for k, v := range sl.fields {
		newLogger.fields[k] = v
	}

	// 添加新字段
	newLogger.fields[key] = value

	return newLogger
}

// WithFields 添加多个字段
func (sl *StructuredLogger) WithFields(fields map[string]interface{}) *StructuredLogger {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	newLogger := &StructuredLogger{
		level:      sl.level,
		output:     sl.output,
		fields:     make(map[string]interface{}),
		formatter:  sl.formatter,
		aggregator: sl.aggregator,
	}

	// 复制现有字段
	for k, v := range sl.fields {
		newLogger.fields[k] = v
	}

	// 添加新字段
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// WithContext 添加上下文
func (sl *StructuredLogger) WithContext(ctx context.Context) *StructuredLogger {
	// 从上下文中提取追踪信息
	traceID := ctx.Value("trace_id")
	spanID := ctx.Value("span_id")

	newLogger := sl.WithFields(map[string]interface{}{
		"trace_id": traceID,
		"span_id":  spanID,
	})

	return newLogger
}

// log 记录日志
func (sl *StructuredLogger) log(level LogLevel, msg string, fields map[string]interface{}) {
	if level < sl.level {
		return
	}

	// 获取调用者信息
	_, file, line, ok := runtime.Caller(2)
	caller := ""
	if ok {
		caller = fmt.Sprintf("%s:%d", file, line)
	}

	// 创建日志条目
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
		Fields:    make(map[string]interface{}),
		Caller:    caller,
		Service:   "platform",
		Version:   "1.0.0",
		Hostname:  getHostname(),
		PID:       os.Getpid(),
	}

	// 复制字段
	for k, v := range sl.fields {
		entry.Fields[k] = v
	}
	for k, v := range fields {
		entry.Fields[k] = v
	}

	// 格式化日志
	formatted, err := sl.formatter.Format(entry)
	if err != nil {
		// 如果格式化失败，使用简单的文本格式
		formatted = []byte(fmt.Sprintf("%s %s %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"), level.String(), msg))
	}

	// 写入输出
	sl.output.Write(formatted)

	// 聚合日志
	if sl.aggregator != nil {
		sl.aggregator.Aggregate(entry)
	}
}

// Debug 记录调试日志
func (sl *StructuredLogger) Debug(msg string, fields ...map[string]interface{}) {
	mergedFields := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			mergedFields[k] = v
		}
	}
	sl.log(LogLevelDebug, msg, mergedFields)
}

// Info 记录信息日志
func (sl *StructuredLogger) Info(msg string, fields ...map[string]interface{}) {
	mergedFields := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			mergedFields[k] = v
		}
	}
	sl.log(LogLevelInfo, msg, mergedFields)
}

// Warn 记录警告日志
func (sl *StructuredLogger) Warn(msg string, fields ...map[string]interface{}) {
	mergedFields := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			mergedFields[k] = v
		}
	}
	sl.log(LogLevelWarn, msg, mergedFields)
}

// Error 记录错误日志
func (sl *StructuredLogger) Error(msg string, fields ...map[string]interface{}) {
	mergedFields := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			mergedFields[k] = v
		}
	}
	sl.log(LogLevelError, msg, mergedFields)
}

// Fatal 记录致命日志
func (sl *StructuredLogger) Fatal(msg string, fields ...map[string]interface{}) {
	mergedFields := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			mergedFields[k] = v
		}
	}
	sl.log(LogLevelFatal, msg, mergedFields)
	os.Exit(1)
}

// Flush 刷新日志
func (sl *StructuredLogger) Flush() error {
	if sl.aggregator != nil {
		return sl.aggregator.Flush()
	}
	return nil
}

// Close 关闭日志器
func (sl *StructuredLogger) Close() error {
	if sl.aggregator != nil {
		return sl.aggregator.Close()
	}
	return nil
}

// getHostname 获取主机名
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// LogAnalyzer 日志分析器
type LogAnalyzer struct {
	aggregator LogAggregator
}

// NewLogAnalyzer 创建日志分析器
func NewLogAnalyzer(aggregator LogAggregator) *LogAnalyzer {
	return &LogAnalyzer{
		aggregator: aggregator,
	}
}

// Analyze 分析日志
func (la *LogAnalyzer) Analyze() (*LogAnalysis, error) {
	// 这里应该实现实际的日志分析逻辑
	// 简化实现，返回空分析结果
	return &LogAnalysis{
		TotalLogs:    0,
		ErrorCount:   0,
		WarningCount: 0,
		InfoCount:    0,
		DebugCount:   0,
		FatalCount:   0,
		ErrorRate:    0.0,
		TopErrors:    make([]string, 0),
		TopWarnings:  make([]string, 0),
		TimeRange:    TimeRange{},
	}, nil
}

// LogAnalysis 日志分析结果
type LogAnalysis struct {
	TotalLogs    int64     `json:"total_logs"`
	ErrorCount   int64     `json:"error_count"`
	WarningCount int64     `json:"warning_count"`
	InfoCount    int64     `json:"info_count"`
	DebugCount   int64     `json:"debug_count"`
	FatalCount   int64     `json:"fatal_count"`
	ErrorRate    float64   `json:"error_rate"`
	TopErrors    []string  `json:"top_errors"`
	TopWarnings  []string  `json:"top_warnings"`
	TimeRange    TimeRange `json:"time_range"`
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
