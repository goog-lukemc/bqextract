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
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/goog-lukemc/tserver"
)

const (
	SQLLIMIT   string = "select * from %s where status = @status and county = @county limit @limit"
	SQLNOLIMIT string = "select * from %s where status = @status and county = @county"
)

func CSVHandler(server *tserver.ServerControl) {
	server.MUX.HandleFunc("/api/v1/getcsv/", func(w http.ResponseWriter, r *http.Request) {
		bqParams := []bigquery.QueryParameter{}
		tv := path.Base(r.URL.Path)
		log.Printf("%+v", r.URL.Query())
		var sql string
		for k, v := range r.URL.Query() {
			if k == "limit" {
				l, _ := strconv.Atoi(v[0])
				if l == 0 {
					sql = fmt.Sprintf(SQLNOLIMIT, tv)
					continue
				}
				bqParams = append(bqParams, bigquery.QueryParameter{
					Name:  k,
					Value: l,
				})
				sql = fmt.Sprintf(SQLLIMIT, tv)
				continue
			}

			bqParams = append(bqParams, bigquery.QueryParameter{
				Name:  k,
				Value: v[0],
			})
		}

		log.Printf("view:%s params:%+v", tv, bqParams)

		// create a bq client
		client, err := bigquery.NewClient(r.Context(), os.Getenv("BQPROJECT"))
		if err != nil {
			tserver.Respond(w, fmt.Errorf("errBQClient:%s", err))
			return
		}
		defer client.Close()

		// Run the query
		tmpTble, err := runQuery(r.Context(), client, sql, bqParams...)
		if err != nil {
			tserver.Respond(w, fmt.Errorf("errBQQuery:%s", err))
			return
		}
		log.Printf("Something:%s", tmpTble)
		// Export the results
		url, err := exportGCS(client, r.Context(), tv, tmpTble)
		if err != nil {
			tserver.Respond(w, fmt.Errorf("errExport:%s", err))
			return
		}
		http.Redirect(w, r, url, 307)
	})
}

func runQuery(ctx context.Context, bqClient *bigquery.Client, sql string, param ...bigquery.QueryParameter) (string, error) {

	tempTable := strconv.FormatInt(time.Now().UnixNano(), 10) + "_tmp"

	q := bqClient.Query("")
	q.QueryConfig = bigquery.QueryConfig{
		DefaultProjectID: os.Getenv("BQPROJECT"),
		DefaultDatasetID: os.Getenv("BQDATASET"),
		WriteDisposition: bigquery.WriteTruncate,
		Q:                sql,
		Dst: &bigquery.Table{
			ProjectID: os.Getenv("BQPROJECT"),
			DatasetID: os.Getenv("BQDATASET"),
			TableID:   tempTable,
		},
		Parameters: param,
	}

	j, err := q.Run(ctx)
	if err != nil {
		return "", err
	}
	_, err = j.Wait(ctx)

	return tempTable, err

}

func exportGCS(bqClient *bigquery.Client, ctx context.Context, tv string, tempTable string) (string, error) {

	bucket := os.Getenv("REPORTBUCKET")

	fileName := fmt.Sprintf("%s-%s.csv", tv, strconv.FormatInt(time.Now().UnixNano(), 10))

	objectPath := fmt.Sprintf("%s/%s", bucket, fileName)

	gcPath := fmt.Sprintf("gs://%s", objectPath)

	gcsRef := bigquery.NewGCSReference(gcPath)

	extract := bqClient.Dataset(os.Getenv("BQDATASET")).Table(tempTable).ExtractorTo(gcsRef)

	ej, err := extract.Run(ctx)
	if err != nil {
		return "", err
	}
	_, err = ej.Wait(ctx)
	if err != nil {
		return "", err
	}

	url, err := storage.SignedURL(bucket, fileName, &storage.SignedURLOptions{
		GoogleAccessID: "dlp-accessor@dlp-secure-dev.iam.gserviceaccount.com",
		PrivateKey:     []byte(PEM),
		Method:         "GET",
		Expires:        time.Now().Add(time.Hour),
	})

	if err != nil {
		return "", fmt.Errorf("errSigning:%s", err)
	}

	return url, nil

}
