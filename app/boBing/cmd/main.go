package main

import (
	"platform/app/boBing/database/cache"
	"platform/app/boBing/database/dao"
	"platform/app/boBing/database/models"
	"platform/app/boBing/service"
	"platform/app/common/database"
	"platform/app/common/grpc"
	BoBingPb "platform/idl/pb/boBing"
)

func main() {
	// 创建gRPC服务器配置
	config := &grpc.ServerConfig{
		ServiceName: "博饼服务",
		ServiceKey:  "bobing",
		DomainKey:   "bobing",
		InitDBFunc: func() error {
			// 使用统一的数据库初始化
			db, err := database.InitDatabase(&database.DatabaseConfig{
				ServiceName: "博饼服务",
				ServiceKey:  "bobing",
				Models: []interface{}{
					&models.Rank{},
					&models.Record{},
					&models.Submission{},
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
			BoBingPb.RegisterBoBingServiceServer(server, service.GetBoBingSrv())
		},
	}

	// 创建并启动服务器
	grpcServer := grpc.NewGRPCServer(config)
	grpcServer.Start()
}
