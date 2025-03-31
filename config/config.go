package config

import (
	"os"

	"github.com/spf13/viper"
)

var Conf *Config

var URL string

type Config struct {
	Server   *Server              `yaml:"server"`
	MySQL    map[string]*MySQL    `yaml:"mysql"`
	Redis    map[string]*Redis    `yaml:"redis"`
	Etcd     *Etcd                `yaml:"etcd"`
	Services map[string]*Service  `yaml:"services"`
	Domain   map[string]*Domain   `yaml:"domain"`
	Mq       map[string]*RabbitMQ `yaml:"mq"`
}

type Server struct {
	Port      string `yaml:"port"`
	Version   string `yaml:"version"`
	JwtSecret string `yaml:"jwtSecret"`
}

type MySQL struct {
	DriverName string `yaml:"driverName"`
	Host       string `yaml:"host"`
	Port       string `yaml:"port"`
	Database   string `yaml:"database"`
	UserName   string `yaml:"username"`
	Password   string `yaml:"password"`
	Charset    string `yaml:"charset"`
}

type Redis struct {
	UserName string `yaml:"userName"`
	Address  string `yaml:"address"`
	Password string `yaml:"password"`
}

type Etcd struct {
	Address  string `yaml:"address"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Service struct {
	Name        string   `yaml:"name"`
	LoadBalance bool     `yaml:"loadBalance"`
	Addr        []string `yaml:"addr"`
}

type Domain struct {
	Name string `yaml:"name"`
}

type RabbitMQ struct {
	Address  string `yaml:"address"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Port     string `yaml:"port"`
}

func InitConfig() {
	workDir, _ := os.Getwd()
	//viper.SetConfigName("test_config")
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(workDir + "/config")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	err = viper.Unmarshal(&Conf)
	if err != nil {
		panic(err)
	}
}
