package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type User struct {
	GlobalName          string `json:"global_name"`
	PublicFlags         int    `json:"public_flags"`
	Flags               int    `json:"flags"`
	AccentColor         string `json:"accent_color"`
	AvatarDecorationURL string `json:"avatar_decoration"`
	BannerURL           string `json:"banner_url"`
	BannerColor         string `json:"banner_color"`
}

func main() {
	r := gin.Default()
	r.GET("/:userIDs", func(c *gin.Context) {
		userIDs := strings.Split(c.Param("userIDs"), ",")

		if len(userIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No user IDs provided"})
			return
		}

		godotenv.Load()
		botToken := os.Getenv("BOT_TOKEN")
		client := &http.Client{}

		userResponses := make(map[string]User)

		for _, userID := range userIDs {
			for {
				req, err := http.NewRequest("GET", "https://discord.com/api/v10/users/"+userID, nil)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				req.Header.Add("Authorization", "Bot "+botToken)
				req.Header.Add("Content-Type", "application/json")

				res, err := client.Do(req)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				defer res.Body.Close()

				if res.StatusCode == http.StatusTooManyRequests {
					retryAfter := parseRetryAfterHeader(res.Header)
					if retryAfter > 0 {
						time.Sleep(retryAfter)
						continue // Retry the request after waiting
					}
				}

				body, err := io.ReadAll(res.Body)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				var jsonMap map[string]interface{}
				if err := json.Unmarshal(body, &jsonMap); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				user := User{
					GlobalName:  getStringFromMap(jsonMap, "global_name"),
					PublicFlags: getIntFromMap(jsonMap, "public_flags"),
					Flags:       getIntFromMap(jsonMap, "flags"),
					AccentColor: getStringFromMap(jsonMap, "accent_color"),
					BannerColor: getStringFromMap(jsonMap, "banner_color"),
				}

				if avatarDecoration, ok := jsonMap["avatar_decoration"].(string); ok {
					user.AvatarDecorationURL = "https://cdn.discordapp.com/avatar-decorations/" + userID + "/" + avatarDecoration + ".png"
				}

				if banner, ok := jsonMap["banner"].(string); ok {
					user.BannerURL = "https://cdn.discordapp.com/banners/" + userID + "/" + banner + ".png"
				}

				userResponses[userID] = user
				break // Exit the loop and process the next user ID
			}
		}

		c.JSON(http.StatusOK, userResponses)
	})

	r.GET("/", func(c *gin.Context) {
		c.Writer.WriteHeader(http.StatusOK)
		c.Writer.Write([]byte("<html><body>Please provide a comma-separated list of Discord user IDs or a single user ID in the URL. Example: /688778053408784415,369216554950328320 <br> <a href=\"https://github.com/vaporvee/discord-sdk-godot/\">GitHub project</a></body></html>"))
	})

	r.Run(":8080")
}

func parseRetryAfterHeader(headers http.Header) time.Duration {
	retryAfterStr := headers.Get("Retry-After")
	if retryAfterStr == "" {
		return 0
	}

	retryAfter, err := strconv.Atoi(retryAfterStr)
	if err != nil {
		return 0
	}

	return time.Duration(retryAfter) * time.Millisecond
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getIntFromMap(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return 0
}
