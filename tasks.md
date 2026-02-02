# 已完成目标（用于压缩存储上下文信息）
1. 请求Grafana API获取Grafana中MySQL慢查询日志的JSON响应数据，将其转换并生成Excel文件，并通过Gitlab API请求将文件上传至特定Issue中。
2. 新增命令选项参数，增强构建二进制命令执行时的健壮性、完整性。

--- 

# 新目标
当前项目有对GitLab、Grafana、机器人Webhook进行HTTP请求，主要优化代码对其进行抽象、封装。

## 要求
1. 保证健壮性、可复用性，降低代码冗余。
