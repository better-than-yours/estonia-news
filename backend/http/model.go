package http

import (
	"github.com/lafin/http"
	"github.com/mmcdole/gofeed"
)

// GetFeed - get feed
func GetFeed(url string) (*gofeed.Feed, error) {
	response, err := http.Get(url, nil)
	if err != nil {
		return nil, err
	}
	fp := gofeed.NewParser()
	feed, err := fp.ParseString(string(response))
	if err != nil {
		return nil, err
	}
	return feed, nil
}

// GetImage - get image
func GetImage(imageURL string) ([]byte, error) {
	response, err := http.Get(imageURL, nil)
	if err != nil {
		return nil, err
	}
	return response, nil
}
