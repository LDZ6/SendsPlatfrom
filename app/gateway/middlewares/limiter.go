package middlewares

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"platform/app/gateway/types"
	"sync"
	"time"
)

type TokenBucket struct {
	capacity  int64
	rate      float64
	tokens    float64
	lastToken time.Time
	mtx       sync.Mutex
}

func (tb *TokenBucket) Allow() bool {
	tb.mtx.Lock()
	defer tb.mtx.Unlock()
	now := time.Now()
	tb.tokens = tb.tokens + tb.rate*now.Sub(tb.lastToken).Seconds()
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}
	if tb.tokens >= 1 {
		tb.tokens--
		tb.lastToken = now
		return true
	} else {
		return false
	}
}

func LimiterHandler(maxCoon int64) gin.HandlerFunc {
	tb := &TokenBucket{
		capacity:  maxCoon,
		rate:      1,
		tokens:    0,
		lastToken: time.Time{},
		mtx:       sync.Mutex{},
	}
	return func(c *gin.Context) {
		if !tb.Allow() {
			types.ResponseErrorWithMsg(c, http.StatusServiceUnavailable, "当前使用人数过多,服务繁忙")
			c.Abort()
			return
		}
		c.Next()
	}
}
