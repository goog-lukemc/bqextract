package bqextract

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/goog-lukemc/tserver"
)

func CSVHandler(server *tserver.ServerControl) {
	server.MUX.HandleFunc("/api/v1/getcsv/", func(w http.ResponseWriter, r *http.Request) {
		dlurl, err := getBQData(r.Context(), path.Base(r.URL.Path))
		if err != nil {
			tserver.Respond(w, err)
		}
		//tserver.Respond(w, []byte(dlurl))
		http.Redirect(w, r, dlurl, 307)
	})
}

func getBQData(ctx context.Context, tv string) (string, error) {
	client, err := bigquery.NewClient(ctx, os.Getenv("BQPROJECT"))
	if err != nil {
		return "", err
	}

	tempTable := strconv.FormatInt(time.Now().UnixNano(), 10) + "_tmp"

	q := client.Query("")
	q.QueryConfig = bigquery.QueryConfig{
		DefaultProjectID: os.Getenv("BQPROJECT"),
		DefaultDatasetID: os.Getenv("BQDATASET"),
		WriteDisposition: bigquery.WriteTruncate,
		Q:                fmt.Sprintf("select * from %s limit 1000", tv),
		Dst: &bigquery.Table{
			ProjectID: os.Getenv("BQPROJECT"),
			DatasetID: os.Getenv("BQDATASET"),
			TableID:   tempTable,
		},
	}

	j, err := q.Run(ctx)
	if err != nil {
		return "", err
	}

	_, err = j.Wait(ctx)
	if err != nil {
		return "", err
	}

	bucket := os.Getenv("REPORTBUCKET")

	fileName := fmt.Sprintf("%s-%s.csv", tv, strconv.FormatInt(time.Now().UnixNano(), 10))

	objectPath := fmt.Sprintf("%s/%s", bucket, fileName)

	gcPath := fmt.Sprintf("gs://%s", objectPath)

	gcsRef := bigquery.NewGCSReference(gcPath)

	extract := client.Dataset(os.Getenv("BQDATASET")).Table(tempTable).ExtractorTo(gcsRef)

	ej, err := extract.Run(ctx)
	if err != nil {
		return "", err
	}
	_, err = ej.Wait(ctx)
	if err != nil {
		return "", err
	}

	bts, err := ioutil.ReadFile("key.pem")
	if err != nil {
		return "", err
	}
	url, err := storage.SignedURL(bucket, fileName, &storage.SignedURLOptions{
		GoogleAccessID: "dlp-accessor@dlp-secure-dev.iam.gserviceaccount.com",
		PrivateKey:     bts,
		Method:         "GET",
		Expires:        time.Now().Add(time.Hour),
	})
	if err != nil {
		return "", err
	}

	return url, nil
}
