package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/hongliang5316/midjourney-apiserver/pkg/api"
	"github.com/hongliang5316/midjourney-apiserver/pkg/store"
	"github.com/hongliang5316/midjourney-apiserver/pkg/webhook"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var apiServiceClient api.APIServiceClient

func init() {
	conn, err := grpc.Dial("127.0.0.1:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	apiServiceClient = api.NewAPIServiceClient(conn)

}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	req := &webhook.WebhookRequest{}
	json.Unmarshal(body, req)

	log.Printf("------------- webhook req: %+v", req)

	if req.Type == store.TypeImagine {
		resp, err := apiServiceClient.Upscale(context.Background(), &api.UpscaleRequest{
			Index:   1,
			TaskId:  req.TaskID,
			Webhook: "http://127.0.0.1:8000/",
		})
		if err != nil {
			panic(err)
		}

		log.Printf("resp: %+v", resp)
	}
}

func main() {
	go func() {
		resp, err := apiServiceClient.Imagine(context.Background(), &api.ImagineRequest{
			Prompt:  "tokyo city",
			Webhook: "http://127.0.0.1:8000/",
		})
		if err != nil {
			panic(err)
		}

		log.Printf("%+v", resp)
	}()

	http.HandleFunc("/", webhookHandler)
	http.ListenAndServe(":8000", nil)
}
