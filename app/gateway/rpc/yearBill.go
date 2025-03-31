package rpc

import (
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
	yearBillPb "platform/idl/pb/yearBill"
)

func YearBillPingRpc(ctx context.Context) (resp *yearBillPb.YearBillPingResponse, err error) {
	resp, err = YearBillClient.YearBillPing(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}

func GetCertificateRpc(ctx context.Context, req *yearBillPb.GetCertificateRequest) (resp *yearBillPb.GetCertificateResponse, err error) {
	resp, err = YearBillClient.GetCertificate(ctx, req)
	if err != nil {
		return
	}
	return
}

func DataInitRpc(ctx context.Context) (err error) {
	_, err = YearBillClient.DataInit(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}
func CheckStateRpc(ctx context.Context, req *yearBillPb.CheckStateRequest) (resp *yearBillPb.CheckStateResponse, err error) {
	resp, err = YearBillClient.CheckState(ctx, req)
	if err != nil {
		return
	}
	return
}

func AppraiseRpc(ctx context.Context, req *yearBillPb.AppraiseRequest) (err error) {
	_, err = YearBillClient.Appraise(ctx, req)
	if err != nil {
		return
	}
	return
}

func GetLearnDataRpc(ctx context.Context, req *yearBillPb.GetLearnDataRequest) (resp *yearBillPb.GetLearnDataResponse, err error) {
	resp, err = YearBillClient.GetLearnData(ctx, req)
	if err != nil {
		return
	}
	return
}

func GetPayDataRpc(ctx context.Context, req *yearBillPb.GetPayDataRequest) (resp *yearBillPb.GetPayDataResponse, err error) {
	resp, err = YearBillClient.GetPayData(ctx, req)
	if err != nil {
		return
	}
	return
}

func InfoCheckRpc(ctx context.Context, req *yearBillPb.InfoCheckRequest) (resp *yearBillPb.InfoCheckResponse, err error) {
	resp, err = YearBillClient.InfoCheck(ctx, req)
	if err != nil {
		return
	}
	return
}

func GetRankRpc(ctx context.Context, req *yearBillPb.GetRankRequest) (resp *yearBillPb.GetRankResponse, err error) {
	resp, err = YearBillClient.GetRank(ctx, req)
	if err != nil {
		return
	}
	return
}
