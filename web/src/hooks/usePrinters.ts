import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { printersApi } from '../api/client'
import type { Printer } from '../types'

// Fetch all printers.
export function usePrinters() {
  return useQuery({
    queryKey: ['printers'],
    queryFn: () => printersApi.list(),
  })
}

// Fetch a single printer.
export function usePrinter(id: string) {
  return useQuery({
    queryKey: ['printers', id],
    queryFn: () => printersApi.get(id),
    enabled: !!id,
  })
}

// Create a new printer.
export function useCreatePrinter() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (data: Partial<Printer>) => printersApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['printers'] })
    },
  })
}

// Update a printer.
export function useUpdatePrinter() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Printer> }) => 
      printersApi.update(id, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['printers'] })
      queryClient.invalidateQueries({ queryKey: ['printers', id] })
    },
  })
}

// Delete a printer.
export function useDeletePrinter() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (id: string) => printersApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['printers'] })
    },
  })
}

// Fetch real-time state for all printers (poll every 5 seconds).
export function usePrinterStates() {
  return useQuery({
    queryKey: ['printer-states'],
    queryFn: () => printersApi.getAllStates(),
    refetchInterval: 5000, // Poll every 5 seconds
  })
}

// Fetch real-time state for a single printer.
export function usePrinterState(id: string) {
  return useQuery({
    queryKey: ['printer-states', id],
    queryFn: () => printersApi.getState(id),
    enabled: !!id,
    refetchInterval: 3000, // Poll every 3 seconds when viewing single printer
  })
}

