package middlewares

import (
	"errors"
	"net/http"
	"strings"
	"sublink/models"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// 随机密钥

// var Secret = []byte("sublink") // 秘钥
// 在 main 中完成配置初始化后赋值，避免首次启动读取到空密钥。
var Secret []byte

const maxTokenLength = 8192

func publicRequest(method, path string) bool {
	if method == http.MethodGet || method == http.MethodHead {
		if path == "/" || path == "/static" || strings.HasPrefix(path, "/static/") || path == "/api/v1/auth/captcha" || path == "/api/v1/version" || path == "/c/" {
			return true
		}
	}
	return method == http.MethodPost && path == "/api/v1/auth/login"
}

// JwtClaims jwt声明
type JwtClaims struct {
	Username       string `json:"username"`
	SessionVersion string `json:"session_version"`
	jwt.StandardClaims
}

// AuthorToken 验证token中间件
func AuthorToken(c *gin.Context) {
	if publicRequest(c.Request.Method, c.Request.URL.Path) {
		c.Next()
		return
	}
	token := c.Request.Header.Get("Authorization")
	if token == "" {
		c.JSON(400, gin.H{"msg": "请求未携带token"})
		c.Abort()
		return
	}
	// Bound work before parsing: Split on an attacker-controlled dot flood allocates
	// a slice proportional to the full header. Count never allocates that slice.
	if len(token) > maxTokenLength || strings.Count(token, ".") != 2 {
		c.JSON(400, gin.H{"msg": "token格式错误"})
		c.Abort()
		return
	}
	// 去掉Bearer前缀
	if len(token) >= 7 && strings.EqualFold(token[:7], "Bearer ") {
		token = token[7:]
	}
	mc, err := ParseToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": "A0230",
			"msg":  "登录已失效，请重新登录",
		})
		c.Abort()
		return
	}
	c.Set("username", mc.Username)
	c.Next()
}

// ParseToken 解析JWT
func ParseToken(tokenString string) (*JwtClaims, error) {
	if len(tokenString) > maxTokenLength || strings.Count(tokenString, ".") != 2 {
		return nil, errors.New("invalid token length or format")
	}
	// 解析token
	token, err := jwt.ParseWithClaims(tokenString, &JwtClaims{}, func(token *jwt.Token) (i interface{}, err error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("invalid signing method")
		}
		return Secret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*JwtClaims); ok && token.Valid { // 校验token
		if claims.SessionVersion == "" || models.DB == nil {
			return nil, errors.New("login revoked")
		}
		user := models.User{Username: claims.Username}
		if err := user.Find(); err != nil || user.SessionVersion != claims.SessionVersion {
			return nil, errors.New("login revoked")
		}
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
