package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/thisisnttheway/hx-monitor/db"
	"github.com/thisisnttheway/hx-monitor/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

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

// Get all areas
func getAreas(w http.ResponseWriter, r *http.Request) {
	logResponse(r)

	hxAreas, err := db.GetDocument[models.HXArea](
		"hx_areas",
		bson.M{},
	)

	var s interface{}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s = ResponseError{
			Error: "Internal error",
			Data:  err.Error(),
		}
	} else {
		s = ResponseOk{
			Message: "Ok",
			Data:    hxAreas,
		}
	}

	res, _ := json.Marshal(s)
	fmt.Fprint(w, string(res))
}

// Get area by name (/areas/{name})
func getAreaByName(w http.ResponseWriter, r *http.Request) {
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
			Data:  err.Error(),
		}
	} else {
		s = ResponseOk{
			Message: "Ok",
			Data:    hxArea[0],
		}
	}

	res, _ := json.Marshal(s)
	fmt.Fprint(w, string(res))
}

// Gets the latest transcript for a given area
func getTranscriptsLatest(w http.ResponseWriter, r *http.Request) {
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
}

// Gets all transcripts for a given area
func getTranscripts(w http.ResponseWriter, r *http.Request) {
	logResponse(r)
	areaName := mux.Vars(r)["name"]

	transcripts, err := db.Aggregate[transcriptAggregation](
		"hx_areas",
		getTranscriptAggregationPipeline(areaName, []bson.D{}),
	)

	response := handleTranscripts(transcripts, err, areaName, w)

	res, _ := json.Marshal(response)
	fmt.Fprint(w, string(res))
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
		var dataObject interface{}
		if len(t) > 1 {
			type MultipleTranscripts struct {
				Amount      int                     `json:"amount"`
				Transcripts []transcriptAggregation `json:"transcripts"`
			}

			dataObject = MultipleTranscripts{
				Amount:      len(t),
				Transcripts: t,
			}
		} else {
			dataObject = t[0]
		}

		s = ResponseOk{
			Message: "Ok",
			Data:    dataObject,
		}
	}

	return s
}
