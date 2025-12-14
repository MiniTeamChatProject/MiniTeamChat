package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// =======================
// 配置区域
// =======================
var jwtSecret = []byte("MySecretKey_0721") // 生产环境请放入环境变量

// =======================
// 数据模型
// =======================
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"type:varchar(50);unique;not null" json:"username"`
	Password  string    `gorm:"type:varchar(255);not null" json:"-"`
	Nickname  string    `gorm:"type:varchar(50)" json:"nickname"`
	CreatedAt time.Time `json:"created_at"`
}

var DB *gorm.DB

func InitDB() {
	dsn := "host=localhost user=postgres password=07210721 dbname=benutzer_db port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("数据库连接失败: ", err)
	}
	DB.AutoMigrate(&User{})
	fmt.Println("✅ 数据库连接成功")
}

// =======================
// 中间件：JWT 认证 (检票员)
// =======================
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取 Authorization Header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "需要认证Token"})
			return
		}

		// 2. 格式通常是 "Bearer <token>"，我们需要去掉 "Bearer "
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token格式错误"})
			return
		}
		tokenString := parts[1]

		// 3. 解析 Token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		// 4. 验证 Token 有效性
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token无效或已过期"})
			return
		}

		// 5. 将 Token 里的 UserID 存入上下文，供后面的函数使用
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("user_id", claims["user_id"])
		}
		
		c.Next() // 放行
	}
}

// =======================
// 业务逻辑 Handlers
// =======================

// 1. 获取个人信息 (需要登录)
func GetProfile(c *gin.Context) {
	// 从上下文中取出中间件放进去的 user_id
	userId, _ := c.Get("user_id")

	var user User
	if err := DB.First(&user, userId).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户未找到"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"msg": "获取成功",
		"user": user,
	})
}

// 2. 注册
type RegisterInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname"`
}

func Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "数据格式不对"})
		return
	}
	// 查重
	var user User
	if DB.Where("username = ?", input.Username).First(&user).RowsAffected > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名已存在"})
		return
	}
	// 加密
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	newUser := User{Username: input.Username, Password: string(hashedPassword), Nickname: input.Nickname}
	
	if err := DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "注册成功", "data": newUser})
}

// 3. 登录
type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	var user User
	if err := DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}
	// 比对密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}
	// 生成 Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, _ := token.SignedString(jwtSecret)

	c.JSON(http.StatusOK, gin.H{"msg": "登录成功", "token": tokenString, "user": user})
}

// =======================
// 主函数
// =======================
func main() {
	InitDB()
	r := gin.Default()

	// 公开接口
	r.POST("/register", Register)
	r.POST("/login", Login)

	// 受保护接口 (API 组)
	userRoutes := r.Group("/user")
	userRoutes.Use(AuthMiddleware()) // 挂载检票员
	{
		userRoutes.GET("/profile", GetProfile)
	}

	fmt.Println("🚀 服务已启动 :8080")
	r.Run(":8080")
}
