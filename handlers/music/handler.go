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
	"github.com/joho/godotenv"
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

func saveToken(rdb *redis.Client, ctx context.Context, key string, token string, ttl time.Duration) error {
	return rdb.Set(ctx, key, token, ttl).Err()
}

// GET Spotify token
func (m *MusicHandler) getSpotifyToken(ctx context.Context) (*SpotifyToken, error) {
	tokenVal, err := getToken(m.Redis, ctx, "spotify_token")
	if err != nil {
		return nil, fmt.Errorf("smth went wrong with redis while getting spt tkn %v", err)
	}

	fmt.Println(tokenVal)

	if tokenVal != "" {
		return &SpotifyToken{
			AccessToken: tokenVal,
		}, nil
	}

	err = godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load .env")
	}

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("missing Spotify credentials")
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", "https://accounts.spotify.com/api/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", getAuthHeader(clientID, clientSecret))

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("spotify error: %s", string(body))
	}

	var token SpotifyToken
	if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
		return nil, err
	}

	saveToken(m.Redis, ctx, "spotify_token", token.AccessToken, time.Duration(token.ExpiresIn)*time.Second)

	return &token, nil
}

// GET my playlist
func (m *MusicHandler) GetMusics(ctx *gin.Context) {
	token, err := m.getSpotifyToken(ctx.Request.Context())
	if err != nil {
		log.Fatal(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "try later"})
		return
	}

	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/playlists/0cwPcui7aGHkmfHZiD3Hb9", nil)
	if err != nil {
		log.Fatal(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "try later"})
		return
	}

	authStr := fmt.Sprintf("Bearer %s", token.AccessToken)

	req.Header.Set("Authorization", authStr)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "try later"})
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

// GET Playlist by playlistID
func (m *MusicHandler) GetPlaylist(ctx *gin.Context) {
	playlistID, ok := ctx.Params.Get("playlistID")
	if !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	token, err := m.getSpotifyToken(ctx.Request.Context())
	if err != nil {
		log.Fatal(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "try later"})
		return
	}

	spotifyPlaylistEndpoint := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s", playlistID)

	req, err := http.NewRequest("GET", spotifyPlaylistEndpoint, nil)
	if err != nil {
		log.Fatal(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "try later"})
		return
	}

	authStr := fmt.Sprintf("Bearer %s", token.AccessToken)

	req.Header.Set("Authorization", authStr)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "try later"})
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

// Get playlist tracks
func (m *MusicHandler) GetPlaylistTracks(ctx *gin.Context) {
	playlistID, ok := ctx.Params.Get("playlistID")
	if !ok {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "playlist not found"})
		return
	}

	token, err := m.getSpotifyToken(ctx.Request.Context())
	if err != nil {
		log.Fatal(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "try later"})
		return
	}

	spotifyPlaylistEndpoint := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", playlistID)

	req, err := http.NewRequest("GET", spotifyPlaylistEndpoint, nil)
	if err != nil {
		log.Fatal(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "try later"})
		return
	}

	authStr := fmt.Sprintf("Bearer %s", token.AccessToken)

	req.Header.Set("Authorization", authStr)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "try later"})
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "try later", "body": string(body)})
		return
	}

	fmt.Println(res)

	var tracks []*Track
	if err := json.NewDecoder(res.Body).Decode(&tracks); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": "try later", "error": err})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "success", "body": tracks})
}
