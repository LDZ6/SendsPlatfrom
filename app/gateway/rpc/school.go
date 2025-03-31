package rpc

import (
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
	SchoolPb "platform/idl/pb/school"
)

func SchoolScheduleRpc(ctx context.Context) (resp *SchoolPb.SchoolScheduleResponse, err error) {
	resp, err = SchoolClient.SchoolSchedule(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}

func SchoolGradeRpc(ctx context.Context) (resp *SchoolPb.SchoolGradeResponse, err error) {
	resp, err = SchoolClient.SchoolGrade(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}

func SchoolGpaRpc(ctx context.Context) (resp *SchoolPb.SchoolGpaResponse, err error) {
	resp, err = SchoolClient.SchoolGpa(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}

func SchoolXuefenRpc(ctx context.Context) (resp *SchoolPb.SchoolXuefenResponse, err error) {
	resp, err = SchoolClient.SchoolXuefen(ctx, &emptypb.Empty{})
	if err != nil {
		return
	}
	return
}
