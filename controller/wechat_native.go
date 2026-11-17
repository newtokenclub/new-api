package controller

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

type wechatNativeCreateResponse struct { Code int `json:"code"`; Msg string `json:"msg"`; CodeURL string `json:"code_url"` }

func wechatNativeSettings() *operation_setting.PaymentSetting { return operation_setting.GetPaymentSetting() }

func wechatNativeSignature(body []byte, token string) string {
	h := sha256.New(); h.Write([]byte(token)); h.Write(body); return hex.EncodeToString(h.Sum(nil))
}

func RequestWechatNativePay(c *gin.Context) {
	settings := wechatNativeSettings()
	if !settings.WechatNativeEnabled || strings.TrimSpace(settings.WechatNativeNodeURL) == "" || strings.TrimSpace(settings.WechatNativeNodeToken) == "" || strings.TrimSpace(settings.WechatNativeNotifyURL) == "" {
		common.ApiErrorMsg(c, "微信Native支付未启用")
		return
	}
	var req AmountRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil || req.Amount < getMinTopup() {
		common.ApiErrorMsg(c, "充值金额无效")
		return
	}
	id := c.GetInt("id")
	group, err := model.GetUserGroup(id, true)
	if err != nil { common.ApiError(c, err); return }
	money := getPayMoney(req.Amount, group)
	tradeNo := fmt.Sprintf("WXN%d%d", id, time.Now().UnixNano())
	topUp := &model.TopUp{UserId:id, Amount:req.Amount, Money:money, TradeNo:tradeNo, PaymentMethod:model.PaymentMethodWechatNative, PaymentProvider:model.PaymentProviderWechatNative, CreateTime:time.Now().Unix(), Status:common.TopUpStatusPending}
	if err := topUp.Insert(); err != nil { common.ApiError(c, err); return }
	payload := []byte(fmt.Sprintf(`{"trade_no":"%s","amount_fen":%d,"description":"new-api wallet topup","notify_url":"%s"}`, tradeNo, int64(money*100+0.5), settings.WechatNativeNotifyURL))
	url := strings.TrimRight(settings.WechatNativeNodeURL, "/") + "/new-api/wechat-native/create"
	httpReq, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, url, bytes.NewReader(payload))
	httpReq.Header.Set("Content-Type", "application/json"); httpReq.Header.Set("X-New-API-Signature", wechatNativeSignature(payload, settings.WechatNativeNodeToken))
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil { common.ApiErrorMsg(c, "微信支付服务暂时不可用"); return }
	defer resp.Body.Close(); body, _ := io.ReadAll(resp.Body)
	var result wechatNativeCreateResponse
	if resp.StatusCode/100 != 2 || common.Unmarshal(body, &result) != nil || result.Code <= 0 || result.CodeURL == "" {
		logger.LogError(c.Request.Context(), fmt.Sprintf("wechat native create failed: status=%d response=%s", resp.StatusCode, common.LocalLogPreview(string(body))))
	}
	if err := common.Unmarshal(body, &result); err != nil || resp.StatusCode/100 != 2 || result.Code <= 0 || result.CodeURL == "" { common.ApiErrorMsg(c, "创建微信支付订单失败"); return }
	common.ApiSuccess(c, gin.H{"trade_no": tradeNo, "code_url": result.CodeURL})
}

func QueryWechatNativeOrders(c *gin.Context) {
	settings := wechatNativeSettings()
	var orders []*model.TopUp
	model.DB.Where("user_id = ? AND payment_provider = ? AND status = ?", c.GetInt("id"), model.PaymentProviderWechatNative, common.TopUpStatusPending).Order("id desc").Limit(20).Find(&orders)
	if len(orders) == 0 || strings.TrimSpace(settings.WechatNativeNodeURL) == "" || strings.TrimSpace(settings.WechatNativeNodeToken) == "" { common.ApiSuccess(c, gin.H{"orders": orders}); return }
	for _, order := range orders {
		payload := []byte(fmt.Sprintf(`{"trade_no":"%s"}`, order.TradeNo))
		url := strings.TrimRight(settings.WechatNativeNodeURL, "/") + "/new-api/wechat-native/query"
		req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, url, bytes.NewReader(payload)); req.Header.Set("Content-Type", "application/json"); req.Header.Set("X-New-API-Signature", wechatNativeSignature(payload, settings.WechatNativeNodeToken))
		resp, err := http.DefaultClient.Do(req); if err != nil { continue }; body, _ := io.ReadAll(resp.Body); resp.Body.Close()
		var result struct { Paid bool `json:"paid"`; AmountFen int64 `json:"amount_fen"` }
		if common.Unmarshal(body, &result) == nil && result.Paid && result.AmountFen == int64(order.Money*100+0.5) { _ = model.CompleteWechatNativeTopUp(order.TradeNo, c.ClientIP()) }
	}
	orders = nil
	model.DB.Where("user_id = ? AND payment_provider = ? AND status = ?", c.GetInt("id"), model.PaymentProviderWechatNative, common.TopUpStatusPending).Order("id desc").Limit(20).Find(&orders)
	common.ApiSuccess(c, gin.H{"orders": orders})
}

func WechatNativeNotify(c *gin.Context) {
	settings := wechatNativeSettings()
	if c.GetHeader("X-New-API-Signature") == "" || settings.WechatNativeNodeToken == "" { c.Status(http.StatusUnauthorized); return }
	body, err := io.ReadAll(c.Request.Body); if err != nil { c.Status(http.StatusBadRequest); return }
	if c.GetHeader("X-New-API-Signature") != wechatNativeSignature(body, settings.WechatNativeNodeToken) { c.Status(http.StatusUnauthorized); return }
	var payload struct { TradeNo string `json:"trade_no"`; Paid bool `json:"paid"`; AmountFen int64 `json:"amount_fen"` }
	if common.Unmarshal(body, &payload) != nil || !payload.Paid { c.Status(http.StatusBadRequest); return }
	order := model.GetTopUpByTradeNo(payload.TradeNo)
	if order == nil || order.PaymentProvider != model.PaymentProviderWechatNative || payload.AmountFen != int64(order.Money*100+0.5) { c.Status(http.StatusBadRequest); return }
	if err := model.CompleteWechatNativeTopUp(payload.TradeNo, c.ClientIP()); err != nil && err != model.ErrTopUpStatusInvalid { c.Status(http.StatusInternalServerError); return }
	c.JSON(http.StatusOK, gin.H{"success": true})
}
