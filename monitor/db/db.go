package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	c "github.com/thisisnttheway/hx-monitor/configuration"
	"github.com/thisisnttheway/hx-monitor/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var contextTimeout time.Duration = 6 * time.Second

func init() {
	c.SetUpMongoConfig()
}

func Connect() *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	slog.Info("DB", "action", "connect", "host", c.GetMongoConfig().Host, "port", c.GetMongoConfig().Port)

	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(c.GetMongoConfig().Uri))
	if err != nil {
		logger.LogErrorFatal("DB", fmt.Sprintf("Error while connecting: %v", err.Error()))
	}

	cmd, result := bson.D{{"ping", 1}}, bson.D{}
	if err := client.Database("admin").RunCommand(ctx, cmd).Decode(&result); err != nil {
		logger.LogErrorFatal("DB", fmt.Sprintf("DB unreachable: %v", err))
	}

	slog.Info("DB", "action", "connect", "success", true)
	return client
}

// Insert single document into database
func InsertDocument(colName string, document interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	collection := client.Database(c.GetMongoConfig().Database).Collection(colName)
	_, err := collection.InsertOne(ctx, document)
	if err != nil {
		slog.Error("DB", "error", fmt.Sprintf("Failed to insert document: %v", err))
		return err
	}

	slog.Debug("DB", "action", "insertDocument", "colName", colName, "document", document)
	return nil
}

// Insert many documents into database
func InsertDocuments(colName string, documents []interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	collection := client.Database(c.GetMongoConfig().Database).Collection(colName)
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

	collection := client.Database(c.GetMongoConfig().Database).Collection(colName)
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		slog.Error("DB", "error", fmt.Sprintf("Failed to update document: %v", err))
		return err
	}

	slog.Debug("DB", "action", "updateDocument", "colName", colName, "filter", filter, "document", update)
	return nil
}

// Perform an aggregation operation
func Aggregate[T any](colName string, pipeline mongo.Pipeline) ([]T, error) {
	var results []T
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	collection := client.Database(c.GetMongoConfig().Database).Collection(colName)
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

	collection := client.Database(c.GetMongoConfig().Database).Collection(colName)
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
