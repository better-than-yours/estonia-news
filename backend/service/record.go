package service

import (
	"estonia-news/config"
	"estonia-news/entity"
	"estonia-news/misc"
)

// DeleteRecord perform delete record
func DeleteRecord(params *config.Params, entry entity.Entry) error {
	// TODO need to fix it
	result := params.DB.Unscoped().Where("entry_id = ?", misc.FormatGUID(entry.GUID)).Delete(&entity.EntryToCategory{})
	if result.Error != nil {
		return result.Error
	}
	result = params.DB.Unscoped().Where("guid = ?", misc.FormatGUID(entry.GUID)).Delete(&entity.Entry{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}
