package services

import (
	"context"
	"dailyDataPanel/internal/api"
	"dailyDataPanel/internal/conf"
	"fmt"
	"time"
)

func Run() {
	logger := conf.GetLogger()
	logger.Info("初始化完成")
	defer conf.CloseLogger()
	logger.Info("开始Grafana MySQL慢查询日志导出与上传...")

	// 创建Grafana客户端并获取数据
	ctx := context.Background()
	grafanaClient := api.NewGrafanaClient()
	logger.Info("正在从Grafana获取MySQL慢查询日志数据...")

	grafanaResp, err := grafanaClient.GetMySQLSlowQueryData(ctx)
	if err != nil {
		logger.Fatal("获取Grafana仪表盘数据失败: " + err.Error())
	}
	logger.Info(fmt.Sprintf("成功获取到 %d 条慢查询记录", grafanaResp.Responses[0].Hits.Total.Value))

	// 转成Csv文件并保存
	logger.Info("开始转换Grafana仪表盘数据")
	csvGen := NewCSVResult(grafanaResp, "mysql_slow_query_weekly")
	filePath, err := csvGen.Convert()
	if err != nil {
		logger.Fatal("生成CSV文件失败: " + err.Error())
	}
	logger.Info("成功转换CSV文件")

	// 上传CSV文件到GitLab
	gitlab := api.NewGitLabAPI()
	uploadRes, err := gitlab.UploadFile(ctx, filePath)
	if err != nil {
		logger.Fatal("上传CSV文件到GitLab发生错误: " + err.Error())
	}

	// 计算时间范围
	appConf := conf.GetAppConfig()
	startDays := time.Now().AddDate(0, 0, 0-appConf.Query.LookBackDays)
	endDays := time.Now().AddDate(0, 0, -1)
	err = gitlab.CommentCreate(ctx, fmt.Sprintf("## %s至%s MySQL慢日志数据导出\n> CSV文件：%s\n", startDays.Format("2006-01-02"), endDays.Format("2006-01-02"), uploadRes))
	if err != nil {
		logger.Fatal("GitLab评论失败: " + err.Error())
	}

	//4. 调用API通知群机器人
	wx := api.NewWeixinRobotAPI()
	err = wx.Call(ctx, "<font color=\"warning\">【生产】MySQL慢查询数据报表已更新</font>\n> [跳转详情](http://172.16.1.82/OP/public/issues/9)\n")
	if err != nil {
		logger.Fatal("通知失败: " + err.Error())
	}
	logger.Info("Grafana MySQL慢查询日志导出与上传已完成")
}
