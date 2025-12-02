package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	// ❗请确认你的 go.mod 第一行是 module minichat/services/user
	"minichat/services/user/model"
)

const (
	// 数据库连接配置 (注意：这里用了 5433 端口，防止和本地冲突)
	DB_DSN  = "host=127.0.0.1 user=postgres password=password123 dbname=minichat port=5433 sslmode=disable TimeZone=Asia/Shanghai"
	JWT_KEY = "my_secret_key_123"
)

var db *gorm.DB

func main() {
	// 1. 连接数据库
	initDB()

	// 2. 配置路由并启动
	r := setupRouter()

	log.Println("🚀 HTTP Server running on :8081")
	// 启动 HTTP 服务，监听 8081 端口
	if err := r.Run(":8081"); err != nil {
		log.Fatal("Server start failed: ", err)
	}
}

func initDB() {
	var err error
	// 连接 Postgres
	db, err = gorm.Open(postgres.Open(DB_DSN), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ 连接数据库失败: ", err)
	}

	// 自动创建表结构 (Auto Migrate)
	db.AutoMigrate(&model.User{})
	log.Println("✅ 数据库连接成功！表结构已迁移。")
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	// --- 关键修改：添加 CORS 中间件 ---
	// 这是为了让你直接打开 HTML 文件也能访问接口
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})
	// ----------------------------------

	r.POST("/register", handleRegister)
	r.POST("/login", handleLogin)

	return r
}

// --- 业务逻辑处理 ---

type RegisterReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func handleRegister(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数格式错误"})
		return
	}

	// 密码加密
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	user := model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPwd),
	}

	// 写入数据库
	if err := db.Create(&user).Error; err != nil {
		log.Println("Insert error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败，用户名或邮箱可能已存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "注册成功", "uid": user.ID})
}

type LoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func handleLogin(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数格式错误"})
		return
	}

	// 查找用户
	var user model.User
	if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "账号不存在"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	// 生成 Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      user.ID.String(),
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenStr, err := token.SignedString([]byte(JWT_KEY))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token 生成失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "登录成功",
		"token":    tokenStr,
		"username": user.Username,
	})
}
