package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	PaymentMethodWeChatJSAPI = "wechat_jsapi"
	PaymentProviderWeChat    = "wechat_jsapi"

	WeChatRefundWindow = 365 * 24 * time.Hour
)

var (
	ErrWeChatPayNotPaid          = errors.New("wechat order is not paid")
	ErrWeChatRefundExpired       = errors.New("wechat refund window expired")
	ErrWeChatRefundInProgress    = errors.New("wechat refund already in progress")
	ErrWeChatAlreadyRefunded     = errors.New("wechat order already refunded")
	ErrWeChatRefundQuotaConsumed = errors.New("wechat topup quota already consumed")
	ErrWeChatAmountMismatch      = errors.New("wechat paid amount mismatch")
)

func wechatTradeNoColumn() string {
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		return `"trade_no"`
	}
	return "`trade_no`"
}

func wechatTopUpQuota(topUp *TopUp) (int, error) {
	quota, err := common.WalletQuotaFromDecimalStrict(
		decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)),
	)
	if err != nil || quota <= 0 {
		return 0, ErrInvalidTopUpQuota
	}
	return quota, nil
}

func GetWeChatTopUpByTradeNo(tradeNo string) (*TopUp, error) {
	if tradeNo == "" {
		return nil, ErrTopUpNotFound
	}
	topUp := &TopUp{}
	if err := DB.Where(wechatTradeNoColumn()+" = ?", tradeNo).First(topUp).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTopUpNotFound
		}
		return nil, err
	}
	if topUp.PaymentProvider != PaymentProviderWeChat {
		return nil, ErrPaymentMethodMismatch
	}
	return topUp, nil
}

func GetUserWeChatTopUp(userId int, tradeNo string) (*TopUp, error) {
	topUp, err := GetWeChatTopUpByTradeNo(tradeNo)
	if err != nil {
		return nil, err
	}
	if topUp.UserId != userId {
		return nil, ErrTopUpNotFound
	}
	return topUp, nil
}

func RechargeWeChat(tradeNo string, wechatTransactionId string, paidCents int64, callerIp string) (alreadyDone bool, err error) {
	if tradeNo == "" {
		return false, errors.New("未提供支付单号")
	}

	var quotaToAdd int
	topUp := &TopUp{}
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where(wechatTradeNoColumn()+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if topUp.PaymentProvider != PaymentProviderWeChat {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status == common.TopUpStatusSuccess || topUp.Status == common.TopUpStatusRefunding || topUp.Status == common.TopUpStatusRefunded {
			alreadyDone = true
			return nil
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}
		expectedCents := decimal.NewFromFloat(topUp.Money).Mul(decimal.NewFromInt(100)).Round(0).IntPart()
		if paidCents != expectedCents {
			return ErrWeChatAmountMismatch
		}
		var quotaErr error
		quotaToAdd, quotaErr = wechatTopUpQuota(topUp)
		if quotaErr != nil {
			return quotaErr
		}
		if wechatTransactionId != "" {
			topUp.PaymentMethod = PaymentMethodWeChatJSAPI
		}
		topUp.CompleteTime = common.GetTimestamp()
		topUp.Status = common.TopUpStatusSuccess
		if err := tx.Save(topUp).Error; err != nil {
			return err
		}
		return creditTopUpQuota(tx, topUp.UserId, quotaToAdd, nil)
	})
	if err != nil {
		if !errors.Is(err, ErrTopUpNotFound) && !errors.Is(err, ErrPaymentMethodMismatch) && !errors.Is(err, ErrTopUpStatusInvalid) {
			common.SysError("wechat topup failed: " + err.Error())
		}
		return false, err
	}
	if alreadyDone {
		return true, nil
	}
	syncCreditUserQuotaCache(topUp.UserId, quotaToAdd, "wechat topup")
	common.SysLog(fmt.Sprintf("微信支付充值成功 trade_no=%s user_id=%d quota_to_add=%d money=%.2f transaction_id=%s", topUp.TradeNo, topUp.UserId, quotaToAdd, topUp.Money, wechatTransactionId))
	RecordTopupLog(topUp.UserId, fmt.Sprintf("使用微信支付充值成功，充值金额: %v，支付金额：%f", logger.LogQuota(quotaToAdd), topUp.Money), callerIp, PaymentMethodWeChatJSAPI, PaymentProviderWeChat)
	return false, nil
}

func PrepareWeChatFullRefund(tradeNo string) (*TopUp, int, error) {
	if tradeNo == "" {
		return nil, 0, errors.New("未提供支付单号")
	}

	var quotaToHold int
	topUp := &TopUp{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where(wechatTradeNoColumn()+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if topUp.PaymentProvider != PaymentProviderWeChat {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status == common.TopUpStatusRefunding {
			return ErrWeChatRefundInProgress
		}
		if topUp.Status == common.TopUpStatusRefunded {
			return ErrWeChatAlreadyRefunded
		}
		if topUp.Status != common.TopUpStatusSuccess {
			return ErrWeChatPayNotPaid
		}
		if topUp.CompleteTime > 0 {
			paidAt := time.Unix(topUp.CompleteTime, 0)
			if time.Since(paidAt) > WeChatRefundWindow {
				return ErrWeChatRefundExpired
			}
		}
		var quotaErr error
		quotaToHold, quotaErr = wechatTopUpQuota(topUp)
		if quotaErr != nil {
			return quotaErr
		}
		var user User
		if err := lockForUpdate(tx).Select("quota").Where("id = ?", topUp.UserId).First(&user).Error; err != nil {
			return err
		}
		if user.Quota < quotaToHold {
			return ErrWeChatRefundQuotaConsumed
		}
		result := tx.Model(&User{}).
			Where("id = ? AND quota >= ?", topUp.UserId, quotaToHold).
			Update("quota", gorm.Expr("quota - ?", quotaToHold))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrWeChatRefundQuotaConsumed
		}
		topUp.Status = common.TopUpStatusRefunding
		return tx.Save(topUp).Error
	})
	if err != nil {
		return nil, 0, err
	}
	if err := cacheDecrUserQuota(topUp.UserId, int64(quotaToHold)); err != nil {
		common.SysLog("failed to sync wechat refund hold to user quota cache: " + err.Error())
	}
	return topUp, quotaToHold, nil
}

func CompleteWeChatRefund(tradeNo string, callerIp string) (alreadyDone bool, err error) {
	if tradeNo == "" {
		return false, errors.New("未提供支付单号")
	}

	var quota int
	topUp := &TopUp{}
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where(wechatTradeNoColumn()+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if topUp.PaymentProvider != PaymentProviderWeChat {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status == common.TopUpStatusRefunded {
			alreadyDone = true
			return nil
		}
		if topUp.Status != common.TopUpStatusRefunding {
			return ErrTopUpStatusInvalid
		}
		var quotaErr error
		quota, quotaErr = wechatTopUpQuota(topUp)
		if quotaErr != nil {
			return quotaErr
		}
		topUp.Status = common.TopUpStatusRefunded
		return tx.Save(topUp).Error
	})
	if err != nil {
		return false, err
	}
	if alreadyDone {
		return true, nil
	}
	common.SysLog(fmt.Sprintf("微信支付退款成功 trade_no=%s user_id=%d quota=%d money=%.2f", topUp.TradeNo, topUp.UserId, quota, topUp.Money))
	RecordTopupLog(topUp.UserId, fmt.Sprintf("微信支付全额退款成功，退回金额: %v，支付金额：%f", logger.LogQuota(quota), topUp.Money), callerIp, PaymentMethodWeChatJSAPI, PaymentProviderWeChat)
	return false, nil
}

func RollbackWeChatRefund(tradeNo string) error {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	var quotaToRestore int
	topUp := &TopUp{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where(wechatTradeNoColumn()+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if topUp.PaymentProvider != PaymentProviderWeChat {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status == common.TopUpStatusSuccess {
			return nil
		}
		if topUp.Status != common.TopUpStatusRefunding {
			return ErrTopUpStatusInvalid
		}
		var quotaErr error
		quotaToRestore, quotaErr = wechatTopUpQuota(topUp)
		if quotaErr != nil {
			return quotaErr
		}
		if err := creditTopUpQuota(tx, topUp.UserId, quotaToRestore, nil); err != nil {
			return err
		}
		topUp.Status = common.TopUpStatusSuccess
		return tx.Save(topUp).Error
	})
	if err != nil {
		return err
	}
	syncCreditUserQuotaCache(topUp.UserId, quotaToRestore, "wechat refund rollback")
	return nil
}
