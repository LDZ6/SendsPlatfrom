package main

import (
	"platform/app/common/database"
	"platform/app/common/grpc"
	"platform/app/user/database/cache"
	"platform/app/user/database/dao"
	"platform/app/user/database/models"
	"platform/app/user/service"
	userPb "platform/idl/pb/user"
)

func main() {
	// 创建gRPC服务器配置
	config := &grpc.ServerConfig{
		ServiceName: "用户服务",
		ServiceKey:  "user",
		DomainKey:   "user",
		InitDBFunc: func() error {
			// 使用统一的数据库初始化
			db, err := database.InitDatabase(&database.DatabaseConfig{
				ServiceName: "用户服务",
				ServiceKey:  "user",
				Models: []interface{}{
					&models.User{},
				},
			})
			if err != nil {
				return err
			}
			// 设置全局数据库实例
			dao.SetDB(db)
			return nil
		},
		InitCacheFunc: func() error {
			return cache.InitRDB()
		},
		RegisterFunc: func(server *grpc.Server) {
			userPb.RegisterUserServiceServer(server, service.GetUserSrv())
		},
	}

	// 创建并启动服务器
	grpcServer := grpc.NewGRPCServer(config)
	grpcServer.Start()
}
