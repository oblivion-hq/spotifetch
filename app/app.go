package app

import (
	"context"
	"fmt"
	"spotify-api/handlers/music"
	"spotify-api/pkgs/utils"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type App struct {
	Router *gin.Engine
	Redis  *redis.Client
}

func InitApp(ctx context.Context) (*App, error) {
	rdb, err := utils.InitRedis(ctx)
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}

	router := utils.InitRouter()

	app := &App{
		Router: router,
		Redis:  rdb,
	}

	app.registerRoutes()

	return app, nil
}

func (a *App) registerRoutes() {
	musicHandler := music.NewMusicHandler(a.Redis)

	a.Router.GET("/health", func(c *gin.Context) {
		if err := a.Redis.Ping(c).Err(); err != nil {
			c.JSON(500, gin.H{"status": "redis down"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	a.Router.GET("/musics/:playlistID", musicHandler.GetPlaylist)
	a.Router.GET("/me/player/recently-played", musicHandler.GetRecentlyPlayedMusic)
	a.Router.GET("/me/player/currently-playing", musicHandler.GetCurrentlyPlayingMusic)
}
