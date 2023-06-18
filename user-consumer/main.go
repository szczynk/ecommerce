package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"user-consumer-go/config"
	"user-consumer-go/handler"
	"user-consumer-go/helper/logging"
	"user-consumer-go/package/db"
	"user-consumer-go/package/rmq"
	"user-consumer-go/repository"
	"user-consumer-go/service"
)

func main() {
	config, errConf := config.LoadConfig()
	if errConf != nil {
		log.Fatalf("load config err:%s", errConf)
	}

	logger := logging.New(config.Debug)

	sqlDB, errDB := db.NewGormDB(config.Debug, config.Database.Driver, config.Database.URL)
	if errDB != nil {
		logger.Fatal().Err(errDB).Msg("db failed to connect")
	}
	logger.Debug().Msg("db connected")

	// connect to rabbitmq using amqp091 and provide publish and subscribe function
	rmqConn, errRmq := rmq.NewConn(config.RabbitMQ.URL)
	if errRmq != nil {
		logger.Fatal().Err(errRmq).Msg("rabbitmq failed to connect")
	}
	logger.Debug().Msg("rabbitmq connected")

	defer func() {
		if errDBC := sqlDB.Close(); errDBC != nil {
			logger.Fatal().Err(errDBC).Msg("db failed to closed")
		}
		logger.Debug().Msg("db closed")

		if errRmqC := rmqConn.Close(); errRmqC != nil {
			logger.Fatal().Err(errRmqC).Msg("rabbitmq failed to closed")
		}
		logger.Debug().Msg("rabbitmq closed")
	}()

	userRepo := repository.NewUserRepository(sqlDB.SQLDB)
	userSvc := service.NewUserService(userRepo)
	userHandler := handler.NewUserHandler(rmqConn, logger, userSvc)

	userSettingRepo := repository.NewUserSettingRepository(sqlDB.SQLDB)
	userSettingSvc := service.NewUserSettingService(userSettingRepo)
	userSettingHandler := handler.NewUserSettingHandler(rmqConn, logger, userSettingSvc)

	uHCreateConsumer, errUHCCon := userHandler.Create()
	if errUHCCon != nil {
		logger.Fatal().Err(errUHCCon).Msg("uHCreateConsumer err")
	}
	logger.Debug().Msg("uHCreateConsumer created")
	defer uHCreateConsumer.Close()

	uHUpdateByIDConsumer, errUHUCon := userHandler.UpdateByID()
	if errUHUCon != nil {
		logger.Fatal().Err(errUHUCon).Msg("uHUpdateByIDConsumer err")
	}
	logger.Debug().Msg("uHUpdateByIDConsumer created")
	defer uHUpdateByIDConsumer.Close()

	uSHUpdateByIDConsumer, errUSHUCon := userSettingHandler.UpdateByUserID()
	if errUSHUCon != nil {
		logger.Fatal().Err(errUSHUCon).Msg("uSHUpdateByIDConsumer err")
	}
	logger.Debug().Msg("uSHUpdateByIDConsumer created")
	defer uSHUpdateByIDConsumer.Close()

	// block main thread - wait for shutdown signal
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logger.Debug().Msgf("sig: %v", sig)
		done <- true
	}()

	logger.Debug().Msg("awaiting signal...")
	<-done
	logger.Debug().Msg("user-consumer shutting down...")
}
