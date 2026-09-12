package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/require"
)

func TestDecryptWeChatTransaction(t *testing.T) {
	key := "1234567890abcdef1234567890abcdef"
	t.Setenv("WECHAT_PAY_APIV3_KEY", key)

	plain := []byte(`{"appid":"wxapp","mchid":"1234567890","out_trade_no":"WXTESTPAY","transaction_id":"wx-txn-1","trade_state":"SUCCESS","amount":{"total":999,"currency":"CNY"}}`)
	nonce := make([]byte, 12)
	_, err := rand.Read(nonce)
	require.NoError(t, err)

	block, err := aes.NewCipher([]byte(key))
	require.NoError(t, err)
	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)
	ciphertext := gcm.Seal(nil, nonce, plain, []byte("transaction"))

	notification := &WeChatResourceNotification{
		EventType: "TRANSACTION.SUCCESS",
	}
	notification.Resource.Algorithm = "AEAD_AES_256_GCM"
	notification.Resource.AssociatedData = "transaction"
	notification.Resource.Nonce = string(nonce)
	notification.Resource.Ciphertext = base64.StdEncoding.EncodeToString(ciphertext)

	parsed, err := DecryptWeChatTransaction(notification)
	require.NoError(t, err)
	require.Equal(t, "WXTESTPAY", parsed.OutTradeNo)
	require.Equal(t, "wx-txn-1", parsed.TransactionID)
	require.Equal(t, int64(999), parsed.Amount.Total)
	require.Equal(t, "SUCCESS", parsed.TradeState)
}

func TestWeChatPayNotifyJSON(t *testing.T) {
	var success map[string]string
	require.NoError(t, common.Unmarshal(WeChatPaySuccessJSON(), &success))
	require.Equal(t, "SUCCESS", success["code"])

	var fail map[string]string
	require.NoError(t, common.Unmarshal(WeChatPayFailJSON("入账失败"), &fail))
	require.Equal(t, "FAIL", fail["code"])
	require.Equal(t, "入账失败", fail["message"])
}
