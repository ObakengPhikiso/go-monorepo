package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/obakengphikiso/go-monorepo/libs/shared"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusConfirmed OrderStatus = "confirmed"
	StatusShipped   OrderStatus = "shipped"
	StatusDelivered OrderStatus = "delivered"
	StatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID          string      `json:"id" bson:"_id"`
	UserID      string      `json:"user_id" bson:"user_id"`
	Amount      float64     `json:"amount" binding:"required" bson:"amount"`
	Status      OrderStatus `json:"status" bson:"status"`
	Items       []OrderItem `json:"items" binding:"required" bson:"items"`
	CreatedAt   time.Time   `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at" bson:"updated_at"`
	Description string      `json:"description" bson:"description"`
}

type OrderItem struct {
	ProductID   string  `json:"product_id" binding:"required" bson:"product_id"`
	Quantity    int     `json:"quantity" binding:"required,min=1" bson:"quantity"`
	UnitPrice   float64 `json:"unit_price" binding:"required,min=0" bson:"unit_price"`
	Description string  `json:"description" bson:"description"`
}

var (
	db               *mongo.Database
	ordersCollection *mongo.Collection
)

func connectDB() error {
	// Use shared.GetMongoCollection for orders
	dbURL := shared.GetEnv("ORDERS_DB_URL", "mongodb://mongo:27017")
	coll, err := shared.GetMongoCollection(dbURL, "orders", "orders")
	if err != nil {
		return err
	}
	db = coll.Database()
	ordersCollection = coll

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Create indexes
	_, err = ordersCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "user_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
	})
	return err
}

//func getUserIDFromHeader(c *gin.Context) string {
//	return c.GetHeader("X-User-ID")
//}

func handleGetOrders(c *gin.Context) {
	userID := getUserIDFromHeader(c)
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Parse query parameters
	status := c.Query("status")
	limit := 10 // Default limit
	if limitStr := c.Query("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
		if limit <= 0 {
			limit = 10
		}
	}

	query := bson.M{"user_id": userID}
	if status != "" {
		query["status"] = status
	}

	// Set up options for pagination and sorting
	options := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := ordersCollection.Find(ctx, query, options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch orders"})
		return
	}

	var orders []Order
	if err := cursor.All(ctx, &orders); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode orders"})
		return
	}

	c.JSON(http.StatusOK, orders)
}

func handleGetOrder(c *gin.Context) {
	id := c.Param("id")
	userID := getUserIDFromHeader(c)
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var order Order
	err := ordersCollection.FindOne(ctx, bson.M{
		"_id":     id,
		"user_id": userID,
	}).Decode(&order)

	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch order"})
		return
	}

	c.JSON(http.StatusOK, order)
}

func handleCreateOrder(c *gin.Context) {
	userID := getUserIDFromHeader(c)
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user ID"})
		return
	}

	var order Order
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order.ID = shared.GenerateID()
	order.UserID = userID
	order.Status = StatusPending
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	// Calculate total amount from items
	var total float64
	for _, item := range order.Items {
		total += item.UnitPrice * float64(item.Quantity)
	}
	order.Amount = total

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := ordersCollection.InsertOne(ctx, order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func handleUpdateOrderStatus(c *gin.Context) {
	id := c.Param("id")
	userID := getUserIDFromHeader(c)
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user ID"})
		return
	}

	var update struct {
		Status OrderStatus `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status transition
	if !isValidStatus(update.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := ordersCollection.UpdateOne(
		ctx,
		bson.M{
			"_id":     id,
			"user_id": userID,
		},
		bson.M{
			"$set": bson.M{
				"status":     update.Status,
				"updated_at": time.Now(),
			},
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update order"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order status updated successfully"})
}

func handleCancelOrder(c *gin.Context) {
	id := c.Param("id")
	userID := getUserIDFromHeader(c)
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Only allow cancellation of pending orders
	result, err := ordersCollection.UpdateOne(
		ctx,
		bson.M{
			"_id":     id,
			"user_id": userID,
			"status":  StatusPending,
		},
		bson.M{
			"$set": bson.M{
				"status":     StatusCancelled,
				"updated_at": time.Now(),
			},
		},
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel order"})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found or cannot be cancelled"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order cancelled successfully"})
}

func isValidStatus(status OrderStatus) bool {
	validStatuses := []OrderStatus{
		StatusPending,
		StatusConfirmed,
		StatusShipped,
		StatusDelivered,
		StatusCancelled,
	}

	for _, s := range validStatuses {
		if status == s {
			return true
		}
	}
	return false
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
		log.Fatalf("Failed to connect to database: %v", err)
	}

	r := gin.Default()

	// Health check endpoint
	r.GET("/health", healthCheck)

	// Order endpoints - no auth middleware needed
	orders := r.Group("/orders")
	{
		orders.GET("", handleGetOrders)
		orders.POST("", handleCreateOrder)
		orders.GET("/:id", handleGetOrder)
		orders.PUT("/:id/status", handleUpdateOrderStatus)
		orders.POST("/:id/cancel", handleCancelOrder)
	}

	port := shared.GetEnv("PORT", "8080")
	r.Run(":" + port)
}
