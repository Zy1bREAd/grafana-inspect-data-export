目标：请求Grafana API获取Grafana中MySQL慢查询日志的JSON响应数据，将其转换并生成Excel文件，并通过Gitlab API请求将文件上传至特定Issue中。

要求：
- 基于当前目录结构扩展与生成。
- 后端技术栈使用Golang。
- 启动时，顺序读取config/config.yaml文件加载环境变量。
