import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { projectsApi, partsApi } from '../api/client'
import type { Project, Part } from '../types'

// Fetch all projects.
export function useProjects() {
  return useQuery({
    queryKey: ['projects'],
    queryFn: () => projectsApi.list(),
  })
}

// Fetch a single project.
export function useProject(id: string) {
  return useQuery({
    queryKey: ['projects', id],
    queryFn: () => projectsApi.get(id),
    enabled: !!id,
  })
}

// Create a new project.
export function useCreateProject() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (data: Partial<Project>) => projectsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

// Update a project.
export function useUpdateProject() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Project> }) => 
      projectsApi.update(id, data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      queryClient.invalidateQueries({ queryKey: ['projects', id] })
    },
  })
}

// Delete a project.
export function useDeleteProject() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (id: string) => projectsApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
}

// Fetch parts for a project.
export function useParts(projectId: string) {
  return useQuery({
    queryKey: ['parts', projectId],
    queryFn: () => partsApi.listByProject(projectId),
    enabled: !!projectId,
  })
}

// Create a new part, optionally with a file attachment.
export function useCreatePart() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ projectId, data, file, notes }: { projectId: string; data: Partial<Part>; file?: File; notes?: string }) =>
      partsApi.createWithFile(projectId, data, file, notes),
    onSuccess: (_, { projectId }) => {
      queryClient.invalidateQueries({ queryKey: ['parts', projectId] })
      queryClient.invalidateQueries({ queryKey: ['designs'] })
    },
  })
}

// Update a part.
export function useUpdatePart() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Part> }) => 
      partsApi.update(id, data),
    onSuccess: (part) => {
      queryClient.invalidateQueries({ queryKey: ['parts', part.project_id] })
    },
  })
}

// Delete a part.
export function useDeletePart() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ id, projectId }: { id: string; projectId: string }) => 
      partsApi.delete(id).then(() => projectId),
    onSuccess: (projectId) => {
      queryClient.invalidateQueries({ queryKey: ['parts', projectId] })
    },
  })
}

