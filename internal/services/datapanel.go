package services

import (
	"context"
	"dailyDataPanel/internal/api"
	"dailyDataPanel/internal/conf"
	"fmt"
	"time"

	"go.uber.org/zap"
)

func Run() {
	logger := conf.GetLogger()
	logger.Info("初始化完成")
	defer conf.CloseLogger()
	logger.Info("开始Grafana MySQL慢查询日志导出与上传...")

	// 创建Grafana客户端并获取数据
	ctx := context.Background()
	grafanaClient := api.NewGrafanaClient()
	logger.Info("获取慢日志数据", zap.String("action", "Request API"))

	grafanaResp, err := grafanaClient.GetMySQLSlowQueryData(ctx)
	if err != nil {
		logger.Fatal("获取Grafana仪表盘数据失败: " + err.Error())
	}
	logger.Info(fmt.Sprintf("成功获取到 %d 条慢日志数据", grafanaResp.Responses[0].Hits.Total.Value), zap.String("who", "阿里云自建数据库"))

	// TODO：使用阿里云API获取服务商的慢日志
	appConf := conf.GetAppConfig()
	aliResp, err := api.DescribeSlowLogRecords(appConf.Ali.RDS)
	if err != nil {
		logger.Fatal("阿里云慢日志获取失败: " + err.Error())
	}
	logger.Info(fmt.Sprintf("成功获取到 %d 条慢日志数据", len(aliResp)), zap.String("who", "阿里云RDS服务商"))

	// 传入数据将其转换成CSV文件
	filesPath := make([]string, 0)
	logger.Info("开始为MySQL慢日志数据制成CSV报表", zap.String("action", "Convert File"))

	aliGen := NewConvertor(aliResp, "service_mysql_slow_log_weekly")
	graGen := NewConvertor(grafanaResp, "main_mysql_slow_log_weekly")
	// 先服务商后自建
	for _, gen := range []Convertor{aliGen, graGen} {
		filePath, err := gen.Convert()
		if err != nil {
			logger.Fatal("生成CSV文件失败: " + err.Error())
		}
		filesPath = append(filesPath, filePath)
	}
	logger.Info("成功转换为CSV文件", zap.String("action", "Convert"))

	// 上传CSV文件到GitLab
	gitlab := api.NewGitLabAPI()
	uploadResults := make([]string, 0)
	for _, path := range filesPath {
		uploadRes, err := gitlab.UploadFile(ctx, path)
		if err != nil {
			logger.Fatal("上传CSV文件到GitLab发生错误: " + err.Error())
		}
		uploadResults = append(uploadResults, uploadRes)
	}
	logger.Info("GitLab上传文件成功")

	// 计算时间范围
	startDays := time.Now().AddDate(0, 0, 0-appConf.Query.LookBackDays)
	endDays := time.Now().AddDate(0, 0, -1)
	comment := fmt.Sprintf("## %s至%s MySQL慢日志数据导出\n", startDays.Format("2006-01-02"), endDays.Format("2006-01-02"))
	// > 服务商：%s\n> 主流： %s\n
	comment += fmt.Sprintf("> 阿里云RDS服务商：%s\n\n> 阿里云自建数据库： %s\n", uploadResults[0], uploadResults[1])

	err = gitlab.CommentCreate(ctx, comment)
	if err != nil {
		logger.Fatal("GitLab评论失败: " + err.Error())
	}

	//4. 调用API通知群机器人
	wx := api.NewWeixinRobotAPI()
	err = wx.Call(ctx, "<font color=\"warning\">【生产】MySQL慢查询数据报表已更新</font>\n> [跳转详情](http://172.16.1.82/OP/public/issues/9)\n")
	if err != nil {
		logger.Fatal("通知失败: " + err.Error())
	}
	logger.Info("企业微信机器人已通知更新")
}
