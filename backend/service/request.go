package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/lafin/http"
	"github.com/mmcdole/gofeed"
	"github.com/thoas/go-funk"
	"golang.org/x/net/html"
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

func getImageURL(link string) (string, error) {
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

	body, _, err := http.Get(link, nil)
	if err != nil {
		return "", err
	}
	doc, _ := html.Parse(bytes.NewReader(body))
	findImageURL(doc)
	return imageURL, nil
}

func translate(query, from, to string) (string, error) {
	body, _, err := http.Get(fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=%s&tl=%s&dt=t&q=%s", from, to, url.QueryEscape(query)), nil)
	if err != nil {
		return "", err
	}
	var data []interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	if data == nil {
		return "", errors.New("empty translation")
	}
	return data[0].([]interface{})[0].([]interface{})[0].(string), nil
}
