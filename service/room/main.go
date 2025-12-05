package main

import (
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"minichat/service/room/model"
)

const (
	// 注意端口是 5433 (Docker)
	DB_DSN = "host=127.0.0.1 user=postgres password=password123 dbname=minichat port=5433 sslmode=disable TimeZone=Asia/Shanghai"
	// 为了快速跑通，这里直接使用和 User Service 相同的密钥
	JWT_KEY   = "my_secret_key_123"
	HTTP_PORT = ":8082" // 房间服务监听 8082
)

var db *gorm.DB

func main() {
	initDB()
	r := setupRouter()

	log.Printf("🚀 Room Service running on %s", HTTP_PORT)
	if err := r.Run(HTTP_PORT); err != nil {
		log.Fatal("Start Error:", err)
	}
}

func initDB() {
	var err error
	db, err = gorm.Open(postgres.Open(DB_DSN), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ DB Connect Error: ", err)
	}
	// 自动迁移 Room 表
	db.AutoMigrate(&model.Room{})
	log.Println("✅ Room Table Migrated!")
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	// CORS 中间件 (允许网页跨域访问)
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 路由配置
	r.GET("/rooms", listRooms)                   // 列出房间 (公开或需登录均可，这里暂设公开)
	r.POST("/rooms", authMiddleware, createRoom) // 创建房间 (必须登录)

	return r
}

// --- 中间件：JWT 验证 ---
// 它的作用是：拦截请求，检查 Header 里有没有带 Token
func authMiddleware(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.AbortWithStatusJSON(401, gin.H{"error": "需要登录 (No Token)"})
		return
	}

	// 提取 Bearer 后面的 token 字符串
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.AbortWithStatusJSON(401, gin.H{"error": "Token 格式错误"})
		return
	}
	tokenStr := parts[1]

	// 解析 Token
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWT_KEY), nil
	})

	if err != nil || !token.Valid {
		c.AbortWithStatusJSON(401, gin.H{"error": "Token 无效或已过期"})
		return
	}

	// 把 Token 里的 UserID 取出来，存到上下文里，给后面的 createRoom 用
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		c.Set("userID", claims["sub"])
	}

	c.Next() // 放行
}

// --- 业务逻辑 ---

// 1. 创建房间
type CreateRoomReq struct {
	Name string `json:"name" binding:"required"`
}

func createRoom(c *gin.Context) {
	var req CreateRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "参数错误"})
		return
	}

	// 从中间件获取当前登录的用户ID
	uidStr, _ := c.Get("userID")
	ownerID, _ := uuid.Parse(uidStr.(string))

	room := model.Room{
		Name:    req.Name,
		OwnerID: ownerID,
	}

	if err := db.Create(&room).Error; err != nil {
		c.JSON(500, gin.H{"error": "创建房间失败"})
		return
	}

	c.JSON(200, gin.H{"message": "创建成功", "id": room.ID, "name": room.Name})
}

// 2. 房间列表
func listRooms(c *gin.Context) {
	var rooms []model.Room
	// 按创建时间倒序排，最新的在前面
	db.Order("created_at desc").Find(&rooms)
	c.JSON(200, gin.H{"rooms": rooms})
}
