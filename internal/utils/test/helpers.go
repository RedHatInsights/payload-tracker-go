package test

import (
	"fmt"
	"net/http"
)

func formattedQuery(params map[string]interface{}) string {
	formatted := ""
	for k, v := range params {
		formatted += fmt.Sprintf("&%v=%v", k, v)
	}
	return formatted[1:]
}

func MakeTestRequest(uri string, queryParams map[string]interface{}) (*http.Request, error) {
	fullURI := uri
	if len(queryParams) > 0 {
		fullURI += "?" + formattedQuery(queryParams)
	}

	req, err := http.NewRequest("GET", fullURI, nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}
