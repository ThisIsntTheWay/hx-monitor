package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ---------------------------------------------
// DATABASE
type Number struct {
	ID             primitive.ObjectID `bson:"_id" json:"_id"`
	Name           string             `bson:"name" json:"name"`
	Number         string             `bson:"number" json:"number"`
	LastCalled     time.Time          `bson:"last_called" json:"last_called"`
	LastCallStatus string             `bson:"last_call_status" json:"last_call_status"`
}

type HXArea struct {
	ID                   primitive.ObjectID `bson:"_id" json:"_id"`
	Name                 string             `bson:"name" json:"name"`
	NextAction           time.Time          `bson:"next_action" json:"next_action"`
	LastAction           time.Time          `bson:"last_action" json:"last_action"`
	LastActionSuccess    bool               `bson:"last_action_success" json:"last_action_success"`
	FlightOperatingHours []time.Time        `bson:"flight_operating_hours" json:"flight_operating_hours"`
	SubAreas             []HXSubArea        `bson:"sub_areas" json:"sub_areas"`
	NumberName           string             `bson:"number_name" json:"number_name"`
	LastError            string             `bson:"last_error" json:"last_error"`
	NumErrors            int8               `bson:"num_errors" json:"num_errors"`
}

type HXSubArea struct {
	FullName string `bson:"full_name" json:"full_name"`
	Name     string `bson:"name" json:"name"`
	Active   bool   `bson:"active" json:"active"`
}

type Call struct {
	ID       primitive.ObjectID `bson:"_id" json:"_id"`
	SID      string             `bson:"sid" json:"sid"`
	Time     time.Time          `bson:"time" json:"time"`
	Status   string             `bson:"status" json:"status"`
	Cost     string             `bson:"cost" json:"cost"`
	NumberID primitive.ObjectID `bson:"number_id" json:"number_id"`
}

type Transcript struct {
	ID         primitive.ObjectID `bson:"_id" json:"_id"`
	Transcript string             `bson:"transcript" json:"transcript"`
	Date       time.Time          `bson:"date" json:"date"`
	Cost       string             `bson:"cost" json:"cost"`
	NumberID   primitive.ObjectID `bson:"number_id" json:"number_id"`
	HXAreaID   primitive.ObjectID `bson:"hx_area_id" json:"hx_area_id"`
	CallSID    string             `bson:"call_sid" json:"call_sid"`
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
	Active bool `json:"active"`
}

type TimeSegment struct {
	Type  string
	Times []time.Time
}
