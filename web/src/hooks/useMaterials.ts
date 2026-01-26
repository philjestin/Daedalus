import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { materialsApi, spoolsApi } from '../api/client'
import type { Material, MaterialSpool } from '../types'

// Fetch all materials.
export function useMaterials() {
  return useQuery({
    queryKey: ['materials'],
    queryFn: () => materialsApi.list(),
  })
}

// Fetch a single material.
export function useMaterial(id: string) {
  return useQuery({
    queryKey: ['materials', id],
    queryFn: () => materialsApi.get(id),
    enabled: !!id,
  })
}

// Create a new material.
export function useCreateMaterial() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: Partial<Material>) => materialsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['materials'] })
    },
  })
}

// Fetch all spools.
export function useSpools() {
  return useQuery({
    queryKey: ['spools'],
    queryFn: () => spoolsApi.list(),
  })
}

// Fetch a single spool.
export function useSpool(id: string) {
  return useQuery({
    queryKey: ['spools', id],
    queryFn: () => spoolsApi.get(id),
    enabled: !!id,
  })
}

// Create a new spool.
export function useCreateSpool() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: Partial<MaterialSpool>) => spoolsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['spools'] })
    },
  })
}

// Fetch spools with their associated material info.
export function useSpoolsWithMaterials() {
  const { data: spools = [], ...spoolsQuery } = useSpools()
  const { data: materials = [] } = useMaterials()

  const spoolsWithMaterials = spools.map((spool) => ({
    ...spool,
    material: materials.find((m) => m.id === spool.material_id),
  }))

  return {
    ...spoolsQuery,
    data: spoolsWithMaterials,
  }
}
