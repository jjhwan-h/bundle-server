package cmd

import (
	"bundle-server/api"
	"bundle-server/database"
	"bundle-server/internal/utils"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
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
		if port, _ = cmd.Flags().GetString("port"); port == "" {
			if port = viper.GetString("SERVER_PORT"); port == "" {
				port = "4001"
			}
		}
		port = fmt.Sprintf(":%s", port)

		var loggerConfig zap.Config
		if appEnv = utils.AppMode(); appEnv == "dev" {
			loggerConfig = zap.NewDevelopmentConfig()
			gin.SetMode(gin.DebugMode)
		} else if appEnv == "prod" {
			gin.SetMode(gin.ReleaseMode)
			loggerConfig = zap.NewProductionConfig()
		} else {
			log.Printf("default mode: production")
		}

		logger, err := loggerConfig.Build(zap.AddCaller())
		if err != nil {
			log.Fatalf("Failed to build logger: %v", err)
		}

		err = database.Init(dbs)
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
