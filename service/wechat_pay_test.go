package service

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"

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
	require.Equal(t, "CNY", parsed.Amount.Currency)
}

func TestDecryptWeChatResourceRejectsInvalidNonce(t *testing.T) {
	_, err := decryptWeChatResource(
		base64.StdEncoding.EncodeToString([]byte("ciphertext")),
		"transaction",
		"short",
		"1234567890abcdef1234567890abcdef",
	)
	require.Error(t, err)
}

func TestWeChatRefundNoIsStableAndWithinLimit(t *testing.T) {
	refundNo := WeChatRefundNo("USR1WXABC123")
	require.Equal(t, refundNo, WeChatRefundNo("USR1WXABC123"))
	require.NotEqual(t, refundNo, WeChatRefundNo("USR1WXABC124"))
	require.Len(t, refundNo, 64)
}

func TestVerifyWeChatSignatureRejectsExpiredAndInvalidSerial(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	t.Setenv("WECHAT_PAY_PLATFORM_SERIAL_NO", "platform-serial")

	body := []byte(`{"id":"event-1"}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := "nonce"
	message := timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	require.NoError(t, err)

	headers := http.Header{
		"Wechatpay-Timestamp": {timestamp},
		"Wechatpay-Nonce":     {nonce},
		"Wechatpay-Serial":    {"platform-serial"},
		"Wechatpay-Signature": {base64.StdEncoding.EncodeToString(signature)},
	}
	client := &wechatPayClient{platformPublicKey: &privateKey.PublicKey}
	require.NoError(t, client.verifySignature(headers, body))

	headers.Set("Wechatpay-Serial", "unexpected-serial")
	require.Error(t, client.verifySignature(headers, body))
	headers.Set("Wechatpay-Serial", "platform-serial")
	headers.Set("Wechatpay-Timestamp", strconv.FormatInt(time.Now().Add(-6*time.Minute).Unix(), 10))
	require.Error(t, client.verifySignature(headers, body))
}

func TestWeChatPayAPIErrorClassification(t *testing.T) {
	rejected := &wechatPayAPIError{statusCode: http.StatusBadRequest, message: "bad request"}
	require.ErrorIs(t, rejected, ErrWeChatPayRejected)
	assert.False(t, errors.Is(rejected, ErrWeChatPayStatusUnknown))

	notFound := &wechatPayAPIError{statusCode: http.StatusNotFound, message: "missing"}
	require.ErrorIs(t, notFound, ErrWeChatPayNotFound)

	unknown := &wechatPayAPIError{statusCode: http.StatusInternalServerError, message: "server error"}
	require.ErrorIs(t, unknown, ErrWeChatPayStatusUnknown)

	rateLimited := &wechatPayAPIError{statusCode: http.StatusTooManyRequests, message: "slow down"}
	require.ErrorIs(t, rateLimited, ErrWeChatPayStatusUnknown)
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
