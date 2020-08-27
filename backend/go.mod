module github.com/lafin/estonia-news

go 1.15

replace github.com/lafin/estonia-news/model => "./model"
replace github.com/lafin/estonia-news/proc => "./proc"
replace github.com/lafin/estonia-news/rest => "./rest"

require (
	github.com/PuerkitoBio/goquery v1.5.1 // indirect
	github.com/andybalholm/cascadia v1.2.0 // indirect
	github.com/go-pkgz/lgr v0.9.0
	github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible
	github.com/mmcdole/gofeed v1.0.0
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/technoweenie/multipartstreamer v1.0.1 // indirect
	go.etcd.io/bbolt v1.3.5
	golang.org/x/net v0.0.0-20200822124328-c89045814202 // indirect
	golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6 // indirect
	golang.org/x/text v0.3.3 // indirect
)
