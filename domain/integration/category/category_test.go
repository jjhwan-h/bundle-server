package category

import (
	"bundle-server/database"
	"bundle-server/domain/casb/policy"
	"context"
	"encoding/json"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"
)

var (
	once sync.Once
	cr   CategoryRepo
)

func TestListCategorySummaries(t *testing.T) {
	t.Helper()

	cr, err := setup()
	if err != nil {
		t.Fatalf("%v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	data, err := cr.ListCategorySummaries(ctx)
	if err != nil {
		t.Fatalf("%v", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("%v", err)
	}

	log.Println("(1)category summaries Test:\n", string(jsonData))
}

func TestListCategoryCids(t *testing.T) {
	t.Helper()

	cr, err := setup()
	if err != nil {
		t.Fatalf("%v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	data, err := cr.ListCategoryServices(ctx, []policy.Pid{1, 2})
	if err != nil {
		t.Fatalf("%v", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("%v", err)
	}

	log.Println("(2)category cids Test:\n", string(jsonData))
}

func setup() (CategoryRepo, error) {
	var setupErr error
	once.Do(func() {
		initConfig()
		err := database.Init([]string{"common"})
		if err != nil {
			setupErr = err
			return
		}
		cr = NewCategoryRepo(database.GetDB("common"))
	})

	return cr, setupErr
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
