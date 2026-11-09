// 产品切换：T100 / TOP GP。
//
// 只影响「站点管理」与「数据字典」这两块读谁的数据：
//   T100   → tt 自己的 config.json + 调试/字典子系统
//   TOP GP → 本机 ttgp 服务（127.0.0.1:28671）的 gp/config.json
// 其余分区（DEBUG / 技能 / 应用设置）与产品无关，切了也不变。

import { useEffect, useState } from 'react'

export type Product = 't100' | 'gp'

const KEY = 'tt.product'
const EVT = 'tt-product-change'

export function getProduct(): Product {
  try {
    return localStorage.getItem(KEY) === 'gp' ? 'gp' : 't100'
  } catch {
    return 't100'
  }
}

export function setProduct(p: Product) {
  try { localStorage.setItem(KEY, p) } catch { /* 隐私模式忽略 */ }
  window.dispatchEvent(new Event(EVT))
}

export function useProduct(): [Product, (p: Product) => void] {
  const [p, setP] = useState<Product>(getProduct)
  useEffect(() => {
    const h = () => setP(getProduct())
    window.addEventListener(EVT, h)
    return () => window.removeEventListener(EVT, h)
  }, [])
  return [p, (x: Product) => { setProduct(x); setP(x) }]
}
