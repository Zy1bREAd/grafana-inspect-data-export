package api

import (
	"context"
	"dailyDataPanel/internal/conf"
	"fmt"
)

// WeixinRobotAPI 封装微信机器人API操作
type WeixinRobotAPI struct {
	client *HTTPClient
	url    string
}

func NewWeixinRobotAPI() *WeixinRobotAPI {
	appConf := conf.GetAppConfig()
	return &WeixinRobotAPI{
		client: NewDefaultHTTPClient(),
		url:    appConf.WeixinRobot.WebhookURL,
	}
}

// Call 发送消息(Markdown形式)
func (wx *WeixinRobotAPI) Call(ctx context.Context, mdMsg string) error {
	payload := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": mdMsg,
		},
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	_, err := wx.client.PostJSON(ctx, wx.url, payload, headers)
	if err != nil {
		return fmt.Errorf("发送微信机器人消息失败: %w", err)
	}

	return nil
}
