package rpc

import (
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
	boBingPb "platform/idl/pb/boBing"
)

func BoBingPingRpc(ctx context.Context) (resp *boBingPb.BoBingPingResponse, err error) {
	resp, err = BoBingClient.BoBingPing(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}

func BoBingKeyRpc(ctx context.Context, req *boBingPb.BoBingKeyRequest) (resp *boBingPb.BoBingKeyResponse, err error) {
	resp, err = BoBingClient.BoBingKey(ctx, req)
	if err != nil {
		return
	}
	return
}

func BoBingPublishRpc(ctx context.Context, req *boBingPb.BoBingPublishRequest) (resp *boBingPb.BoBingPublishResponse, err error) {
	resp, err = BoBingClient.BoBingPublish(ctx, req)
	if err != nil {
		return
	}
	return
}

func BoBingToTalTenRpc(ctx context.Context) (resp *boBingPb.BoBingToTalTenResponse, err error) {
	resp, err = BoBingClient.BoBingToTalTen(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}

func BoBingDayRankRpc(ctx context.Context) (resp *boBingPb.BoBingDayRankResponse, err error) {
	resp, err = BoBingClient.BoBingDayRank(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}

func BoBingDayInitRpc(ctx context.Context, req *boBingPb.BoBingInitRequest) (err error) {
	_, err = BoBingClient.BoBingDayInit(ctx, req)
	if err != nil {
		return
	}
	return
}

func BoBingTianXuanRpc(ctx context.Context) (resp *boBingPb.BoBingTianXuanResponse, err error) {
	resp, err = BoBingClient.BoBingTianXuan(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}

func BoBingGetCountRpc(ctx context.Context) (resp *boBingPb.BoBingGetCountResponse, err error) {
	resp, err = BoBingClient.BoBingGetCount(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}

func BoBingRetransmissionRpc(ctx context.Context) (err error) {
	_, err = BoBingClient.BoBingRetransmission(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}

func BoBingBroadcastCheckRpc(ctx context.Context, req *boBingPb.BoBingBroadcastCheckRequest) (err error) {
	_, err = BoBingClient.BoBingBroadcastCheck(ctx, req)
	if err != nil {
		return
	}
	return
}

func BoBingRecordRpc(ctx context.Context) (resp *boBingPb.BoBingRecordResponse, err error) {
	resp, err = BoBingClient.BoBingRecord(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}

func BoBingBlacklistRpc(ctx context.Context) (err error) {
	_, err = BoBingClient.BoBingBlacklist(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}

func BoBingGetNumberRp(ctx context.Context) (resp *boBingPb.BoBingGetNumberResponse, err error) {
	resp, err = BoBingClient.BoBingGetNumber(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}
