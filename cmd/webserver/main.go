package main

import (
	"fmt"
	"os"

	"github.com/goog-lukemc/bqextract"
	"github.com/goog-lukemc/tserver"
)

func main() {
	var port string
	var ok bool
	if port, ok = os.LookupEnv("PORT"); !ok {
		port = "8080"
	}

	port = fmt.Sprintf(":%s", port)
	server := tserver.NewServer(&tserver.ServerConfig{
		// Listening Port
		Addr: port,

		// Static content directory
		StaticDir: "site",
	})
	server.Start(bqextract.CSVHandler, tserver.DefaultHandlers)
}
