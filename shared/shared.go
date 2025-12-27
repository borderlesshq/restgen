package shared

import (
	"encoding/json"
	"log"
	"net/http"
	"reflect"
)

type ApiResponse[T any] struct {
	Data    T      `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
	Success bool   `json:"success"`
}

func WriteResponse[T any](wr http.ResponseWriter, statusCode int, apiResponse *ApiResponse[T]) {
	wr.Header().Set("Content-Type", "application/json")
	wr.WriteHeader(statusCode)
	_ = json.NewEncoder(wr).Encode(apiResponse)
}

func AssertDependencies(service interface{}, name string) {
	sType := reflect.TypeOf(service)
	fields := reflect.VisibleFields(sType)
	for _, v := range fields {
		if reflect.ValueOf(service).FieldByName(v.Name).IsZero() {
			log.Printf("%s has an uninitialized dependency: %s.%s \n", name, sType.Name(), v.Name)
		}
	}
}
