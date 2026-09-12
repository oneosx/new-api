package setting

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func resetWeChatPaySettings() {
	WeChatPayEnabled = false
	WeChatPayAppIDValue = ""
	WeChatPayMchIDValue = ""
	WeChatPayAPIv3KeyValue = ""
	WeChatPaySerialNoValue = ""
	WeChatPayPrivateKeyPEM = ""
	WeChatPayPrivateKeyFile = ""
	WeChatPayPlatformCertificatePEM = ""
	WeChatPayPlatformSerialNoValue = ""
	WeChatPayNotifyURLValue = ""
	WeChatPayRefundNotifyURLValue = ""
	WeChatPayMinTopUpValue = WeChatPayMinTopUp
}

func TestIsWeChatPayEnabledRequiresEnvAndCert(t *testing.T) {
	t.Cleanup(resetWeChatPaySettings)
	t.Setenv("WECHAT_PAY_ENABLED", "true")
	t.Setenv("WECHAT_PAY_APPID", "wxapp")
	t.Setenv("WECHAT_PAY_MCHID", "1234567890")
	t.Setenv("WECHAT_PAY_APIV3_KEY", "1234567890abcdef1234567890abcdef")
	t.Setenv("WECHAT_PAY_SERIAL_NO", "ABC")
	t.Setenv("WECHAT_PAY_NOTIFY_URL", "https://example.com/api/user/wechat/notify")
	t.Setenv("WECHAT_PAY_PLATFORM_CERTIFICATE", "-----BEGIN CERTIFICATE-----\ndummy\n-----END CERTIFICATE-----")
	t.Setenv("WECHAT_PAY_PLATFORM_SERIAL_NO", "PLATFORM-ABC")

	missing := filepath.Join(t.TempDir(), "missing.pem")
	t.Setenv("WECHAT_PAY_PRIVATE_KEY_PATH", missing)
	require.False(t, IsWeChatPayEnabled())

	keyPath := filepath.Join(t.TempDir(), "apiclient_key.pem")
	require.NoError(t, os.WriteFile(keyPath, []byte("dummy"), 0o600))
	t.Setenv("WECHAT_PAY_PRIVATE_KEY_PATH", keyPath)
	require.True(t, IsWeChatPayEnabled())

	t.Setenv("WECHAT_PAY_ENABLED", "false")
	require.False(t, IsWeChatPayEnabled())
}

func TestIsWeChatPayEnabledUsesAdminOptions(t *testing.T) {
	t.Cleanup(resetWeChatPaySettings)

	WeChatPayEnabled = true
	WeChatPayAppIDValue = "wxapp"
	WeChatPayMchIDValue = "1234567890"
	WeChatPayAPIv3KeyValue = "1234567890abcdef1234567890abcdef"
	WeChatPaySerialNoValue = "ABC"
	WeChatPayPrivateKeyPEM = "-----BEGIN PRIVATE KEY-----\ndummy\n-----END PRIVATE KEY-----"
	WeChatPayPlatformCertificatePEM = "-----BEGIN CERTIFICATE-----\ndummy\n-----END CERTIFICATE-----"
	WeChatPayPlatformSerialNoValue = "PLATFORM-ABC"
	WeChatPayNotifyURLValue = "https://example.com/api/user/wechat/notify"

	require.True(t, IsWeChatPayEnabled())
	require.Equal(t, "wxapp", WeChatPayAppID())
	require.Equal(t, int64(1), WeChatPayMinTopup())
}

func TestWeChatPayEnvOverridesAdminOptions(t *testing.T) {
	t.Cleanup(resetWeChatPaySettings)

	WeChatPayEnabled = true
	WeChatPayAppIDValue = "stored-app"
	WeChatPayMinTopUpValue = 5
	t.Setenv("WECHAT_PAY_APPID", "env-app")
	t.Setenv("WECHAT_PAY_MIN_TOPUP", "9")
	t.Setenv("WECHAT_PAY_ENABLED", "false")

	require.Equal(t, "env-app", WeChatPayAppID())
	require.Equal(t, int64(9), WeChatPayMinTopup())
	require.False(t, wechatPayEnabled())
}
