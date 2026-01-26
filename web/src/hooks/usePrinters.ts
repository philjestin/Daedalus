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

// Fetch real-time state for all printers.
// WebSocket pushes updates; polling is a fallback for reconnection scenarios.
export function usePrinterStates() {
  return useQuery({
    queryKey: ['printer-states'],
    queryFn: () => printersApi.getAllStates(),
    refetchInterval: 30000, // Reduced polling - WebSocket handles real-time updates
    staleTime: 10000, // Consider data fresh for 10 seconds
  })
}

// Fetch real-time state for a single printer.
// WebSocket pushes updates; polling is a fallback.
export function usePrinterState(id: string) {
  return useQuery({
    queryKey: ['printer-states', id],
    queryFn: () => printersApi.getState(id),
    enabled: !!id,
    refetchInterval: 15000, // Reduced polling - WebSocket handles real-time updates
    staleTime: 5000,
  })
}

