package services

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"dailyDataPanel/internal/api"
	"dailyDataPanel/internal/conf"
)

// 结果集的结构体
type Field struct {
	Key     string // map key
	ColName string // CSV 列名
}

type CSVResult struct {
	Data     []api.GrafanaSourceData
	BasePath string
	FileName string
	FullPath string
}

func NewCSVResult(data *api.GrafanaResponse, fileName string) CSVResult {
	appConf := conf.GetAppConfig()
	if appConf.Global.ExportFilePath == "" {
		appConf.Global.ExportFilePath = "/tmp"
	}
	return CSVResult{
		Data:     data.Responses[0].Hits.Hits,
		BasePath: appConf.Global.ExportFilePath,
		FileName: fileName,
	}
}

// 慢查询日志的字段的中文意思映射
func (cr *CSVResult) slowQueryFieldsMap() []Field {
	// [object Object]      类型    占有内存        Key数量 元素数量
	return []Field{
		{"@timestamp", "时间戳"},
		{"db_name", "数据库"},
		{"db_user", "数据库用户名"},
		{"lock_time", "锁等待时间"},
		{"query_time", "查询耗时"},
		{"rows_examined", "扫描行数"},
		{"rows_sent", "返回行数"},
		{"sql_statement", "SQL语句"},
		{"message", "日志消息"},
	}
}

// 构造返回慢查询的中文列名
func (cr *CSVResult) slowQueryColNames() []string {
	// [object Object]      类型    占有内存        Key数量 元素数量
	fields := cr.slowQueryFieldsMap()
	colNames := make([]string, len(fields))
	for k, f := range fields {
		colNames[k] = f.ColName
	}
	return colNames
}

// 转换成CSV文件并存储在本地
func (cr *CSVResult) Convert() (string, error) {
	err := pathIsExist(cr.BasePath)
	if err != nil {
		return "", err
	}

	if beforePath, ok := strings.CutSuffix(cr.BasePath, "/"); ok {
		cr.BasePath = beforePath
	}
	now := time.Now().Format("20060102150405")
	if cr.FileName == "" {
		cr.FileName = "unknown_mysql_slow_query"
	}
	cr.FileName = cr.FileName + "_" + now + ".csv" // 完整文件名
	absFilePath := cr.BasePath + "/" + cr.FileName // 绝对路径
	cr.FullPath = absFilePath
	f, err := os.Create(absFilePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// 避免Window Excel打开中文乱码
	f.WriteString("\xEF\xBB\xBF")

	// 制作表头数据
	w := csv.NewWriter(f)
	defer w.Flush()
	if cr.Data == nil {
		// 空数据直接返回
		return "", errors.New("无数据")
	}

	var colNames []string = cr.slowQueryColNames()
	var colKeys []Field = cr.slowQueryFieldsMap()

	// 写入表头
	if err := w.Write(colNames); err != nil {
		return "", errors.New("写入表头发生错误: " + err.Error())
	}
	// 写入结果集数据
	for _, row := range cr.Data {
		rowData := generateRowsData(row.Source, colKeys)
		err := w.Write(rowData)
		if err != nil {
			return "", errors.New("写入表头发生错误: " + err.Error())
		}
	}
	return absFilePath, nil
}

// 提取行数据成切片(当前行)
func generateRowsData(record map[string]any, headers []Field) []string {
	row := make([]string, 0, len(headers))
	for _, col := range headers {
		var colData string
		if val, exist := record[col.Key]; exist {
			switch v := val.(type) {
			case string:
				colData = v
			case float64:
				colData = strconv.FormatFloat(v, 'f', -1, 64)
			case int:
				colData = strconv.Itoa(v)
			case int64:
				colData = strconv.FormatInt(v, 10)
			default:
				colData = fmt.Sprintf("%v", v)
			}
		} else {
			colData = "N/A"
		}
		row = append(row, colData)
	}
	return row
}

// 判断路径是否存在
func pathIsExist(base string) error {
	// 创建文件，不存在目录则创建
	_, err := os.Stat(base)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(base, 0755)
			if err != nil {
				return errors.New("创建文件失败, " + err.Error())
			}
		} else {
			return errors.New("unknown error, " + err.Error())
		}
	}
	return nil
}
