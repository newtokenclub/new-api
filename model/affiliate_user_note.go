package model

type AffiliateUserNote struct {
	Id           int    `json:"id" gorm:"primaryKey"`
	SalesId      int    `json:"sales_id" gorm:"not null;uniqueIndex:ux_affiliate_user_note"`
	TargetUserId int    `json:"target_user_id" gorm:"not null;uniqueIndex:ux_affiliate_user_note"`
	Remark       string `json:"remark" gorm:"type:varchar(500);not null"`
	CreatedAt    int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (AffiliateUserNote) TableName() string { return "affiliate_user_notes" }
