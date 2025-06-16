package cmd

import (
	"bundle-server/api"
	"bundle-server/database"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dbs = []string{"casb", "common"}
)

var serveCmd = &cobra.Command{
	Use:   `serve`,
	Short: ``,
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var port string
		if port, _ = cmd.Flags().GetString("port"); port == "" {
			if port = viper.GetString("SERVER_PORT"); port == "" {
				port = "4001"
			}
		}
		port = fmt.Sprintf(":%s", port)

		err := database.Init(dbs)
		if err != nil {
			log.Fatalf("[ERROR] failed to configure database: %v", err)
		}

		api, err := api.NewServer(port)
		if err != nil {
			log.Fatalf("[ERROR] failed to configure server: %v", err)
		}
		api.Start()
	},
}

func init() {
	serveCmd.Flags().StringP("port", "p", "4001", "Number of port")
	RootCmd.AddCommand(serveCmd)
}
