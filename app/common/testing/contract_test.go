package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ContractTest 契约测试
type ContractTest struct {
	provider  ContractProvider
	consumer  ContractConsumer
	validator ContractValidator
}

// ContractProvider 契约提供者
type ContractProvider interface {
	GetName() string
	GetVersion() string
	GetEndpoints() []ContractEndpoint
	GetSchemas() map[string]ContractSchema
}

// ContractConsumer 契约消费者
type ContractConsumer interface {
	GetName() string
	GetVersion() string
	GetExpectations() []ContractExpectation
}

// ContractValidator 契约验证器
type ContractValidator interface {
	ValidateContract(provider ContractProvider, consumer ContractConsumer) (*ContractValidationResult, error)
	ValidateRequest(request *ContractRequest) error
	ValidateResponse(response *ContractResponse) error
}

// ContractEndpoint 契约端点
type ContractEndpoint struct {
	Path        string            `json:"path"`
	Method      string            `json:"method"`
	Description string            `json:"description"`
	Request     *ContractRequest  `json:"request,omitempty"`
	Response    *ContractResponse `json:"response,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// ContractRequest 契约请求
type ContractRequest struct {
	Headers map[string]string      `json:"headers,omitempty"`
	Body    map[string]interface{} `json:"body,omitempty"`
	Query   map[string]string      `json:"query,omitempty"`
	Path    map[string]string      `json:"path,omitempty"`
}

// ContractResponse 契约响应
type ContractResponse struct {
	Status  int                    `json:"status"`
	Headers map[string]string      `json:"headers,omitempty"`
	Body    map[string]interface{} `json:"body,omitempty"`
}

// ContractSchema 契约模式
type ContractSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
	Example    interface{}            `json:"example,omitempty"`
}

// ContractExpectation 契约期望
type ContractExpectation struct {
	Endpoint    string            `json:"endpoint"`
	Method      string            `json:"method"`
	Request     *ContractRequest  `json:"request,omitempty"`
	Response    *ContractResponse `json:"response,omitempty"`
	Description string            `json:"description"`
}

// ContractValidationResult 契约验证结果
type ContractValidationResult struct {
	Valid         bool                        `json:"valid"`
	Errors        []ContractValidationError   `json:"errors,omitempty"`
	Warnings      []ContractValidationWarning `json:"warnings,omitempty"`
	Compatibility float64                     `json:"compatibility"`
}

// ContractValidationError 契约验证错误
type ContractValidationError struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	Endpoint string `json:"endpoint,omitempty"`
	Field    string `json:"field,omitempty"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
}

// ContractValidationWarning 契约验证警告
type ContractValidationWarning struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	Endpoint   string `json:"endpoint,omitempty"`
	Field      string `json:"field,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// NewContractTest 创建契约测试
func NewContractTest(provider ContractProvider, consumer ContractConsumer, validator ContractValidator) *ContractTest {
	return &ContractTest{
		provider:  provider,
		consumer:  consumer,
		validator: validator,
	}
}

// Run 运行契约测试
func (ct *ContractTest) Run(ctx context.Context) (*ContractValidationResult, error) {
	return ct.validator.ValidateContract(ct.provider, ct.consumer)
}

// HTTPContractProvider HTTP契约提供者
type HTTPContractProvider struct {
	name      string
	version   string
	baseURL   string
	client    *http.Client
	endpoints []ContractEndpoint
	schemas   map[string]ContractSchema
}

// NewHTTPContractProvider 创建HTTP契约提供者
func NewHTTPContractProvider(name, version, baseURL string) *HTTPContractProvider {
	return &HTTPContractProvider{
		name:      name,
		version:   version,
		baseURL:   baseURL,
		client:    &http.Client{Timeout: 30 * time.Second},
		endpoints: make([]ContractEndpoint, 0),
		schemas:   make(map[string]ContractSchema),
	}
}

// GetName 获取名称
func (hcp *HTTPContractProvider) GetName() string {
	return hcp.name
}

// GetVersion 获取版本
func (hcp *HTTPContractProvider) GetVersion() string {
	return hcp.version
}

// GetEndpoints 获取端点
func (hcp *HTTPContractProvider) GetEndpoints() []ContractEndpoint {
	return hcp.endpoints
}

// GetSchemas 获取模式
func (hcp *HTTPContractProvider) GetSchemas() map[string]ContractSchema {
	return hcp.schemas
}

// AddEndpoint 添加端点
func (hcp *HTTPContractProvider) AddEndpoint(endpoint ContractEndpoint) {
	hcp.endpoints = append(hcp.endpoints, endpoint)
}

// AddSchema 添加模式
func (hcp *HTTPContractProvider) AddSchema(name string, schema ContractSchema) {
	hcp.schemas[name] = schema
}

// TestEndpoint 测试端点
func (hcp *HTTPContractProvider) TestEndpoint(ctx context.Context, endpoint ContractEndpoint) (*ContractResponse, error) {
	url := hcp.baseURL + endpoint.Path

	// 构建请求
	var body io.Reader
	if endpoint.Request != nil && endpoint.Request.Body != nil {
		jsonData, err := json.Marshal(endpoint.Request.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = strings.NewReader(string(jsonData))
	}

	req, err := http.NewRequestWithContext(ctx, endpoint.Method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	if endpoint.Request != nil {
		for key, value := range endpoint.Request.Headers {
			req.Header.Set(key, value)
		}
	}

	// 发送请求
	resp, err := hcp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 解析响应体
	var respData map[string]interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &respData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
		}
	}

	// 构建响应
	response := &ContractResponse{
		Status:  resp.StatusCode,
		Headers: make(map[string]string),
		Body:    respData,
	}

	// 设置响应头
	for key, values := range resp.Header {
		if len(values) > 0 {
			response.Headers[key] = values[0]
		}
	}

	return response, nil
}

// JSONSchemaValidator JSON模式验证器
type JSONSchemaValidator struct {
	schemas map[string]ContractSchema
}

// NewJSONSchemaValidator 创建JSON模式验证器
func NewJSONSchemaValidator() *JSONSchemaValidator {
	return &JSONSchemaValidator{
		schemas: make(map[string]ContractSchema),
	}
}

// ValidateContract 验证契约
func (jsv *JSONSchemaValidator) ValidateContract(provider ContractProvider, consumer ContractConsumer) (*ContractValidationResult, error) {
	result := &ContractValidationResult{
		Valid:         true,
		Errors:        make([]ContractValidationError, 0),
		Warnings:      make([]ContractValidationWarning, 0),
		Compatibility: 1.0,
	}

	// 验证端点兼容性
	if err := jsv.validateEndpoints(provider, consumer, result); err != nil {
		return nil, err
	}

	// 验证模式兼容性
	if err := jsv.validateSchemas(provider, consumer, result); err != nil {
		return nil, err
	}

	// 计算兼容性分数
	result.Compatibility = jsv.calculateCompatibility(result)

	return result, nil
}

// validateEndpoints 验证端点
func (jsv *JSONSchemaValidator) validateEndpoints(provider ContractProvider, consumer ContractConsumer, result *ContractValidationResult) error {
	providerEndpoints := provider.GetEndpoints()
	consumerExpectations := consumer.GetExpectations()

	// 创建提供者端点映射
	providerEndpointMap := make(map[string]ContractEndpoint)
	for _, endpoint := range providerEndpoints {
		key := endpoint.Method + ":" + endpoint.Path
		providerEndpointMap[key] = endpoint
	}

	// 验证消费者期望
	for _, expectation := range consumerExpectations {
		key := expectation.Method + ":" + expectation.Endpoint
		providerEndpoint, exists := providerEndpointMap[key]

		if !exists {
			result.Errors = append(result.Errors, ContractValidationError{
				Type:     "missing_endpoint",
				Message:  fmt.Sprintf("Endpoint %s %s not found in provider", expectation.Method, expectation.Endpoint),
				Endpoint: expectation.Endpoint,
			})
			result.Valid = false
			continue
		}

		// 验证请求兼容性
		if expectation.Request != nil && providerEndpoint.Request != nil {
			jsv.validateRequestCompatibility(expectation.Request, providerEndpoint.Request, expectation.Endpoint, result)
		}

		// 验证响应兼容性
		if expectation.Response != nil && providerEndpoint.Response != nil {
			jsv.validateResponseCompatibility(expectation.Response, providerEndpoint.Response, expectation.Endpoint, result)
		}
	}

	return nil
}

// validateSchemas 验证模式
func (jsv *JSONSchemaValidator) validateSchemas(provider ContractProvider, consumer ContractConsumer, result *ContractValidationResult) error {
	providerSchemas := provider.GetSchemas()

	// 这里可以添加模式验证逻辑
	// 简化实现，只检查模式是否存在
	for schemaName, schema := range providerSchemas {
		if schema.Type == "" {
			result.Warnings = append(result.Warnings, ContractValidationWarning{
				Type:       "missing_schema_type",
				Message:    fmt.Sprintf("Schema %s missing type", schemaName),
				Field:      schemaName,
				Suggestion: "Add type field to schema",
			})
		}
	}

	return nil
}

// validateRequestCompatibility 验证请求兼容性
func (jsv *JSONSchemaValidator) validateRequestCompatibility(consumerReq, providerReq *ContractRequest, endpoint string, result *ContractValidationResult) {
	// 验证必需字段
	for key, value := range consumerReq.Body {
		if _, exists := providerReq.Body[key]; !exists {
			result.Errors = append(result.Errors, ContractValidationError{
				Type:     "missing_required_field",
				Message:  fmt.Sprintf("Required field %s not found in provider request", key),
				Endpoint: endpoint,
				Field:    key,
			})
			result.Valid = false
		} else if value != nil && providerReq.Body[key] != nil {
			// 验证字段类型兼容性
			if !jsv.isCompatibleType(value, providerReq.Body[key]) {
				result.Warnings = append(result.Warnings, ContractValidationWarning{
					Type:       "type_mismatch",
					Message:    fmt.Sprintf("Field %s type mismatch", key),
					Endpoint:   endpoint,
					Field:      key,
					Suggestion: "Check field type compatibility",
				})
			}
		}
	}
}

// validateResponseCompatibility 验证响应兼容性
func (jsv *JSONSchemaValidator) validateResponseCompatibility(consumerResp, providerResp *ContractResponse, endpoint string, result *ContractValidationResult) {
	// 验证状态码
	if consumerResp.Status != providerResp.Status {
		result.Warnings = append(result.Warnings, ContractValidationWarning{
			Type:       "status_code_mismatch",
			Message:    fmt.Sprintf("Status code mismatch: expected %d, got %d", consumerResp.Status, providerResp.Status),
			Endpoint:   endpoint,
			Suggestion: "Check status code compatibility",
		})
	}

	// 验证响应体字段
	for key, value := range consumerResp.Body {
		if _, exists := providerResp.Body[key]; !exists {
			result.Warnings = append(result.Warnings, ContractValidationWarning{
				Type:       "missing_response_field",
				Message:    fmt.Sprintf("Response field %s not found in provider", key),
				Endpoint:   endpoint,
				Field:      key,
				Suggestion: "Check if field is optional or add to provider",
			})
		} else if value != nil && providerResp.Body[key] != nil {
			// 验证字段类型兼容性
			if !jsv.isCompatibleType(value, providerResp.Body[key]) {
				result.Warnings = append(result.Warnings, ContractValidationWarning{
					Type:       "response_type_mismatch",
					Message:    fmt.Sprintf("Response field %s type mismatch", key),
					Endpoint:   endpoint,
					Field:      key,
					Suggestion: "Check response field type compatibility",
				})
			}
		}
	}
}

// isCompatibleType 检查类型兼容性
func (jsv *JSONSchemaValidator) isCompatibleType(consumer, provider interface{}) bool {
	consumerType := fmt.Sprintf("%T", consumer)
	providerType := fmt.Sprintf("%T", provider)

	// 简化类型兼容性检查
	return consumerType == providerType
}

// calculateCompatibility 计算兼容性分数
func (jsv *JSONSchemaValidator) calculateCompatibility(result *ContractValidationResult) float64 {
	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		return 1.0
	}

	// 根据错误和警告数量计算兼容性分数
	errorPenalty := float64(len(result.Errors)) * 0.1
	warningPenalty := float64(len(result.Warnings)) * 0.05

	compatibility := 1.0 - errorPenalty - warningPenalty
	if compatibility < 0 {
		compatibility = 0
	}

	return compatibility
}

// ValidateRequest 验证请求
func (jsv *JSONSchemaValidator) ValidateRequest(request *ContractRequest) error {
	// 实现请求验证逻辑
	return nil
}

// ValidateResponse 验证响应
func (jsv *JSONSchemaValidator) ValidateResponse(response *ContractResponse) error {
	// 实现响应验证逻辑
	return nil
}

// ContractTestSuite 契约测试套件
type ContractTestSuite struct {
	tests []*ContractTest
	mutex sync.RWMutex
}

// NewContractTestSuite 创建契约测试套件
func NewContractTestSuite() *ContractTestSuite {
	return &ContractTestSuite{
		tests: make([]*ContractTest, 0),
	}
}

// AddTest 添加测试
func (cts *ContractTestSuite) AddTest(test *ContractTest) {
	cts.mutex.Lock()
	defer cts.mutex.Unlock()
	cts.tests = append(cts.tests, test)
}

// RunAll 运行所有测试
func (cts *ContractTestSuite) RunAll(ctx context.Context) ([]*ContractValidationResult, error) {
	cts.mutex.RLock()
	tests := make([]*ContractTest, len(cts.tests))
	copy(tests, cts.tests)
	cts.mutex.RUnlock()

	results := make([]*ContractValidationResult, 0, len(tests))

	for _, test := range tests {
		result, err := test.Run(ctx)
		if err != nil {
			return nil, fmt.Errorf("test failed: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// GetTestCount 获取测试数量
func (cts *ContractTestSuite) GetTestCount() int {
	cts.mutex.RLock()
	defer cts.mutex.RUnlock()
	return len(cts.tests)
}
