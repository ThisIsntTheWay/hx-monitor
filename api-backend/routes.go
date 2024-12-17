package main

import (
	"time"

	"github.com/gorilla/mux"
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
	Transcript string    `bson:"transcript" json:"transcript"`
	Date       time.Time `bson:"date" json:"date"`
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
