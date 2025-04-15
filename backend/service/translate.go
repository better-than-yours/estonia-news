package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/lafin/http"
)

func translate(query, from, to string) (string, error) {
	body, _, err := http.Get(fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=%s&tl=%s&dt=t&q=%s", from, to, url.QueryEscape(query)), nil)
	if err != nil {
		return "", fmt.Errorf("failed to get translation from '%s' to '%s' for query '%s': %v", from, to, query, err)
	}
	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("failed to get translation from '%s' to '%s' for query '%s': %v", from, to, query, err)
	}
	if data == nil {
		return "", errors.New("empty translation")
	}
	return data[0].([]any)[0].([]any)[0].(string), nil
}
