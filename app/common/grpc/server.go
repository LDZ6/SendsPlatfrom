package grpc

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"platform/config"
	"platform/utils"
	"platform/utils/discovery"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// ServerConfig gRPC服务器配置
type ServerConfig struct {
	ServiceName   string
	ServiceKey    string
	DomainKey     string
	Port          string
	RegisterFunc  func(*grpc.Server)
	InitDBFunc    func() error
	InitCacheFunc func() error
	InitMQFunc    func() error
}

// GRPCServer gRPC服务器封装
type GRPCServer struct {
	config   *ServerConfig
	server   *grpc.Server
	register *discovery.Register
	logger   *logrus.Logger
}

// NewGRPCServer 创建gRPC服务器
func NewGRPCServer(config *ServerConfig) *GRPCServer {
	logger := utils.GetLogger()

	return &GRPCServer{
		config: config,
		logger: logger,
	}
}

// Start 启动gRPC服务器
func (s *GRPCServer) Start() error {
	// 初始化配置
	config.InitConfig()

	// 初始化数据库
	if s.config.InitDBFunc != nil {
		if err := s.config.InitDBFunc(); err != nil {
			s.logger.Fatalf("数据库初始化失败: %v", err)
		}
	}

	// 初始化缓存
	if s.config.InitCacheFunc != nil {
		if err := s.config.InitCacheFunc(); err != nil {
			s.logger.Fatalf("缓存初始化失败: %v", err)
		}
	}

	// 初始化消息队列
	if s.config.InitMQFunc != nil {
		if err := s.config.InitMQFunc(); err != nil {
			s.logger.Fatalf("消息队列初始化失败: %v", err)
		}
	}

	// 获取服务配置
	grpcAddress := config.Conf.Services[s.config.ServiceKey].Addr
	etcdAddress := []string{config.Conf.Etcd.Address}
	username := config.Conf.Etcd.Username
	password := config.Conf.Etcd.Password

	// 创建etcd注册器
	s.register = discovery.NewRegister(etcdAddress, username, password, s.logger)
	defer s.register.Stop()

	// 注册服务
	taskNode := discovery.Server{
		Name: config.Conf.Domain[s.config.DomainKey].Name,
		Addr: grpcAddress,
	}

	if _, err := s.register.Register(taskNode, 10); err != nil {
		s.logger.Fatalf("服务注册失败: %v", err)
	}

	// 创建gRPC服务器
	s.server = grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    60 * time.Second,
			Timeout: 5 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)

	// 绑定服务
	if s.config.RegisterFunc != nil {
		s.config.RegisterFunc(s.server)
	}

	// 监听端口
	lis, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		s.logger.Fatalf("监听端口失败: %v", err)
	}

	// 启动服务器
	go func() {
		s.logger.Infof("%s服务启动，监听地址: %s", s.config.ServiceName, grpcAddress)
		if err := s.server.Serve(lis); err != nil {
			s.logger.Fatalf("gRPC服务器启动失败: %v", err)
		}
	}()

	// 优雅关闭
	s.gracefulShutdown()

	return nil
}

// gracefulShutdown 优雅关闭
func (s *GRPCServer) gracefulShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("收到关闭信号，开始优雅关闭...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("gRPC服务器已优雅关闭")
	case <-ctx.Done():
		s.logger.Warn("优雅关闭超时，强制关闭")
		s.server.Stop()
	}
}
