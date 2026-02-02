package conf

import (
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func GetLogger() *zap.Logger {
	if Logger == nil {
		log.Fatalln("[WARN] Zap Logger not init")
	}
	return Logger
}

func CloseLogger() {
	err := Logger.Sync()
	if err != nil {
		log.Fatal(err)
	}
}

// 初始化日志记录
func InitLogger() func() error {
	appConf := GetAppConfig()
	if appConf.Global.LogFilePath == "" {
		log.Fatal("日志文件名为空")
	}
	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	f, err := os.OpenFile(appConf.Global.LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	fileWriteSyncer := zapcore.AddSync(f)
	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, fileWriteSyncer, zapcore.InfoLevel),
	)
	Logger = zap.New(core, zap.AddCaller())
	log.Println("[INFO] 日志记录初始化完成")
	return f.Close
}
