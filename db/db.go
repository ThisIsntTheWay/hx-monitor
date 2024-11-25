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
var contextTimeout time.Duration = 6 * time.Second
var mongoDatabase = getEnv("MONGODB_DATABASE", "hx")

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

	slog.Info("DB", "action", "connect", "success", true)
	return client
}

// Insert single document into database
func InsertDocument(colName string, document interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	collection := client.Database(mongoDatabase).Collection(colName)
	_, err := collection.InsertOne(ctx, document)
	if err != nil {
		slog.Error("DB", "error", fmt.Sprintf("Failed to insert document: %v", err))
		return err
	}

	slog.Info("DB", "action", "insertDocument", "colName", colName, "document", document)
	return nil
}

// Insert many documents into database
func InsertDocuments(colName string, documents []interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	collection := client.Database(mongoDatabase).Collection(colName)
	_, err := collection.InsertMany(ctx, documents)
	if err != nil {
		slog.Error("DB", "error", fmt.Sprintf("Failed to insert documents: %v", err))
		return err
	}

	slog.Info("DB", "action", "insertDocuments", "colName", colName, "documents", documents)
	return nil
}

// Update a document in the database
func UpdateDocument(colName string, filter interface{}, update interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	collection := client.Database(mongoDatabase).Collection(colName)
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		slog.Error("DB", "error", fmt.Sprintf("Failed to update document: %v", err))
		return err
	}

	slog.Info("DB", "action", "updateDocument", "colName", colName, "filter", filter, "document", update)
	return nil
}

// Perform an aggregation operation
func Aggregate[T any](colName string, pipeline mongo.Pipeline) ([]T, error) {
	var results []T
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	collection := client.Database(mongoDatabase).Collection(colName)
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		slog.Error("DB", "error", fmt.Sprintf("Error querying document: %v", err.Error()))
		return results, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &results); err != nil {
		return results, err
	}

	return results, nil
}

// Get document from database
func GetDocument[T any](colName string, filter interface{}) ([]T, error) {
	var results []T
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	collection := client.Database(mongoDatabase).Collection(colName)
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		slog.Error("DB", "error", fmt.Sprintf("Error querying document: %v", err.Error()))
		return results, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &results); err != nil {
		logger.LogErrorFatal("DB", fmt.Sprintf("Problem with MongoDB cursor: %v", err.Error()))
	}

	if len(results) == 0 {
		err = fmt.Errorf("the database returned nothing for the given query: %v", filter)
	}

	return results, err
}
