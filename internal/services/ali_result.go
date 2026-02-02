package services

import (
	"dailyDataPanel/internal/conf"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	rds20140815 "github.com/alibabacloud-go/rds-20140815/v16/client"
)

type AliResult struct {
	Data     []*rds20140815.DescribeSlowLogRecordsResponseBodyItemsSQLSlowRecord
	Fields   []Field
	BasePath string
	FileName string
	FullPath string
}

func NewAliResult(data []*rds20140815.DescribeSlowLogRecordsResponseBodyItemsSQLSlowRecord, fileName string) AliResult {
	appConf := conf.GetAppConfig()
	if appConf.Global.ExportFilePath == "" {
		appConf.Global.ExportFilePath = "/tmp"
	}
	res := AliResult{
		Data:     data,
		BasePath: appConf.Global.ExportFilePath,
		FileName: fileName,
	}
	return res
}

// 维护一个慢查询日志的字段的中文意思映射
func (ali *AliResult) FieldsMap() {
	// [object Object]      类型    占有内存        Key数量 元素数量
	ali.Fields = []Field{
		{"ExecutionStartTime", "执行开始时间"},
		{"DBName", "数据库"},
		{"LockTimes", "锁等待时间"},
		{"QueryTimes", "查询耗时"},
		{"ParseRowCounts", "解析行数"},
		{"ReturnRowCounts", "返回行数"},
		{"SQLText", "SQL语句"},
		{"HostAddress", "数据库客户端及地址"},
		{"QueryTimeMS", "查询时间（毫秒）"},
		{"SQLHash", "SQL唯一标识"},
	}
}

// 构造返回慢查询的中文列名
func (ali *AliResult) generateColNames() []string {
	// [object Object]      类型    占有内存        Key数量 元素数量
	colNames := make([]string, len(ali.Fields))
	for i, field := range ali.Fields {
		colNames[i] = field.ColName
	}
	return colNames
}

// 提取行数据成切片(手动构造一行，顺序一一对应)
func (ali *AliResult) generateRowsData(record *rds20140815.DescribeSlowLogRecordsResponseBodyItemsSQLSlowRecord) []string {
	return []string{
		*record.ExecutionStartTime,
		*record.DBName,
		fmt.Sprintf("%d", *record.LockTimes),
		fmt.Sprintf("%d", *record.QueryTimes),
		fmt.Sprintf("%d", *record.ParseRowCounts),
		fmt.Sprintf("%d", *record.ReturnRowCounts),
		*record.SQLText,
		*record.HostAddress,
		fmt.Sprintf("%d", *record.QueryTimeMS),
		*record.SQLHash,
	}
}

// 转换成CSV文件并存储在本地
func (ali *AliResult) Convert() (string, error) {
	err := pathIsExist(ali.BasePath)
	if err != nil {
		return "", err
	}

	if beforePath, ok := strings.CutSuffix(ali.BasePath, "/"); ok {
		ali.BasePath = beforePath
	}
	now := time.Now().Format("20060102150405")
	if ali.FileName == "" {
		ali.FileName = "unknown_service_mysql_slow_log"
	}
	ali.FileName = ali.FileName + "_" + now + ".csv" // 完整文件名
	absFilePath := ali.BasePath + "/" + ali.FileName // 绝对路径
	ali.FullPath = absFilePath
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
	if ali.Data == nil {
		// 空数据直接返回
		return "", errors.New("无数据")
	}

	var colNames []string = ali.generateColNames()

	// 写入表头
	if err := w.Write(colNames); err != nil {
		return "", errors.New("写入表头发生错误: " + err.Error())
	}
	// 写入结果集数据
	for _, row := range ali.Data {
		rowData := ali.generateRowsData(row)
		err := w.Write(rowData)
		if err != nil {
			return "", errors.New("写入表头发生错误: " + err.Error())
		}
	}
	return absFilePath, nil
}
