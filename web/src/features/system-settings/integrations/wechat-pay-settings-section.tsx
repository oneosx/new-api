/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

import { SettingsSwitchField } from '../components/settings-form-layout'

export interface WeChatPaySettingsValues {
  WeChatPayEnabled: boolean
  WeChatPayAppID: string
  WeChatPayMchID: string
  WeChatPayAPIv3Key: string
  WeChatPaySerialNo: string
  WeChatPayPrivateKey: string
  WeChatPayPrivateKeyPath: string
  WeChatPayNotifyURL: string
  WeChatPayRefundNotifyURL: string
  WeChatPayMinTopUp: number
}

interface Props {
  values: WeChatPaySettingsValues
  onValueChange: <K extends keyof WeChatPaySettingsValues>(
    key: K,
    value: WeChatPaySettingsValues[K]
  ) => void
}

export function WeChatPaySettingsSection({ values, onValueChange }: Props) {
  const { t } = useTranslation()

  return (
    <div className='space-y-4 pt-4'>
      <div>
        <h3 className='text-lg font-medium'>{t('Official WeChat Pay')}</h3>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Official WeChat JSAPI top-up. This is independent from Epay WeChat Pay.'
          )}
        </p>
      </div>

      <Alert>
        <AlertTitle>{t('Webhook Configuration:')}</AlertTitle>
        <AlertDescription>
          <ul className='list-inside list-disc space-y-1 text-xs'>
            <li>
              {t('Payment notify URL:')}{' '}
              <code className='bg-muted rounded px-1 py-0.5'>
                {'<ServerAddress>/api/user/wechat/notify'}
              </code>
            </li>
            <li>
              {t('Refund notify URL:')}{' '}
              <code className='bg-muted rounded px-1 py-0.5'>
                {'<ServerAddress>/api/user/wechat/refund-notify'}
              </code>
            </li>
            <li>
              {t(
                'JSAPI requires the payer openid from the same WeChat AppID used for payment. Leave notify URLs blank to use the server address or custom callback origin.'
              )}
            </li>
            <li>
              {t(
                'Environment variables override these fields when set. Secrets are not shown after saving; leave them blank unless rotating.'
              )}
            </li>
          </ul>
        </AlertDescription>
      </Alert>

      <SettingsSwitchField
        checked={values.WeChatPayEnabled}
        onCheckedChange={(v) => onValueChange('WeChatPayEnabled', v)}
        label={t('Enable official WeChat Pay')}
        description={t(
          'Requires AppID, merchant ID, APIv3 key, certificate serial number, and a merchant private key.'
        )}
        className='py-0'
      />

      <div className='grid gap-4 sm:grid-cols-2'>
        <div className='grid gap-1.5'>
          <Label>{t('AppID')}</Label>
          <Input
            value={values.WeChatPayAppID}
            onChange={(event) =>
              onValueChange('WeChatPayAppID', event.target.value)
            }
            placeholder='wx...'
            autoComplete='off'
          />
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Merchant ID')}</Label>
          <Input
            value={values.WeChatPayMchID}
            onChange={(event) =>
              onValueChange('WeChatPayMchID', event.target.value)
            }
            placeholder='1600000000'
            autoComplete='off'
          />
        </div>
      </div>

      <div className='grid gap-4 sm:grid-cols-2'>
        <div className='grid gap-1.5'>
          <Label>{t('APIv3 key')}</Label>
          <Input
            type='password'
            value={values.WeChatPayAPIv3Key}
            onChange={(event) =>
              onValueChange('WeChatPayAPIv3Key', event.target.value)
            }
            placeholder={t('Leave blank unless updating')}
            autoComplete='new-password'
          />
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Certificate serial number')}</Label>
          <Input
            value={values.WeChatPaySerialNo}
            onChange={(event) =>
              onValueChange('WeChatPaySerialNo', event.target.value)
            }
            autoComplete='off'
          />
        </div>
      </div>

      <div className='grid gap-1.5'>
        <Label>{t('Merchant private key PEM')}</Label>
        <Textarea
          rows={6}
          value={values.WeChatPayPrivateKey}
          onChange={(event) =>
            onValueChange('WeChatPayPrivateKey', event.target.value)
          }
          placeholder={t(
            'Paste apiclient_key.pem here, or leave blank and use the file path below.'
          )}
          autoComplete='off'
        />
      </div>

      <div className='grid gap-1.5'>
        <Label>{t('Private key file path')}</Label>
        <Input
          value={values.WeChatPayPrivateKeyPath}
          onChange={(event) =>
            onValueChange('WeChatPayPrivateKeyPath', event.target.value)
          }
          placeholder='/data/certs/wechat/apiclient_key.pem'
          autoComplete='off'
        />
        <p className='text-muted-foreground text-xs'>
          {t(
            'Used only when the PEM field is empty. Mount the merchant certificate into the container if you use a file path.'
          )}
        </p>
      </div>

      <div className='grid gap-4 sm:grid-cols-2'>
        <div className='grid gap-1.5'>
          <Label>{t('Payment notify URL')}</Label>
          <Input
            value={values.WeChatPayNotifyURL}
            onChange={(event) =>
              onValueChange('WeChatPayNotifyURL', event.target.value)
            }
            placeholder='https://example.com/api/user/wechat/notify'
            autoComplete='off'
          />
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Refund notify URL')}</Label>
          <Input
            value={values.WeChatPayRefundNotifyURL}
            onChange={(event) =>
              onValueChange('WeChatPayRefundNotifyURL', event.target.value)
            }
            placeholder='https://example.com/api/user/wechat/refund-notify'
            autoComplete='off'
          />
        </div>
      </div>

      <div className='grid gap-1.5 sm:max-w-xs'>
        <Label>{t('Minimum top-up')}</Label>
        <Input
          type='number'
          min={1}
          value={values.WeChatPayMinTopUp}
          onChange={(event) =>
            onValueChange(
              'WeChatPayMinTopUp',
              Number.parseInt(event.target.value, 10) || 1
            )
          }
        />
      </div>
    </div>
  )
}
