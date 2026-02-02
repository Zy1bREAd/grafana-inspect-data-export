package api

import (
	"dailyDataPanel/internal/conf"
	"fmt"
	"log"
	"testing"
)

func Test_DescribeSlowLogRecords(T *testing.T) {
	conf.InitConfigWithPath("/opt/sekorm/dailyDataPanel/config/config.yaml")
	res, err := DescribeSlowLogRecords("rm-xxxx")
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(">>", res[:1])
}
