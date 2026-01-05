package utils

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func InitRedis(ctx context.Context) (*redis.Client, error) {
	redis_addr := os.Getenv("REDIS_ADDR")
	redis_pass := os.Getenv("REDIS_PASSWORD")
	redis_db := 0 // os.Getenv("REDIS_DB")

	rdb := redis.NewClient(&redis.Options{
		Addr:     redis_addr,
		Password: redis_pass,
		DB:       redis_db,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return rdb, nil
}

func InitRouter() *gin.Engine {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Authorization"},
		MaxAge:          12 * time.Hour,
	}))

	return router
}
