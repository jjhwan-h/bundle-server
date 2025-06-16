package router

import (
	"bundle-server/api/app/handler"
	"bundle-server/database"
	"bundle-server/domain/casb/policy"
	"bundle-server/domain/common/org"
	"bundle-server/domain/common/profile"
	"bundle-server/domain/integration/category"
	"bundle-server/domain/usecase"
	"time"

	"github.com/gin-gonic/gin"
)

func NewDataRouter(r *gin.Engine, timeout time.Duration) error {
	casbUsecase := usecase.NewCasbUsecase(
		policy.NewPolicySaasRepo(database.GetDB("casb")),
		org.NewOrgGroupRepo(database.GetDB("common")),
		profile.NewProfileUserSubRepo(database.GetDB("common")),
		category.NewCategoryRepo(database.GetDB("casb")),
		policy.NewPolicySaasConfigRepo(database.GetDB("casb")))

	dh := &handler.DataHandler{
		CasbUsecase: casbUsecase,
	}

	dataRouter := r.Group("/data", TimeOutMiddleware(timeout))
	{
		// POST /data
		dataRouter.POST("", dh.BuildDataJson)
	}

	return nil
}
