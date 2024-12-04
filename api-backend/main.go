package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/thisisnttheway/hx-checker/configuration"
)

const port string = "8080"

func main() {
	configuration.SetUpMongoConfig()

	slog.Info("MAIN", "action", "startServer", "port", port, "apiBase", apiBase)
	http.ListenAndServe(":"+port, muxRouter)

	fmt.Println("DONE EXECUTING")
}
