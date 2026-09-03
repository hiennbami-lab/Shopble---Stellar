package api

import (
	"net/http"
	"time"

	"shopble/api/middleware"
	"shopble/config"
	shopblev1 "shopble/project/shopble/api/v1"

	_ "shopble/project/shopble/docs" // swagger generated docs

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func InitRouter(router *gin.Engine) {
	router.Use(gin.CustomRecovery(middleware.RecoverPanic))
	if config.ReleaseMode != "prod" {
		router.Use(cors.New(cors.Config{
			AllowOrigins:  []string{"*"},
			AllowMethods:  []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:  []string{"Origin", "Content-Type", "Authorization"},
			ExposeHeaders: []string{"Content-Length"},
			MaxAge:        12 * time.Hour,
		}))
	}
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "healthy")
	})

	// Swagger UI tại /swagger/index.html (chỉ enable ngoài prod).
	if config.ReleaseMode != "prod" {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	rootGroup := router.Group("/api/v1", middleware.WrapError)
	shopblev1.Init(rootGroup)
}
