package service

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/lafin/http"
	"github.com/thoas/go-funk"
	"golang.org/x/net/html"
)

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
		return nil, fmt.Errorf("failed to fetch meta for link '%s': %v", link, err)
	}
	doc, _ := html.Parse(bytes.NewReader(body))
	meta.ImageURL = findMeta(doc, "og:image")
	meta.Description = findMeta(doc, "og:description")
	if meta.ImageURL == "" || meta.Description == "" {
		return nil, errors.New("Meta is empty")
	}
	meta.Paywall = funk.Contains([]string{"fragment--teaser", `"isPremium":true`}, func(feature string) bool {
		return strings.Contains(string(body), feature)
	})
	return &meta, nil
}
