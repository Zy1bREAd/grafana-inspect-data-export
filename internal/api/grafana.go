package api

import (
	"context"
	"dailyDataPanel/internal/conf"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// GrafanaClient 封装Grafana API操作
type GrafanaClient struct {
	client *HTTPClient
	config conf.AppConfig
}

// GrafanaSourceData Grafana数据源结构
type GrafanaSourceData struct {
	Source map[string]any `json:"_source"`
}

// GrafanaResponse Grafana响应结构
type GrafanaResponse struct {
	Responses []struct {
		Hits struct {
			Total struct {
				Value int
			}
			Hits []GrafanaSourceData `json:"hits"`
		} `json:"hits"`
	} `json:"responses"`
}

// ReqBodyParams 查询请求体参数
type ReqBodyParams struct {
	sTimeUnix         int64
	eTimeUnix         int64
	Interval          string
	QueryTimeGtFilter string // 大于指定query_time的过滤器
}

func NewGrafanaClient() *GrafanaClient {
	config := conf.GetAppConfig()
	return &GrafanaClient{
		client: NewDefaultHTTPClient(),
		config: config,
	}
}

// buildURL 构建完整的API URL
func (g *GrafanaClient) buildURL() string {
	grafanaURL, ok := strings.CutSuffix(g.config.Grafana.URL, "/")
	if !ok {
		grafanaURL = g.config.Grafana.URL
	}
	grafanaAPI, ok := strings.CutPrefix(g.config.Grafana.MySQLSlowQueryAPI, "/")
	if !ok {
		grafanaAPI = g.config.Grafana.MySQLSlowQueryAPI
	}
	return fmt.Sprintf("%s/%s", grafanaURL, grafanaAPI)
}

// newReqBodyParams 创建请求体参数
func newReqBodyParams() ReqBodyParams {
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
	startMillis := time.Date(startDay.Year(), startDay.Month(), startDay.Day(), 0, 0, 0, 0, loc).UnixMilli()
	endMillis := time.Date(endDay.Year(), endDay.Month(), endDay.Day(), 23, 59, 59, 0, loc).UnixMilli()

	return ReqBodyParams{
		sTimeUnix:         startMillis,
		eTimeUnix:         endMillis,
		QueryTimeGtFilter: appConf.Query.QueryTimeThreshold,
		Interval:          appConf.Query.Interval,
	}
}

// buildReqBody 构建请求体
func (g *GrafanaClient) buildReqBody(params ReqBodyParams) string {
	header := `{"search_type":"query_then_fetch","ignore_unavailable":true,"index":"mysql_slow_log-*"}`
	query := fmt.Sprintf(
		`{"size":10000,"query":{"bool":{"filter":[{"range":{"@timestamp":{"gte":%d,"lte":%d,"format":"epoch_millis"}}},{"range":{"query_time":{"gt":"%s"}}},{"query_string":{"analyze_wildcard":true,"query":"*"}}]}},"sort":[{"@timestamp":{"order":"desc","unmapped_type":"boolean"}},{"_doc":{"order":"desc"}}],"script_fields":{},"aggs":{"1":{"date_histogram":{"interval":"%s","field":"@timestamp","min_doc_count":0,"extended_bounds":{"min":%d,"max":%d},"format":"epoch_millis"},"aggs":{}}},"highlight":{"fields":{"*":{}},"pre_tags":["@HIGHLIGHT@"],"post_tags":["@/HIGHLIGHT@"],"fragment_size":2147483647}}`,
		params.sTimeUnix, params.eTimeUnix, params.QueryTimeGtFilter, params.Interval, params.sTimeUnix, params.eTimeUnix,
	)

	return fmt.Sprintf("%s\n%s\n", header, query)
}

// GetMySQLSlowQueryData 获取MySQL慢查询数据
func (g *GrafanaClient) GetMySQLSlowQueryData(ctx context.Context) (*GrafanaResponse, error) {
	url := g.buildURL()

	// 准备请求体
	params := newReqBodyParams()
	requestBody := g.buildReqBody(params)

	// 设置请求头
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + g.config.Grafana.AuthToken,
	}

	// 发送请求
	options := &RequestOptions{
		Headers: headers,
		Body:    []byte(requestBody),
	}

	resp, err := g.client.Post(ctx, url, options)
	if err != nil {
		return nil, fmt.Errorf("请求Grafana API失败: %w", err)
	}

	// 解析JSON响应
	var grafanaResp GrafanaResponse
	if err := json.Unmarshal(resp.Body, &grafanaResp); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %w", err)
	}

	return &grafanaResp, nil
}
