package music

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type MusicHandler struct {
	Redis *redis.Client
}

func NewMusicHandler(rdb *redis.Client) *MusicHandler {
	return &MusicHandler{
		Redis: rdb,
	}
}

func getAuthHeader(clientID, clientSecret string) string {
	authString := fmt.Sprintf("%s:%s", clientID, clientSecret)
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(authString))
	return "Basic " + encodedAuth
}

func getToken(rdb *redis.Client, ctx context.Context, key string) (string, error) {
	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// GET Spotify token
func (m *MusicHandler) getSpotifyToken(ctx context.Context) (*SpotifyToken, error) {
	const tokenKey = "spotify:access_token"
	const lockKey = "spotify:token:lock"

	tokenVal, err := getToken(m.Redis, ctx, tokenKey)
	if err != nil {
		return nil, err
	}
	if tokenVal != "" {
		return &SpotifyToken{AccessToken: tokenVal}, nil
	}

	ok, err := m.Redis.SetNX(ctx, lockKey, "1", 10*time.Second).Result()
	if err != nil {
		return nil, err
	}

	if !ok {
		time.Sleep(200 * time.Millisecond)
		return m.getSpotifyToken(ctx)
	}
	defer m.Redis.Del(ctx, lockKey)

	tokenVal, err = getToken(m.Redis, ctx, tokenKey)
	if err != nil {
		return nil, err
	}
	if tokenVal != "" {
		return &SpotifyToken{AccessToken: tokenVal}, nil
	}

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	refreshToken := os.Getenv("SPOTIFY_REFRESH_TOKEN")

	if clientID == "" || clientSecret == "" || refreshToken == "" {
		return nil, fmt.Errorf("missing spotify env vars")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest(
		"POST",
		"https://accounts.spotify.com/api/token",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", getAuthHeader(clientID, clientSecret))

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("spotify token error: %s", body)
	}

	var token SpotifyToken
	if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
		return nil, err
	}

	ttl := time.Duration(token.ExpiresIn-60) * time.Second
	if err := m.Redis.Set(ctx, tokenKey, token.AccessToken, ttl).Err(); err != nil {
		return nil, err
	}

	return &token, nil
}

func handleErr(ctx *gin.Context, err error) {
	log.Println("handler error:", err)
	ctx.JSON(http.StatusInternalServerError, gin.H{
		"error": "internal server error",
	})
}

// GET Playlist by playlistID
func (m *MusicHandler) GetPlaylist(ctx *gin.Context) {
	playlistID, ok := ctx.Params.Get("playlistID")
	if !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	token, err := m.getSpotifyToken(ctx.Request.Context())
	if err != nil {
		handleErr(ctx, err)
		return
	}

	spotifyPlaylistEndpoint := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s", playlistID)

	req, err := http.NewRequest("GET", spotifyPlaylistEndpoint, nil)
	if err != nil {
		handleErr(ctx, err)
		return
	}

	authStr := fmt.Sprintf("Bearer %s", token.AccessToken)

	req.Header.Set("Authorization", authStr)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		handleErr(ctx, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "try later", "body": string(body)})
		return
	}

	var playlist Playlist
	if err := json.NewDecoder(res.Body).Decode(&playlist); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "try later", "error": err})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "success", "body": playlist})
}

func (m *MusicHandler) GetRecentlyPlayedMusic(ctx *gin.Context) {
	token, err := m.getSpotifyToken(ctx.Request.Context())
	if err != nil {
		handleErr(ctx, err)
		return
	}

	req, _ := http.NewRequest(
		"GET",
		"https://api.spotify.com/v1/me/player/recently-played?limit=1",
		nil,
	)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		handleErr(ctx, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": string(body)})
		return
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		handleErr(ctx, err)
		return
	}

	log.Println("Spotify recently-played raw response:")
	log.Println(string(bodyBytes))

	var data RecentlyPlayedResponse
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		handleErr(ctx, err)
		return
	}

	if len(data.Items) == 0 {
		ctx.JSON(http.StatusOK, gin.H{"message": "silence"})
		return
	}

	item := data.Items[0]

	ctx.JSON(http.StatusOK, gin.H{
		"track":     item.Track,
		"played_at": item.PlayedAt,
		"context":   item.Context,
	})
}
