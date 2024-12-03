package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

func BlackList(ips []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if lo.Contains(ips, ip) {
			log.Printf("ban request from ip %s", ip)
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
	}
}
