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

	rds20140815 "github.com/alibabacloud-go/rds-20140815/v16/client"
)

// 兼容map[string]any和阿里云API数据的CSV转换方式。
type Convertor interface {
	Convert() (string, error)
	FieldsMap()
}

// 结果集的结构体
type Field struct {
	Key     string // map key
	ColName string // CSV 列名
}

type GrafanaResult struct {
	Data     []api.GrafanaSourceData
	Fields   []Field
	BasePath string
	FileName string
	FullPath string
}

func NewConvertor(data any, fileName string) Convertor {
	appConf := conf.GetAppConfig()
	if appConf.Global.ExportFilePath == "" {
		appConf.Global.ExportFilePath = "/tmp"
	}

	var conv Convertor
	switch v := data.(type) {
	case *api.GrafanaResponse:
		conv = &GrafanaResult{
			Data:     v.Responses[0].Hits.Hits,
			BasePath: appConf.Global.ExportFilePath,
			FileName: fileName,
		}
	case []*rds20140815.DescribeSlowLogRecordsResponseBodyItemsSQLSlowRecord:
		conv = &AliResult{
			Data:     v,
			BasePath: appConf.Global.ExportFilePath,
			FileName: fileName,
		}
	default:
		logger := conf.GetLogger()
		logger.Warn("不支持，无法转换")
		return nil
	}
	conv.FieldsMap()
	return conv
}

func NewGrafanaResult(data *api.GrafanaResponse, fileName string) GrafanaResult {
	appConf := conf.GetAppConfig()
	if appConf.Global.ExportFilePath == "" {
		appConf.Global.ExportFilePath = "/tmp"
	}

	res := GrafanaResult{
		Data:     data.Responses[0].Hits.Hits,
		BasePath: appConf.Global.ExportFilePath,
		FileName: fileName,
	}
	return res
}

// 慢查询日志的字段的中文意思映射
func (gra *GrafanaResult) FieldsMap() {
	// [object Object]      类型    占有内存        Key数量 元素数量
	gra.Fields = []Field{
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
func (gra *GrafanaResult) generateColNames() []string {
	// [object Object]      类型    占有内存        Key数量 元素数量
	colNames := make([]string, len(gra.Fields))
	for i, field := range gra.Fields {
		colNames[i] = field.ColName
	}
	return colNames
}

// 转换成CSV文件并存储在本地
func (gra *GrafanaResult) Convert() (string, error) {
	err := pathIsExist(gra.BasePath)
	if err != nil {
		return "", err
	}

	if beforePath, ok := strings.CutSuffix(gra.BasePath, "/"); ok {
		gra.BasePath = beforePath
	}
	now := time.Now().Format("20060102150405")
	if gra.FileName == "" {
		gra.FileName = "unknown_mysql_slow_query"
	}
	gra.FileName = gra.FileName + "_" + now + ".csv" // 完整文件名
	absFilePath := gra.BasePath + "/" + gra.FileName // 绝对路径
	gra.FullPath = absFilePath
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
	if gra.Data == nil {
		// 空数据直接返回
		return "", errors.New("无数据")
	}

	var colNames []string = gra.generateColNames()

	// 写入表头
	if err := w.Write(colNames); err != nil {
		return "", errors.New("写入表头发生错误: " + err.Error())
	}
	// 写入结果集数据
	for _, row := range gra.Data {
		rowData := gra.generateRowsData(row.Source)
		err := w.Write(rowData)
		if err != nil {
			return "", errors.New("写入表头发生错误: " + err.Error())
		}
	}
	return absFilePath, nil
}

// 提取行数据成切片(当前行)
func (gra *GrafanaResult) generateRowsData(record map[string]any) []string {
	row := make([]string, 0, len(gra.Fields))
	for _, col := range gra.Fields {
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
