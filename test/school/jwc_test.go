package school

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
	schoolPb "platform/idl/pb/school"
	"testing"
)

var client schoolPb.SchoolServiceClient

func init() {
	addr := "127.0.0.1:10004"
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err.Error())
	}
	client = schoolPb.NewSchoolServiceClient(conn)
}

func Test_Schedule(t *testing.T) {
	md := metadata.Pairs("stu_num", "2125102042")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.SchoolSchedule(ctx, &emptypb.Empty{})
	fmt.Println(resp)
	if err != nil {
		logrus.Info(err)
	}
}

func Test_Gpa(t *testing.T) {
	md := metadata.Pairs("gs_session", "8c5b6e03fca0557a629be4eff84b1e63",
		"stu_num", "2125102013",
		"semester", "2023-2024-1")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.SchoolGpa(ctx, &emptypb.Empty{})
	logrus.Info(resp.Gpa)
	if err != nil {
		logrus.Info(err)
	}
}

func Test_XUEFEN(t *testing.T) {
	md := metadata.Pairs("gs_session", "1c6a4777ee58080f22d0976ff43256d7",
		"stu_num", "2224105014",
		"semester", "2023-2024-1")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.SchoolGrade(ctx, &emptypb.Empty{})
	logrus.Info(resp)
	if err != nil {
		logrus.Info(err)
	}
}

func Test_GRADE(t *testing.T) {
	md := metadata.Pairs("stu_num", "2125102042")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	resp, err := client.SchoolGrade(ctx, &emptypb.Empty{})
	logrus.Info(resp)
	if err != nil {
		logrus.Info(err)
	}
}
