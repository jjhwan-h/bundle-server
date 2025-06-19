package cmd

import (
	"bundle-server/api"
	"bundle-server/database"
	"bundle-server/internal/utils"
	"fmt"
	"log"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	dbs = []string{"casb", "common"}
)

var serveCmd = &cobra.Command{
	Use:   `serve`,
	Short: ``,
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			port   string
			appEnv string
		)
		if appEnv = utils.AppMode(); appEnv == "dev" {
			gin.SetMode(gin.DebugMode)
		} else if appEnv == "prod" {
			gin.SetMode(gin.ReleaseMode)
		} else {
			log.Printf("default mode: production")
			gin.SetMode(gin.ReleaseMode)
		}

		if port, _ = cmd.Flags().GetString("port"); port == "" {
			if port = viper.GetString("SERVER_PORT"); port == "" {
				port = "4001"
			}
		}
		port = fmt.Sprintf(":%s", port)

		logPath := viper.GetString("LOG_PATH")
		if logPath == "" {
			logPath = "" // TODO: 수정 필요

		}
		logger := newZapWithLumberjack(appEnv, logPath)

		err := database.Init(dbs)
		if err != nil {
			logger.Fatal("Failed to configure database", zap.Error(err))
		}

		api, err := api.NewServer(port, logger)
		if err != nil {
			logger.Fatal("Failed to configure server", zap.Error(err))
		}
		api.Start(logger)
	},
}

func init() {
	serveCmd.Flags().StringP("port", "p", "4001", "Number of port")
	RootCmd.AddCommand(serveCmd)
}

func newZapWithLumberjack(appEnv string, logPath string) *zap.Logger {
	var (
		writer     zapcore.WriteSyncer
		encoder    zapcore.Encoder
		logLevel   zapcore.Level
		encoderCfg zapcore.EncoderConfig
	)

	if appEnv == "dev" {
		writer = zapcore.AddSync(os.Stdout)
		logLevel = zapcore.DebugLevel
		encoderCfg = zap.NewDevelopmentEncoderConfig()
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	} else {
		writer = zapcore.AddSync(&lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    100, // MB
			MaxBackups: 3,
			MaxAge:     28, // days
			Compress:   true,
		})
		logLevel = zapcore.InfoLevel
		encoderCfg = zap.NewProductionEncoderConfig()
		encoderCfg.TimeKey = "timestamp"
		encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderCfg.CallerKey = "caller"
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	core := zapcore.NewCore(encoder, writer, logLevel)

	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
}
