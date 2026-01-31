package conf

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

var globalConfig AppConfig

type AppConfig struct {
	Grafana struct {
		URL               string `yaml:"URL"`
		MySQLSlowQueryAPI string `yaml:"MYSQL_SLOW_QUERY_API"`
		RequestBody       string `yaml:"REQUEST_BODY"`
		AuthToken         string `yaml:"AUTH_TOKEN"`
	} `yaml:"GRAFANA"`

	Global struct {
		ExportFilePath string `yaml:"EXPORT_FILE_PATH"`
		LogFilePath    string `yaml:"LOG_FILE"`
	} `yaml:"GLOBAL"`

	Gitlab struct {
		URL         string `yaml:"URL"`
		ProjectID   int    `yaml:"PROJECT_ID"`
		IssueIID    int    `yaml:"ISSUE_IID"`
		AccessToken string `yaml:"ACCESS_TOKEN"`
	} `yaml:"GITLAB"`

	WeixinRobot struct {
		WebhookURL string `yaml:"WEBHOOK_URL"`
	} `yaml:"WEIXIN_ROBOT"`

	Query struct {
		Interval           string `yaml:"INTERVAL"`
		QueryTimeThreshold string `yaml:"QUERY_TIME_THRESHOLD"`
		LookBackDays       int    `yaml:"TIME_RANGE_DAYS_AGO"`
	} `yaml:"QUERY"`
}

// 初始化配置文件（从配置文件读取）
func InitConfig() error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cfgFileName := "config.yaml"
	cfgPath := pwd + "/config/" + cfgFileName
	// cfgPath := "/opt/sekorm/dailyDataPanel/config/config.yaml"

	cfgF, err := os.ReadFile(cfgPath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(cfgF, &globalConfig)
	if err != nil {
		return err
	}
	return nil
}

// 初始化配置文件（从指定路径读取）
func InitConfigWithPath(configPath string) error {
	cfgF, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	err = yaml.Unmarshal(cfgF, &globalConfig)
	if err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}
	fmt.Printf("[INFO] 配置文件初始化完成: %s\n", configPath)
	return nil
}

func GetAppConfig() AppConfig {
	return globalConfig
}
