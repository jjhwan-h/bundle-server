package cmd

import (
	"github.com/jjhwan-h/bundle-server/api"
	"github.com/jjhwan-h/bundle-server/database"

	"fmt"
	"log"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/jjhwan-h/bundle-server/config"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var serveCmd = &cobra.Command{
	Use:   "serve -p <port>",
	Short: "Start the API server",
	Run: func(cmd *cobra.Command, args []string) {
		runServer(cmd)
	},
}

func init() {
	serveCmd.Flags().StringP("port", "p", "4001", "Port to run the server on")
	RootCmd.AddCommand(serveCmd)
}

func runServer(cmd *cobra.Command) {
	err := config.LoadConfig("./config.yaml")
	if err != nil {
		log.Fatal("config.yaml is missing or invalid format")
	}

	appEnv := config.Cfg.AppEnv
	setGinMode(appEnv)

	port := resolvePort(cmd)
	logPath := config.Cfg.Logger.FileName

	if appEnv == "prod" && logPath == "" {
		log.Fatal("logger.file_name is empty")
	}

	logger := newZapLogger(appEnv, logPath)

	dbs := config.Cfg.DB.DataBase
	if len(dbs) == 0 {
		logger.Fatal(
			"Failed to configure database",
			zap.Error(
				fmt.Errorf("database to connect to is not configured in the config.yaml file")),
		)
	}
	if err := database.Init(dbs); err != nil {
		logger.Fatal("Failed to configure database", zap.Error(err))
	}

	server, err := api.NewServer(port, logger)
	if err != nil {
		logger.Fatal("Failed to configure server", zap.Error(err))
	}

	server.Start(logger)
}

func setGinMode(env string) {
	switch env {
	case "dev":
		gin.SetMode(gin.DebugMode)
	case "prod":
		gin.SetMode(gin.ReleaseMode)
	default:
		log.Printf("Unknown appEnv '%s', defaulting to ReleaseMode", env)
		gin.SetMode(gin.ReleaseMode)
	}
}

func resolvePort(cmd *cobra.Command) string {
	port, _ := cmd.Flags().GetString("port")
	if port == "" {
		port = os.Getenv("SERVER_PORT")
		if port == "" {
			port = "4001"
		}
	}
	return fmt.Sprintf(":%s", port)
}

func newZapLogger(appEnv string, logPath string) *zap.Logger {
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
			MaxSize:    config.Cfg.Logger.MaxSize, // MB
			MaxBackups: config.Cfg.Logger.MaxBackups,
			MaxAge:     config.Cfg.Logger.MaxAge, // days
			Compress:   config.Cfg.Logger.Compress,
		})
		logLevel = zapcore.InfoLevel
		encoderCfg = zap.NewProductionEncoderConfig()
		encoderCfg.TimeKey = "timestamp"
		encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderCfg.CallerKey = "caller"
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	core := zapcore.NewCore(encoder, writer, logLevel)

	return zap.New(core, zap.AddCaller())
}
