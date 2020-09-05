module github.com/lafin/estonia-news

go 1.15

replace github.com/lafin/estonia-news/model => ./model

replace github.com/lafin/estonia-news/proc => ./proc

replace github.com/lafin/estonia-news/rest => ./rest

require (
	github.com/PuerkitoBio/goquery v1.5.1 // indirect
	github.com/andybalholm/cascadia v1.2.0 // indirect
	github.com/go-telegram-bot-api/telegram-bot-api v4.6.4+incompatible
	github.com/jackc/pgproto3/v2 v2.0.4 // indirect
	github.com/joho/godotenv v1.3.0
	github.com/mmcdole/gofeed v1.0.0
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/technoweenie/multipartstreamer v1.0.1 // indirect
	golang.org/x/crypto v0.0.0-20200820211705-5c72a883971a // indirect
	golang.org/x/net v0.0.0-20200822124328-c89045814202 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gorm.io/driver/postgres v1.0.0
	gorm.io/gorm v1.20.0
)
