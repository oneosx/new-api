package controller

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type WeChatPayRequest struct {
	Amount int64  `json:"amount"`
	OpenID string `json:"openid"`
}

type WeChatRefundRequest struct {
	TradeNo string `json:"trade_no"`
	Reason  string `json:"reason"`
}

func wechatPayMethod() map[string]string {
	return map[string]string{
		"name":      "微信支付",
		"type":      model.PaymentMethodWeChatJSAPI,
		"color":     setting.WeChatPayColor,
		"min_topup": strconv.FormatInt(wechatPayMinTopup(), 10),
	}
}

func appendWeChatPayMethod(payMethods []map[string]string) []map[string]string {
	if !isWeChatPayTopUpEnabled() {
		return payMethods
	}
	for _, method := range payMethods {
		if method["type"] == model.PaymentMethodWeChatJSAPI {
			return payMethods
		}
	}
	return append(payMethods, wechatPayMethod())
}

func wechatPayMoney(amount int64, group string) float64 {
	return getPayMoney(amount, group)
}

func wechatAmountCents(payMoney float64) int64 {
	return decimal.NewFromFloat(payMoney).Mul(decimal.NewFromInt(100)).Round(0).IntPart()
}

func wechatPayMinTopup() int64 {
	minTopup := setting.WeChatPayMinTopup()
	if operation_setting.GetQuotaDisplayType() != operation_setting.QuotaDisplayTypeTokens {
		return minTopup
	}
	quota, err := common.WalletQuotaFromDecimalStrict(
		decimal.NewFromInt(minTopup).Mul(decimal.NewFromFloat(common.QuotaPerUnit)),
	)
	if err != nil {
		return int64(common.MaxWalletQuota)
	}
	return int64(quota)
}

func wechatStoredTopUpAmount(amount int64) (int64, error) {
	if operation_setting.GetQuotaDisplayType() != operation_setting.QuotaDisplayTypeTokens {
		return amount, nil
	}
	if common.QuotaPerUnit <= 0 {
		return 0, errors.New("充值额度配置无效")
	}
	storedAmount := decimal.NewFromInt(amount).Div(decimal.NewFromFloat(common.QuotaPerUnit))
	if !storedAmount.Equal(storedAmount.Truncate(0)) || storedAmount.LessThanOrEqual(decimal.Zero) {
		return 0, errors.New("Token 充值数量必须是额度单位的整数倍")
	}
	return storedAmount.IntPart(), nil
}

func RequestWeChatAmount(c *gin.Context) {
	if !isWeChatPayTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "未启用微信支付"})
		return
	}
	var req WeChatPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if minTopup := wechatPayMinTopup(); req.Amount < minTopup {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", minTopup)})
		return
	}
	id := c.GetInt("id")
	user, err := model.GetUserById(id, false)
	if err != nil || user == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "用户不存在"})
		return
	}
	if rejectInvalidTopUpQuota(c, id, req.Amount) {
		return
	}
	if _, err := wechatStoredTopUpAmount(req.Amount); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return
	}
	payMoney := wechatPayMoney(req.Amount, user.Group)
	if payMoney <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": strconv.FormatFloat(payMoney, 'f', 2, 64)})
}

func RequestWeChatPay(c *gin.Context) {
	if !isWeChatPayTopUpEnabled() {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "未启用微信支付"})
		return
	}
	var req WeChatPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "参数错误"})
		return
	}
	if minTopup := wechatPayMinTopup(); req.Amount < minTopup {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": fmt.Sprintf("充值数量不能小于 %d", minTopup)})
		return
	}
	id := c.GetInt("id")
	user, err := model.GetUserById(id, false)
	if err != nil || user == nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "用户不存在"})
		return
	}
	openID := strings.TrimSpace(user.WeChatId)
	if openID == "" {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "请先在微信内授权或绑定微信后再支付"})
		return
	}
	if rejectInvalidTopUpQuota(c, id, req.Amount) {
		return
	}
	storedAmount, err := wechatStoredTopUpAmount(req.Amount)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": err.Error()})
		return
	}
	payMoney := wechatPayMoney(req.Amount, user.Group)
	if payMoney <= 0.01 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	amountCents := wechatAmountCents(payMoney)
	if amountCents < 1 {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "充值金额过低"})
		return
	}
	tradeNo := fmt.Sprintf("USR%dWX%s%d", id, common.GetRandomString(6), time.Now().Unix())
	topUp := &model.TopUp{
		UserId:          id,
		Amount:          storedAmount,
		Money:           payMoney,
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodWeChatJSAPI,
		PaymentProvider: model.PaymentProviderWeChat,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付创建订单失败 user_id=%d trade_no=%s error=%q", id, tradeNo, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}
	prepay, err := service.CreateWeChatJSAPIPrepay(tradeNo, fmt.Sprintf("钱包充值 TUC%d", req.Amount), openID, amountCents)
	if err != nil {
		_ = model.UpdatePendingTopUpStatus(tradeNo, model.PaymentProviderWeChat, common.TopUpStatusFailed)
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付下单失败 user_id=%d amount=%d error=%q", id, req.Amount, err.Error()))
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "拉起微信支付失败"})
		return
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付订单创建成功 user_id=%d trade_no=%s amount=%d money=%.2f", id, tradeNo, req.Amount, payMoney))
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"trade_no":  tradeNo,
			"app_id":    setting.WeChatPayAppID(),
			"timeStamp": prepay.TimeStamp,
			"nonceStr":  prepay.NonceStr,
			"package":   prepay.Package,
			"signType":  prepay.SignType,
			"paySign":   prepay.PaySign,
		},
	})
}

func GetWeChatPayOrder(c *gin.Context) {
	id := c.GetInt("id")
	tradeNo := strings.TrimSpace(c.Param("trade_no"))
	topUp, err := model.GetUserWeChatTopUp(id, tradeNo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "error", "data": "订单不存在"})
		return
	}
	if topUp.Status == common.TopUpStatusPending {
		txInfo, queryErr := service.QueryWeChatTransaction(tradeNo)
		if queryErr != nil {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付查单失败 trade_no=%s error=%q", tradeNo, queryErr.Error()))
		} else if txInfo.TradeState == "SUCCESS" && txInfo.MchID == setting.WeChatPayMchID() && txInfo.AppID == setting.WeChatPayAppID() && txInfo.TradeType == setting.WeChatPayTradeTypeJSAPI && txInfo.Amount.Currency == "CNY" {
			if _, rechargeErr := model.RechargeWeChat(tradeNo, txInfo.TransactionID, txInfo.Amount.Total, c.ClientIP()); rechargeErr != nil {
				logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付查单入账失败 trade_no=%s error=%q", tradeNo, rechargeErr.Error()))
			} else if refreshed, refreshErr := model.GetUserWeChatTopUp(id, tradeNo); refreshErr == nil {
				topUp = refreshed
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"trade_no": topUp.TradeNo,
			"status":   topUp.Status,
			"amount":   topUp.Amount,
			"money":    topUp.Money,
		},
	})
}

func writeWeChatNotify(c *gin.Context, success bool, message string) {
	c.Header("Content-Type", "application/json")
	if success {
		c.Status(http.StatusOK)
		_, _ = c.Writer.Write(service.WeChatPaySuccessJSON())
		return
	}
	c.Status(http.StatusBadRequest)
	_, _ = c.Writer.Write(service.WeChatPayFailJSON(message))
}

func WeChatPayNotify(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeWeChatNotify(c, false, "读取回调失败")
		return
	}
	if err := service.VerifyWeChatNotification(c.Request.Header, body); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付回调验签失败 error=%q", err.Error()))
		writeWeChatNotify(c, false, "回调验签失败")
		return
	}
	notification, err := service.ParseWeChatNotification(body)
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付回调解析失败 error=%q", err.Error()))
		writeWeChatNotify(c, false, "回调解析失败")
		return
	}
	if notification.EventType != "" && notification.EventType != "TRANSACTION.SUCCESS" {
		writeWeChatNotify(c, true, "")
		return
	}
	txInfo, err := service.DecryptWeChatTransaction(notification)
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付回调解密失败 error=%q", err.Error()))
		writeWeChatNotify(c, false, "回调解密失败")
		return
	}
	if txInfo.MchID != setting.WeChatPayMchID() || txInfo.AppID != setting.WeChatPayAppID() || txInfo.TradeType != setting.WeChatPayTradeTypeJSAPI || txInfo.Amount.Currency != "CNY" {
		writeWeChatNotify(c, false, "支付信息不匹配")
		return
	}
	if txInfo.TradeState != "SUCCESS" {
		writeWeChatNotify(c, true, "")
		return
	}
	alreadyDone, err := model.RechargeWeChat(txInfo.OutTradeNo, txInfo.TransactionID, txInfo.Amount.Total, c.ClientIP())
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付入账失败 trade_no=%s error=%q", txInfo.OutTradeNo, err.Error()))
		writeWeChatNotify(c, false, "入账失败")
		return
	}
	if alreadyDone {
		logger.LogInfo(c.Request.Context(), fmt.Sprintf("微信支付重复回调忽略 trade_no=%s", txInfo.OutTradeNo))
	}
	writeWeChatNotify(c, true, "")
}

func WeChatRefundNotify(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		writeWeChatNotify(c, false, "读取回调失败")
		return
	}
	if err := service.VerifyWeChatNotification(c.Request.Header, body); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信退款回调验签失败 error=%q", err.Error()))
		writeWeChatNotify(c, false, "回调验签失败")
		return
	}
	notification, err := service.ParseWeChatNotification(body)
	if err != nil {
		writeWeChatNotify(c, false, "回调解析失败")
		return
	}
	if notification.EventType != "" && notification.EventType != "REFUND.SUCCESS" && notification.EventType != "REFUND.CLOSED" && notification.EventType != "REFUND.ABNORMAL" {
		writeWeChatNotify(c, true, "")
		return
	}
	refundInfo, err := service.DecryptWeChatRefund(notification)
	if err != nil {
		writeWeChatNotify(c, false, "回调解密失败")
		return
	}
	if refundInfo.MchID != setting.WeChatPayMchID() || refundInfo.OutRefundNo != service.WeChatRefundNo(refundInfo.OutTradeNo) {
		writeWeChatNotify(c, false, "退款信息不匹配")
		return
	}
	switch strings.ToUpper(refundInfo.RefundStatus) {
	case "SUCCESS":
		topUp, err := model.GetWeChatTopUpByTradeNo(refundInfo.OutTradeNo)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("微信退款订单不存在 trade_no=%s error=%q", refundInfo.OutTradeNo, err.Error()))
			writeWeChatNotify(c, false, "退款订单不存在")
			return
		}
		expectedCents := wechatAmountCents(topUp.Money)
		if refundInfo.Amount.Total != expectedCents || refundInfo.Amount.Refund != expectedCents {
			logger.LogError(c.Request.Context(), fmt.Sprintf("微信退款金额不匹配 trade_no=%s total=%d refund=%d expected=%d", refundInfo.OutTradeNo, refundInfo.Amount.Total, refundInfo.Amount.Refund, expectedCents))
			writeWeChatNotify(c, false, "退款金额不匹配")
			return
		}
		if _, err := model.CompleteWeChatRefund(refundInfo.OutTradeNo, c.ClientIP()); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("微信退款完成失败 trade_no=%s error=%q", refundInfo.OutTradeNo, err.Error()))
			writeWeChatNotify(c, false, "退款入账失败")
			return
		}
	case "CLOSED":
		if err := model.RollbackWeChatRefund(refundInfo.OutTradeNo); err != nil && !errors.Is(err, model.ErrTopUpStatusInvalid) {
			logger.LogError(c.Request.Context(), fmt.Sprintf("微信退款关闭回滚失败 trade_no=%s error=%q", refundInfo.OutTradeNo, err.Error()))
			writeWeChatNotify(c, false, "退款回滚失败")
			return
		}
	}
	writeWeChatNotify(c, true, "")
}

func AdminRefundWeChatTopUp(c *gin.Context) {
	if !setting.IsWeChatPayConfigured() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未配置微信支付商户凭证"})
		return
	}
	var req WeChatRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TradeNo) == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "管理员全额退款"
	}
	topUp, _, err := model.PrepareWeChatFullRefund(req.TradeNo)
	if errors.Is(err, model.ErrWeChatRefundInProgress) {
		message, reconcileErr := reconcileWeChatRefund(req.TradeNo, c.ClientIP())
		if errors.Is(reconcileErr, service.ErrWeChatPayNotFound) {
			topUp, getErr := model.GetWeChatTopUpByTradeNo(req.TradeNo)
			if getErr != nil {
				c.JSON(http.StatusOK, gin.H{"success": false, "message": wechatRefundErrorMessage(getErr)})
				return
			}
			refundNo := service.WeChatRefundNo(topUp.TradeNo)
			if createErr := service.CreateWeChatRefund(topUp.TradeNo, refundNo, reason, "", wechatAmountCents(topUp.Money), wechatAmountCents(topUp.Money)); createErr != nil {
				handleWeChatRefundCreateError(c, topUp.TradeNo, refundNo, createErr)
				return
			}
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "退款已提交，额度已预扣，等待微信确认"})
			return
		}
		if reconcileErr != nil {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信退款查单失败 trade_no=%s error=%q", req.TradeNo, reconcileErr.Error()))
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "退款处理中，暂无法确认微信退款状态"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": wechatRefundErrorMessage(err)})
		return
	}
	refundNo := service.WeChatRefundNo(topUp.TradeNo)
	if err := service.CreateWeChatRefund(topUp.TradeNo, refundNo, reason, "", wechatAmountCents(topUp.Money), wechatAmountCents(topUp.Money)); err != nil {
		handleWeChatRefundCreateError(c, topUp.TradeNo, refundNo, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "退款已提交，额度已预扣，等待微信确认"})
}

func wechatRefundErrorMessage(err error) string {
	switch {
	case errors.Is(err, model.ErrTopUpNotFound):
		return "订单不存在"
	case errors.Is(err, model.ErrPaymentMethodMismatch):
		return "不是微信支付订单"
	case errors.Is(err, model.ErrWeChatPayNotPaid):
		return "订单尚未支付成功"
	case errors.Is(err, model.ErrWeChatRefundExpired):
		return "已超过微信支付 365 天退款期限"
	case errors.Is(err, model.ErrWeChatRefundInProgress):
		return "退款处理中"
	case errors.Is(err, model.ErrWeChatAlreadyRefunded):
		return "订单已退款"
	case errors.Is(err, model.ErrWeChatRefundQuotaConsumed):
		return "该笔充值额度已被消费，无法全额退款"
	default:
		return err.Error()
	}
}

func handleWeChatRefundCreateError(c *gin.Context, tradeNo, refundNo string, err error) {
	if errors.Is(err, service.ErrWeChatPayRejected) {
		if rollbackErr := model.RollbackWeChatRefund(tradeNo); rollbackErr != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("微信拒绝退款后额度恢复失败 trade_no=%s refund_no=%s error=%q rollback=%q", tradeNo, refundNo, err.Error(), rollbackErr.Error()))
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "微信拒绝退款，额度恢复失败，请人工核对"})
			return
		}
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信明确拒绝退款 trade_no=%s refund_no=%s error=%q", tradeNo, refundNo, err.Error()))
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "向微信申请退款失败"})
		return
	}
	logger.LogError(c.Request.Context(), fmt.Sprintf("微信退款申请结果未知 trade_no=%s refund_no=%s error=%q", tradeNo, refundNo, err.Error()))
	c.JSON(http.StatusOK, gin.H{"success": false, "message": "退款请求结果待确认，额度将保持冻结"})
}

func reconcileWeChatRefund(tradeNo string, callerIP string) (string, error) {
	refundNo := service.WeChatRefundNo(tradeNo)
	refundInfo, err := service.QueryWeChatRefund(refundNo)
	if err != nil {
		return "", err
	}
	if refundInfo.OutTradeNo != tradeNo || refundInfo.OutRefundNo != refundNo {
		return "", errors.New("微信退款查询结果不匹配")
	}
	topUp, err := model.GetWeChatTopUpByTradeNo(tradeNo)
	if err != nil {
		return "", err
	}
	expectedCents := wechatAmountCents(topUp.Money)
	if refundInfo.Amount.Total != expectedCents || refundInfo.Amount.Refund != expectedCents {
		return "", errors.New("微信退款查询金额不匹配")
	}
	switch strings.ToUpper(refundInfo.Status) {
	case "SUCCESS":
		if _, err := model.CompleteWeChatRefund(tradeNo, callerIP); err != nil {
			return "", err
		}
		return "退款已成功完成", nil
	case "CLOSED":
		if err := model.RollbackWeChatRefund(tradeNo); err != nil {
			return "", err
		}
		return "微信退款已关闭，额度已恢复", nil
	case "ABNORMAL":
		return "微信退款异常，额度将保持冻结，请在微信商户平台处理", nil
	default:
		return "退款处理中，额度将保持冻结", nil
	}
}
