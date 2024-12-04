package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/thisisnttheway/hx-checker/db"
	"github.com/thisisnttheway/hx-checker/models"
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
var dbClient *mongo.Client = db.Connect()

// Prints various information about a request to stdout
func logResponse(r *http.Request) {
	headers, _ := json.Marshal(r.Header)
	slog.Info("SERVER",
		"path", strings.TrimPrefix(r.URL.Path, apiBase),
		"method", r.Method,
		"body", r.Body,
		"headers", headers,
	)
}

func init() {
	// Base
	muxRouter.HandleFunc(apiBase, func(w http.ResponseWriter, r *http.Request) {
		logResponse(r)
		s := ResponseOk{
			Message: "Ok",
			Data:    nil,
		}

		res, _ := json.Marshal(s)
		fmt.Fprint(w, string(res))
	}).Methods("GET")

	// HX areas
	muxRouter.HandleFunc(apiBase+"areas/{name}", func(w http.ResponseWriter, r *http.Request) {
		logResponse(r)
		areaName := mux.Vars(r)["name"]

		hxArea, err := db.GetDocument[models.HXArea](
			"hx_areas",
			bson.M{"name": areaName},
		)

		var s interface{}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			s = ResponseError{
				Error: "Internal error",
				Data:  err,
			}
		} else {
			s = ResponseOk{
				Message: "Ok",
				Data:    hxArea[0],
			}
		}

		res, _ := json.Marshal(s)
		fmt.Fprint(w, string(res))
	}).Methods("GET")

	// Transcripts
	muxRouter.HandleFunc(apiBase+"transcripts/{name:[^/]+}/latest", func(w http.ResponseWriter, r *http.Request) {
		logResponse(r)
		areaName := mux.Vars(r)["name"]

		transcripts, err := db.Aggregate[transcriptAggregation](
			"hx_areas",
			getTranscriptAggregationPipeline(areaName, []bson.D{
				bson.D{
					{"$sort", bson.D{
						{"related_transcripts.date", -1},
					}},
				},
				bson.D{{"$limit", 1}},
			}),
		)

		response := handleTranscripts(transcripts, err, areaName, w)

		res, _ := json.Marshal(response)
		fmt.Fprint(w, string(res))
	})

	muxRouter.HandleFunc(apiBase+"transcripts/{name:[^/]+}", func(w http.ResponseWriter, r *http.Request) {
		logResponse(r)
		areaName := mux.Vars(r)["name"]

		transcripts, err := db.Aggregate[transcriptAggregation](
			"hx_areas",
			getTranscriptAggregationPipeline(areaName, []bson.D{}),
		)

		response := handleTranscripts(transcripts, err, areaName, w)

		res, _ := json.Marshal(response)
		fmt.Fprint(w, string(res))
	}).Methods("GET")
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
	if inject != nil && len(inject) > 0 {
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
