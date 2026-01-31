package api

import (
	"bytes"
	"context"
	"dailyDataPanel/internal/conf"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// 主要是封装一个操作wx API的Handler
type WeixinRobotAPI struct {
	URL string
}

func NewWeixinRobotAPI() *WeixinRobotAPI {
	appConf := conf.GetAppConfig()
	return &WeixinRobotAPI{
		URL: appConf.WeixinRobot.WebhookURL,
	}
}

// 发送消息(Markdown形式)
func (wx *WeixinRobotAPI) Call(ctx context.Context, mdMsg string) error {
	payload := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": mdMsg,
		},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	// 评论API接口地址
	req, err := http.NewRequestWithContext(ctx, "POST", wx.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	// req.Header.Set("PRIVATE-TOKEN", wx.AccessToken)
	// 设置请求头，携带JSON形式的POST请求体
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// 获取响应结果
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// 状态码检查（成功是201）
	if resp.StatusCode > 299 || resp.StatusCode < 200 {
		return errors.New("request error: " + string(respBody))
	}

	return nil
}
