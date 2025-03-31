package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"platform/app/yearBill/database/cache"
	"platform/app/yearBill/database/dao"
	"platform/app/yearBill/database/mq"
	"platform/app/yearBill/service"
	"platform/config"
	YearBillPb "platform/idl/pb/yearBill"
	"platform/utils/discovery"
)

func main() {
	config.InitConfig()
	dao.InitDB()
	cache.InitRDB()
	mq.InitMQ()
	// etcd 地址
	etcdAddress := []string{config.Conf.Etcd.Address}
	username := config.Conf.Etcd.Username
	password := config.Conf.Etcd.Password
	// 服务注册
	etcdRegister := discovery.NewRegister(etcdAddress, username, password, logrus.New())
	grpcAddress := config.Conf.Services["year_bill"].Addr[0]
	defer etcdRegister.Stop()
	taskNode := discovery.Server{
		Name: config.Conf.Domain["year_bill"].Name,
		Addr: grpcAddress,
	}
	server := grpc.NewServer()
	defer server.Stop()
	// 绑定service
	YearBillPb.RegisterYearBillServiceServer(server, service.GetYearBillSrv())
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
