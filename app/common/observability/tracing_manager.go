package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TracingManager 链路追踪管理器
type TracingManager struct {
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
	sampler    trace.Sampler
	exporter   trace.SpanExporter
	processor  trace.SpanProcessor
	mutex      sync.RWMutex
	config     *TracingConfig
}

// TracingConfig 链路追踪配置
type TracingConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	SampleRate     float64
	Endpoint       string
	Headers        map[string]string
	BatchTimeout   time.Duration
	MaxExportBatch int
}

// BusinessContext 业务上下文
type BusinessContext struct {
	UserID    string
	SessionID string
	RequestID string
	Operation string
	Resource  string
	Action    string
	ClientIP  string
	UserAgent string
	Metadata  map[string]interface{}
}

// BusinessTracer 业务追踪器
type BusinessTracer struct {
	tracer   trace.Tracer
	business BusinessContext
}

// NewTracingManager 创建链路追踪管理器
func NewTracingManager(config *TracingConfig) (*TracingManager, error) {
	tm := &TracingManager{
		config: config,
	}

	// 创建采样器
	tm.sampler = trace.TraceIDRatioBased(config.SampleRate)

	// 创建传播器
	tm.propagator = propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	// 创建导出器
	exporter, err := tm.createExporter()
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}
	tm.exporter = exporter

	// 创建处理器
	tm.processor = trace.NewBatchSpanProcessor(exporter)

	// 创建追踪器提供者
	tp := trace.NewTracerProvider(
		trace.WithSampler(tm.sampler),
		trace.WithSpanProcessor(tm.processor),
	)

	// 设置全局追踪器提供者
	otel.SetTracerProvider(tp)

	// 创建追踪器
	tm.tracer = tp.Tracer(config.ServiceName)

	return tm, nil
}

// createExporter 创建导出器
func (tm *TracingManager) createExporter() (trace.SpanExporter, error) {
	// 这里应该根据配置创建具体的导出器
	// 例如：Jaeger、Zipkin、OTLP等
	// 简化实现，返回nil
	return nil, nil
}

// StartSpan 开始跨度
func (tm *TracingManager) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tm.tracer.Start(ctx, name, opts...)
}

// StartBusinessSpan 开始业务跨度
func (tm *TracingManager) StartBusinessSpan(ctx context.Context, name string, business BusinessContext, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	// 添加业务属性
	attrs := []attribute.KeyValue{
		attribute.String("user.id", business.UserID),
		attribute.String("session.id", business.SessionID),
		attribute.String("request.id", business.RequestID),
		attribute.String("operation", business.Operation),
		attribute.String("resource", business.Resource),
		attribute.String("action", business.Action),
		attribute.String("client.ip", business.ClientIP),
		attribute.String("user.agent", business.UserAgent),
	}

	// 添加元数据
	for key, value := range business.Metadata {
		attrs = append(attrs, attribute.String(fmt.Sprintf("metadata.%s", key), fmt.Sprintf("%v", value)))
	}

	// 添加属性到选项
	opts = append(opts, trace.WithAttributes(attrs...))

	return tm.tracer.Start(ctx, name, opts...)
}

// Inject 注入追踪上下文
func (tm *TracingManager) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	tm.propagator.Inject(ctx, carrier)
}

// Extract 提取追踪上下文
func (tm *TracingManager) Extract(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return tm.propagator.Extract(ctx, carrier)
}

// GetTracer 获取追踪器
func (tm *TracingManager) GetTracer() trace.Tracer {
	return tm.tracer
}

// GetPropagator 获取传播器
func (tm *TracingManager) GetPropagator() propagation.TextMapPropagator {
	return tm.propagator
}

// Close 关闭追踪管理器
func (tm *TracingManager) Close() error {
	if tm.processor != nil {
		return tm.processor.Shutdown(context.Background())
	}
	return nil
}

// NewBusinessTracer 创建业务追踪器
func NewBusinessTracer(tracer trace.Tracer, business BusinessContext) *BusinessTracer {
	return &BusinessTracer{
		tracer:   tracer,
		business: business,
	}
}

// StartSpan 开始跨度
func (bt *BusinessTracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	// 添加业务属性
	attrs := []attribute.KeyValue{
		attribute.String("user.id", bt.business.UserID),
		attribute.String("session.id", bt.business.SessionID),
		attribute.String("request.id", bt.business.RequestID),
		attribute.String("operation", bt.business.Operation),
		attribute.String("resource", bt.business.Resource),
		attribute.String("action", bt.business.Action),
		attribute.String("client.ip", bt.business.ClientIP),
		attribute.String("user.agent", bt.business.UserAgent),
	}

	// 添加元数据
	for key, value := range bt.business.Metadata {
		attrs = append(attrs, attribute.String(fmt.Sprintf("metadata.%s", key), fmt.Sprintf("%v", value)))
	}

	// 添加属性到选项
	opts = append(opts, trace.WithAttributes(attrs...))

	return bt.tracer.Start(ctx, name, opts...)
}

// RecordEvent 记录事件
func (bt *BusinessTracer) RecordEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// RecordError 记录错误
func (bt *BusinessTracer) RecordError(span trace.Span, err error, attrs ...attribute.KeyValue) {
	span.RecordError(err, trace.WithAttributes(attrs...))
	span.SetStatus(codes.Error, err.Error())
}

// SetAttributes 设置属性
func (bt *BusinessTracer) SetAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	span.SetAttributes(attrs...)
}

// SetStatus 设置状态
func (bt *BusinessTracer) SetStatus(span trace.Span, code codes.Code, description string) {
	span.SetStatus(code, description)
}

// Finish 完成跨度
func (bt *BusinessTracer) Finish(span trace.Span) {
	span.End()
}

// TracingMiddleware 链路追踪中间件
type TracingMiddleware struct {
	tracer      trace.Tracer
	propagator  propagation.TextMapPropagator
	serviceName string
}

// NewTracingMiddleware 创建链路追踪中间件
func NewTracingMiddleware(tracer trace.Tracer, propagator propagation.TextMapPropagator, serviceName string) *TracingMiddleware {
	return &TracingMiddleware{
		tracer:      tracer,
		propagator:  propagator,
		serviceName: serviceName,
	}
}

// HTTPMiddleware HTTP中间件
func (tm *TracingMiddleware) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 提取追踪上下文
		ctx := tm.propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// 开始跨度
		spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		ctx, span := tm.tracer.Start(ctx, spanName)

		// 设置属性
		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.user_agent", r.UserAgent()),
			attribute.String("http.remote_addr", r.RemoteAddr),
		)

		// 创建响应包装器
		wrapped := &responseWrapper{ResponseWriter: w, statusCode: 200}

		// 调用下一个处理器
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		// 设置状态码属性
		span.SetAttributes(attribute.Int("http.status_code", wrapped.statusCode))

		// 完成跨度
		span.End()
	})
}

// GRPCMiddleware gRPC中间件
func (tm *TracingMiddleware) GRPCMiddleware(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// 开始跨度
	spanName := info.FullMethod
	ctx, span := tm.tracer.Start(ctx, spanName)

	// 设置属性
	span.SetAttributes(
		attribute.String("rpc.method", info.FullMethod),
		attribute.String("rpc.service", info.Server),
	)

	// 调用处理器
	resp, err := handler(ctx, req)

	// 记录错误
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	// 完成跨度
	span.End()

	return resp, err
}

// responseWrapper 响应包装器
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 写入状态码
func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// TracingContext 追踪上下文
type TracingContext struct {
	TraceID    string
	SpanID     string
	ParentID   string
	Sampled    bool
	Flags      byte
	Attributes map[string]interface{}
}

// NewTracingContext 创建追踪上下文
func NewTracingContext(traceID, spanID string) *TracingContext {
	return &TracingContext{
		TraceID:    traceID,
		SpanID:     spanID,
		Sampled:    true,
		Attributes: make(map[string]interface{}),
	}
}

// SetAttribute 设置属性
func (tc *TracingContext) SetAttribute(key string, value interface{}) {
	tc.Attributes[key] = value
}

// GetAttribute 获取属性
func (tc *TracingContext) GetAttribute(key string) (interface{}, bool) {
	value, exists := tc.Attributes[key]
	return value, exists
}

// ToCarrier 转换为载体
func (tc *TracingContext) ToCarrier() propagation.TextMapCarrier {
	carrier := make(propagation.TextMapCarrier)
	carrier.Set("traceparent", fmt.Sprintf("00-%s-%s-01", tc.TraceID, tc.SpanID))
	return carrier
}

// FromCarrier 从载体创建
func FromCarrier(carrier propagation.TextMapCarrier) *TracingContext {
	traceparent := carrier.Get("traceparent")
	if traceparent == "" {
		return nil
	}

	// 解析traceparent格式
	// 格式：00-{trace-id}-{span-id}-{trace-flags}
	parts := strings.Split(traceparent, "-")
	if len(parts) != 4 {
		return nil
	}

	return &TracingContext{
		TraceID:    parts[1],
		SpanID:     parts[2],
		Sampled:    parts[3] == "01",
		Attributes: make(map[string]interface{}),
	}
}

// TracingManager 全局追踪管理器
var (
	globalTracingManager *TracingManager
	tracingMutex         sync.RWMutex
)

// SetGlobalTracingManager 设置全局追踪管理器
func SetGlobalTracingManager(tm *TracingManager) {
	tracingMutex.Lock()
	defer tracingMutex.Unlock()
	globalTracingManager = tm
}

// GetGlobalTracingManager 获取全局追踪管理器
func GetGlobalTracingManager() *TracingManager {
	tracingMutex.RLock()
	defer tracingMutex.RUnlock()
	return globalTracingManager
}

// StartGlobalSpan 开始全局跨度
func StartGlobalSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tm := GetGlobalTracingManager()
	if tm == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return tm.StartSpan(ctx, name, opts...)
}

// StartGlobalBusinessSpan 开始全局业务跨度
func StartGlobalBusinessSpan(ctx context.Context, name string, business BusinessContext, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tm := GetGlobalTracingManager()
	if tm == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return tm.StartBusinessSpan(ctx, name, business, opts...)
}
