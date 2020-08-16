// Package model describes data models
package model

import (
	"crypto/sha1"
	"fmt"
	"time"
)

// Entry is a entry structure
type Entry struct {
	GUID      string    `xml:"guid"`
	Link      string    `xml:"link"`
	Title     string    `xml:"title"`
	Published time.Time `xml:"published"`
	Date      string    `xml:"date"`
	MessageID int       `xml:"message_id"`
}

// GetKey is a method to get a bolt key
func (item Entry) GetKey(key []byte) ([]byte, error) {
	fmt.Println(string(key))
	h := sha1.New()
	if _, err := h.Write(key); err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("%x", h.Sum(nil))), nil
}

// GetDate is a method to get a date
func (item Entry) GetDate() string {
	if item.Date == "" {
		item.Date = time.Now().Format(time.RFC1123Z)
	}
	return item.Date
}
