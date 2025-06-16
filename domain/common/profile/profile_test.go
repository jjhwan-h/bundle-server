package profile

import (
	"bundle-server/database"
	"context"
	"encoding/json"
	"log"
	"sync"
	"testing"

	"github.com/spf13/viper"
)

var (
	once sync.Once
	pr   ProfileUserSubRepo
)

func TestListGcodes(t *testing.T) {
	t.Helper()

	repo, err := setup()
	if err != nil {
		t.Fatalf("%v", err)
	}

	/*===================== user gcode list =====================*/
	gcodes, err := repo.ListGcodes(context.Background(), 3, 2)
	if err != nil {
		t.Fatalf("%v", err)
	}

	jsonGcodes, err := json.MarshalIndent(gcodes, "", "  ")
	if err != nil {
		t.Fatalf("%v", err)
	}

	log.Println("(1)user gcode list Test:\n", string(jsonGcodes))

	/*===================== group gcode list =====================*/
	gcodes, err = repo.ListGcodes(context.Background(), 3, 1)
	if err != nil {
		t.Fatalf("%v", err)
	}

	jsonGcodes, err = json.MarshalIndent(gcodes, "", "  ")
	if err != nil {
		t.Fatalf("%v", err)
	}

	log.Println("(2)group gcode list Test:\n", string(jsonGcodes))
}

func setup() (ProfileUserSubRepo, error) {
	var setupErr error
	once.Do(func() {
		initConfig()
		err := database.Init([]string{"common"})
		if err != nil {
			setupErr = err
			return
		}
		pr = NewProfileUserSubRepo(database.GetDB("common"))
	})
	return pr, setupErr
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
