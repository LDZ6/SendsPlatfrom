package tcc

import "context"

// TCCComponent TCC组件接口
type TCCComponent interface {
	ID() string
	Try(ctx context.Context, req *TCCReq) (*TCCResp, error)
	Confirm(ctx context.Context, txID string) (*TCCResp, error)
	Cancel(ctx context.Context, txID string) (*TCCResp, error)
}

// TCCReq TCC请求结构
type TCCReq struct {
	ComponentID string
	TXID        string
	Data        map[string]interface{}
}

// TCCResp TCC响应结构
type TCCResp struct {
	ComponentID string
	TXID        string
	ACK         bool
}
