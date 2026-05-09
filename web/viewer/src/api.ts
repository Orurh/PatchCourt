import type {
  ContractsReport,
  CreateReviewRequest,
  CreateReviewResponse,
  DependenciesReport,
  FindingsReport,
  GitBranchesResponse,
  GitCommitsResponse,
  GitRefsResponse,
  GitStatus,
  ReviewResult,
  RuntimeReport,
  TreeReport,
} from './types'

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(path)

  if (!response.ok) {
    throw new Error(`${path}: HTTP ${response.status}`)
  }

  return response.json() as Promise<T>
}

async function postJSON<TResponse, TBody>(path: string, body: TBody): Promise<TResponse> {
  const response = await fetch(path, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  })

  if (!response.ok) {
    const text = await response.text()
    throw new Error(`${path}: HTTP ${response.status}: ${text}`)
  }

  return response.json() as Promise<TResponse>
}

export function fetchGitStatus(): Promise<GitStatus> {
  return getJSON<GitStatus>('/api/git/status')
}

export function fetchGitBranches(): Promise<GitBranchesResponse> {
  return getJSON<GitBranchesResponse>('/api/git/branches')
}

export function fetchGitRefs(): Promise<GitRefsResponse> {
  return getJSON<GitRefsResponse>('/api/git/refs')
}

export function fetchGitCommits(limit = 50, ref = ''): Promise<GitCommitsResponse> {
  const params = new URLSearchParams()
  params.set('limit', String(limit))

  if (ref.trim() !== '') {
    params.set('ref', ref)
  }

  return getJSON<GitCommitsResponse>(`/api/git/commits?${params.toString()}`)
}

export function fetchGitCommitsAll(limit = 100): Promise<GitCommitsResponse> {
  return getJSON<GitCommitsResponse>(`/api/git/commits?all=true&limit=${limit}`)
}

export function createReview(req: CreateReviewRequest): Promise<CreateReviewResponse> {
  return postJSON<CreateReviewResponse, CreateReviewRequest>('/api/reviews', req)
}

export function fetchReview(): Promise<ReviewResult> {
  return getJSON<ReviewResult>('/api/reviews/latest/review')
}

export function fetchTree(): Promise<TreeReport> {
  return getJSON<TreeReport>('/api/reviews/latest/tree')
}

export function fetchRuntime(): Promise<RuntimeReport> {
  return getJSON<RuntimeReport>('/api/reviews/latest/runtime')
}

export function fetchFindings(): Promise<FindingsReport> {
  return getJSON<FindingsReport>('/api/reviews/latest/findings')
}

export function fetchContracts(): Promise<ContractsReport> {
  return getJSON<ContractsReport>('/api/reviews/latest/contracts')
}

export function fetchDependencies(): Promise<DependenciesReport> {
  return getJSON<DependenciesReport>('/api/reviews/latest/dependencies')
}
