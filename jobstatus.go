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
package bqextract

import (
	"context"
	"net/http"
	"os"
	"path"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/goog-lukemc/tserver"
)

func CheckJob(server *tserver.ServerControl) {
	server.MUX.HandleFunc("/api/v1/checkjob/", func(w http.ResponseWriter, r *http.Request) {
		jobid := path.Base(r.URL.Path)
		jobstatus, err := getBQJobStatus(r.Context(), jobid)
		if err != nil {
			tserver.Respond(w, err)
			return
		}

		if jobstatus.Done() {
			tserver.Respond(w, jobstatus)
		}
		time.Sleep(time.Second * 2)
		http.Redirect(w, r, r.URL.Path, 307)
		return
	})
}

func getBQJobStatus(ctx context.Context, id string) (*bigquery.JobStatus, error) {
	client, err := bigquery.NewClient(ctx, os.Getenv("BQPROJECT"))
	if err != nil {
		return nil, err
	}
	j, err := client.JobFromID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return j.LastStatus(), nil
}
