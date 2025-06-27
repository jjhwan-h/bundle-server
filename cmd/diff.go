package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/jjhwan-h/bundle-server/config"
	"github.com/jjhwan-h/bundle-server/database"
	"github.com/jjhwan-h/bundle-server/domain/casb/policy"
	"github.com/jjhwan-h/bundle-server/domain/integration/category"
	"github.com/jjhwan-h/bundle-server/domain/sse/org"
	"github.com/jjhwan-h/bundle-server/domain/sse/profile"
	"github.com/jjhwan-h/bundle-server/domain/usecase"
	"github.com/jjhwan-h/bundle-server/internal/utils"
	"github.com/spf13/cobra"
)

var oldBundle string
var newBundle string
var output string

var diffCmd = &cobra.Command{
	Use:   "diff --old <old_bundle> --new <new_bundle>",
	Short: "A command that compares two bundles' data.json files and generates a patch.json.",
	Long: `A command that compares two bundles' data.json files and generates a patch.json.
	Each bundle must contain exactly one data.json file.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("starting..")
		if oldBundle == "" || newBundle == "" {
			log.Fatalf("--old and --new must both be provided")
		}
		casbUsecase := usecase.NewCasbUsecase(
			policy.NewPolicySaasRepo(database.GetDB(config.Cfg.DB.Repository["policy_repo"])),
			org.NewOrgGroupRepo(database.GetDB(config.Cfg.DB.Repository["org_repo"])),
			profile.NewProfileUserSubRepo(database.GetDB(config.Cfg.DB.Repository["profile_repo"])),
			category.NewCategoryRepo(database.GetDB(config.Cfg.DB.Repository["category_repo"])),
			policy.NewPolicySaasConfigRepo(database.GetDB(config.Cfg.DB.Repository["policy_repo"])),
		)

		oldB, err := ExtractTarGz(oldBundle)
		if err != nil {
			log.Fatalf("failed to unmarshal %s : %v", oldBundle, err)
		}
		newB, err := ExtractTarGz(newBundle)
		if err != nil {
			log.Fatalf("failed to unmarshal %s : %v", newBundle, err)
		}

		patch, err := casbUsecase.BuildPatchJson(oldB, newB)
		if err != nil {
			log.Fatalf("failed to build patch : %v", err)
		}

		buf := new(bytes.Buffer)
		err = utils.EncodeJson(buf, patch)
		if err != nil {
			log.Fatalf("failed to encoding data : %v", err)
		}

		if output == "" {
			io.Copy(os.Stdout, buf)
		} else {
			utils.SaveToFile(context.Background(), buf, output)
		}
	},
}

func init() {
	diffCmd.Flags().StringVar(&oldBundle, "old", "", "Path or name of the old bundle (e.g., regular-v1.0)")
	diffCmd.Flags().StringVar(&newBundle, "new", "", "Path or name of the new bundle (e.g., regular-v1.1)")
	diffCmd.Flags().StringVar(&output, "output", "", "Name of the patch.json (e.g., patch.json)")

	RootCmd.AddCommand(diffCmd)
}

func ExtractTarGz(src string) (*usecase.Data, error) {
	file, err := os.Open(src)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // 끝
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read tar entry: %w", err)
		}

		if strings.HasSuffix(header.Name, "data.json") {
			var data usecase.Data
			dec := json.NewDecoder(tr)
			if err := dec.Decode(&data); err != nil {
				return nil, fmt.Errorf("json decode error: %w", err)
			}
			return &data, nil
		}
	}

	return nil, fmt.Errorf("data.json not found in archive")
}
