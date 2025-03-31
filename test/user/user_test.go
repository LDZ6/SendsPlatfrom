package user

import (
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	userPb "platform/idl/pb/user"
	"testing"
)

var client userPb.UserServiceClient

func init() {
	addr := "127.0.0.1:10002"
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err.Error())
	}
	client = userPb.NewUserServiceClient(conn)
}
func TestSchool_user(t *testing.T) {
	req := userPb.UserLoginRequest{
		Code: "071kcGFa1p2m9G0FfCIa1jjR7X0kcGFY",
	}
	resp, err := client.SchoolUserLogin(context.Background(), &req)
	logrus.Info(resp)
	logrus.Info(err)
}
