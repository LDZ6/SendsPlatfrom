package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// 追踪配置
type TracingConfig struct {
	Enabled     bool
	ServiceName string
	SampleRate  float64
	Endpoint    string
	Exporter    string // jaeger, zipkin
}

var (
	tracer     trace.Tracer
	tracingCfg *TracingConfig
)

// 初始化追踪系统
func init() {
	initTracing()
}

func initTracing() {
	// 从配置获取追踪设置
	enabled := viper.GetBool("observability.tracing.enabled")
	if !enabled {
		return
	}

	serviceName := viper.GetString("observability.tracing.serviceName")
	if serviceName == "" {
		serviceName = "sends-platform"
	}

	sampleRate := viper.GetFloat64("observability.tracing.sampleRate")
	if sampleRate <= 0 {
		sampleRate = 0.1 // 默认10%采样
	}

	endpoint := viper.GetString("observability.tracing.endpoint")
	if endpoint == "" {
		endpoint = "http://localhost:14268/api/traces" // Jaeger默认端点
	}

	exporter := viper.GetString("observability.tracing.exporter")
	if exporter == "" {
		exporter = "jaeger"
	}

	tracingCfg = &TracingConfig{
		Enabled:     enabled,
		ServiceName: serviceName,
		SampleRate:  sampleRate,
		Endpoint:    endpoint,
		Exporter:    exporter,
	}

	if !enabled {
		return
	}

	// 创建资源
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
	if err != nil {
		panic(fmt.Errorf("创建追踪资源失败: %w", err))
	}

	// 创建导出器
	var exp sdktrace.SpanExporter
	switch exporter {
	case "jaeger":
		exp, err = jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
	case "zipkin":
		exp, err = zipkin.New(endpoint)
	default:
		panic(fmt.Errorf("不支持的追踪导出器: %s", exporter))
	}

	if err != nil {
		panic(fmt.Errorf("创建追踪导出器失败: %w", err))
	}

	// 创建采样器
	sampler := sdktrace.TraceIDRatioBased(sampleRate)

	// 创建追踪提供者
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// 设置全局追踪提供者
	otel.SetTracerProvider(tp)

	// 设置全局传播器
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 创建追踪器
	tracer = otel.Tracer(serviceName)
}

// 获取追踪器
func GetTracer() trace.Tracer {
	if tracer == nil {
		// 返回noop追踪器
		return trace.NewNoopTracerProvider().Tracer("noop")
	}
	return tracer
}

// 创建根span
func StartRootSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return GetTracer().Start(ctx, name, opts...)
}

// 创建子span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return GetTracer().Start(ctx, name, opts...)
}

// 从上下文获取追踪信息
func GetTraceInfo(ctx context.Context) (traceID, spanID string) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String(), span.SpanContext().SpanID().String()
	}
	return "", ""
}

// 设置追踪信息到上下文
func SetTraceInfo(ctx context.Context, traceID, spanID string) context.Context {
	// 这里需要根据实际的trace ID和span ID创建span context
	// 简化实现，实际应该解析trace ID和span ID
	return ctx
}

// 添加span属性
func AddSpanAttributes(span trace.Span, attrs map[string]interface{}) {
	var attributes []attribute.KeyValue
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			attributes = append(attributes, attribute.String(k, val))
		case int:
			attributes = append(attributes, attribute.Int(k, val))
		case int64:
			attributes = append(attributes, attribute.Int64(k, val))
		case float64:
			attributes = append(attributes, attribute.Float64(k, val))
		case bool:
			attributes = append(attributes, attribute.Bool(k, val))
		default:
			attributes = append(attributes, attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}
	span.SetAttributes(attributes...)
}

// 记录span事件
func RecordSpanEvent(span trace.Span, name string, attrs map[string]interface{}) {
	var attributes []attribute.KeyValue
	for k, v := range attrs {
		switch val := v.(type) {
		case string:
			attributes = append(attributes, attribute.String(k, val))
		case int:
			attributes = append(attributes, attribute.Int(k, val))
		case int64:
			attributes = append(attributes, attribute.Int64(k, val))
		case float64:
			attributes = append(attributes, attribute.Float64(k, val))
		case bool:
			attributes = append(attributes, attribute.Bool(k, val))
		default:
			attributes = append(attributes, attribute.String(k, fmt.Sprintf("%v", val)))
		}
	}
	span.AddEvent(name, trace.WithAttributes(attributes...))
}

// 记录span错误
func RecordSpanError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(1, err.Error()) // 1表示错误状态
}

// HTTP中间件追踪
func TraceHTTPHandler(handler func(ctx context.Context, req interface{}) (interface{}, error)) func(ctx context.Context, req interface{}) (interface{}, error) {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		ctx, span := StartSpan(ctx, "http_handler")
		defer span.End()

		// 添加HTTP相关属性
		AddSpanAttributes(span, map[string]interface{}{
			"http.method": "POST",
			"http.scheme": "http",
		})

		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		// 记录耗时
		AddSpanAttributes(span, map[string]interface{}{
			"http.duration_ms": duration.Milliseconds(),
		})

		if err != nil {
			RecordSpanError(span, err)
		} else {
			span.SetStatus(0, "OK") // 0表示成功状态
		}

		return resp, err
	}
}

// gRPC中间件追踪
func TraceGRPCHandler(handler func(ctx context.Context, req interface{}) (interface{}, error)) func(ctx context.Context, req interface{}) (interface{}, error) {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		ctx, span := StartSpan(ctx, "grpc_handler")
		defer span.End()

		// 添加gRPC相关属性
		AddSpanAttributes(span, map[string]interface{}{
			"rpc.system":  "grpc",
			"rpc.service": "unknown",
			"rpc.method":  "unknown",
		})

		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		// 记录耗时
		AddSpanAttributes(span, map[string]interface{}{
			"rpc.duration_ms": duration.Milliseconds(),
		})

		if err != nil {
			RecordSpanError(span, err)
		} else {
			span.SetStatus(0, "OK")
		}

		return resp, err
	}
}

// Redis操作追踪
func TraceRedisOperation(ctx context.Context, operation string, fn func() error) error {
	ctx, span := StartSpan(ctx, "redis_operation")
	defer span.End()

	AddSpanAttributes(span, map[string]interface{}{
		"db.system":    "redis",
		"db.operation": operation,
	})

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	AddSpanAttributes(span, map[string]interface{}{
		"db.duration_ms": duration.Milliseconds(),
	})

	if err != nil {
		RecordSpanError(span, err)
	} else {
		span.SetStatus(0, "OK")
	}

	return err
}

// MySQL操作追踪
func TraceMySQLOperation(ctx context.Context, operation, table string, fn func() error) error {
	ctx, span := StartSpan(ctx, "mysql_operation")
	defer span.End()

	AddSpanAttributes(span, map[string]interface{}{
		"db.system":    "mysql",
		"db.operation": operation,
		"db.sql.table": table,
	})

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	AddSpanAttributes(span, map[string]interface{}{
		"db.duration_ms": duration.Milliseconds(),
	})

	if err != nil {
		RecordSpanError(span, err)
	} else {
		span.SetStatus(0, "OK")
	}

	return err
}

// RabbitMQ操作追踪
func TraceRabbitMQOperation(ctx context.Context, operation, queue string, fn func() error) error {
	ctx, span := StartSpan(ctx, "rabbitmq_operation")
	defer span.End()

	AddSpanAttributes(span, map[string]interface{}{
		"messaging.system":      "rabbitmq",
		"messaging.operation":   operation,
		"messaging.destination": queue,
	})

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	AddSpanAttributes(span, map[string]interface{}{
		"messaging.duration_ms": duration.Milliseconds(),
	})

	if err != nil {
		RecordSpanError(span, err)
	} else {
		span.SetStatus(0, "OK")
	}

	return err
}

// 业务操作追踪
func TraceBusinessOperation(ctx context.Context, operation string, attrs map[string]interface{}, fn func() error) error {
	ctx, span := StartSpan(ctx, "business_operation")
	defer span.End()

	// 添加业务相关属性
	AddSpanAttributes(span, map[string]interface{}{
		"business.operation": operation,
	})

	// 添加自定义属性
	AddSpanAttributes(span, attrs)

	start := time.Now()
	err := fn()
	duration := time.Since(start)

	AddSpanAttributes(span, map[string]interface{}{
		"business.duration_ms": duration.Milliseconds(),
	})

	if err != nil {
		RecordSpanError(span, err)
	} else {
		span.SetStatus(0, "OK")
	}

	return err
}

// 跨边界传播追踪上下文
func PropagateTraceContext(ctx context.Context) context.Context {
	// 这里应该实现跨边界的追踪上下文传播
	// 例如通过HTTP头、gRPC元数据、消息队列头等
	return ctx
}

// 从消息头提取追踪上下文
func ExtractTraceContextFromMessage(headers map[string]string) context.Context {
	// 从消息头提取追踪信息并创建上下文
	// 这里需要根据实际的消息格式实现
	return context.Background()
}

// 注入追踪上下文到消息头
func InjectTraceContextToMessage(ctx context.Context, headers map[string]string) map[string]string {
	// 将追踪上下文注入到消息头
	// 这里需要根据实际的消息格式实现
	return headers
}

// 获取追踪配置
func GetTracingConfig() *TracingConfig {
	return tracingCfg
}

// 关闭追踪系统
func ShutdownTracing(ctx context.Context) error {
	if tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
		return tp.Shutdown(ctx)
	}
	return nil
}
