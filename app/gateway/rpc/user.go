package rpc

import (
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
	userPb "platform/idl/pb/user"
)

func UserPingRpc(ctx context.Context) (resp *userPb.UserPingResponse, err error) {
	resp, err = UserClient.UserPing(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}

func UserLogin(ctx context.Context, req *userPb.UserLoginRequest) (resp *userPb.UserLoginResponse, err error) {
	resp, err = UserClient.UserLogin(ctx, req)
	if err != nil {
		return
	}
	return
}

func MassesLogin(ctx context.Context, req *userPb.UserLoginRequest) (resp *userPb.UserLoginResponse, err error) {
	resp, err = UserClient.MassesLogin(ctx, req)
	if err != nil {
		return
	}
	return
}

func UserJsTicketRpc(ctx context.Context) (resp *userPb.UserJsTicketResponse, err error) {
	resp, err = UserClient.UserJsTicket(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}

func SchoolUserLoginRpc(ctx context.Context, req *userPb.UserLoginRequest) (resp *userPb.UserLoginResponse, err error) {
	resp, err = UserClient.SchoolUserLogin(ctx, req)
	if err != nil {
		return
	}
	return
}

func YearBillLoginRpc(ctx context.Context, req *userPb.UserLoginRequest) (resp *userPb.UserLoginResponse, err error) {
	resp, err = UserClient.YearBillLogin(ctx, req)
	if err != nil {
		return
	}
	return
}

func WxJSSDKRpc(ctx context.Context, req *userPb.WxJSSDKRequest) (resp *userPb.WxJSSDKResponse, err error) {
	resp, err = UserClient.WxJSSDK(ctx, req)
	if err != nil {
		return
	}
	return
}
