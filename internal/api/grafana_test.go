package api

import (
	"dailyDataPanel/internal/conf"
	"fmt"
	"log"
	"testing"
)

func Test_MatchURL(T *testing.T) {
	if err := conf.InitConfig(); err != nil {
		log.Fatalf("配置初始化失败: %v", err)
	} else {
		log.Println("配置初始化完成")
	}
	g := NewGrafanaClient()
	params := newReqBodyParams()
	a := g.buildReqBody(params)
	fmt.Println(a)

}
