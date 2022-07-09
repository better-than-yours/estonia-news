package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

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

func findMeta(node *html.Node, key string) string {
	if node.Type == html.ElementNode && node.Data == "meta" && len(node.Attr) > 0 {
		if funk.Contains(node.Attr, func(attr html.Attribute) bool {
			return attr.Key == "property" && attr.Val == key
		}) {
			found := funk.Find(node.Attr, func(attr html.Attribute) bool {
				return attr.Key == "content"
			})
			return found.(html.Attribute).Val
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		found := findMeta(child, key)
		if found != "" {
			return found
		}
	}
	return ""
}

// Meta is meta struct
type Meta struct {
	ImageURL    string
	Description string
	Paywall     bool
}

// GetMeta return meta info by url
func GetMeta(link string) (*Meta, error) {
	var meta Meta
	body, _, err := http.Get(link, nil)
	if err != nil {
		return nil, err
	}
	doc, _ := html.Parse(bytes.NewReader(body))
	meta.ImageURL = findMeta(doc, "og:image")
	meta.Description = findMeta(doc, "og:description")
	if meta.ImageURL == "" || meta.Description == "" {
		return nil, errors.New("Meta is empty")
	}
	meta.Paywall = funk.Contains([]string{`paywall-component="paywall"`, "C-fragment--teaser", `class="article-paywall-buttons-block__price"`}, func(feature string) bool {
		return strings.Contains(string(body), feature)
	})
	return &meta, nil
}

func translate(query, from, to string) (string, error) {
	body, _, err := http.Get(fmt.Sprintf("https://translate.googleapis.com/translate_a/single?client=gtx&sl=%s&tl=%s&dt=t&q=%s", from, to, url.QueryEscape(query)), nil)
	if err != nil {
		return "", err
	}
	var data []any
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	if data == nil {
		return "", errors.New("empty translation")
	}
	return data[0].([]any)[0].([]any)[0].(string), nil
}
