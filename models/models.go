package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---------------------------------------------
// DATABASE

type Number struct {
	ID             primitive.ObjectID `bson:"_id"`
	Name           string             `bson:"name"`
	Number         string             `bson:"number"`
	LastCalled     time.Time          `bson:"last_called"`
	LastCallStatus string             `bson:"last_call_status"`
}

type HXArea struct {
	ID                primitive.ObjectID `bson:"_id"`
	Name              string             `bson:"name"`
	NextAction        time.Time          `bson:"next_action"`
	LastAction        time.Time          `bson:"last_action"`
	LastActionSuccess bool               `bson:"last_action_success"`
	SubAreas          []HXSubArea        `bson:"sub_areas"`
	NumberName        string             `bson:"number_name"`
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
	NumberID   primitive.ObjectID `bson:"number_id"`
	HXAreaID   primitive.ObjectID `bson:"hx_area_id"`
	CallSID    string             `bson:"call_sid"`
}

// ---------------------------------------------
// PARSER

type AirspaceStatus struct {
	Areas          []Area      `json:"areas"`
	NextUpdate     time.Time   `json:"nextUpdate"`
	OperatingHours []time.Time `json:"operatingHours"`
}

type Area struct {
	Index  int  `json:"index"`
	Status bool `json:"status"`
}

type TimeSegment struct {
	Type  string
	Times []time.Time
}
