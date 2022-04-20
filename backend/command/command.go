package command

import (
	"strconv"

	"estonia-news/misc"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

// ExecCommand is exec command
func ExecCommand(dbConnect *gorm.DB, bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
	command := update.Message.CommandArguments()
	switch update.Message.Command() {
	case "info":
		msg.Text = getEntryByID(dbConnect, command)
	case "add":
		categoryID, _ := strconv.Atoi(command)
		msg.Text = addCategoryToBlock(dbConnect, categoryID)
	case "delete":
		categoryID, _ := strconv.Atoi(command)
		msg.Text = deleteCategoryFromBlock(dbConnect, categoryID)
	case "list":
		msg.Text = getListBlocks(dbConnect)
	default:
		return
	}
	if len(msg.Text) > 0 {
		_, err := bot.Send(msg)
		if err != nil {
			misc.Error("send_message", "send message", err)
		}
	}
}

func getEntryByID(dbConnect *gorm.DB, entryID string) string {
	return ""
}

func addCategoryToBlock(dbConnect *gorm.DB, categoryID int) string {
	return ""
}

func deleteCategoryFromBlock(dbConnect *gorm.DB, categoryID int) string {
	return ""
}

func getListBlocks(dbConnect *gorm.DB) string {
	return ""
}
