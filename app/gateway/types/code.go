package types

import "net/http"

type ResCode int32

// Error categories and codes
const (
	CodeSuccess      ResCode = 1000 + iota
	CodeInvalidParam         // 1001 client error
	CodeUnauthorized         // 1002 client error
	CodeForbidden            // 1003 client error
	CodeNotFound             // 1004 client error

	CodeUpstreamError // 1005 upstream dependency error
	CodeTimeout       // 1006 timeout

	CodeServerBusy // 1007 server error
)

// Legacy compatibility
const AuthenticationTimeout ResCode = 1419

var codeMsgMap = map[ResCode]string{
	CodeSuccess:       "success",
	CodeInvalidParam:  "请求参数错误",
	CodeUnauthorized:  "未认证",
	CodeForbidden:     "无权限",
	CodeNotFound:      "未找到",
	CodeUpstreamError: "上游服务错误",
	CodeTimeout:       "请求超时",
	CodeServerBusy:    "服务繁忙",

	AuthenticationTimeout: "身份认证过期",
}

func (c ResCode) Msg() string {
	msg, ok := codeMsgMap[c]
	if !ok {
		msg = codeMsgMap[CodeServerBusy]
	}
	return msg
}

// ToHTTPStatus maps internal codes to HTTP status codes
func (c ResCode) ToHTTPStatus() int {
	switch c {
	case CodeSuccess:
		return http.StatusOK
	case CodeInvalidParam:
		return http.StatusBadRequest
	case CodeUnauthorized, AuthenticationTimeout:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeTimeout:
		return http.StatusGatewayTimeout
	case CodeUpstreamError:
		return http.StatusBadGateway
	case CodeServerBusy:
		fallthrough
	default:
		return http.StatusInternalServerError
	}
}
