package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// apigen:api {"url": "/user/profile", "auth": false}
// apigen:api {"url": "/user/create", "auth": true, "method": "POST"}
// apigen:api {"url": "/user/create", "auth": true, "method": "POST"}

func (ma *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handlerMap := make(map[string]func(w http.ResponseWriter, r *http.Request), 5)
	handlerMap["/user/profile"] = func(w http.ResponseWriter, r *http.Request) {
		response, err := ma.Profile(nil, ProfileParams{Login: "rvasily"})
		if err == nil {
			re, err := json.Marshal(map[string]interface{}{
				"response": response,
				"error":    "",
			})
			if err == nil {
				w.Write(re)
			}
		}
	}
	handlerMap["/user/profile"] = func(w http.ResponseWriter, r *http.Request) {
		response, err := ma.Profile(nil, ProfileParams{Login: "rvasily"})
		if err == nil {
			re, err := json.Marshal(map[string]interface{}{
				"response": response,
				"error":    "",
			})
			if err == nil {
				w.Write(re)
			}
		}
	}

}

func (oa *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Work_It")
}
