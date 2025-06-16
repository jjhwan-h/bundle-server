package org

import (
	"bundle-server/database"
	"context"
	"log"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

var (
	once sync.Once
	gr   OrgGroupRepo
)

func TestListGidsRecursive(t *testing.T) {
	t.Helper()

	repo, err := setup()
	if err != nil {
		t.Fatalf("%v", err)
	}

	/*=====================Descendant Pids list =====================*/
	pids, err := repo.ListGidsRecursive(context.Background(), []string{"j1_10", "j1_2"})
	if err != nil {
		t.Fatalf("%v", err)
	}

	log.Println("(1)Descendant pids list Test:\n", pids)
}

func setup() (OrgGroupRepo, error) {
	var setupErr error
	once.Do(func() {
		initConfig()
		err := database.Init([]string{"common"})
		if err != nil {
			setupErr = err
			return
		}
		gr = NewOrgGroupRepo(database.GetDB("common"))
	})
	return gr, setupErr
}

func initConfig() {
	viper.AddConfigPath("../../../")
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		log.Printf("Error reading config file: %v \n", err)
	}
}
