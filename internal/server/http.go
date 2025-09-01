package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andykhv/lanpeerdiscovery/internal/table"
)

func StartHttpServer(ctx context.Context, port int, bus *table.Bus) {

	http.HandleFunc("/peers", func(w http.ResponseWriter, r *http.Request) {
		request := table.ListPeersRequest{}
		bus.ListPeersRequestCh <- request
		select {
		case response := <-bus.ListPeersResponseCh:
			bytes, err := json.Marshal(response)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			w.Write(bytes)
		case <-ctx.Done():
			http.Error(w, ctx.Err().Error(), http.StatusInternalServerError)
		}
	})

	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
