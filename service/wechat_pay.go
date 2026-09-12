package service

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
)

const wechatPayAPIBase = "https://api.mch.weixin.qq.com"

var (
	ErrWeChatPayNotFound      = errors.New("wechat pay resource not found")
	ErrWeChatPayRejected      = errors.New("wechat pay request rejected")
	ErrWeChatPayStatusUnknown = errors.New("wechat pay request status unknown")
)

type wechatPayAPIError struct {
	statusCode int
	message    string
}

func (e *wechatPayAPIError) Error() string {
	return e.message
}

func (e *wechatPayAPIError) Unwrap() error {
	switch {
	case e.statusCode == http.StatusNotFound:
		return ErrWeChatPayNotFound
	case e.statusCode >= 400 && e.statusCode < 500 && e.statusCode != http.StatusTooManyRequests:
		return ErrWeChatPayRejected
	default:
		return ErrWeChatPayStatusUnknown
	}
}

type WeChatJSAPIPrepay struct {
	PrepayID  string
	TimeStamp string
	NonceStr  string
	Package   string
	SignType  string
	PaySign   string
}

type wechatPayClient struct {
	httpClient        *http.Client
	privateKey        *rsa.PrivateKey
	platformPublicKey *rsa.PublicKey
}

var (
	wechatPayMu     sync.Mutex
	wechatPayCli    *wechatPayClient
	wechatPayCliErr error
	wechatPayCliVer int64
)

func loadWeChatPrivateKeyPEM() ([]byte, error) {
	if pemValue := setting.WeChatPayPrivateKeyPEMValue(); pemValue != "" {
		return []byte(pemValue), nil
	}
	path := setting.WeChatPayPrivateKeyPath()
	if path == "" {
		return nil, errors.New("wechat private key is empty")
	}
	keyPEM, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read wechat private key: %w", err)
	}
	return keyPEM, nil
}

func parseWeChatPrivateKey(keyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, errors.New("invalid wechat private key pem")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key, pkcs1Err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if pkcs1Err != nil {
			return nil, fmt.Errorf("parse wechat private key: %w", err)
		}
		parsed = key
	}
	privateKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("wechat private key is not rsa")
	}
	return privateKey, nil
}

func parseWeChatPlatformPublicKey(certificatePEM []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(certificatePEM)
	if block == nil {
		return nil, errors.New("invalid wechat platform certificate pem")
	}

	var publicKey any
	var err error
	switch block.Type {
	case "CERTIFICATE":
		certificate, certificateErr := x509.ParseCertificate(block.Bytes)
		if certificateErr != nil {
			return nil, fmt.Errorf("parse wechat platform certificate: %w", certificateErr)
		}
		publicKey = certificate.PublicKey
	case "PUBLIC KEY":
		publicKey, err = x509.ParsePKIXPublicKey(block.Bytes)
	case "RSA PUBLIC KEY":
		publicKey, err = x509.ParsePKCS1PublicKey(block.Bytes)
	default:
		return nil, errors.New("unsupported wechat platform certificate pem")
	}
	if err != nil {
		return nil, fmt.Errorf("parse wechat platform public key: %w", err)
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("wechat platform public key is not rsa")
	}
	return rsaPublicKey, nil
}

func (c *wechatPayClient) verifySignature(headers http.Header, body []byte) error {
	timestamp := strings.TrimSpace(headers.Get("Wechatpay-Timestamp"))
	nonce := strings.TrimSpace(headers.Get("Wechatpay-Nonce"))
	serialNo := strings.TrimSpace(headers.Get("Wechatpay-Serial"))
	signature := strings.TrimSpace(headers.Get("Wechatpay-Signature"))
	if timestamp == "" || nonce == "" || serialNo == "" || signature == "" {
		return errors.New("wechat signature headers are incomplete")
	}
	if serialNo != setting.WeChatPayPlatformSerialNo() {
		return errors.New("wechat platform certificate serial does not match")
	}

	signedAt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("invalid wechat signature timestamp")
	}
	now := time.Now()
	if signedAt < now.Add(-5*time.Minute).Unix() || signedAt > now.Add(5*time.Minute).Unix() {
		return errors.New("wechat signature timestamp is outside the allowed window")
	}

	rawSignature, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("decode wechat signature: %w", err)
	}
	message := timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	digest := sha256.Sum256([]byte(message))
	if err := rsa.VerifyPKCS1v15(c.platformPublicKey, crypto.SHA256, digest[:], rawSignature); err != nil {
		return fmt.Errorf("verify wechat signature: %w", err)
	}
	return nil
}

func VerifyWeChatNotification(headers http.Header, body []byte) error {
	client, err := getWeChatPayClient()
	if err != nil {
		return err
	}
	return client.verifySignature(headers, body)
}

func getWeChatPayClient() (*wechatPayClient, error) {
	version := setting.WeChatPayConfigVersion()
	wechatPayMu.Lock()
	defer wechatPayMu.Unlock()
	if wechatPayCli != nil && wechatPayCliErr == nil && wechatPayCliVer == version {
		return wechatPayCli, nil
	}
	keyPEM, err := loadWeChatPrivateKeyPEM()
	if err != nil {
		wechatPayCli = nil
		wechatPayCliErr = err
		wechatPayCliVer = version
		return nil, err
	}
	privateKey, err := parseWeChatPrivateKey(keyPEM)
	if err != nil {
		wechatPayCli = nil
		wechatPayCliErr = err
		wechatPayCliVer = version
		return nil, err
	}
	platformPublicKey, err := parseWeChatPlatformPublicKey([]byte(setting.WeChatPayPlatformCertificatePEMValue()))
	if err != nil {
		wechatPayCli = nil
		wechatPayCliErr = err
		wechatPayCliVer = version
		return nil, err
	}
	wechatPayCli = &wechatPayClient{
		httpClient:        &http.Client{Timeout: 15 * time.Second},
		privateKey:        privateKey,
		platformPublicKey: platformPublicKey,
	}
	wechatPayCliErr = nil
	wechatPayCliVer = version
	return wechatPayCli, nil
}

func wechatNonce() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("%x", buf)
}

func (c *wechatPayClient) signMessage(message string) (string, error) {
	hashed := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func (c *wechatPayClient) authorization(method, path, body string) (string, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := wechatNonce()
	message := method + "\n" + path + "\n" + timestamp + "\n" + nonce + "\n" + body + "\n"
	signature, err := c.signMessage(message)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",timestamp="%s",serial_no="%s",signature="%s"`,
		setting.WeChatPayMchID(),
		nonce,
		timestamp,
		setting.WeChatPaySerialNo(),
		signature,
	), nil
}

func (c *wechatPayClient) doJSON(method, path string, payload any) ([]byte, error) {
	var body string
	if payload != nil {
		raw, err := common.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = string(raw)
	}
	auth, err := c.authorization(method, path, body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, wechatPayAPIBase+path, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := c.verifySignature(resp.Header, respBody); err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, &wechatPayAPIError{
			statusCode: resp.StatusCode,
			message:    fmt.Sprintf("wechat pay api %s: %s", resp.Status, strings.TrimSpace(string(respBody))),
		}
	}
	return respBody, nil
}

func CreateWeChatJSAPIPrepay(tradeNo, description, openID string, amountCents int64) (*WeChatJSAPIPrepay, error) {
	client, err := getWeChatPayClient()
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"appid":        setting.WeChatPayAppID(),
		"mchid":        setting.WeChatPayMchID(),
		"description":  description,
		"out_trade_no": tradeNo,
		"notify_url":   setting.WeChatPayNotifyURL(),
		"amount": map[string]any{
			"total":    amountCents,
			"currency": "CNY",
		},
		"payer": map[string]any{
			"openid": openID,
		},
	}
	respBody, err := client.doJSON(http.MethodPost, "/v3/pay/transactions/jsapi", payload)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		PrepayID string `json:"prepay_id"`
	}
	if err := common.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	if parsed.PrepayID == "" {
		return nil, errors.New("wechat prepay_id missing")
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := wechatNonce()
	pack := "prepay_id=" + parsed.PrepayID
	message := setting.WeChatPayAppID() + "\n" + timestamp + "\n" + nonce + "\n" + pack + "\n"
	paySign, err := client.signMessage(message)
	if err != nil {
		return nil, err
	}
	return &WeChatJSAPIPrepay{
		PrepayID:  parsed.PrepayID,
		TimeStamp: timestamp,
		NonceStr:  nonce,
		Package:   pack,
		SignType:  "RSA",
		PaySign:   paySign,
	}, nil
}

func QueryWeChatTransaction(tradeNo string) (*WeChatTransactionNotification, error) {
	client, err := getWeChatPayClient()
	if err != nil {
		return nil, err
	}
	path := "/v3/pay/transactions/out-trade-no/" + url.PathEscape(tradeNo) + "?mchid=" + url.QueryEscape(setting.WeChatPayMchID())
	respBody, err := client.doJSON(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var parsed WeChatTransactionNotification
	if err := common.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func CreateWeChatRefund(tradeNo, refundNo, reason, transactionID string, refundCents, totalCents int64) error {
	client, err := getWeChatPayClient()
	if err != nil {
		return err
	}
	payload := map[string]any{
		"out_trade_no":  tradeNo,
		"out_refund_no": refundNo,
		"reason":        reason,
		"notify_url":    setting.WeChatPayRefundNotifyURL(),
		"amount": map[string]any{
			"refund":   refundCents,
			"total":    totalCents,
			"currency": "CNY",
		},
	}
	if transactionID != "" {
		payload["transaction_id"] = transactionID
	}
	_, err = client.doJSON(http.MethodPost, "/v3/refund/domestic/refunds", payload)
	return err
}

func WeChatRefundNo(tradeNo string) string {
	digest := sha256.Sum256([]byte(tradeNo))
	return "R" + fmt.Sprintf("%x", digest)[:63]
}

type WeChatRefundQuery struct {
	MchID       string `json:"mchid"`
	OutTradeNo  string `json:"out_trade_no"`
	OutRefundNo string `json:"out_refund_no"`
	RefundID    string `json:"refund_id"`
	Status      string `json:"status"`
	Amount      struct {
		Total  int64 `json:"total"`
		Refund int64 `json:"refund"`
	} `json:"amount"`
}

func QueryWeChatRefund(refundNo string) (*WeChatRefundQuery, error) {
	client, err := getWeChatPayClient()
	if err != nil {
		return nil, err
	}
	path := "/v3/refund/domestic/refunds/" + url.PathEscape(refundNo)
	respBody, err := client.doJSON(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var parsed WeChatRefundQuery
	if err := common.Unmarshal(respBody, &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

type WeChatResourceNotification struct {
	ID           string `json:"id"`
	CreateTime   string `json:"create_time"`
	EventType    string `json:"event_type"`
	Summary      string `json:"summary"`
	ResourceType string `json:"resource_type"`
	Resource     struct {
		Algorithm      string `json:"algorithm"`
		Ciphertext     string `json:"ciphertext"`
		AssociatedData string `json:"associated_data"`
		Nonce          string `json:"nonce"`
		OriginalType   string `json:"original_type"`
	} `json:"resource"`
}

type WeChatTransactionNotification struct {
	AppID         string `json:"appid"`
	MchID         string `json:"mchid"`
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id"`
	TradeType     string `json:"trade_type"`
	TradeState    string `json:"trade_state"`
	SuccessTime   string `json:"success_time"`
	Amount        struct {
		Total    int64  `json:"total"`
		Currency string `json:"currency"`
	} `json:"amount"`
}

type WeChatRefundNotification struct {
	MchID         string `json:"mchid"`
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id"`
	OutRefundNo   string `json:"out_refund_no"`
	RefundID      string `json:"refund_id"`
	RefundStatus  string `json:"refund_status"`
	SuccessTime   string `json:"success_time"`
	Amount        struct {
		Total  int64 `json:"total"`
		Refund int64 `json:"refund"`
	} `json:"amount"`
}

func decryptWeChatResource(ciphertext, associatedData, nonce, apiV3Key string) ([]byte, error) {
	if len(apiV3Key) != 32 {
		return nil, errors.New("invalid wechat api v3 key length")
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher([]byte(apiV3Key))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, errors.New("invalid wechat resource nonce length")
	}
	return gcm.Open(nil, []byte(nonce), raw, []byte(associatedData))
}

func ParseWeChatNotification(body []byte) (*WeChatResourceNotification, error) {
	var notification WeChatResourceNotification
	if err := common.Unmarshal(body, &notification); err != nil {
		return nil, err
	}
	if notification.Resource.Algorithm != "AEAD_AES_256_GCM" || notification.Resource.Ciphertext == "" || notification.Resource.Nonce == "" {
		return nil, errors.New("invalid wechat encrypted resource")
	}
	return &notification, nil
}

func DecryptWeChatTransaction(notification *WeChatResourceNotification) (*WeChatTransactionNotification, error) {
	plain, err := decryptWeChatResource(
		notification.Resource.Ciphertext,
		notification.Resource.AssociatedData,
		notification.Resource.Nonce,
		setting.WeChatPayAPIv3Key(),
	)
	if err != nil {
		return nil, err
	}
	var parsed WeChatTransactionNotification
	if err := common.Unmarshal(plain, &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func DecryptWeChatRefund(notification *WeChatResourceNotification) (*WeChatRefundNotification, error) {
	plain, err := decryptWeChatResource(
		notification.Resource.Ciphertext,
		notification.Resource.AssociatedData,
		notification.Resource.Nonce,
		setting.WeChatPayAPIv3Key(),
	)
	if err != nil {
		return nil, err
	}
	var parsed WeChatRefundNotification
	if err := common.Unmarshal(plain, &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

func WeChatPaySuccessJSON() []byte {
	raw, _ := common.Marshal(map[string]string{"code": "SUCCESS", "message": "成功"})
	return raw
}

func WeChatPayFailJSON(message string) []byte {
	if message == "" {
		message = "失败"
	}
	raw, _ := common.Marshal(map[string]string{"code": "FAIL", "message": message})
	return raw
}
