package model

import "gorm.io/gorm"

// AffiliateSalesLink adds a sales attribution without changing the user's
// original inviter_id, which remains the source for legacy invite rewards.
type AffiliateSalesLink struct {
	Id        int `json:"id"`
	UserId    int `json:"user_id" gorm:"uniqueIndex:ux_user_sales_link"`
	SalesId   int `json:"sales_id" gorm:"uniqueIndex:ux_user_sales_link;index"`
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime"`
}

func (AffiliateSalesLink) TableName() string { return "affiliate_sales_links" }

func IsAffiliateSalesForUser(db *gorm.DB, userID, salesID int) (bool, error) {
	var count int64
	linkedUsers := db.Table("affiliate_sales_links").Select("user_id").Where("sales_id = ?", salesID)
	err := db.Model(&User{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Where("(inviter_id = ? OR id IN (?))", salesID, linkedUsers).
		Count(&count).Error
	return count > 0, err
}
