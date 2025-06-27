package usecase

import (
	"github.com/jjhwan-h/bundle-server/domain/integration/category"
)

type (
	Data struct {
		DefaultEffect string   `json:"default_effect"`
		Policies      []Policy `json:"policies"`
	}

	Policy struct {
		Priority   int16                      `json:"priority"`
		PolicyID   uint                       `json:"id"`
		PolicyName string                     `json:"name"`
		Subject    Subject                    `json:"subject"`
		Services   []category.CategoryService `json:"services"`
		Effect     string                     `json:"effect"`
	}

	Subject struct {
		Users  []string `json:"users"`
		Groups []string `json:"groups"`
	}

	Patch struct {
		Data []PatchData `json:"data"`
	}

	PatchData struct {
		Op    string      `json:"op"`
		Path  string      `json:"path"`
		Value interface{} `json:"value,omitempty"`
	}
)
