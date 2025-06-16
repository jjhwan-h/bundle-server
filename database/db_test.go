package database

import (
	"log"
	"testing"

	"github.com/spf13/viper"
)

func TestDBConn(t *testing.T) {
	t.Helper()

	initConfig()
	err := Init([]string{"casb", "common"})
	if err != nil {
		t.Fatalf("failed to connect to DB: %v", err)
	}
}

func TestDBCloseALL(t *testing.T) {

}

func initConfig() {

	viper.AddConfigPath("../")
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		log.Printf("Error reading config file: %v \n", err)
	}
}
