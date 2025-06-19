package router

import (
	"bundle-server/api/app/handler"
	"bundle-server/database"
	"bundle-server/domain/casb/policy"
	"bundle-server/domain/common/org"
	"bundle-server/domain/common/profile"
	"bundle-server/domain/integration/category"
	"bundle-server/domain/usecase"
	"bundle-server/pkg/middleware"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewDataRouter(r *gin.Engine, logger *zap.Logger, timeout time.Duration) error {
	casbUsecase := usecase.NewCasbUsecase(
		policy.NewPolicySaasRepo(database.GetDB("casb")),
		org.NewOrgGroupRepo(database.GetDB("common")),
		profile.NewProfileUserSubRepo(database.GetDB("common")),
		category.NewCategoryRepo(database.GetDB("casb")),
		policy.NewPolicySaasConfigRepo(database.GetDB("casb")))

	dh := &handler.DataHandler{
		CasbUsecase: casbUsecase,
		Logger:      logger,
	}

	dataRouter := r.Group("/data", middleware.TimeOutMiddleware(timeout))
	{
		// POST /data
		dataRouter.POST("", dh.BuildDataJson)
	}

	return nil
}
