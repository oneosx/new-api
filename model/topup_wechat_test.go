package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertWeChatTopUp(t *testing.T, userID int, tradeNo string, amount int64, money float64, status string) *TopUp {
	t.Helper()
	topUp := &TopUp{
		UserId:          userID,
		Amount:          amount,
		Money:           money,
		TradeNo:         tradeNo,
		PaymentMethod:   PaymentMethodWeChatJSAPI,
		PaymentProvider: PaymentProviderWeChat,
		Status:          status,
		CreateTime:      time.Now().Unix(),
	}
	if status == common.TopUpStatusSuccess {
		topUp.CompleteTime = time.Now().Unix()
	}
	require.NoError(t, topUp.Insert())
	return topUp
}

func TestRechargeWeChatCreditsPendingOrderOnce(t *testing.T) {
	truncateTables(t)

	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	user := insertUserForPaymentGuardTest(t, 801, 0)
	insertWeChatTopUp(t, user.Id, "WXTESTPAY", 2, 9.99, common.TopUpStatusPending)

	alreadyDone, err := RechargeWeChat("WXTESTPAY", "wx-txn-1", 999, "127.0.0.1")
	require.NoError(t, err)
	assert.False(t, alreadyDone)

	reloaded := GetTopUpByTradeNo("WXTESTPAY")
	require.NotNil(t, reloaded)
	assert.Equal(t, common.TopUpStatusSuccess, reloaded.Status)
	assert.Equal(t, PaymentMethodWeChatJSAPI, reloaded.PaymentMethod)
	assert.Equal(t, 1_000_000, getUserQuotaForPaymentGuardTest(t, user.Id))

	alreadyDone, err = RechargeWeChat("WXTESTPAY", "wx-txn-1", 999, "127.0.0.1")
	require.NoError(t, err)
	assert.True(t, alreadyDone)
	assert.Equal(t, 1_000_000, getUserQuotaForPaymentGuardTest(t, user.Id))
}

func TestRechargeWeChatRejectsMismatchedProviderAmountAndStatus(t *testing.T) {
	truncateTables(t)

	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	user := insertUserForPaymentGuardTest(t, 802, 7)

	t.Run("order from another payment provider", func(t *testing.T) {
		insertTopUpForPaymentGuardTest(t, "WXTESTSTRIPE", user.Id, PaymentProviderStripe)
		_, err := RechargeWeChat("WXTESTSTRIPE", "wx-txn", 999, "127.0.0.1")
		assert.ErrorIs(t, err, ErrPaymentMethodMismatch)
		assert.Equal(t, 7, getUserQuotaForPaymentGuardTest(t, user.Id))
	})

	t.Run("paid amount mismatch", func(t *testing.T) {
		insertWeChatTopUp(t, user.Id, "WXTESTAMOUNT", 2, 9.99, common.TopUpStatusPending)
		_, err := RechargeWeChat("WXTESTAMOUNT", "wx-txn", 1000, "127.0.0.1")
		assert.ErrorIs(t, err, ErrWeChatAmountMismatch)
		assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, "WXTESTAMOUNT"))
		assert.Equal(t, 7, getUserQuotaForPaymentGuardTest(t, user.Id))
	})

	t.Run("order that is not pending", func(t *testing.T) {
		insertWeChatTopUp(t, user.Id, "WXTESTEXPIRED", 2, 9.99, common.TopUpStatusExpired)
		_, err := RechargeWeChat("WXTESTEXPIRED", "wx-txn", 999, "127.0.0.1")
		assert.ErrorIs(t, err, ErrTopUpStatusInvalid)
		assert.Equal(t, 7, getUserQuotaForPaymentGuardTest(t, user.Id))
	})
}

func TestWeChatFullRefundHoldsUnusedQuotaAndCompletes(t *testing.T) {
	truncateTables(t)

	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	user := insertUserForPaymentGuardTest(t, 803, 0)
	insertWeChatTopUp(t, user.Id, "WXTESTREFUND", 2, 9.99, common.TopUpStatusPending)
	_, err := RechargeWeChat("WXTESTREFUND", "wx-txn-2", 999, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, 1_000_000, getUserQuotaForPaymentGuardTest(t, user.Id))

	topUp, held, err := PrepareWeChatFullRefund("WXTESTREFUND")
	require.NoError(t, err)
	require.NotNil(t, topUp)
	assert.Equal(t, 1_000_000, held)
	assert.Equal(t, common.TopUpStatusRefunding, getTopUpStatusForPaymentGuardTest(t, "WXTESTREFUND"))
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, user.Id))

	alreadyDone, err := CompleteWeChatRefund("WXTESTREFUND", "127.0.0.1")
	require.NoError(t, err)
	assert.False(t, alreadyDone)
	assert.Equal(t, common.TopUpStatusRefunded, getTopUpStatusForPaymentGuardTest(t, "WXTESTREFUND"))
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, user.Id))

	alreadyDone, err = CompleteWeChatRefund("WXTESTREFUND", "127.0.0.1")
	require.NoError(t, err)
	assert.True(t, alreadyDone)
}

func TestWeChatFullRefundRejectsConsumedQuotaAndRollsBackHold(t *testing.T) {
	truncateTables(t)

	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	user := insertUserForPaymentGuardTest(t, 804, 0)
	insertWeChatTopUp(t, user.Id, "WXTESTCONSUMED", 2, 9.99, common.TopUpStatusPending)
	_, err := RechargeWeChat("WXTESTCONSUMED", "wx-txn-3", 999, "127.0.0.1")
	require.NoError(t, err)

	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("quota", 100).Error)
	_, _, err = PrepareWeChatFullRefund("WXTESTCONSUMED")
	require.ErrorIs(t, err, ErrWeChatRefundQuotaConsumed)
	assert.Equal(t, common.TopUpStatusSuccess, getTopUpStatusForPaymentGuardTest(t, "WXTESTCONSUMED"))
	assert.Equal(t, 100, getUserQuotaForPaymentGuardTest(t, user.Id))

	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("quota", 1_000_000).Error)
	_, _, err = PrepareWeChatFullRefund("WXTESTCONSUMED")
	require.NoError(t, err)
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, user.Id))

	require.NoError(t, RollbackWeChatRefund("WXTESTCONSUMED"))
	assert.Equal(t, common.TopUpStatusSuccess, getTopUpStatusForPaymentGuardTest(t, "WXTESTCONSUMED"))
	assert.Equal(t, 1_000_000, getUserQuotaForPaymentGuardTest(t, user.Id))
}

func TestManualCompleteTopUpRejectsWeChatOrders(t *testing.T) {
	truncateTables(t)

	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 500000
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	user := insertUserForPaymentGuardTest(t, 805, 0)
	insertWeChatTopUp(t, user.Id, "WXTESTMANUAL", 2, 9.99, common.TopUpStatusPending)

	err := ManualCompleteTopUp("WXTESTMANUAL", "127.0.0.1")
	require.Error(t, err)
	assert.Equal(t, common.TopUpStatusPending, getTopUpStatusForPaymentGuardTest(t, "WXTESTMANUAL"))
	assert.Equal(t, 0, getUserQuotaForPaymentGuardTest(t, user.Id))
}
