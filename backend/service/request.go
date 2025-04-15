package service

import (
	"fmt"
	"strings"

	"github.com/lafin/http"
	"github.com/mmcdole/gofeed"
)

// GetFeed return feed
func GetFeed(feedURL string) (*gofeed.Feed, error) {
	body, _, err := http.Get(feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed from URL '%s': %v", feedURL, err)
	}
	fp := gofeed.NewParser()
	feed, err := fp.ParseString(string(body))
	if err != nil {
		return nil, fmt.Errorf("failed to get feed from URL '%s': %v", feedURL, err)
	}
	return feed, nil
}

func getImage(imageURL string) ([]byte, error) {
	body, _, err := http.Get(imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get image from URL '%s': %v", imageURL, err)
	}
	return body, nil
}

// IsLinkUnavailable return availability of link
func IsLinkUnavailable(link string) bool {
	body, res, _ := http.Get(link, nil)
	return res != nil && (res.StatusCode == 404 || res.StatusCode == 200 && strings.Contains(string(body), "Artiklit ei leitud"))
}
