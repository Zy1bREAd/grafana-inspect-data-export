package main

import (
	"dailyDataPanel/internal/conf"
	"dailyDataPanel/internal/services"
	"flag"
	"fmt"
	"log"
	"os"
)

var appVersion string = "v1.0.0"

func main() {
	// 定义命令行参数
	var (
		configPath = flag.String("config", "config/config.yaml", "配置文件路径")
		help       = flag.Bool("help", false, "显示帮助信息")
		version    = flag.Bool("version", false, "显示版本信息")
	)

	flag.Parse()

	// 显示版本信息
	if *version {
		fmt.Println(appVersion)
		os.Exit(0)
	}

	// 显示帮助信息
	if *help {
		fmt.Println("用法:")
		fmt.Println("  dataPanelExport [选项]")
		fmt.Println("")
		fmt.Println("选项:")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("示例:")
		fmt.Println("  dataPanelExport --config /path/to/config.yaml")
		os.Exit(0)
	}

	// 使用自定义配置路径初始化配置
	if err := conf.InitConfigWithPath(*configPath); err != nil {
		log.Fatalf("配置初始化失败: %v", err)
	} else {
		log.Println("配置初始化完成")
	}

	fileCloseFn := conf.InitLogger()
	defer fileCloseFn()
	services.Run()
}
