package middlewares

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// 可观测性中间件
type ObservabilityMiddleware struct {
	service string
}

func NewObservabilityMiddleware(service string) *ObservabilityMiddleware {
	return &ObservabilityMiddleware{service: service}
}

// HTTP可观测性中间件
func (om *ObservabilityMiddleware) HTTPMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 开始时间
		start := time.Now()

		// 创建追踪上下文
		ctx := om.extractTraceContext(c)
		ctx, span := om.startHTTPSpan(ctx, c)
		defer span.End()

		// 创建日志上下文
		logCtx := om.createLogContext(ctx, c)

		// 记录请求开始
		logCtx.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"user_agent": c.Request.UserAgent(),
			"remote_ip":  c.ClientIP(),
		}).Info("HTTP请求开始")

		// 增加正在处理的请求数
		IncrementHTTPRequestsInFlight(om.service)
		defer DecrementHTTPRequestsInFlight(om.service)

		// 处理请求
		c.Next()

		// 计算耗时
		duration := time.Since(start)

		// 记录指标
		statusCode := strconv.Itoa(c.Writer.Status())
		RecordHTTPRequest(c.Request.Method, c.FullPath(), statusCode, om.service, duration)

		// 记录响应
		logCtx.WithFields(logrus.Fields{
			"status_code":   c.Writer.Status(),
			"duration_ms":   duration.Milliseconds(),
			"response_size": c.Writer.Size(),
		}).Info("HTTP请求完成")

		// 更新span属性
		span.SetAttributes(
			attribute.String("http.status_code", statusCode),
			attribute.Int64("http.duration_ms", duration.Milliseconds()),
			attribute.Int("http.response_size", c.Writer.Size()),
		)

		// 记录错误
		if c.Writer.Status() >= 400 {
			span.RecordError(c.Err)
			span.SetStatus(1, c.Err.Error())
		} else {
			span.SetStatus(0, "OK")
		}
	})
}

// 提取追踪上下文
func (om *ObservabilityMiddleware) extractTraceContext(c *gin.Context) context.Context {
	// 从HTTP头提取追踪信息
	carrier := propagation.HeaderCarrier(c.Request.Header)
	ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), carrier)
	return ctx
}

// 开始HTTP span
func (om *ObservabilityMiddleware) startHTTPSpan(ctx context.Context, c *gin.Context) (context.Context, trace.Span) {
	tracer := otel.Tracer(om.service)
	ctx, span := tracer.Start(ctx, "http_request",
		trace.WithAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.user_agent", c.Request.UserAgent()),
			attribute.String("http.remote_ip", c.ClientIP()),
		),
	)
	return ctx, span
}

// 创建日志上下文
func (om *ObservabilityMiddleware) createLogContext(ctx context.Context, c *gin.Context) *logrus.Entry {
	// 从span获取追踪信息
	span := trace.SpanFromContext(ctx)
	traceID := ""
	spanID := ""
	if span.SpanContext().IsValid() {
		traceID = span.SpanContext().TraceID().String()
		spanID = span.SpanContext().SpanID().String()
	}

	// 创建日志条目
	entry := logrus.WithFields(logrus.Fields{
		"service":   om.service,
		"trace_id":  traceID,
		"span_id":   spanID,
		"route":     c.FullPath(),
		"method":    c.Request.Method,
		"remote_ip": c.ClientIP(),
	})

	return entry
}

// gRPC可观测性中间件
func (om *ObservabilityMiddleware) GRPCMiddleware() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 开始时间
		start := time.Now()

		// 创建追踪span
		tracer := otel.Tracer(om.service)
		ctx, span := tracer.Start(ctx, "grpc_request",
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", info.FullMethod),
				attribute.String("rpc.method", info.FullMethod),
			),
		)
		defer span.End()

		// 创建日志上下文
		logCtx := om.createGRPCLogContext(ctx, info)

		// 记录请求开始
		logCtx.Info("gRPC请求开始")

		// 处理请求
		resp, err := handler(ctx, req)

		// 计算耗时
		duration := time.Since(start)

		// 记录指标
		code := "OK"
		if err != nil {
			code = "INTERNAL"
		}
		RecordGRPCRequest(info.FullMethod, om.service, code, duration)

		// 记录响应
		logCtx.WithFields(logrus.Fields{
			"duration_ms": duration.Milliseconds(),
			"error":       err,
		}).Info("gRPC请求完成")

		// 更新span属性
		span.SetAttributes(
			attribute.String("rpc.status_code", code),
			attribute.Int64("rpc.duration_ms", duration.Milliseconds()),
		)

		// 记录错误
		if err != nil {
			span.RecordError(err)
			span.SetStatus(1, err.Error())
		} else {
			span.SetStatus(0, "OK")
		}

		return resp, err
	}
}

// 创建gRPC日志上下文
func (om *ObservabilityMiddleware) createGRPCLogContext(ctx context.Context, info *grpc.UnaryServerInfo) *logrus.Entry {
	// 从span获取追踪信息
	span := trace.SpanFromContext(ctx)
	traceID := ""
	spanID := ""
	if span.SpanContext().IsValid() {
		traceID = span.SpanContext().TraceID().String()
		spanID = span.SpanContext().SpanID().String()
	}

	// 创建日志条目
	entry := logrus.WithFields(logrus.Fields{
		"service":     om.service,
		"trace_id":    traceID,
		"span_id":     spanID,
		"grpc_method": info.FullMethod,
	})

	return entry
}

// 数据库可观测性中间件
func (om *ObservabilityMiddleware) DatabaseMiddleware() func(db *gorm.DB) {
	return func(db *gorm.DB) {
		// 开始时间
		start := time.Now()

		// 创建追踪span
		ctx := db.Statement.Context
		tracer := otel.Tracer(om.service)
		ctx, span := tracer.Start(ctx, "database_query",
			trace.WithAttributes(
				attribute.String("db.system", "mysql"),
				attribute.String("db.operation", db.Statement.SQL.String()),
				attribute.String("db.sql.table", db.Statement.Table),
			),
		)
		defer span.End()

		// 执行查询
		err := db.Error
		duration := time.Since(start)

		// 记录指标
		operation := "unknown"
		if db.Statement != nil {
			operation = db.Statement.SQL.String()
		}
		RecordGORMQuery(operation, db.Statement.Table, om.service, duration)

		// 记录慢查询
		if err == nil && duration > 200*time.Millisecond {
			logCtx := logrus.WithFields(logrus.Fields{
				"service":     om.service,
				"query":       operation,
				"table":       db.Statement.Table,
				"duration_ms": duration.Milliseconds(),
			})
			logCtx.Warn("慢查询检测")
		}

		// 更新span属性
		span.SetAttributes(
			attribute.Int64("db.duration_ms", duration.Milliseconds()),
		)

		// 记录错误
		if err != nil {
			span.RecordError(err)
			span.SetStatus(1, err.Error())
		} else {
			span.SetStatus(0, "OK")
		}
	}
}

// Redis可观测性中间件
func (om *ObservabilityMiddleware) RedisMiddleware() redis.Hook {
	return &redisObservabilityHook{
		service: om.service,
	}
}

type redisObservabilityHook struct {
	service string
}

func (h *redisObservabilityHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	return context.WithValue(ctx, "start_time", time.Now()), nil
}

func (h *redisObservabilityHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	startTime, ok := ctx.Value("start_time").(time.Time)
	if !ok {
		return nil
	}

	duration := time.Since(startTime)

	// 创建追踪span
	tracer := otel.Tracer(h.service)
	ctx, span := tracer.Start(ctx, "redis_operation",
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", cmd.Name()),
		),
	)
	defer span.End()

	// 记录指标
	RecordRedisOperation(cmd.Name(), "success", h.service, duration)

	// 记录慢命令
	if duration > 10*time.Millisecond {
		logCtx := logrus.WithFields(logrus.Fields{
			"service":     h.service,
			"command":     cmd.Name(),
			"duration_ms": duration.Milliseconds(),
			"args":        cmd.Args(),
		})
		logCtx.Warn("慢Redis命令检测")
	}

	// 更新span属性
	span.SetAttributes(
		attribute.Int64("db.duration_ms", duration.Milliseconds()),
	)

	// 记录错误
	if cmd.Err() != nil {
		span.RecordError(cmd.Err())
		span.SetStatus(1, cmd.Err().Error())
	} else {
		span.SetStatus(0, "OK")
	}

	return nil
}

func (h *redisObservabilityHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	return context.WithValue(ctx, "start_time", time.Now()), nil
}

func (h *redisObservabilityHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	startTime, ok := ctx.Value("start_time").(time.Time)
	if !ok {
		return nil
	}

	duration := time.Since(startTime)

	// 创建追踪span
	tracer := otel.Tracer(h.service)
	ctx, span := tracer.Start(ctx, "redis_pipeline",
		trace.WithAttributes(
			attribute.String("db.system", "redis"),
			attribute.String("db.operation", "pipeline"),
			attribute.Int("db.command_count", len(cmds)),
		),
	)
	defer span.End()

	// 记录指标
	for _, cmd := range cmds {
		RecordRedisOperation(cmd.Name(), "pipeline", h.service, duration/time.Duration(len(cmds)))
	}

	// 记录慢管道操作
	if duration > 50*time.Millisecond {
		logCtx := logrus.WithFields(logrus.Fields{
			"service":       h.service,
			"command_count": len(cmds),
			"duration_ms":   duration.Milliseconds(),
		})
		logCtx.Warn("慢Redis管道操作检测")
	}

	// 更新span属性
	span.SetAttributes(
		attribute.Int64("db.duration_ms", duration.Milliseconds()),
	)

	// 记录错误
	var lastErr error
	for _, cmd := range cmds {
		if cmd.Err() != nil {
			lastErr = cmd.Err()
		}
	}

	if lastErr != nil {
		span.RecordError(lastErr)
		span.SetStatus(1, lastErr.Error())
	} else {
		span.SetStatus(0, "OK")
	}

	return nil
}

// 业务操作可观测性中间件
func (om *ObservabilityMiddleware) BusinessMiddleware(operation string) func(ctx context.Context, fn func() error) error {
	return func(ctx context.Context, fn func() error) error {
		// 开始时间
		start := time.Now()

		// 创建追踪span
		tracer := otel.Tracer(om.service)
		ctx, span := tracer.Start(ctx, "business_operation",
			trace.WithAttributes(
				attribute.String("business.operation", operation),
			),
		)
		defer span.End()

		// 创建日志上下文
		logCtx := logrus.WithFields(logrus.Fields{
			"service":   om.service,
			"operation": operation,
		})

		// 记录操作开始
		logCtx.Info("业务操作开始")

		// 执行操作
		err := fn()

		// 计算耗时
		duration := time.Since(start)

		// 记录指标
		if err != nil {
			RecordYearBillTaskFailure(operation, "unknown", om.service)
		} else {
			RecordYearBillTaskSuccess(operation, om.service)
		}

		// 记录操作完成
		logCtx.WithFields(logrus.Fields{
			"duration_ms": duration.Milliseconds(),
			"error":       err,
		}).Info("业务操作完成")

		// 更新span属性
		span.SetAttributes(
			attribute.Int64("business.duration_ms", duration.Milliseconds()),
		)

		// 记录错误
		if err != nil {
			span.RecordError(err)
			span.SetStatus(1, err.Error())
		} else {
			span.SetStatus(0, "OK")
		}

		return err
	}
}

// 设置Prometheus指标端点
func SetupMetricsEndpoint(r *gin.Engine) {
	// 检查是否启用指标
	if !viper.GetBool("observability.metrics.enabled") {
		return
	}

	path := viper.GetString("observability.metrics.path")
	if path == "" {
		path = "/metrics"
	}

	// 设置指标端点
	r.GET(path, gin.WrapH(promhttp.Handler()))
}

// 设置健康检查端点
func SetupHealthCheckEndpoint(r *gin.Engine) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "sends-platform",
		})
	})
}

// 设置性能报告端点
func SetupPerformanceReportEndpoint(r *gin.Engine) {
	r.GET("/performance", func(c *gin.Context) {
		report := GeneratePerformanceReport()
		c.JSON(http.StatusOK, report)
	})
}
