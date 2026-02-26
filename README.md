# MySQL 慢查询日志导出工具

定期从 Grafana 和阿里云 RDS 获取 MySQL 慢查询日志，生成 CSV 报表并上传至 GitLab，同时发送企业微信通知。

## 功能特性

- 从 Grafana 获取自建数据库慢日志
- 从阿里云 RDS 获取服务商慢日志
- 转换为 CSV 文件（支持中文列名，解决 Excel 乱码）
- 上传 CSV 至 GitLab Issue
- 企业微信机器人自动通知

## 技术栈

- Go 1.24.9
- 阿里云 RDS SDK

## 项目结构

```
.
├── cmd/
│   └── main.go              # 程序入口
├── internal/
│   ├── api/                 # API 客户端
│   │   ├── grafana.go       # Grafana API
│   │   ├── ali_cloud.go     # 阿里云 RDS API
│   │   ├── gitlab.go        # GitLab API
│   │   └── weixin_robot.go  # 企业微信机器人
│   ├── conf/                # 配置管理
│   │   ├── config.go        # YAML 配置解析
│   │   └── logger.go        # 日志管理
│   └── services/           # 核心业务逻辑
│       ├── datapanel.go     # 主流程
│       ├── result.go        # CSV 转换
│       └── ali_result.go    # 阿里云数据转换
└── config/
    └── config.yaml（          # 配置文件需自行配置）
```

## 配置说明

配置文件 `config/config.yaml`：

```yaml
GRAFANA:
  URL: "<Grafana 地址>"
  MYSQL_SLOW_QUERY_API: "<慢查询 API 路径>"
  AUTH_TOKEN: "<认证令牌>"

GLOBAL:
  EXPORT_FILE_PATH: "/tmp"
  LOG_FILE: "<日志文件路径>"

GITLAB:
  URL: "<GitLab 地址>"
  PROJECT_ID: <项目 ID>
  ISSUE_IID: <Issue 编号>
  ACCESS_TOKEN: "<访问令牌>"

WEIXIN_ROBOT:
  WEBHOOK_URL: "<机器人 Webhook 地址>"

QUERY:
  INTERVAL: "1h"
  QUERY_TIME_THRESHOLD: "1s"
  TIME_RANGE_DAYS_AGO: 7

ALI:
  RDS: "<RDS 实例 ID>"
  ACCESS_KEY: "<阿里云 AccessKey>"
  ACCESS_SECRET: "<阿里云 AccessSecret>"
  ENDPOINT: "<阿里云 RDS 端点>"
```

## 构建与运行

```bash
# 构建
go build -o datapanel-export ./cmd/main.go

# 运行
./datapanel-export --config config/config.yaml

# 查看版本
./datapanel-export --version

# 查看帮助
./datapanel-export --help
```

## 安全注意事项

- 配置文件包含敏感凭证，请妥善保管
- 建议使用环境变量或密钥管理服务替代硬编码
- 添加到 `.gitignore` 避免提交到版本库

## License

MIT
