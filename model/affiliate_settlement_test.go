package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAffiliateCommissionPlanOverridesLegacyRulesWithoutChangingSettlementSnapshot(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(
		&AffiliateCommissionRule{},
		&AffiliateCommissionPlan{},
		&AffiliateCommissionPlanRate{},
		&AffiliateSalesPlanAssignment{},
		&AffiliateSettlementApplication{},
		&AffiliateSettlementItem{},
	))

	const salesID = 791024
	legacyRule := AffiliateCommissionRule{SalesId: salesID, GroupName: "legacy-test", RateBps: 500, Enabled: true}
	require.NoError(t, DB.Create(&legacyRule).Error)
	t.Cleanup(func() {
		DB.Where("application_id IN (SELECT id FROM affiliate_settlement_applications WHERE sales_id = ?)", salesID).Delete(&AffiliateSettlementItem{})
		DB.Where("sales_id = ?", salesID).Delete(&AffiliateSettlementApplication{})
		DB.Where("sales_id = ?", salesID).Delete(&AffiliateCommissionRule{})
		DB.Where("sales_id = ?", salesID).Delete(&AffiliateSalesPlanAssignment{})
		var plan AffiliateCommissionPlan
		if DB.Where("name = ?", "affiliate-test-plan").First(&plan).Error == nil {
			DB.Where("plan_id = ?", plan.Id).Delete(&AffiliateCommissionPlanRate{})
			DB.Delete(&plan)
		}
	})

	legacyRules, err := GetAffiliateCommissionRules(DB, salesID)
	require.NoError(t, err)
	require.Len(t, legacyRules, 1)
	assert.Equal(t, 500, legacyRules[0].RateBps)

	plan := AffiliateCommissionPlan{Name: "affiliate-test-plan", Enabled: true}
	require.NoError(t, DB.Create(&plan).Error)
	require.NoError(t, DB.Create(&AffiliateCommissionPlanRate{
		PlanId: plan.Id, GroupName: "plan-test", RateBps: 1250, Enabled: true,
	}).Error)
	require.NoError(t, DB.Create(&AffiliateSalesPlanAssignment{SalesId: salesID, PlanId: plan.Id}).Error)

	app := AffiliateSettlementApplication{SalesId: salesID, TargetUserId: 1, Status: AffiliateSettlementPending}
	require.NoError(t, DB.Create(&app).Error)
	snapshot := AffiliateSettlementItem{ApplicationId: app.Id, GroupName: "legacy-test", Quota: 10000, RateBps: 500, Commission: 500}
	require.NoError(t, DB.Create(&snapshot).Error)

	planRules, err := GetAffiliateCommissionRules(DB, salesID)
	require.NoError(t, err)
	require.Len(t, planRules, 1)
	assert.Equal(t, "plan-test", planRules[0].GroupName)
	assert.Equal(t, 1250, planRules[0].RateBps)

	require.NoError(t, DB.Delete(&legacyRule).Error)
	planRules, err = GetAffiliateCommissionRules(DB, salesID)
	require.NoError(t, err)
	require.Len(t, planRules, 1)
	var savedSnapshot AffiliateSettlementItem
	require.NoError(t, DB.First(&savedSnapshot, snapshot.Id).Error)
	assert.Equal(t, snapshot, savedSnapshot)
}

func TestAffiliateSalesLinkGrantsSubordinateAccessWithoutChangingPrimaryInviter(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&AffiliateSalesLink{}, &User{}))
	target := User{Username: "affiliate-link-target-test", Password: "test-password", InviterId: 791031}
	secondary := User{Username: "affiliate-link-secondary-test", Password: "test-password"}
	require.NoError(t, DB.Create(&target).Error)
	require.NoError(t, DB.Create(&secondary).Error)
	t.Cleanup(func() {
		DB.Where("user_id = ?", target.Id).Delete(&AffiliateSalesLink{})
		DB.Delete(&User{}, target.Id)
		DB.Delete(&User{}, secondary.Id)
	})

	primaryAccess, err := IsAffiliateSalesForUser(DB, target.Id, target.InviterId)
	require.NoError(t, err)
	assert.True(t, primaryAccess)

	require.NoError(t, DB.Create(&AffiliateSalesLink{UserId: target.Id, SalesId: secondary.Id}).Error)
	secondaryAccess, err := IsAffiliateSalesForUser(DB, target.Id, secondary.Id)
	require.NoError(t, err)
	assert.True(t, secondaryAccess)

	var unchanged User
	require.NoError(t, DB.First(&unchanged, target.Id).Error)
	assert.Equal(t, target.InviterId, unchanged.InviterId)
}

func TestAffiliateSettlementReservationIsPerSalesUser(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&AffiliateSettlementApplication{}, &QuotaData{}))
	const targetID = 791041
	application := AffiliateSettlementApplication{
		SalesId: 791042, TargetUserId: targetID, StartAt: 100, EndAt: 200,
		Status: AffiliateSettlementPending,
	}
	require.NoError(t, DB.Create(&application).Error)
	quota := QuotaData{UserID: targetID, CreatedAt: 150, UseGroup: "default", Quota: 1000}
	require.NoError(t, DB.Create(&quota).Error)
	t.Cleanup(func() {
		DB.Delete(&quota)
		DB.Delete(&application)
	})

	reservedRows, err := GetAvailableAffiliateSettlementQuota(DB, application.SalesId, targetID, 100, 200)
	require.NoError(t, err)
	assert.Empty(t, reservedRows)

	otherSalesRows, err := GetAvailableAffiliateSettlementQuota(DB, application.SalesId+1, targetID, 100, 200)
	require.NoError(t, err)
	require.Len(t, otherSalesRows, 1)
	assert.Equal(t, int64(1000), otherSalesRows[0].Quota)
}
