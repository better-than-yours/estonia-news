// Package rest contains useful methods for working with http
package rest

import (
	"io/ioutil"
	"net/http"
	"time"
)

// Get - wrapper to execute http GET request
func Get(url string) ([]byte, error) {
	transport := &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     10 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}
