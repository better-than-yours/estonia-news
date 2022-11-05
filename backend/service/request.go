package service

import (
	"strings"

	"github.com/lafin/http"
	"github.com/mmcdole/gofeed"
)

// GetFeed return feed
func GetFeed(feedURL string) (*gofeed.Feed, error) {
	body, _, err := http.Get(feedURL, nil)
	if err != nil {
		return nil, err
	}
	fp := gofeed.NewParser()
	feed, err := fp.ParseString(string(body))
	if err != nil {
		return nil, err
	}
	return feed, nil
}

func getImage(imageURL string) ([]byte, error) {
	body, _, err := http.Get(imageURL, nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// IsLinkUnavailable return availability of link
func IsLinkUnavailable(link string) bool {
	body, res, _ := http.Get(link, nil)
	return res.StatusCode == 404 || strings.Contains(string(body), "Artiklit ei leitud")
}
