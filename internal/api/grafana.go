package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"dailyDataPanel/internal/conf"
)

type GrafanaClient struct {
	config     conf.AppConfig
	httpClient *http.Client
}

type GrafanaSourceData struct {
	Source map[string]any `json:"_source"`
}

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

func NewGrafanaClient() *GrafanaClient {
	config := conf.GetAppConfig()
	return &GrafanaClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// test
func (g *GrafanaClient) MatchURL() {
	var grafanaURL, grafanaAPI string
	grafanaURL, ok := strings.CutSuffix(g.config.Grafana.URL, "/")
	if !ok {
		grafanaURL = g.config.Grafana.URL
	}
	grafanaAPI, ok = strings.CutPrefix(g.config.Grafana.MySQLSlowQueryAPI, "/")
	if !ok {
		grafanaAPI = g.config.Grafana.MySQLSlowQueryAPI
	}
	url := fmt.Sprintf("%s/%s", grafanaURL, grafanaAPI)
	fmt.Println("debug pirnt", url)
}

// 查询请求体
type ReqBodyParams struct {
	sTimeUnix         int64
	eTimeUnix         int64
	Interval          string
	QueryTimeGtFilter string // 大于指定query_time的过滤器
	// 后续引入排序
}

func newReqBodyParams() ReqBodyParams {
	conf := conf.GetAppConfig()
	fmt.Println(">>>", conf.Query.Interval)

	// 获取前一天完整的起始和结束时间戳
	loc, _ := time.LoadLocation("Asia/Shanghai")
	now := time.Now().In(loc)
	d := 0 - conf.Query.LookBackDays
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
		QueryTimeGtFilter: conf.Query.QueryTimeThreshold,
		Interval:          conf.Query.Interval,
	}
}

func (g *GrafanaClient) buildReqBody(parmas ReqBodyParams) string {
	header := `{"search_type":"query_then_fetch","ignore_unavailable":true,"index":"mysql_slow_log-*"}`
	query := fmt.Sprintf(
		`{"size":10000,"query":{"bool":{"filter":[{"range":{"@timestamp":{"gte":%d,"lte":%d,"format":"epoch_millis"}}},{"range":{"query_time":{"gt":"%s"}}},{"query_string":{"analyze_wildcard":true,"query":"*"}}]}},"sort":[{"@timestamp":{"order":"desc","unmapped_type":"boolean"}},{"_doc":{"order":"desc"}}],"script_fields":{},"aggs":{"1":{"date_histogram":{"interval":"%s","field":"@timestamp","min_doc_count":0,"extended_bounds":{"min":%d,"max":%d},"format":"epoch_millis"},"aggs":{}}},"highlight":{"fields":{"*":{}},"pre_tags":["@HIGHLIGHT@"],"post_tags":["@/HIGHLIGHT@"],"fragment_size":2147483647}}`,
		parmas.sTimeUnix, parmas.eTimeUnix, parmas.QueryTimeGtFilter, parmas.Interval, parmas.sTimeUnix, parmas.eTimeUnix,
	)

	return fmt.Sprintf("%s\n%s\n", header, query)
}

// 传入起始和结束的Unix时间戳参数。示例：1769529600000
func (g *GrafanaClient) GetMySQLSlowQueryData(ctx context.Context) (*GrafanaResponse, error) {
	var grafanaURL, grafanaAPI string
	grafanaURL, ok := strings.CutSuffix(g.config.Grafana.URL, "/")
	if !ok {
		grafanaURL = g.config.Grafana.URL
	}
	grafanaAPI, ok = strings.CutPrefix(g.config.Grafana.MySQLSlowQueryAPI, "/")
	if !ok {
		grafanaAPI = g.config.Grafana.MySQLSlowQueryAPI
	}
	url := fmt.Sprintf("%s/%s", grafanaURL, grafanaAPI)

	// 准备请求体(当前以时间戳来排序)
	params := newReqBodyParams()
	requestBody := g.buildReqBody(params)

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 设置认证
	// auth := base64.StdEncoding.EncodeToString(
	// 	fmt.Appendf([]byte{}, "%s:%s", g.config.Grafana.AuthUser, g.config.Grafana.AuthPwd),
	// )
	req.Header.Set("Authorization", "Bearer "+g.config.Grafana.AuthToken)

	// 发送请求
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求Grafana API失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Grafana API返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", err)
	}

	// 解析JSON响应
	var grafanaResp GrafanaResponse
	if err := json.Unmarshal(body, &grafanaResp); err != nil {
		return nil, fmt.Errorf("解析JSON响应失败: %v", err)
	}

	return &grafanaResp, nil
}
