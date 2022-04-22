package command

import (
	"fmt"
	"strconv"
	"strings"

	"estonia-news/entity"
	"estonia-news/misc"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/thoas/go-funk"
	"gorm.io/gorm"
)

// ExecCommand is exec command
func ExecCommand(dbConnect *gorm.DB, bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
	command := update.Message.CommandArguments()
	switch update.Message.Command() {
	case "info":
		res, err := entity.GetEntryByID(dbConnect, command)
		if err == nil {
			categories := funk.Map(res.Categories, func(item entity.EntryToCategory) string {
				return fmt.Sprintf("%d - %s", item.CategoryID, item.Category.Name)
			}).([]string)
			msg.Text = strings.Join(append([]string{fmt.Sprintf("%s %s", res.GUID, res.Title)}, categories...), "\n")
		}
	case "add_block":
		categoryID, err := strconv.Atoi(command)
		if err == nil {
			_ = entity.AddCategoryToBlock(dbConnect, categoryID)
			msg.Text = "done"
		}
	case "delete_block":
		categoryID, err := strconv.Atoi(command)
		if err == nil {
			_ = entity.DeleteCategoryFromBlock(dbConnect, categoryID)
			msg.Text = "done"
		}
	case "list_blocks":
		res, err := entity.GetListBlocks(dbConnect)
		if err == nil {
			msg.Text = strings.Join(funk.Map(res, func(block entity.BlockedCategory) string {
				return fmt.Sprintf("%d %s %s", block.CategoryID, block.Category.Name, block.Category.Provider.Lang)
			}).([]string), "\n")
		}
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
