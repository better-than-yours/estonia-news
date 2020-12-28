// Package http handle work with http
package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/lafin/http"
)

// Translate - return translated string
func Translate(query, from, to string) (string, error) {
	response, err := http.Get(fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=%s&tl=%s&dt=t&q=%s", from, to, url.QueryEscape(query)), nil)
	if err != nil {
		return "", err
	}
	var data []interface{}
	if err := json.Unmarshal(response, &data); err != nil {
		return "", err
	}
	if data == nil {
		return "", errors.New("empty translation")
	}
	return data[0].([]interface{})[0].([]interface{})[0].(string), nil
}
