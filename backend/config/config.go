package config

import (
	"time"
)

// TimeoutBetweenLoops is main loop timeout
var TimeoutBetweenLoops = 5 * time.Minute

// TimeoutBetweenMessages is timeout between attempts to send a message
var TimeoutBetweenMessages = time.Second

// TimeShift get messages from the last hours
var TimeShift = 3 * time.Hour

// PurgeOldEntriesEvery is time for purge old entries
var PurgeOldEntriesEvery = time.Hour

// PushMetricsEvery is time for push metrics
var PushMetricsEvery = 5 * time.Second

// FeedItem is feed item struct
type FeedItem struct {
	GUID          string
	Link          string
	Title         string
	ImageURL      string
	Description   string
	Published     string
	Categories    []string
	CategoriesIds []int
	Paywall       bool
}

type ctxKey int

const (
	// CtxDBKey is ctx db key
	CtxDBKey ctxKey = iota
	// CtxBotKey is ctx bot key
	CtxBotKey
	// CtxProviderKey is ctx provider key
	CtxProviderKey
	// CtxChatIDKey is ctx chat id key
	CtxChatIDKey
	// CtxFeedTitleKey is ctx feed title key
	CtxFeedTitleKey
)
