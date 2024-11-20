package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Number struct {
	ID             primitive.ObjectID `bson:"_id"`
	Name           string             `bson:"name"`
	Number         string             `bson:"number"`
	LastCalled     time.Time          `bson:"last_called"`
	LastCallStatus string             `bson:"last_call_status"`
}

type HXArea struct {
	ID         primitive.ObjectID `bson:"_id"`
	Name       string             `bson:"name"`
	NextAction time.Time          `bson:"next_action"`
	LastAction time.Time          `bson:"last_action"`
	SubAreas   []HXSubArea        `bson:"sub_areas"`
	NumberName string             `bson:"number_name"`
}

type HXSubArea struct {
	Fullname string `bson:"full_name"`
	Name     string `bson:"name"`
	Status   bool   `bson:"status"`
}

type Call struct {
	ID       primitive.ObjectID `bson:"_id"`
	SID      string             `bson:"sid"`
	Time     time.Time          `bson:"time"`
	Status   string             `bson:"status"`
	Cost     string             `bson:"cost"`
	NumberID primitive.ObjectID `bson:"number_id"`
}

type Transcript struct {
	ID         primitive.ObjectID `bson:"_id"`
	Transcript string             `bson:"transcript"`
	Date       time.Time          `bson:"date"`
	Cost       string             `bson:"cost"`
	NumberID   int                `bson:"number_id"`
	HXAreaID   int                `bson:"hx_area_id"`
	CallID     int                `bson:"call_id"`
}
