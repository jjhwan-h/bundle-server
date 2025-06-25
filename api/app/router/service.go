package router

import (
	"net/http"
	"time"

	"github.com/jjhwan-h/bundle-server/api/app/handler"
	"github.com/jjhwan-h/bundle-server/config"
	"github.com/jjhwan-h/bundle-server/database"
	"github.com/jjhwan-h/bundle-server/domain/casb/policy"
	"github.com/jjhwan-h/bundle-server/domain/common/org"
	"github.com/jjhwan-h/bundle-server/domain/common/profile"
	"github.com/jjhwan-h/bundle-server/domain/integration/category"
	"github.com/jjhwan-h/bundle-server/domain/usecase"
	"github.com/jjhwan-h/bundle-server/internal/clients"
	appErr "github.com/jjhwan-h/bundle-server/internal/errors"
	"github.com/jjhwan-h/bundle-server/pkg/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewServiceRouter(r *gin.Engine, logger *zap.Logger, timeout time.Duration) error {
	casbUsecase := usecase.NewCasbUsecase(
		policy.NewPolicySaasRepo(database.GetDB(config.Cfg.DB.Repository["policy_repo"])),
		org.NewOrgGroupRepo(database.GetDB(config.Cfg.DB.Repository["org_repo"])),
		profile.NewProfileUserSubRepo(database.GetDB(config.Cfg.DB.Repository["profile_repo"])),
		category.NewCategoryRepo(database.GetDB(config.Cfg.DB.Repository["category_repo"])),
		policy.NewPolicySaasConfigRepo(database.GetDB(config.Cfg.DB.Repository["policy_repo"])),
	)
	sh := &handler.ServiceHandler{
		CasbUsecase: casbUsecase,
		Client:      clients.NewClient(config.Cfg.Clients.Service),
		Logger:      logger,
	}

	serviceRouter := r.Group("/services", middleware.TimeOutMiddleware(timeout))
	{
		// POST /services/:service/data/trigger
		serviceRouter.POST("/:service/data/trigger", checkAllowedService, sh.BuildDataNBundles)

		// POST /services/:service/policy/trigger
		serviceRouter.POST("/:service/policy/trigger", checkAllowedService, sh.CreateBundle)

		// POST /services/:serivce/policy
		serviceRouter.POST("/:service/policy", checkAllowedService, sh.RegisterPolicy)

		// GET /services/:service/bundle?type=
		serviceRouter.GET("/:service/bundle", checkAllowedService, sh.ServeBundle)

		// POST /services/:service/clients
		serviceRouter.POST("/:service/clients", checkAllowedService, sh.RegisterClients)

		// GET /services/clients
		serviceRouter.GET("/clients", checkAllowedService, sh.ServeClients)

		// GET /services/:service/clients
		serviceRouter.GET("/:service/clients", checkAllowedService, sh.ServeServiceClients)

		// DELETE /services/:service/clients?client=
		serviceRouter.DELETE("/:service/clients", checkAllowedService, sh.DeleteClients)
	}

	return nil
}

func checkAllowedService(c *gin.Context) {
	service := c.Param("service")
	for s := range config.Cfg.Clients.Service {
		if service == s {
			c.Next()
			return
		}
	}
	c.Error(appErr.NewHttpError(
		"bad_request",
		http.StatusBadRequest,
		"Invalid service parameter",
	))

	c.Abort()
}
