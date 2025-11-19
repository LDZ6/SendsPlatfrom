package utils

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// 结构化日志字段
type LogFields struct {
	TraceID    string `json:"traceId,omitempty"`
	SpanID     string `json:"spanId,omitempty"`
	Service    string `json:"service,omitempty"`
	Route      string `json:"route,omitempty"`
	User       string `json:"user,omitempty"` // openId/stuNum
	HTTPStatus int    `json:"httpStatus,omitempty"`
	GRPCCode   string `json:"grpcCode,omitempty"`
	CostMs     int64  `json:"costMs,omitempty"`
	RemoteIP   string `json:"remoteIP,omitempty"`
	Error      string `json:"error,omitempty"`
	KeyID      string `json:"kid,omitempty"` // JWT密钥版本
	Level      string `json:"level"`
	Message    string `json:"message"`
	Timestamp  string `json:"timestamp"`
}

// 日志采样器
type LogSampler struct {
	errorRate  float64
	normalRate float64
}

var (
	logger      *logrus.Logger
	sampler     *LogSampler
	serviceName string
)

// 初始化日志系统
func init() {
	initLogger()
}

func initLogger() {
	logger = logrus.New()

	// 从配置获取日志设置
	level := viper.GetString("observability.logging.level")
	if level == "" {
		level = "info"
	}

	format := viper.GetString("observability.logging.format")
	if format == "" {
		format = "json"
	}

	// 设置日志级别
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// 设置日志格式
	if format == "json" {
		logger.SetFormatter(&StructuredFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
		})
	}

	// 设置输出
	logger.SetOutput(os.Stdout)

	// 初始化采样器
	initSampler()

	// 设置服务名
	serviceName = viper.GetString("observability.tracing.serviceName")
	if serviceName == "" {
		serviceName = "sends-platform"
	}
}

// 初始化采样器
func initSampler() {
	errorRate := viper.GetFloat64("observability.logging.sampling.errorRate")
	if errorRate <= 0 {
		errorRate = 1.0 // 错误日志全量记录
	}

	normalRate := viper.GetFloat64("observability.logging.sampling.normalRate")
	if normalRate <= 0 {
		normalRate = 0.1 // 正常日志10%采样
	}

	sampler = &LogSampler{
		errorRate:  errorRate,
		normalRate: normalRate,
	}
}

// 结构化日志格式化器
type StructuredFormatter struct{}

func (f *StructuredFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	fields := LogFields{
		Level:     entry.Level.String(),
		Message:   entry.Message,
		Timestamp: entry.Time.Format(time.RFC3339),
	}

	// 提取常用字段
	if traceID, ok := entry.Data["traceId"].(string); ok {
		fields.TraceID = traceID
	}
	if spanID, ok := entry.Data["spanId"].(string); ok {
		fields.SpanID = spanID
	}
	if service, ok := entry.Data["service"].(string); ok {
		fields.Service = service
	} else {
		fields.Service = serviceName
	}
	if route, ok := entry.Data["route"].(string); ok {
		fields.Route = route
	}
	if user, ok := entry.Data["user"].(string); ok {
		fields.User = maskSensitiveData(user)
	}
	if httpStatus, ok := entry.Data["httpStatus"].(int); ok {
		fields.HTTPStatus = httpStatus
	}
	if grpcCode, ok := entry.Data["grpcCode"].(string); ok {
		fields.GRPCCode = grpcCode
	}
	if costMs, ok := entry.Data["costMs"].(int64); ok {
		fields.CostMs = costMs
	}
	if remoteIP, ok := entry.Data["remoteIP"].(string); ok {
		fields.RemoteIP = remoteIP
	}
	if err, ok := entry.Data["error"].(string); ok {
		fields.Error = err
	}
	if keyID, ok := entry.Data["kid"].(string); ok {
		fields.KeyID = keyID
	}

	return json.Marshal(fields)
}

// 脱敏处理
func maskSensitiveData(data string) string {
	if len(data) <= 4 {
		return "***"
	}

	// 保留前4位，其余用*替代
	masked := data[:4] + strings.Repeat("*", len(data)-4)
	return masked
}

// 采样决策
func (s *LogSampler) shouldSample(level logrus.Level) bool {
	if level >= logrus.ErrorLevel {
		return rand.Float64() < s.errorRate
	}
	return rand.Float64() < s.normalRate
}

// 带上下文的日志记录器
type ContextLogger struct {
	ctx    context.Context
	fields logrus.Fields
}

// 从上下文创建日志记录器
func WithContext(ctx context.Context) *ContextLogger {
	fields := make(logrus.Fields)

	// 从上下文提取trace信息
	if traceID := getTraceIDFromContext(ctx); traceID != "" {
		fields["traceId"] = traceID
	}
	if spanID := getSpanIDFromContext(ctx); spanID != "" {
		fields["spanId"] = spanID
	}

	fields["service"] = serviceName

	return &ContextLogger{
		ctx:    ctx,
		fields: fields,
	}
}

// 添加字段
func (cl *ContextLogger) WithFields(fields logrus.Fields) *ContextLogger {
	newFields := make(logrus.Fields)
	for k, v := range cl.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &ContextLogger{
		ctx:    cl.ctx,
		fields: newFields,
	}
}

// 添加用户信息
func (cl *ContextLogger) WithUser(userID string) *ContextLogger {
	return cl.WithFields(logrus.Fields{
		"user": maskSensitiveData(userID),
	})
}

// 添加路由信息
func (cl *ContextLogger) WithRoute(route string) *ContextLogger {
	return cl.WithFields(logrus.Fields{
		"route": route,
	})
}

// 添加HTTP状态码
func (cl *ContextLogger) WithHTTPStatus(status int) *ContextLogger {
	return cl.WithFields(logrus.Fields{
		"httpStatus": status,
	})
}

// 添加gRPC状态码
func (cl *ContextLogger) WithGRPCCode(code string) *ContextLogger {
	return cl.WithFields(logrus.Fields{
		"grpcCode": code,
	})
}

// 添加耗时
func (cl *ContextLogger) WithCost(cost time.Duration) *ContextLogger {
	return cl.WithFields(logrus.Fields{
		"costMs": cost.Milliseconds(),
	})
}

// 添加远程IP
func (cl *ContextLogger) WithRemoteIP(ip string) *ContextLogger {
	return cl.WithFields(logrus.Fields{
		"remoteIP": ip,
	})
}

// 添加错误信息
func (cl *ContextLogger) WithError(err error) *ContextLogger {
	return cl.WithFields(logrus.Fields{
		"error": err.Error(),
	})
}

// 添加JWT密钥ID
func (cl *ContextLogger) WithKeyID(keyID string) *ContextLogger {
	return cl.WithFields(logrus.Fields{
		"kid": keyID,
	})
}

// 记录日志
func (cl *ContextLogger) log(level logrus.Level, msg string) {
	if !sampler.shouldSample(level) {
		return
	}

	entry := logger.WithFields(cl.fields)
	switch level {
	case logrus.DebugLevel:
		entry.Debug(msg)
	case logrus.InfoLevel:
		entry.Info(msg)
	case logrus.WarnLevel:
		entry.Warn(msg)
	case logrus.ErrorLevel:
		entry.Error(msg)
	case logrus.FatalLevel:
		entry.Fatal(msg)
	case logrus.PanicLevel:
		entry.Panic(msg)
	}
}

// 便捷方法
func (cl *ContextLogger) Debug(msg string) {
	cl.log(logrus.DebugLevel, msg)
}

func (cl *ContextLogger) Info(msg string) {
	cl.log(logrus.InfoLevel, msg)
}

func (cl *ContextLogger) Warn(msg string) {
	cl.log(logrus.WarnLevel, msg)
}

func (cl *ContextLogger) Error(msg string) {
	cl.log(logrus.ErrorLevel, msg)
}

func (cl *ContextLogger) Fatal(msg string) {
	cl.log(logrus.FatalLevel, msg)
}

func (cl *ContextLogger) Panic(msg string) {
	cl.log(logrus.PanicLevel, msg)
}

// 全局日志方法（兼容性）
func Debug(msg string) {
	if sampler.shouldSample(logrus.DebugLevel) {
		logger.Debug(msg)
	}
}

func Info(msg string) {
	if sampler.shouldSample(logrus.InfoLevel) {
		logger.Info(msg)
	}
}

func Warn(msg string) {
	if sampler.shouldSample(logrus.WarnLevel) {
		logger.Warn(msg)
	}
}

func Error(msg string) {
	if sampler.shouldSample(logrus.ErrorLevel) {
		logger.Error(msg)
	}
}

func Fatal(msg string) {
	if sampler.shouldSample(logrus.FatalLevel) {
		logger.Fatal(msg)
	}
}

func Panic(msg string) {
	if sampler.shouldSample(logrus.PanicLevel) {
		logger.Panic(msg)
	}
}

// 从上下文提取trace信息（需要根据实际tracing实现调整）
func getTraceIDFromContext(ctx context.Context) string {
	// TODO: 根据实际使用的tracing库实现
	if traceID := ctx.Value("traceId"); traceID != nil {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return ""
}

func getSpanIDFromContext(ctx context.Context) string {
	// TODO: 根据实际使用的tracing库实现
	if spanID := ctx.Value("spanId"); spanID != nil {
		if id, ok := spanID.(string); ok {
			return id
		}
	}
	return ""
}

// 设置trace信息到上下文
func SetTraceInfo(ctx context.Context, traceID, spanID string) context.Context {
	ctx = context.WithValue(ctx, "traceId", traceID)
	ctx = context.WithValue(ctx, "spanId", spanID)
	return ctx
}

// 获取日志记录器实例
func GetLogger() *logrus.Logger {
	return logger
}
