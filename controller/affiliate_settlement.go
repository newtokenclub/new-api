package controller

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func ListAffiliateRules(c *gin.Context) {
	salesID := c.GetInt("id")
	if c.GetInt("role") == common.RoleRootUser {
		salesID, _ = strconv.Atoi(c.Query("sales_id"))
	}
	type ruleRow struct {
		model.AffiliateCommissionRule
		SalesUsername string `json:"sales_username"`
		SalesDisplayName string `json:"sales_display_name"`
	}
	if c.GetInt("role") != common.RoleRootUser {
		rules, err := model.GetAffiliateCommissionRules(model.DB, salesID)
		if err != nil { common.ApiError(c, err); return }
		c.JSON(http.StatusOK, gin.H{"success": true, "data": rules})
		return
	}
	var rows []ruleRow
	q := model.DB.Table("affiliate_commission_rules AS r").Select("r.*, u.username AS sales_username, u.display_name AS sales_display_name").Joins("LEFT JOIN users u ON u.id = r.sales_id").Order("r.sales_id, r.group_name")
	if salesID > 0 { q = q.Where("r.sales_id = ?", salesID) }
	if err := q.Scan(&rows).Error; err != nil { common.ApiError(c, err); return }
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

func SearchAffiliateSalesUsers(c *gin.Context) {
	if c.GetInt("role") != common.RoleRootUser { common.ApiErrorMsg(c, "root permission required"); return }
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" { c.JSON(http.StatusOK, gin.H{"success": true, "data": []interface{}{}}); return }
	query := model.DB.Model(&model.User{}).Select("id, username, display_name, role, status").Where("deleted_at IS NULL")
	like := "%" + keyword + "%"
	if id, err := strconv.Atoi(keyword); err == nil { query = query.Where("id = ? OR username LIKE ? OR display_name LIKE ?", id, like, like) } else { query = query.Where("username LIKE ? OR display_name LIKE ?", like, like) }
	var users []struct { Id int `json:"id"`; Username string `json:"username"`; DisplayName string `json:"display_name"`; Role int `json:"role"`; Status int `json:"status"` }
	if err := query.Order("id DESC").Limit(20).Find(&users).Error; err != nil { common.ApiError(c, err); return }
	c.JSON(http.StatusOK, gin.H{"success": true, "data": users})
}

func SaveAffiliateRule(c *gin.Context) { var req struct{ SalesID int `json:"sales_id"`; GroupName string `json:"group_name"`; RateBps int `json:"rate_bps"`; Enabled *bool `json:"enabled"`; Remark string `json:"remark"` }; if err:=common.DecodeJson(c.Request.Body,&req); err!=nil { common.ApiError(c,err); return }; if c.GetInt("role") != common.RoleRootUser { common.ApiErrorMsg(c,"root permission required"); return }; if req.SalesID<=0 || strings.TrimSpace(req.GroupName)=="" { common.ApiErrorMsg(c,"sales and group are required"); return }; if err:=model.ValidateAffiliateRate(req.RateBps); err!=nil { common.ApiError(c,err); return }; enabled:=true; if req.Enabled!=nil { enabled=*req.Enabled }; var row model.AffiliateCommissionRule; err:=model.DB.Where("sales_id = ? AND group_name = ?",req.SalesID,strings.TrimSpace(req.GroupName)).First(&row).Error; if err==nil { row.RateBps=req.RateBps; row.Enabled=enabled; row.Remark=req.Remark; err=model.DB.Save(&row).Error } else { row=model.AffiliateCommissionRule{SalesId:req.SalesID,GroupName:strings.TrimSpace(req.GroupName),RateBps:req.RateBps,Enabled:enabled,Remark:req.Remark}; err=model.DB.Create(&row).Error }; if err!=nil { common.ApiError(c,err); return }; c.JSON(http.StatusOK,gin.H{"success":true,"data":row}) }

func DeleteAffiliateRule(c *gin.Context) {
	if c.GetInt("role") != common.RoleRootUser { common.ApiErrorMsg(c, "root permission required"); return }
	id, err := strconv.Atoi(c.Param("id")); if err != nil || id <= 0 { common.ApiErrorMsg(c, "invalid rule id"); return }
	if err := model.DB.Delete(&model.AffiliateCommissionRule{}, id).Error; err != nil { common.ApiError(c, err); return }
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func ListAffiliateCommissionPlans(c *gin.Context) {
	var plans []model.AffiliateCommissionPlan
	if err := model.DB.Preload("Rates").Order("id DESC").Find(&plans).Error; err != nil { common.ApiError(c, err); return }
	type assignmentRow struct { SalesId int `json:"sales_id"`; PlanId int `json:"plan_id"`; Username string `json:"username"`; DisplayName string `json:"display_name"` }
	type planRow struct { Plan model.AffiliateCommissionPlan `json:"plan"`; Assignments []assignmentRow `json:"assignments"` }
	result := make([]planRow, 0, len(plans))
	for i := range plans {
		var assignments []assignmentRow
		if err := model.DB.Table("affiliate_sales_plan_assignments AS a").Select("a.sales_id, a.plan_id, u.username, u.display_name").Joins("LEFT JOIN users u ON u.id = a.sales_id").Where("a.plan_id = ?", plans[i].Id).Find(&assignments).Error; err != nil { common.ApiError(c, err); return }
		result = append(result, planRow{Plan: plans[i], Assignments: assignments})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func SaveAffiliateCommissionPlan(c *gin.Context) {
	if c.GetInt("role") != common.RoleRootUser { common.ApiErrorMsg(c, "root permission required"); return }
	var req struct { Id int `json:"id"`; Name string `json:"name"`; Enabled bool `json:"enabled"`; Rates []model.AffiliateCommissionPlanRate `json:"rates"` }
	if err := common.DecodeJson(c.Request.Body, &req); err != nil { common.ApiError(c, err); return }
	req.Name = strings.TrimSpace(req.Name); if req.Name == "" { common.ApiErrorMsg(c, "plan name is required"); return }
	if req.Id < 0 { common.ApiErrorMsg(c, "invalid plan id"); return }
	seen := map[string]bool{}
	for _, rate := range req.Rates { group := strings.TrimSpace(rate.GroupName); if group == "" || seen[group] { common.ApiErrorMsg(c, "invalid or duplicate group"); return }; if rate.Enabled { if err := model.ValidateAffiliateRate(rate.RateBps); err != nil { common.ApiError(c, err); return } }; seen[group] = true }
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		plan := model.AffiliateCommissionPlan{Id: req.Id, Name: req.Name, Enabled: req.Enabled}
		if req.Id == 0 { if err := tx.Create(&plan).Error; err != nil { return err } } else {
			if err := tx.First(&plan, req.Id).Error; err != nil { return err }
			if err := tx.Model(&plan).Updates(map[string]interface{}{"name": req.Name, "enabled": req.Enabled}).Error; err != nil { return err }
		}
		planId := req.Id; if planId == 0 { planId = plan.Id }
		if err := tx.Where("plan_id = ?", planId).Delete(&model.AffiliateCommissionPlanRate{}).Error; err != nil { return err }
		for _, rate := range req.Rates { rate.Id = 0; rate.PlanId = planId; rate.GroupName = strings.TrimSpace(rate.GroupName); if err := tx.Create(&rate).Error; err != nil { return err } }
		return nil
	})
	if err != nil { common.ApiError(c, err); return }
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func DeleteAffiliateCommissionPlan(c *gin.Context) {
	if c.GetInt("role") != common.RoleRootUser { common.ApiErrorMsg(c, "root permission required"); return }
	id, err := strconv.Atoi(c.Param("id")); if err != nil || id <= 0 { common.ApiErrorMsg(c, "invalid plan id"); return }
	var count int64; if err := model.DB.Model(&model.AffiliateSalesPlanAssignment{}).Where("plan_id = ?", id).Count(&count).Error; err != nil { common.ApiError(c, err); return }
	if count > 0 { common.ApiErrorMsg(c, "remove sales assignments before deleting this plan"); return }
	if err := model.DB.Transaction(func(tx *gorm.DB) error { if err := tx.Where("plan_id = ?", id).Delete(&model.AffiliateCommissionPlanRate{}).Error; err != nil { return err }; return tx.Delete(&model.AffiliateCommissionPlan{}, id).Error }); err != nil { common.ApiError(c, err); return }
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AssignAffiliateCommissionPlan(c *gin.Context) {
	if c.GetInt("role") != common.RoleRootUser { common.ApiErrorMsg(c, "root permission required"); return }
	var req struct { SalesID int `json:"sales_id"`; PlanID int `json:"plan_id"` }; if err := common.DecodeJson(c.Request.Body, &req); err != nil { common.ApiError(c, err); return }
	if req.SalesID <= 0 { common.ApiErrorMsg(c, "invalid sales id"); return }
	var count int64; if err := model.DB.Model(&model.User{}).Where("id = ? AND deleted_at IS NULL", req.SalesID).Count(&count).Error; err != nil || count == 0 { common.ApiErrorMsg(c, "sales user not found"); return }
	if req.PlanID > 0 { if err := model.DB.Where("id = ? AND enabled = ?", req.PlanID, true).First(&model.AffiliateCommissionPlan{}).Error; err != nil { common.ApiErrorMsg(c, "enabled plan not found"); return } }
	var assignment model.AffiliateSalesPlanAssignment
	err := model.DB.Where("sales_id = ?", req.SalesID).First(&assignment).Error
	if err == nil {
		assignment.PlanId = req.PlanID
		err = model.DB.Save(&assignment).Error
	} else if err == gorm.ErrRecordNotFound {
		err = model.DB.Create(&model.AffiliateSalesPlanAssignment{SalesId: req.SalesID, PlanId: req.PlanID}).Error
	}
	if err != nil { common.ApiError(c, err); return }; c.JSON(http.StatusOK, gin.H{"success": true})
}

type settlementItemRequest struct { GroupName string `json:"group_name"`; Quota int64 `json:"quota"`; RateBps int `json:"rate_bps"`; Commission int64 `json:"commission"` }
func PreviewAffiliateSettlement(c *gin.Context) { target,ok:=parseAffiliateTarget(c); if !ok{return}; end,_:=strconv.ParseInt(c.Query("end_at"),10,64); if end==0 { end=time.Now().AddDate(0,0,-1).Truncate(24*time.Hour).Unix()+24*60*60 }; start,_:=strconv.ParseInt(c.Query("start_at"),10,64); if start==0 { start=0 }; rules,err:=model.GetAffiliateCommissionRules(model.DB,c.GetInt("id")); if err!=nil {common.ApiError(c,err);return}; allowed:=map[string]model.AffiliateCommissionRule{}; for _,r:=range rules {allowed[r.GroupName]=r}; rows,err:=model.GetAvailableAffiliateSettlementQuota(model.DB,target,start,end); if err!=nil {common.ApiError(c,err);return}; items:=[]settlementItemRequest{}; var total,commission int64; for _,r:=range rows { rule,yes:=allowed[r.GroupName]; if !yes || r.Quota<=0 {continue}; amount:=r.Quota*int64(rule.RateBps)/10000; items=append(items,settlementItemRequest{GroupName:r.GroupName,Quota:r.Quota,RateBps:rule.RateBps,Commission:amount}); total+=r.Quota; commission+=amount }; c.JSON(http.StatusOK,gin.H{"success":true,"data":gin.H{"start_at":start,"end_at":end,"total_quota":total,"total_commission":commission,"currency":operation_setting.GetQuotaDisplayType(),"items":items}}) }

func CreateAffiliateSettlement(c *gin.Context) { var req struct{ TargetUserID int `json:"target_user_id"`; StartAt int64 `json:"start_at"`; EndAt int64 `json:"end_at"`; Remark string `json:"remark"`; PaymentType string `json:"payment_type"`; PaymentNote string `json:"payment_note"`; AttachmentKeys []string `json:"attachment_keys"`; TotalQuota int64 `json:"total_quota"`; TotalCommission int64 `json:"total_commission"`; Items []settlementItemRequest `json:"items"` }; if err:=common.DecodeJson(c.Request.Body,&req); err!=nil {common.ApiError(c,err);return}; if len(req.Items)==0 || req.EndAt<=req.StartAt {common.ApiErrorMsg(c,"no eligible consumption");return}; if req.EndAt > time.Now().Truncate(24*time.Hour).Unix() { common.ApiErrorMsg(c,"settlement end date must be before today"); return }; target,ok:=parseAffiliateTargetByID(c,req.TargetUserID); if !ok{return}; _=target; rules,err:=model.GetAffiliateCommissionRules(model.DB,c.GetInt("id")); if err!=nil {common.ApiError(c,err);return}; ruleMap:=map[string]model.AffiliateCommissionRule{}; for _,r:=range rules {ruleMap[r.GroupName]=r}; dbRows,err:=model.GetAvailableAffiliateSettlementQuota(model.DB,req.TargetUserID,req.StartAt,req.EndAt); if err!=nil {common.ApiError(c,err);return}; actual:=map[string]int64{}; for _,r:=range dbRows {actual[r.GroupName]=r.Quota}; var computedTotal,computedCommission int64; for _,i:=range req.Items {rule,yes:=ruleMap[i.GroupName]; if !yes || actual[i.GroupName]<=0 || i.RateBps!=rule.RateBps || i.Quota!=actual[i.GroupName] {common.ApiErrorMsg(c,"settlement details have changed, please refresh");return}; expected:=i.Quota*int64(i.RateBps)/10000; if i.Commission!=expected {common.ApiErrorMsg(c,"invalid commission amount");return}; computedTotal+=i.Quota; computedCommission+=expected }; if computedTotal!=req.TotalQuota || computedCommission!=req.TotalCommission {common.ApiErrorMsg(c,"invalid settlement total");return}; var keys string; b,_:=json.Marshal(req.AttachmentKeys); keys=string(b); app:=&model.AffiliateSettlementApplication{SalesId:c.GetInt("id"),TargetUserId:req.TargetUserID,StartAt:req.StartAt,EndAt:req.EndAt,TotalQuota:req.TotalQuota,TotalCommission:req.TotalCommission,Currency:operation_setting.GetQuotaDisplayType(),Remark:req.Remark,PaymentType:req.PaymentType,PaymentNote:req.PaymentNote,AttachmentKeys:keys}; for _,i:=range req.Items { app.Items=append(app.Items,model.AffiliateSettlementItem{GroupName:i.GroupName,Quota:i.Quota,RateBps:i.RateBps,Commission:i.Commission}) }; if err:=model.CreateAffiliateSettlement(app);err!=nil{common.ApiError(c,err);return}; c.JSON(http.StatusOK,gin.H{"success":true,"data":app}) }

func parseAffiliateTargetByID(c *gin.Context,id int)(int,bool){ var n int64; model.DB.Model(&model.User{}).Where("id = ? AND inviter_id = ? AND deleted_at IS NULL",id,c.GetInt("id")).Count(&n); if n==0 {common.ApiErrorMsg(c,"target is not your direct subordinate");return 0,false}; return id,true }

type affiliateSettlementListItem struct {
	model.AffiliateSettlementApplication
	SalesUsername string `json:"sales_username"`
	SalesDisplayName string `json:"sales_display_name"`
	TargetUsername string `json:"target_username"`
	TargetDisplayName string `json:"target_display_name"`
	AttachmentURLs []string `json:"attachment_urls"`
}

func ListAffiliateSettlements(c *gin.Context){
	q:=model.DB.Model(&model.AffiliateSettlementApplication{}).Preload("Items").Order("id DESC")
	if c.GetInt("role")!=common.RoleRootUser {
		q=q.Where("sales_id = ?",c.GetInt("id"))
	} else if rawSalesID := strings.TrimSpace(c.Query("sales_id")); rawSalesID != "" && rawSalesID != "all" {
		salesID, err := strconv.Atoi(rawSalesID)
		if err != nil || salesID <= 0 {
			common.ApiErrorMsg(c, "invalid sales id")
			return
		}
		var count int64
		if err := model.DB.Model(&model.AffiliateSettlementApplication{}).Where("sales_id = ?", salesID).Count(&count).Error; err != nil {
			common.ApiError(c, err)
			return
		}
		if count == 0 {
			common.ApiErrorMsg(c, "sales user has no settlement applications")
			return
		}
		q=q.Where("sales_id = ?",salesID)
	}
	var rows []model.AffiliateSettlementApplication
	if err:=q.Find(&rows).Error;err!=nil{common.ApiError(c,err);return}
	config:=service.QuotaApplicationR2ConfigFromOptions()
	result:=make([]affiliateSettlementListItem,0,len(rows))
	for _,row:=range rows {
		item:=affiliateSettlementListItem{AffiliateSettlementApplication:row,AttachmentURLs:[]string{}}
		var sales,target model.User
		model.DB.Select("id, username, display_name").First(&sales,row.SalesId)
		model.DB.Select("id, username, display_name").First(&target,row.TargetUserId)
		item.SalesUsername=sales.Username; item.SalesDisplayName=sales.DisplayName
		item.TargetUsername=target.Username; item.TargetDisplayName=target.DisplayName
		var keys []string
		if json.Unmarshal([]byte(row.AttachmentKeys),&keys)==nil { for _,key:=range keys { if key!="" { item.AttachmentURLs=append(item.AttachmentURLs,service.R2ObjectURL(config,key)) } } }
		result=append(result,item)
	}
	c.JSON(http.StatusOK,gin.H{"success":true,"data":result})
}

func GetAffiliateSettlementConsumption(c *gin.Context) {
	id,err:=strconv.Atoi(c.Param("id")); if err!=nil || id<=0 { common.ApiErrorMsg(c,"invalid settlement id"); return }
	var app model.AffiliateSettlementApplication
	if err:=model.DB.First(&app,id).Error; err!=nil { common.ApiError(c,err); return }
	if c.GetInt("role")!=common.RoleRootUser && app.SalesId!=c.GetInt("id") { c.JSON(http.StatusForbidden,gin.H{"success":false,"message":"permission denied"}); return }
	start,_:=strconv.ParseInt(c.Query("start_timestamp"),10,64); end,_:=strconv.ParseInt(c.Query("end_timestamp"),10,64)
	dates,err:=model.GetQuotaDataByUserId(app.TargetUserId,start,end,true); if err!=nil {common.ApiError(c,err);return}
	c.JSON(http.StatusOK,gin.H{"success":true,"data":dates})
}
func ReviewAffiliateSettlement(c *gin.Context) {
	if c.GetInt("role") != common.RoleRootUser { common.ApiErrorMsg(c, "root permission required"); return }
	id, err := strconv.Atoi(c.Param("id")); if err != nil || id <= 0 { common.ApiErrorMsg(c, "invalid settlement id"); return }
	var req struct { Approved bool `json:"approved"`; Comment string `json:"comment"`; Paid bool `json:"paid"` }
	if err := common.DecodeJson(c.Request.Body, &req); err != nil { common.ApiError(c, err); return }
	var app model.AffiliateSettlementApplication
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&app, id).Error; err != nil { return err }
		if app.Status == model.AffiliateSettlementPaid {
			if req.Paid { return nil }
			return errors.New("paid settlements cannot be changed")
		}
		if app.Status == model.AffiliateSettlementRejected { return errors.New("rejected settlements cannot be changed") }
		if req.Paid {
			if app.Status != model.AffiliateSettlementApproved { return errors.New("only approved settlements can be marked paid") }
			app.Status = model.AffiliateSettlementPaid; app.PaidAt = time.Now().Unix(); app.PaidBy = c.GetInt("id")
		} else if req.Approved {
			if app.Status != model.AffiliateSettlementPending && app.Status != model.AffiliateSettlementApproved { return errors.New("invalid settlement status transition") }
			app.Status = model.AffiliateSettlementApproved; app.ReviewedAt = time.Now().Unix(); app.ReviewedBy = c.GetInt("id")
		} else {
			if app.Status != model.AffiliateSettlementPending { return errors.New("only pending settlements can be rejected") }
			app.Status = model.AffiliateSettlementRejected; app.ReviewedAt = time.Now().Unix(); app.ReviewedBy = c.GetInt("id")
		}
		app.ReviewComment = req.Comment
		return tx.Save(&app).Error
	})
	if err != nil { common.ApiError(c, err); return }
	if err := model.DB.Preload("Items").First(&app, id).Error; err != nil { common.ApiError(c, err); return }
	c.JSON(http.StatusOK, gin.H{"success": true, "data": app})
}
