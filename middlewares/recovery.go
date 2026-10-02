package middlewares

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
)

// Request dumps, panic values and stack source lines may contain credentials.
func SafeRecovery(writer io.Writer) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recover() != nil {
				message := fmt.Sprintf("请求内部错误: method=%q path=%q\n", c.Request.Method, c.Request.URL.Path)
				if writer != nil {
					fmt.Fprint(writer, message)
				} else {
					log.Print(message)
				}
				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}
