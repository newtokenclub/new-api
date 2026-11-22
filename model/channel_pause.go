package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	ChannelPauseEnabledOption = "channel_pause.enabled"
	ChannelPauseMessageOption = "channel_pause.message"
)

func GetChannelPauseState() (bool, string) {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	return strings.EqualFold(common.OptionMap[ChannelPauseEnabledOption], "true"),
		common.OptionMap[ChannelPauseMessageOption]
}

func UpdateChannelPause(enabled bool, message string) error {
	err := DB.Transaction(func(tx *gorm.DB) error {
		for key, value := range map[string]string{
			ChannelPauseEnabledOption: common.Interface2String(enabled),
			ChannelPauseMessageOption: message,
		} {
			option := Option{Key: key}
			if err := tx.FirstOrCreate(&option, Option{Key: key}).Error; err != nil {
				return err
			}
			if err := tx.Model(&Option{}).Where(commonKeyCol+" = ?", key).Update("value", value).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		loadOptionsFromDatabase()
	}
	return err
}
