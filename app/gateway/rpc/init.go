package rpc

import (
	"context"
	"fmt"
	"log"
	"platform/config"
	"platform/idl/pb/boBing"
	"platform/idl/pb/school"
	"platform/idl/pb/user"
	"platform/idl/pb/yearBill"
	"platform/utils/discovery"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
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

func Init() error {
	Register = discovery.NewResolver([]string{config.Conf.Etcd.Address}, logrus.New())
	resolver.Register(Register)
	ctx, CancelFunc = context.WithTimeout(context.Background(), 3*time.Second)

	// initialize clients; bubble up first error
	if err := initClient(config.Conf.Domain["user"].Name, &UserClient); err != nil {
		return err
	}
	if err := initClient(config.Conf.Domain["bobing"].Name, &BoBingClient); err != nil {
		return err
	}
	if err := initClient(config.Conf.Domain["school"].Name, &SchoolClient); err != nil {
		return err
	}
	if err := initClient(config.Conf.Domain["year_bill"].Name, &YearBillClient); err != nil {
		return err
	}
	return nil
}

// Close releases resolver and cancels dialing context; call on process shutdown
func Close() {
	if CancelFunc != nil {
		CancelFunc()
	}
	if Register != nil {
		_ = Register.Close()
	}
}

func initClient(serviceName string, client interface{}) error {
	conn, err := connectServer(serviceName)
	if err != nil {
		return err
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
		return fmt.Errorf("unsupported client type")
	}
	return nil
}

func connectServer(serviceName string) (conn *grpc.ClientConn, err error) {
	//设置凭据,认证鉴权
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  100 * time.Millisecond,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   2 * time.Second,
			},
			MinConnectTimeout: 2 * time.Second,
		}),
		grpc.WithDefaultServiceConfig(`{"methodConfig":[{"name":[{"service":"*"}],"retryPolicy":{"MaxAttempts":4,"InitialBackoff":"0.1s","MaxBackoff":"2s","BackoffMultiplier":1.6,"RetryableStatusCodes":["UNAVAILABLE","DEADLINE_EXCEEDED"]}}]}`),
		// otel interceptors for tracing propagation and metrics
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	}
	addr := fmt.Sprintf("%s:///%s", Register.Scheme(), serviceName)

	// Load balance
	if config.Conf.Services[serviceName].LoadBalance {
		log.Printf("load balance enabled for %s\n", serviceName)
		//负载均衡策略轮询
		opts = append(opts, grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"LoadBalancingPolicy": "%s"}`, "round_robin")))
	}

	dctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	conn, err = grpc.DialContext(dctx, addr, opts...)
	return
}
