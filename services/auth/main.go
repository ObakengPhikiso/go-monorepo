package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"log"

	"github.com/gin-gonic/gin"
	"github.com/ObakengPhikiso/monorepo/libs/shared"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

const (
	// Security settings
	minPasswordLength    = 8
	maxLoginAttempts     = 5
	loginLockoutDuration = 15 * time.Minute
)

type User struct {
	ID            string    `json:"id" bson:"_id"`
	Username      string    `json:"username" binding:"required" bson:"username"`
	Password      string    `json:"password" binding:"required" bson:"-"`
	Hash          string    `json:"-" bson:"password"`
	Created       time.Time `json:"created" bson:"created"`
	LoginAttempts int       `json:"-" bson:"login_attempts"`
	LastAttempt   time.Time `json:"-" bson:"last_attempt"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

var (
	db              *mongo.Database
	usersCollection *mongo.Collection
)

func connectDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbURL := shared.GetEnv("AUTH_DB_URL", "mongodb://mongo:27017")
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbURL))
	if err != nil {
		return err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return err
	}

	db = client.Database("auth")
	usersCollection = db.Collection("users")

	// Create unique index for username
	_, err = usersCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}
	// Add more password validation rules as needed
	return nil
}

func handleRegister(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate password
	if err := validatePassword(user.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	user.ID = shared.GenerateID()
	user.Hash = string(hash)
	user.Created = time.Now()
	user.LoginAttempts = 0

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = usersCollection.InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// Generate JWT token for the new user
	token, err := shared.GenerateJWT(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, LoginResponse{Token: token})
}

func handleLogin(c *gin.Context) {
	var loginReq User
	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Normalize username
	loginReq.Username = strings.TrimSpace(strings.ToLower(loginReq.Username))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	err := usersCollection.FindOne(ctx, bson.M{"username": loginReq.Username}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}

	// Check for lockout
	if user.LoginAttempts >= maxLoginAttempts {
		lockoutEnds := user.LastAttempt.Add(loginLockoutDuration)
		if time.Now().Before(lockoutEnds) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "account is temporarily locked",
				"retry_after": lockoutEnds.Sub(time.Now()).Seconds(),
			})
			return
		}
		// Reset attempts if lockout period has passed
		user.LoginAttempts = 0
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Hash), []byte(loginReq.Password)); err != nil {
		// Increment login attempts
		update := bson.M{
			"$inc": bson.M{"login_attempts": 1},
			"$set": bson.M{"last_attempt": time.Now()},
		}
		_, _ = usersCollection.UpdateOne(ctx, bson.M{"_id": user.ID}, update)

		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Reset login attempts on successful login
	_, _ = usersCollection.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
		"$set": bson.M{
			"login_attempts": 0,
			"last_attempt":   time.Now(),
		},
	})

	token, err := shared.GenerateJWT(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{Token: token})
}

func handleValidate(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no token provided"})
		return
	}

	// Remove "Bearer " prefix if present
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	claims, err := shared.ValidateJWT(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":  claims.UserID,
		"username": claims.Username,
	})
}

func healthCheck(c *gin.Context) {
	var status string
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Check MongoDB connection
	err := db.Client().Ping(ctx, nil)
	if err != nil {
		status = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  status,
			"message": "database connection failed",
		})
		return
	}

	status = "healthy"
	c.JSON(http.StatusOK, gin.H{
		"status":  status,
		"version": shared.Version(),
	})
}

func main() {
	if err := connectDB(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	r := gin.Default()

	// Health check endpoint
	r.GET("/health", healthCheck)

	r.POST("/register", handleRegister)
	r.POST("/login", handleLogin)
	r.POST("/validate", handleValidate)

	port := shared.GetEnv("PORT", "8084")
	r.Run(":" + port)
}
