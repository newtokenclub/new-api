package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

type AccountMonitorPair struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	UserAId   int    `json:"user_a_id" gorm:"uniqueIndex:idx_monitor_pair;not null"`
	UserBId   int    `json:"user_b_id" gorm:"uniqueIndex:idx_monitor_pair;not null"`
	TokenHash string `json:"-" gorm:"type:varchar(64);index;not null"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
}

func (AccountMonitorPair) TableName() string { return "account_monitor_pairs" }

func CreateAccountMonitorPair(userAId, userBId int) (*AccountMonitorPair, string, error) {
	if userAId <= 0 || userBId <= 0 || userAId == userBId {
		return nil, "", errors.New("invalid account pair")
	}
	if userAId > userBId {
		userAId, userBId = userBId, userAId
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", err
	}
	token := hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	pair := &AccountMonitorPair{UserAId: userAId, UserBId: userBId, TokenHash: hex.EncodeToString(hash[:]), CreatedAt: time.Now().Unix()}
	if err := DB.Create(pair).Error; err != nil {
		return nil, "", err
	}
	return pair, token, nil
}

func GetAccountMonitorPairForUser(userId int) (*AccountMonitorPair, error) {
	var pair AccountMonitorPair
	err := DB.Where("user_a_id = ? OR user_b_id = ?", userId, userId).First(&pair).Error
	return &pair, err
}

func GetAccountMonitorPairByToken(token string) (*AccountMonitorPair, error) {
	hash := sha256.Sum256([]byte(token))
	var pair AccountMonitorPair
	err := DB.Where("token_hash = ?", hex.EncodeToString(hash[:])).First(&pair).Error
	return &pair, err
}

func DeleteAccountMonitorPairForUser(userId int) error {
	return DB.Where("user_a_id = ? OR user_b_id = ?", userId, userId).Delete(&AccountMonitorPair{}).Error
}
