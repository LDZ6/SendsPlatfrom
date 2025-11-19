package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"platform/app/school/database/cache"
	"platform/app/school/service"
	"platform/config"
	SchoolPb "platform/idl/pb/school"
	"platform/utils"
	"platform/utils/discovery"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

func main() {
	config.InitConfig()
	cache.InitRDB()
	// etcd 地址
	etcdAddress := []string{config.Conf.Etcd.Address}
	username := config.Conf.Etcd.Username
	password := config.Conf.Etcd.Password
	// 服务注册
	etcdRegister := discovery.NewRegister(etcdAddress, username, password, logrus.New())
	grpcAddress := config.Conf.Services["school"].Addr
	defer etcdRegister.Stop()
	taskNode := discovery.Server{
		Name: config.Conf.Domain["school"].Name,
		Addr: grpcAddress,
	}
	server := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    60 * time.Second,
			Timeout: 5 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	// 绑定service
	SchoolPb.RegisterSchoolServiceServer(server, service.GetSchoolSrv())
	lis, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		panic(err)
	}
	if _, err := etcdRegister.Register(taskNode, 10); err != nil {
		panic(fmt.Sprintf("start server failed, err: %v", err))
	}
	logrus.Info("server started listen on ", grpcAddress)
	go func() {
		if err := server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger := utils.GetLogger()
	logger.Info("收到关闭信号，开始优雅关闭...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
		logger.Info("gRPC服务器已优雅关闭")
	case <-ctx.Done():
		logger.Warn("优雅关闭超时，强制关闭")
		server.Stop()
	}
}
