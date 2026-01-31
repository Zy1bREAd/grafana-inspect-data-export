package main

import (
	"dailyDataPanel/internal/conf"
	"dailyDataPanel/internal/services"
	"log"
)

func main() {
	// Init
	if err := conf.InitConfig(); err != nil {
		log.Fatalf("配置初始化失败: %v", err)
	} else {
		log.Println("配置初始化完成")
	}
	fileCloseFn := conf.InitLogger()
	defer fileCloseFn()
	services.Run()
}
