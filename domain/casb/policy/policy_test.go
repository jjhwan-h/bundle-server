package policy

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"testing"

	"github.com/jjhwan-h/bundle-server/database"

	"github.com/spf13/viper"
)

var (
	once sync.Once
)

func TestListPolicies(t *testing.T) {
	t.Helper()

	err := setup()
	if err != nil {
		t.Fatalf("%v", err)
	}

	pr := NewPolicySaasRepo(database.GetDB("casb"))

	data, err := pr.ListPolicies(context.Background())
	if err != nil {
		t.Fatalf("%v", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("%v", err)
	}

	log.Println("(1)list policies Test:\n", string(jsonData))
}

func TestListGroupAttrs(t *testing.T) {
	t.Helper()

	err := setup()
	if err != nil {
		t.Fatalf("%v", err)
	}

	pr := NewPolicySaasRepo(database.GetDB("casb"))

	data, err := pr.ListGroupAttrs(context.Background(), 1)
	if err != nil {
		t.Fatalf("%v", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("%v", err)
	}

	log.Println("(2)list group attrs Test:\n", string(jsonData))
}

func TestListCatePids(t *testing.T) {
	t.Helper()

	err := setup()
	if err != nil {
		t.Fatalf("%v", err)
	}

	pr := NewPolicySaasRepo(database.GetDB("casb"))

	data, err := pr.ListCatePids(context.Background(), 1)
	if err != nil {
		t.Fatalf("%v", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("%v", err)
	}

	log.Println("(3)list cate pids Test:\n", string(jsonData))
}

func TestGetConfig(t *testing.T) {
	t.Helper()

	err := setup()
	if err != nil {
		t.Fatalf("%v", err)
	}

	pr := NewPolicySaasConfigRepo(database.GetDB("casb"))

	data, err := pr.GetConfig(context.Background())
	if err != nil {
		t.Fatalf("%v", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("%v", err)
	}

	log.Println("(1)get config Test:\n", string(jsonData))
}

func setup() error {
	var setupErr error
	once.Do(func() {
		initConfig()
		err := database.Init([]string{"casb"})
		if err != nil {
			setupErr = err
			return
		}
	})
	return setupErr
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
