package service

import (
	"estonia-news/config"
	"estonia-news/entity"
	"estonia-news/misc"
)

// DeleteRecord perform delete record
func DeleteRecord(params *config.Params, entry entity.Entry) error {
	result := params.DB.Unscoped().Where("guid = ?", misc.FormatGUID(entry.GUID)).Delete(&entity.Entry{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}
