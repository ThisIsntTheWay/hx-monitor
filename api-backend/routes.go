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

const apiBase string = "/api/v1/"

var muxRouter *mux.Router = mux.NewRouter()
var dbClient *mongo.Client = db.Connect()

// Prints various information about a request to stdout
func logResponse(r *http.Request) {
	headers, _ := json.Marshal(r.Header)
	slog.Info("SERVER",
		"host", r.Host,
		"path", strings.TrimSuffix(r.URL.Host, apiBase),
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
}
