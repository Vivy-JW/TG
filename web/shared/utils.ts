import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

// 类名合并:clsx 处理条件/数组,twMerge 消解冲突(后写的同类工具类胜出)。
//
// 合并前两套 SPA 各有一份 cn —— debug 是这一版,dict 是朴素的 filter(Boolean).join(' '),
// 后者遇到冲突类只能靠书写顺序碰运气。合并后只有这一份。
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
