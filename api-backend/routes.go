package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ResponseOk struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ResponseError struct {
	Error string      `json:"error"`
	Data  interface{} `json:"data"`
}

type transcriptAggregation struct {
	Transcript string    `bson:"transcript"`
	Date       time.Time `bson:"date"`
}

const apiBase string = "/api/v1/"

var muxRouter *mux.Router = mux.NewRouter()

func init() {
	// HX areas
	muxRouter.HandleFunc(apiBase+"areas/{name}", getAreaByName).Methods("GET")
	muxRouter.HandleFunc(apiBase+"areas", getAreas).Methods("GET")

	// Transcripts
	muxRouter.HandleFunc(apiBase+"transcripts/{name:[^/]+}/latest", getTranscriptsLatest).Methods("GET")
	muxRouter.HandleFunc(apiBase+"transcripts/{name:[^/]+}", getTranscripts).Methods("GET")
}

// Get a mongo.Pipeline for transcript lookups. If specified, inject will be placed inbetween $unwind and $project.
func getTranscriptAggregationPipeline(areaName string, inject []bson.D) mongo.Pipeline {
	match := bson.D{{"$match", bson.M{"name": areaName}}}
	lookup := bson.D{{"$lookup", bson.D{
		{"from", "transcripts"},
		{"localField", "_id"},
		{"foreignField", "hx_area_id"},
		{"as", "related_transcripts"},
	}}}
	unwind := bson.D{{"$unwind", "$related_transcripts"}}
	project := bson.D{{"$project", bson.M{
		"transcript": "$related_transcripts.transcript",
		"date":       "$related_transcripts.date",
	}}}

	p := mongo.Pipeline{match, lookup, unwind}
	if len(inject) > 0 {
		for _, in := range inject {
			p = append(p, in)
		}
	}
	p = append(p, project)

	return p
}

// Handles a transcript aggregation result
func handleTranscripts(t []transcriptAggregation, e error, a string, w http.ResponseWriter) interface{} {
	var errType string
	if e == nil {
		if len(t) == 0 {
			errType = "notfound"
			e = fmt.Errorf("Found no transcripts for area '%s'", a)
		}
	}

	var s interface{}
	if e != nil {
		e := ResponseError{
			Data: e.Error(),
		}

		switch errType {
		case "notfound":
			e.Error = "No transcripts"
			w.WriteHeader(http.StatusNotFound)
		default:
			e.Error = "Internal error"
			w.WriteHeader(http.StatusInternalServerError)
		}

		s = e
	} else {
		type Data struct {
			Amount      int         `json:"amount"`
			Transcripts interface{} `json:"transcripts"`
		}
		s = ResponseOk{
			Message: "Ok",
			Data: Data{
				Amount:      len(t),
				Transcripts: t,
			},
		}
	}

	return s
}
