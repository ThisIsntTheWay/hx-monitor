package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/thisisnttheway/hx-checker/db"
	"github.com/thisisnttheway/hx-checker/models"
	"go.mongodb.org/mongo-driver/bson"
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
