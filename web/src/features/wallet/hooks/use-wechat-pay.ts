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
import i18next from 'i18next'
import { useState, useCallback } from 'react'
import { toast } from 'sonner'

import { requestWeChatPay, getWeChatPayOrder, isApiSuccess } from '../api'
import type { UserWalletData, WeChatPayJSAPIData } from '../types'

interface WeixinJSBridgeResponse {
  err_msg: string
}

interface WeixinJSBridgeInstance {
  invoke: (
    action: string,
    params: Record<string, unknown>,
    callback: (res: WeixinJSBridgeResponse) => void
  ) => void
}

declare global {
  interface Window {
    WeixinJSBridge?: WeixinJSBridgeInstance
  }
}

export function isWeChatBrowser(): boolean {
  if (typeof window === 'undefined' || !navigator?.userAgent) {
    return false
  }
  return /MicroMessenger/i.test(navigator.userAgent)
}

function getWeChatErrorMessage(
  message: string | undefined,
  data: unknown
): string {
  if (typeof data === 'string' && data.trim()) {
    return data
  }
  return message || i18next.t('Payment request failed')
}

function waitForWeixinJSBridge(
  timeoutMs = 8000
): Promise<WeixinJSBridgeInstance | null> {
  if (typeof window === 'undefined') {
    return Promise.resolve(null)
  }
  if (
    window.WeixinJSBridge &&
    typeof window.WeixinJSBridge.invoke === 'function'
  ) {
    return Promise.resolve(window.WeixinJSBridge)
  }
  return new Promise((resolve) => {
    const timer = window.setTimeout(() => {
      document.removeEventListener('WeixinJSBridgeReady', onReady)
      resolve(window.WeixinJSBridge ?? null)
    }, timeoutMs)
    const onReady = () => {
      window.clearTimeout(timer)
      document.removeEventListener('WeixinJSBridgeReady', onReady)
      resolve(window.WeixinJSBridge ?? null)
    }
    document.addEventListener('WeixinJSBridgeReady', onReady, false)
  })
}

function invokeWeChatJSAPI(data: WeChatPayJSAPIData): Promise<boolean> {
  return waitForWeixinJSBridge().then((bridge) => {
    if (!bridge || typeof bridge.invoke !== 'function') {
      toast.error(
        i18next.t(
          'Official WeChat Pay currently requires opening the wallet inside WeChat.'
        )
      )
      return false
    }
    return new Promise<boolean>((resolve) => {
      bridge.invoke(
        'getBrandWCPayRequest',
        {
          appId: data.app_id,
          timeStamp: data.timeStamp,
          nonceStr: data.nonceStr,
          package: data.package,
          signType: data.signType,
          paySign: data.paySign,
        },
        (res) => {
          if (res.err_msg === 'get_brand_wcpay_request:ok') {
            resolve(true)
          } else if (res.err_msg === 'get_brand_wcpay_request:cancel') {
            toast.info(i18next.t('Payment cancelled'))
            resolve(false)
          } else {
            toast.error(
              i18next.t('WeChat payment failed: {{msg}}', {
                msg: res.err_msg || '',
              })
            )
            resolve(false)
          }
        }
      )
    })
  })
}

async function pollWeChatOrder(
  tradeNo: string,
  timeoutMs: number = 30000,
  intervalMs: number = 2000
): Promise<boolean> {
  const startTime = Date.now()
  while (Date.now() - startTime < timeoutMs) {
    try {
      const res = await getWeChatPayOrder(tradeNo)
      if (isApiSuccess(res) && res.data?.status === 'success') {
        return true
      }
      if (
        isApiSuccess(res) &&
        (res.data?.status === 'failed' || res.data?.status === 'expired')
      ) {
        return false
      }
    } catch {
      // transient network error, continue polling
    }
    await new Promise((r) => setTimeout(r, intervalMs))
  }
  return false
}

export function useWeChatPay() {
  const [processing, setProcessing] = useState(false)

  const processWeChatPay = useCallback(
    async (topupAmount: number, user?: UserWalletData | null) => {
      const inWeChat = isWeChatBrowser()

      if (!inWeChat) {
        toast.info(
          i18next.t(
            'Official WeChat Pay currently requires opening the wallet inside WeChat.'
          )
        )
        return false
      }

      if (!user?.wechat_id) {
        toast.error(
          i18next.t(
            'Official WeChat Pay requires binding your WeChat account first. Please bind WeChat in Security Settings, or open this page inside WeChat.'
          )
        )
        return false
      }

      setProcessing(true)
      try {
        const response = await requestWeChatPay({
          amount: Math.floor(topupAmount),
        })

        if (!isApiSuccess(response) || !response.data) {
          toast.error(getWeChatErrorMessage(response.message, response.data))
          return false
        }

        const paid = await invokeWeChatJSAPI(response.data)
        if (!paid) {
          return false
        }

        toast.info(i18next.t('Confirming payment result...'))
        const confirmed = await pollWeChatOrder(response.data.trade_no)
        if (confirmed) {
          toast.success(i18next.t('Payment successful'))
          return true
        }

        toast.warning(
          i18next.t(
            'Payment completed, order is being updated. Please refresh shortly.'
          )
        )
        return true
      } catch {
        toast.error(i18next.t('Payment request failed'))
        return false
      } finally {
        setProcessing(false)
      }
    },
    []
  )

  return { processing, processWeChatPay }
}
