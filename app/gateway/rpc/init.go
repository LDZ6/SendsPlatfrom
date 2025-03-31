package rpc

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"log"
	"platform/config"
	"platform/idl/pb/boBing"
	"platform/idl/pb/school"
	"platform/idl/pb/user"
	"platform/idl/pb/yearBill"
	"platform/utils/discovery"
	"time"
)

var (
	Register   *discovery.Resolver
	ctx        context.Context
	CancelFunc context.CancelFunc

	UserClient     user.UserServiceClient
	BoBingClient   boBing.BoBingServiceClient
	SchoolClient   school.SchoolServiceClient
	YearBillClient yearBill.YearBillServiceClient
)

func Init() {
	Register = discovery.NewResolver([]string{config.Conf.Etcd.Address}, logrus.New())
	resolver.Register(Register)
	ctx, CancelFunc = context.WithTimeout(context.Background(), 3*time.Second)

	defer Register.Close()
	initClient(config.Conf.Domain["user"].Name, &UserClient)
	initClient(config.Conf.Domain["bobing"].Name, &BoBingClient)
	initClient(config.Conf.Domain["school"].Name, &SchoolClient)
	initClient(config.Conf.Domain["year_bill"].Name, &YearBillClient)
}

func initClient(serviceName string, client interface{}) {
	conn, err := connectServer(serviceName)

	if err != nil {
		panic(err)
	}

	switch c := client.(type) {
	case *user.UserServiceClient:
		*c = user.NewUserServiceClient(conn)
	case *boBing.BoBingServiceClient:
		*c = boBing.NewBoBingServiceClient(conn)
	case *school.SchoolServiceClient:
		*c = school.NewSchoolServiceClient(conn)
	case *yearBill.YearBillServiceClient:
		*c = yearBill.NewYearBillServiceClient(conn)
	default:
		panic("unsupported client type")
	}
}

func connectServer(serviceName string) (conn *grpc.ClientConn, err error) {
	//设置凭据,认证鉴权
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	addr := fmt.Sprintf("%s:///%s", Register.Scheme(), serviceName)

	// Load balance
	if config.Conf.Services[serviceName].LoadBalance {
		log.Printf("load balance enabled for %s\n", serviceName)
		//负载均衡策略轮询
		opts = append(opts, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, "round_robin")))
	}

	conn, err = grpc.DialContext(ctx, addr, opts...)
	return
}
