package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/thisisnttheway/hx-checker/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

// Get environment variable with a default value
func getEnv(key string, defaultValue string) string {
	val, ok := os.LookupEnv(key)
	if ok {
		return val
	} else {
		return defaultValue
	}
}

func Connect() *mongo.Client {
	mongoUser := getEnv("MONGO_USER", "")
	mongoPassword := getEnv("MONGO_PASSWORD", "")
	mongoHost := getEnv("MONGO_HOST", "")
	mongoPort := getEnv("MONGO_PORT", "")

	if mongoHost == "" || mongoPort == "" {
		logger.LogErrorFatal("DB", "MongoDB connection details are missing in environment variables")
	}
	if mongoUser == "" || mongoPassword == "" {
		logger.LogErrorFatal("DB", "MongoDB connection credentials are missing in environment variables")
	}

	mongoUri := fmt.Sprintf("mongodb://%s:%s@%s:%s", mongoUser, mongoPassword, mongoHost, mongoPort)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoUri))
	if err != nil {
		logger.LogErrorFatal("DB", "MongoDB connection details are missing in environment variables")
	}

	slog.Info("DB", "message", "MongoDB client has been set up")
	return client
}

// Insert document into database
func InsertDocument(colName string, document interface{}) error {
	mongoDatabase := getEnv("MONGODB_DATABASE", "hx")

	collection := client.Database(mongoDatabase).Collection(colName)
	_, err := collection.InsertOne(context.Background(), document)
	if err != nil {
		slog.Error("DB", "error", fmt.Sprintf("Failed to insert document: %v", err))
		return err
	}

	slog.Info("DB", "message", "Document inserted")
	return nil
}

// Get document from database
func GetDocument[T any](colName string, filter interface{}) ([]T, error) {
	var results []T
	mongoDatabase := getEnv("MONGODB_DATABASE", "hx")

	collection := client.Database(mongoDatabase).Collection(colName)
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		slog.Error("DB", "error", fmt.Sprintf("Error querying document: %v", err.Error()))
	}
	defer cursor.Close(context.TODO())

	if err = cursor.All(context.TODO(), &results); err != nil {
		logger.LogErrorFatal("CALLER", fmt.Sprintf("Problem with MongoDB cursor: %v", err.Error()))
	}

	return results, err
}
