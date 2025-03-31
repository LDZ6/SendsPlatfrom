package types

type ResCode int32

const (
	CodeSuccess ResCode = 1000 + iota
	CodeServerBusy
	CodeInvalidParam
	AuthenticationTimeout = 419
)

var codeMsgMap = map[ResCode]string{
	CodeSuccess: "success",
	// 一般不暴露服务器内部错误，对外统一暴露“服务繁忙”
	CodeServerBusy:        "服务繁忙",
	CodeInvalidParam:      "请求参数错误",
	AuthenticationTimeout: "身份认证过期",
}

func (c ResCode) Msg() string {
	msg, ok := codeMsgMap[c]
	if !ok {
		msg = codeMsgMap[CodeServerBusy]
	}
	return msg
}
