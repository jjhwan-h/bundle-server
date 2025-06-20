package cmd

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var RootCmd = &cobra.Command{
	Use:   ``,
	Short: ``,
	Long:  ``,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initEnv)
}

func initEnv() {
	_ = godotenv.Load("./.env")
	viper.AutomaticEnv()
}
