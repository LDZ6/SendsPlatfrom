package service

import (
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
	YearBillPb "platform/idl/pb/yearBill"
	"sync"
)

type YearBillSrv struct {
	YearBillPb.UnimplementedYearBillServiceServer
}

var YearBillSrvIns *YearBillSrv

var YearBillSrvOnce sync.Once

func GetYearBillSrv() *YearBillSrv {
	YearBillSrvOnce.Do(func() {
		YearBillSrvIns = &YearBillSrv{}
	})
	return YearBillSrvIns
}

func (*YearBillSrv) YearBillPing(ctx context.Context, empty *emptypb.Empty) (resp *YearBillPb.YearBillPingResponse, err error) {
	resp = new(YearBillPb.YearBillPingResponse)
	resp.Message = "yearBill微服务ping通"
	return
}
