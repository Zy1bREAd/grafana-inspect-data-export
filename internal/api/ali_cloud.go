package api

import (
	"dailyDataPanel/internal/conf"
	"fmt"
	"log"
	"slices"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	rds20140815 "github.com/alibabacloud-go/rds-20140815/v16/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	credential "github.com/aliyun/credentials-go/credentials"
)

type AliSlowLogResp struct {
	TotalRecordCount int
	PageSize         int
	Records          []*rds20140815.DescribeSlowLogRecordsResponseBodyItemsSQLSlowRecord
}

func CreateClient() (_result *rds20140815.Client, _err error) {
	// 工程代码建议使用更安全的无AK方式，凭据配置方式请参见：https://help.aliyun.com/document_detail/378661.html。
	appConf := conf.GetAppConfig()
	credential, _err := credential.NewCredential(nil)
	if _err != nil {
		return _result, _err
	}

	config := &openapi.Config{
		Credential:      credential,
		AccessKeyId:     &appConf.Ali.AccessKey,
		AccessKeySecret: &appConf.Ali.AccessSecret,
	}
	// Endpoint 请参考 https://api.aliyun.com/product/Rds
	config.Endpoint = tea.String(appConf.Ali.Endpoion)
	fmt.Println(conf.GetAppConfig().Ali)
	_result = &rds20140815.Client{}
	_result, _err = rds20140815.NewClient(config)
	return _result, _err
}

func describeSlowLogRecordsAPI(istID string, pageSize, PageNumber int32) (*AliSlowLogResp, error) {
	appConf := conf.GetAppConfig()
	// 获取前一天完整的起始和结束时间戳
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(loc)
	d := 0 - appConf.Query.LookBackDays
	if d >= -1 {
		d = -1
	}
	startDay := now.AddDate(0, 0, d)
	endDay := now.AddDate(0, 0, -1)
	startTime := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, loc).Format("2006-01-02T15:04Z")
	endTime := time.Date(endDay.Year(), endDay.Month(), endDay.Day(), 23, 59, 59, 0, loc).Format("2006-01-02T15:04Z")

	// 阿里云Client请求API
	client, _err := CreateClient()
	if _err != nil {
		return nil, _err
	}

	describeSlowLogRecordsRequest := &rds20140815.DescribeSlowLogRecordsRequest{
		DBInstanceId: tea.String(istID),
		StartTime:    tea.String(startTime),
		EndTime:      tea.String(endTime),
		PageSize:     tea.Int32(pageSize),
		PageNumber:   tea.Int32(PageNumber),
	}
	fmt.Println("describeSlowLogRecordsRequest", describeSlowLogRecordsRequest)
	runtime := &util.RuntimeOptions{}
	resp, err := client.DescribeSlowLogRecordsWithOptions(describeSlowLogRecordsRequest, runtime)
	if err != nil {
		var error = &tea.SDKError{}
		if _t, ok := err.(*tea.SDKError); ok {
			error = _t
		} else {
			error.Message = tea.String(err.Error())
		}
		// logger := conf.GetLogger()
		// logger.Error(tea.StringValue(error.Message))
		log.Println(tea.StringValue(error.Message))
		return nil, err
	}
	res := resp.Body.Items.SQLSlowRecord
	return &AliSlowLogResp{
		TotalRecordCount: int(*resp.Body.TotalRecordCount),
		PageSize:         int(*resp.Body.PageRecordCount),
		Records:          res,
	}, nil
}

func DescribeSlowLogRecords(istID string) ([]*rds20140815.DescribeSlowLogRecordsResponseBodyItemsSQLSlowRecord, error) {
	// 核心：计算总页数，然后进入循环，最后组合切片数据结果。
	pageSize := 100
	aliResp, err := describeSlowLogRecordsAPI(istID, int32(pageSize), 1)
	if err != nil {
		return nil, err
	}
	allRes := slices.Clone(aliResp.Records)
	totalPages := (aliResp.TotalRecordCount / pageSize) + 1
	for i := 2; i <= totalPages; i++ {
		aliResp, err := describeSlowLogRecordsAPI(istID, int32(pageSize), int32(i))
		if err != nil {
			return nil, err
		}
		allRes = append(allRes, aliResp.Records...)
	}
	return allRes, nil
}
