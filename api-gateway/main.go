package main

import (
	"api-gateway-go/config"
	"api-gateway-go/handler"
	"api-gateway-go/helper/logging"
	"api-gateway-go/helper/middleware"
	"api-gateway-go/package/db"
	"api-gateway-go/package/redisclient"
	"api-gateway-go/repository"
	"api-gateway-go/server"
	"api-gateway-go/service"
	"log"
	"net/http"
	"time"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

//nolint:funlen // hard to avoid
func main() {
	// * load config from .env using viper
	config, errConf := config.LoadConfig()
	if errConf != nil {
		log.Fatalf("load config err:%s", errConf)
	}

	// * setup logger config
	logger := logging.New(config.Debug)

	// * connect to postgres using gorm
	sqlDB, errDB := db.NewGormDB(config.Debug, config.Database.Driver, config.Database.URL)
	if errDB != nil {
		logger.Fatal().Err(errDB).Msg("db failed to connect")
	}
	logger.Debug().Msg("db connected")

	// * connect to redis using redis client
	redisClient, errRedis := redisclient.NewRedisClient(
		config.Redis.Addr, config.Redis.ClientName,
		config.Redis.Username, config.Redis.Password,
		config.Redis.DB,
	)
	if errDB != nil {
		logger.Fatal().Err(errRedis).Msg("redis failed to connect")
	}
	logger.Debug().Msg("redis connected")

	// * Initialize the Casbin Gorm adapter and the Casbin enforcer with the model
	if !config.Debug {
		gormadapter.TurnOffAutoMigrate(sqlDB.Gorm)
	}
	adapter, _ := gormadapter.NewAdapterByDB(sqlDB.Gorm)
	enforcer, errEnforcer := casbin.NewEnforcer("./model.conf", adapter)
	if errEnforcer != nil {
		logger.Fatal().Err(errEnforcer).Msg("enforcer failed")
	}
	logger.Debug().Msg("enforcer connected")

	// * closing all connection after get interrupt signal
	defer func() {
		errDBC := sqlDB.Close()
		if errDBC != nil {
			logger.Fatal().Err(errDBC).Msg("db failed to closed")
		}
		logger.Debug().Msg("db closed")

		errRedisC := redisClient.Close()
		if errRedisC != nil {
			logger.Fatal().Err(errRedisC).Msg("redis failed to closed")
		}
		logger.Debug().Msg("redis closed")
	}()

	pingHandler := handler.NewPingGinHandler()

	shortenRepo := repository.NewShortenRepo(sqlDB.SQLDB, redisClient.Redis)
	shortenSvc := service.NewShortenService(shortenRepo)
	shortenHandler := handler.NewShortenHandler(shortenSvc)

	router := gin.New()
	router.Use(cors.Default())

	// * 9. Error and panic handling
	router.Use(middleware.Logger(logger))
	router.Use(gin.Recovery())

	if config.Debug {
		sGroup := router.Group("/s")
		{
			sGroup.GET("/ping", pingHandler.Ping)
			sGroup.POST("/shorten", shortenHandler.Shorten)
		}
	}

	// * 1. Paramater validation,
	// * 6. Dynamic routing using path parameters,
	// * 7. Service discovery using database,
	router.Use(middleware.HashedURLConverter(shortenSvc))

	// * 2. Allow-path
	// allowedPaths := []string{
	// 	"auth/ping",
	// 	"auth/register",
	// 	"auth/login",
	// 	"auth/google/login",

	// 	"login/cms",
	// }
	// * 3. Authentication
	router.Use(middleware.AuthMiddleware(config.JWTSecretKey))
	// * 4. Authorization
	router.Use(middleware.AuthzMiddleware(enforcer))

	router.Use(requestid.New())
	// * 5. Request counter
	router.Use(middleware.RequestCounter(redisClient.Redis))

	// * things outside my capabilities:
	// * 2. Allow-list based on IPs
	// * 10. Circuit Breaker
	// * 11. Monitoring
	// * 12. Cache

	// if config.Debug {
	// 	pprof.Register(short)
	// }

	router.Any("/:hash", shortenHandler.Get)

	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.Debug().Msgf("service will be start at port: %v", config.Port)

	// * run the ListenAndServe() of a server
	if errSrv := server.Run(srv, logger); errSrv != nil {
		logger.Fatal().Err(errSrv).Msg("server shutdown failed")
	}
}
