import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { printersApi, printJobsApi } from '../api/client'
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

// Fetch print jobs for a specific printer.
export function usePrinterJobs(id: string) {
  return useQuery({
    queryKey: ['printer-jobs', id],
    queryFn: () => printersApi.getJobs(id),
    enabled: !!id,
  })
}

// Fetch job statistics for a specific printer.
export function usePrinterStats(id: string) {
  return useQuery({
    queryKey: ['printer-stats', id],
    queryFn: () => printersApi.getStats(id),
    enabled: !!id,
  })
}

// Fetch comprehensive analytics for a specific printer.
export function usePrinterAnalytics(id: string) {
  return useQuery({
    queryKey: ['printer-analytics', id],
    queryFn: () => printersApi.getAnalytics(id),
    enabled: !!id,
    staleTime: 60000, // Data is fresh for 1 minute
  })
}

// Fetch events for a specific print job.
export function useJobEvents(jobId: string) {
  return useQuery({
    queryKey: ['job-events', jobId],
    queryFn: () => printJobsApi.getEvents(jobId),
    enabled: !!jobId,
  })
}

