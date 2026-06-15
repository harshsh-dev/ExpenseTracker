import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from './client'
import { useFeature } from '../features'
import type { Category, Expense, Income, Investment } from '../types'

type Map = {
  incomes: Income
  expenses: Expense
  investments: Investment
  categories: Category
}

function useList<K extends keyof Map>(key: K, enabled = true) {
  return useQuery({
    queryKey: [key],
    queryFn: () => api[key].list() as Promise<Map[K][]>,
    enabled,
  })
}

function useCrud<K extends keyof Map>(key: K) {
  const qc = useQueryClient()
  const invalidate = () => qc.invalidateQueries({ queryKey: [key] })
  const create = useMutation({
    mutationFn: (body: Partial<Map[K]>) => api[key].create(body as never) as Promise<Map[K]>,
    onSuccess: invalidate,
  })
  const update = useMutation({
    mutationFn: ({ id, body }: { id: string; body: Partial<Map[K]> }) =>
      api[key].update(id, body as never) as Promise<Map[K]>,
    onSuccess: invalidate,
  })
  const remove = useMutation({
    mutationFn: (id: string) => api[key].remove(id),
    onSuccess: invalidate,
  })
  return { create, update, remove }
}

export const useIncomes = () => useList('incomes', useFeature('income'))
export const useExpenses = () => useList('expenses', useFeature('expenses'))
export const useInvestments = () => useList('investments', useFeature('investments'))
export const useCategories = () => useList('categories', useFeature('categories'))

export const useIncomeCrud = () => useCrud('incomes')
export const useExpenseCrud = () => useCrud('expenses')
export const useInvestmentCrud = () => useCrud('investments')
export const useCategoryCrud = () => useCrud('categories')

export function useRefreshPrices() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.refreshPrices(),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['investments'] }),
  })
}
