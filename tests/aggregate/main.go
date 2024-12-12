package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/thisisnttheway/hx-monitor/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoDatabase = getEnv("MONGODB_DATABASE", "hx")
var client *mongo.Client
var contextTimeout time.Duration = 6 * time.Second

func init() {
	client = Connect()
}

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
		panic("MongoDB connection details are missing in environment variables")
	}
	if mongoUser == "" || mongoPassword == "" {
		panic("MongoDB connection credentials are missing in environment variables")
	}

	mongoUri := fmt.Sprintf("mongodb://%s:%s@%s:%s", mongoUser, mongoPassword, mongoHost, mongoPort)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoUri))
	if err != nil {
		panic("MongoDB connection details are missing in environment variables")
	}

	slog.Info("DB", "action", "connect", "success", true)
	return client
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

func main() {
	type AggregateResult struct {
		AreaID      primitive.ObjectID `bson:"_id"`
		AreaName    string             `bson:"name"`
		CallDetails []models.Call      `bson:"call_details"`
	}

	idString := "674050e7a91502ab9a1681ee"
	referenceID, _ := primitive.ObjectIDFromHex(idString)
	referenceTime := time.Now().Add(-(10 * 24 * time.Hour))

	// Will be run on collection "hx_areas"
	aggregationPipeline := mongo.Pipeline{
		bson.D{{"$match", bson.M{"_id": referenceID}}},

		// Enumerate numbers and calls
		bson.D{{"$lookup", bson.D{
			{"from", "numbers"},
			{"localField", "number_name"},
			{"foreignField", "name"},
			{"as", "number_details"},
		}}},
		bson.D{{"$unwind", "$number_details"}},
		bson.D{{"$lookup", bson.D{
			{"from", "calls"},
			{"localField", "number_details._id"},
			{"foreignField", "number_id"},
			{"as", "call_details"},
		}}},

		// Only return select fields and further filter call_details
		bson.D{{"$project", bson.D{
			{"_id", true},
			{"name", true},
			{"last_action", true},
			{"call_details", bson.D{
				{"$filter", bson.D{
					{"input", "$call_details"},
					{"cond", bson.D{
						{"$gte", bson.A{"$$this.time", referenceTime}},
					}},
				}}},
			}},
		}},
	}

	var o []byte
	o, _ = json.MarshalIndent(aggregationPipeline, "", " ")
	fmt.Println(string(o))

	results, err := Aggregate[AggregateResult]("hx_areas", aggregationPipeline)
	if err != nil {
		panic(err)
	}

	fmt.Printf("[i] Filter by ID: %v\n[i] len results: %d\n", referenceID, len(results))

	hasCompletedSuccessfully := false
	for _, s := range results[0].CallDetails {
		if s.Status == "completed" {
			hasCompletedSuccessfully = true
			break
		}
	}

	o, _ = json.MarshalIndent(results, "", "   ")
	fmt.Println(string(o))
	fmt.Println(hasCompletedSuccessfully)
}
