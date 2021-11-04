package http

import (
	"bytes"

	"github.com/lafin/http"
	"github.com/mmcdole/gofeed"
	"github.com/thoas/go-funk"
	"golang.org/x/net/html"
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

// GetImage - get a image
func GetImage(imageURL string) ([]byte, error) {
	response, err := http.Get(imageURL, nil)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// GetImageURL - get a image url by a link
func GetImageURL(link string) (string, error) {
	var imageURL string
	var findImageURL func(*html.Node)
	findImageURL = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "meta" && len(node.Attr) > 0 {
			if funk.Contains(node.Attr, func(attr html.Attribute) bool {
				return attr.Key == "property" && attr.Val == "og:image"
			}) {
				found := funk.Find(node.Attr, func(attr html.Attribute) bool {
					return attr.Key == "content"
				})
				imageURL = found.(html.Attribute).Val
				return
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			findImageURL(child)
		}
	}

	response, err := http.Get(link, nil)
	if err != nil {
		return "", err
	}
	doc, _ := html.Parse(bytes.NewReader(response))
	findImageURL(doc)
	return imageURL, nil
}
