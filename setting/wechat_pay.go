package setting

import (
	"os"
	"strings"
	"sync/atomic"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

const (
	WeChatPayTradeTypeJSAPI = "JSAPI"
	WeChatPayColor          = "#07C160"
	WeChatPayMinTopUp       = 1
)

var (
	WeChatPayEnabled              bool
	WeChatPayAppIDValue           string
	WeChatPayMchIDValue           string
	WeChatPayAPIv3KeyValue        string
	WeChatPaySerialNoValue        string
	WeChatPayPrivateKeyPEM        string
	WeChatPayPrivateKeyFile       string
	WeChatPayNotifyURLValue       string
	WeChatPayRefundNotifyURLValue string
	WeChatPayMinTopUpValue        int64 = WeChatPayMinTopUp
	wechatPayConfigVersion        int64
)

func BumpWeChatPayConfig() {
	atomic.AddInt64(&wechatPayConfigVersion, 1)
}

func WeChatPayConfigVersion() int64 {
	return atomic.LoadInt64(&wechatPayConfigVersion)
}

func wechatPayEnvOrStored(env, stored string) string {
	if value := strings.TrimSpace(os.Getenv(env)); value != "" {
		return value
	}
	return strings.TrimSpace(stored)
}

func WeChatPayAppID() string {
	return wechatPayEnvOrStored("WECHAT_PAY_APPID", WeChatPayAppIDValue)
}

func WeChatPayMchID() string {
	return wechatPayEnvOrStored("WECHAT_PAY_MCHID", WeChatPayMchIDValue)
}

func WeChatPayAPIv3Key() string {
	return wechatPayEnvOrStored("WECHAT_PAY_APIV3_KEY", WeChatPayAPIv3KeyValue)
}

func WeChatPaySerialNo() string {
	return wechatPayEnvOrStored("WECHAT_PAY_SERIAL_NO", WeChatPaySerialNoValue)
}

func WeChatPayPrivateKeyPEMValue() string {
	return wechatPayEnvOrStored("WECHAT_PAY_PRIVATE_KEY", WeChatPayPrivateKeyPEM)
}

func WeChatPayPrivateKeyPath() string {
	return wechatPayEnvOrStored("WECHAT_PAY_PRIVATE_KEY_PATH", WeChatPayPrivateKeyFile)
}

func wechatPayCallbackBase() string {
	base := strings.TrimSpace(operation_setting.CustomCallbackAddress)
	if base == "" {
		base = strings.TrimSpace(system_setting.ServerAddress)
	}
	return strings.TrimRight(base, "/")
}

func WeChatPayNotifyURL() string {
	explicit := wechatPayEnvOrStored("WECHAT_PAY_NOTIFY_URL", WeChatPayNotifyURLValue)
	if explicit != "" {
		return strings.TrimRight(explicit, "/")
	}
	base := wechatPayCallbackBase()
	if base == "" {
		return ""
	}
	return base + "/api/user/wechat/notify"
}

func WeChatPayRefundNotifyURL() string {
	explicit := wechatPayEnvOrStored("WECHAT_PAY_REFUND_NOTIFY_URL", WeChatPayRefundNotifyURLValue)
	if explicit != "" {
		return strings.TrimRight(explicit, "/")
	}
	notifyURL := WeChatPayNotifyURL()
	if notifyURL == "" {
		return ""
	}
	return strings.TrimSuffix(notifyURL, "/notify") + "/refund-notify"
}

func WeChatPayMinTopup() int64 {
	if env := strings.TrimSpace(os.Getenv("WECHAT_PAY_MIN_TOPUP")); env != "" {
		minTopup := int64(common.GetEnvOrDefault("WECHAT_PAY_MIN_TOPUP", WeChatPayMinTopUp))
		if minTopup < 1 {
			return 1
		}
		return minTopup
	}
	if WeChatPayMinTopUpValue < 1 {
		return 1
	}
	return WeChatPayMinTopUpValue
}

func wechatPayEnabled() bool {
	if _, set := os.LookupEnv("WECHAT_PAY_ENABLED"); set {
		return common.GetEnvOrDefaultBool("WECHAT_PAY_ENABLED", false)
	}
	return WeChatPayEnabled
}

func IsWeChatPayConfigured() bool {
	if WeChatPayAppID() == "" || WeChatPayMchID() == "" || WeChatPayAPIv3Key() == "" || WeChatPaySerialNo() == "" {
		return false
	}
	if WeChatPayPrivateKeyPEMValue() == "" {
		path := WeChatPayPrivateKeyPath()
		if path == "" {
			return false
		}
		if _, err := os.Stat(path); err != nil {
			return false
		}
	}
	return WeChatPayNotifyURL() != ""
}

func IsWeChatPayEnabled() bool {
	return wechatPayEnabled() && IsWeChatPayConfigured()
}
