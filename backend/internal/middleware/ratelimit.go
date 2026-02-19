package middleware

import (
  "net/http"
  "sync"
  "time"

  "github.com/folstingx/server/internal/database"
  "github.com/folstingx/server/internal/models"
  "github.com/gin-gonic/gin"
)

type counter struct {
  Count    int
  ResetAt  time.Time
  LockTill time.Time
}

var rateStore = struct {
  sync.Mutex
  m map[string]*counter
}{m: map[string]*counter{}}

func RateLimitMiddleware() gin.HandlerFunc {
  return func(c *gin.Context) {
    key := c.ClientIP()
    limit := 30
    if u := c.GetString("username"); u != "" {
      key = "user:" + u
      limit = 1000
    }
    if c.Request.URL.Path == "/api/v1/auth/login" {
      key = "login:" + c.ClientIP()
      limit = 5
    }

    now := time.Now()
    rateStore.Lock()
    entry, ok := rateStore.m[key]
    if !ok || now.After(entry.ResetAt) {
      entry = &counter{ResetAt: now.Add(time.Minute)}
      rateStore.m[key] = entry
    }

    if now.Before(entry.LockTill) {
      rateStore.Unlock()
      c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "account locked, retry later"})
      return
    }

    entry.Count++
    count := entry.Count
    if count > limit {
      rateStore.Unlock()
      c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
      return
    }
    rateStore.Unlock()

    c.Next()

    if c.Request.URL.Path == "/api/v1/auth/login" && c.Writer.Status() == http.StatusUnauthorized {
      rateStore.Lock()
      e := rateStore.m[key]
      if e != nil && e.Count >= 5 {
        e.LockTill = time.Now().Add(15 * time.Minute)
      }
      rateStore.Unlock()
    }
  }
}

func APIKeyMiddleware() gin.HandlerFunc {
  return func(c *gin.Context) {
    apiKey := c.GetHeader("X-API-Key")
    if apiKey == "" {
      c.Next()
      return
    }

    var user models.User
    if err := database.DB.Where("api_key = ?", apiKey).First(&user).Error; err != nil {
      c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
      return
    }
    c.Set("user_id", user.ID)
    c.Set("username", user.Username)
    c.Set("role", string(user.Role))
    c.Next()
  }
}

func QuotaMiddleware() gin.HandlerFunc {
  return func(c *gin.Context) {
    userID := c.GetUint("user_id")
    if userID == 0 {
      c.Next()
      return
    }

    var user models.User
    if err := database.DB.First(&user, userID).Error; err != nil {
      c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
      return
    }

    if !user.IsActive || (!user.ExpireAt.IsZero() && user.ExpireAt.Before(time.Now())) {
      c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user expired or inactive"})
      return
    }
    if user.TrafficLimit > 0 && user.TrafficUsed >= user.TrafficLimit {
      c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "traffic quota exceeded"})
      return
    }
    c.Next()
  }
}
