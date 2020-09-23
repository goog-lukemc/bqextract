// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
