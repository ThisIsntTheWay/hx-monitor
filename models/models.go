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
	FullName   string             `bson:"full_name"`
	Area       string             `bson:"area"`
	NextAction time.Time          `bson:"next_action"`
	NumberName string             `bson:"number_name"`
}

type HXStatus struct {
	ID     primitive.ObjectID `bson:"_id"`
	Status string             `bson:"status"`
	Date   time.Time          `bson:"date"`
	AreaID int                `bson:"area_id"`
}

type Call struct {
	ID            primitive.ObjectID `bson:"_id"`
	SID           string             `bson:"sid"`
	Time          time.Time          `bson:"time"`
	Status        string             `bson:"status"`
	StatusVerbose string             `bson:"status_verbose"`
	Cost          string             `bson:"cost"`
	NumberID      int                `bson:"number_id"`
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
