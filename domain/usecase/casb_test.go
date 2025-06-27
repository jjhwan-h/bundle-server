package usecase

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"testing"

	"github.com/jjhwan-h/bundle-server/database"
	"github.com/jjhwan-h/bundle-server/domain/casb/policy"
	"github.com/jjhwan-h/bundle-server/domain/integration/category"
	"github.com/jjhwan-h/bundle-server/domain/sse/org"
	"github.com/jjhwan-h/bundle-server/domain/sse/profile"

	"github.com/spf13/viper"
)

var (
	once sync.Once
	cu   CasbUsecase
)

func TestBuildDataJson(t *testing.T) {
	cu, err := setup()
	if err != nil {
		t.Fatalf("%v", err)
	}

	data, err := cu.BuildDataJson(context.Background())
	if err != nil {
		t.Fatalf("%v", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("%v", err)
	}
	log.Println(string(jsonData))
}

func setup() (CasbUsecase, error) {
	var setupErr error
	once.Do(func() {
		initConfig()
		err := database.Init([]string{"casb", "common"})
		if err != nil {
			setupErr = err
			return
		}
		pr := policy.NewPolicySaasRepo(database.GetDB("casb"))
		or := org.NewOrgGroupRepo(database.GetDB("common"))
		pur := profile.NewProfileUserSubRepo(database.GetDB("common"))
		cr := category.NewCategoryRepo(database.GetDB("casb"))
		pcr := policy.NewPolicySaasConfigRepo(database.GetDB("casb"))
		cu = &casbUsecase{
			policySaasRepo:       pr,
			orgGroupRepo:         or,
			profileUserSubRepo:   pur,
			categoryRepo:         cr,
			policySaasConfigRepo: pcr,
		}
	})
	return cu, setupErr
}

func initConfig() {
	viper.AddConfigPath("../../")
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		log.Printf("Error reading config file: %v \n", err)
	}
}
