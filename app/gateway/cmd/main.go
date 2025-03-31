package main

import (
	"platform/app/gateway/internal"
	"platform/app/gateway/router"
	"platform/app/gateway/rpc"
	"platform/config"
)

func main() {
	config.InitConfig()
	rpc.Init()
	r := router.Router()
	//go internal.SendOnlineCount()
	go internal.YearBillDataInitSync()
	r.Run("0.0.0.0:8889")
}
