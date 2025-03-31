package main

import (
	"fmt"
	"net"
	"platform/app/boBing/database/cache"
	"platform/app/boBing/database/dao"
	"platform/app/boBing/service"
	"platform/config"
	BoBingPb "platform/idl/pb/boBing"
	"platform/utils/discovery"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	config.InitConfig()
	dao.InitDB()
	cache.InitRDB()
	// etcd 地址
	etcdAddress := []string{config.Conf.Etcd.Address}
	username := config.Conf.Etcd.Username
	password := config.Conf.Etcd.Password
	// 服务注册
	etcdRegister := discovery.NewRegister(etcdAddress, username, password, logrus.New())
	grpcAddress := config.Conf.Services["bobing"].Addr[0]
	defer etcdRegister.Stop()
	taskNode := discovery.Server{
		Name: config.Conf.Domain["bobing"].Name,
		Addr: grpcAddress,
	}
	server := grpc.NewServer()
	defer server.Stop()
	// 绑定service
	BoBingPb.RegisterBoBingServiceServer(server, service.GetBoBingSrv())
	lis, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		panic(err)
	}
	if _, err := etcdRegister.Register(taskNode, 10); err != nil {
		panic(fmt.Sprintf("start server failed, err: %v", err))
	}
	logrus.Info("server started listen on ", grpcAddress)
	if err := server.Serve(lis); err != nil {
		panic(err)
	}
}
