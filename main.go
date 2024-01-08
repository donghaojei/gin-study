package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB
var uploadDir = "./uploads"

type User struct {
	gorm.Model
	jwt.StandardClaims
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

func init() {
	var err error
	dsn := "user=postgres password=a13071611613 dbname=test sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Database connected")
	}

	// 迁移 schema
	db.AutoMigrate(&User{})
}

func generateToken(username string, password string) (string, error) {
	// 创建一个新的 Claims
	User := &User{
		Username: username,
		Password: password,
		StandardClaims: jwt.StandardClaims{
			// 设置 token 的过期时间
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
		},
	}

	// 创建一个新的 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, User)

	// 使用一个密钥对 token 进行签名
	// 你应该使用一个更安全的方式来存储这个密钥，例如使用环境变量
	tokenString, err := token.SignedString([]byte("your-secret-key"))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func createUser(c *gin.Context) {
	var newUser User
	if err := c.ShouldBindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 你可以在这里生成一个新的 token，然后保存到 newUser.Token
	// 这只是一个示例，你应该使用一个更安全的方式来生成 token
	var err error
	newUser.Token, err = generateToken(newUser.Username, newUser.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if result := db.Create(&newUser); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": newUser.Token})
}

func deleteUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	username := user.Username

	// 首先，我们需要找到要删除的用户
	if result := db.First(&user, "username = ?", username); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 然后，我们可以删除用户
	if result := db.Delete(&user); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func updateUser(c *gin.Context) {
	var user User
	var updateUser User

	if err := c.ShouldBindJSON(&updateUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	username := updateUser.Username

	// 首先，我们需要找到要更新的用户
	if result := db.First(&user, "username = ?", username); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if result := db.Model(&user).Updates(updateUser); result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func queryUserByUsername(c *gin.Context) {
	var user User
	username := c.Param("username") // 从 URL 参数中获取用户的 username

	if result := db.First(&user, "username = ?", username); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"token":    user.Token,
	}})
}

func getAllUsers(c *gin.Context) {
	var users []User

	result := db.Find(&users) // 通过 db.Find 方法查询所有的用户

	// 检查是否有错误
	if result.Error != nil {
		log.Fatalf("Failed to get users, error: %v", result.Error)
	} else {
		for _, user := range users {
			fmt.Printf("\n------------------------\n")
			fmt.Printf("User ID: %s\n", user.ID)
			fmt.Printf("Username: %s\n", user.Username)
			fmt.Printf("Password: %s\n", user.Password)
			fmt.Printf("Token: %s\n", user.Token)
			fmt.Printf("------------------------\n")
		}
		c.JSON(http.StatusOK, gin.H{"usersList": users})
	}
}

func uploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 生成文件保存路径
	filename := filepath.Join(uploadDir, file.Filename)

	// 保存文件到服务器
	if err := c.SaveUploadedFile(file, filename); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构造文件的URL
	fileURL := fmt.Sprintf("/uploads/%s", file.Filename)

	// 返回文件的URL给前端
	c.JSON(http.StatusOK, gin.H{"fileURL": fileURL})
}

func main() {
	router := gin.Default()
	// 设置上传文件保存目录
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		fmt.Println("Failed to create upload directory:", err)
		return
	}
	router.POST("/user", createUser)
	router.GET("/users", getAllUsers)
	router.GET("/user/:username", queryUserByUsername)
	router.PUT("/user", updateUser)
	router.DELETE("/user", deleteUser)
	router.POST("/upload", uploadFile)
	// 设置静态文件目录
	router.Static("/uploads", "./uploads")
	router.Run(":8080")
}
