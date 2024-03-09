package main

import (
	"github.com/gin-gonic/gin"
	"github.com/kamva/mgm/v3"
	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	_ "speed-typing-auth-service/docs"
	"speed-typing-auth-service/internal/adapters/handler"
	"speed-typing-auth-service/internal/adapters/mq/rabbitmq"
	"speed-typing-auth-service/internal/adapters/repository/mongodb"
	"speed-typing-auth-service/internal/core/servises"
	"speed-typing-auth-service/pkg/broker"
	"speed-typing-auth-service/pkg/discovery"
	"speed-typing-auth-service/pkg/env"
	"speed-typing-auth-service/pkg/logging"
)

const (
	Dev  = "dev"
	Prod = "prod"
)

func init() {
	env.LoadEnvVariables()
	if os.Getenv("PROFILE") == Prod {
		gin.SetMode(gin.ReleaseMode)
	}
	discovery.InitServiceDiscovery()
}

//	@title		Auth Service API
//	@version	1.0

// @host		localhost:8090
// @BasePath	/api/v1
func main() {
	log := logging.GetLogger()

	initDatabase(log)

	channel := broker.InitMessageBroker()
	defer broker.Close()

	r := gin.Default()
	router := initRouter(log, channel)
	router.InitRoutes(r)

	log.Fatalf("error while running server due to: %s", r.Run())
}

func initDatabase(log logging.Logger) {
	err := mgm.SetDefaultConfig(nil, "auth", options.Client().ApplyURI(os.Getenv("DB_URL")))
	if err != nil {
		log.Fatal("failed connect to database")
	}
}

func initRouter(log logging.Logger, channel *amqp.Channel) *handler.Router {
	refreshTokenRepository := mongodb.NewRefreshTokenRepository()
	userRepository := mongodb.NewUserRepository()

	resultsMigrator := rabbitmq.NewResultsMigrator(channel, log)
	authService := servises.NewAuthService(
		userRepository, refreshTokenRepository,
		resultsMigrator,
		log,
	)
	return handler.NewRouter(
		log,
		handler.NewAuthHandler(
			authService, log,
		),
	)
}
