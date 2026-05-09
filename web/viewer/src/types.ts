export type RiskLevel = 'low' | 'medium' | 'high' | 'critical' | string

export interface ReviewSummary {
  contract_changes: number
  dependency_changes: number
  layer_edge_changes: number
  finding_changes: number
  added_findings: number
  removed_findings: number
  added_high_findings: number
  added_policy_findings: number
}

export interface RiskReason {
  message: string
  points: number
}

export interface ReviewResult {
  schema_version: string
  risk: {
    points: number
    level: RiskLevel
    reasons?: RiskReason[]
  }
  summary: ReviewSummary
  changed_files?: string[]
}

export interface TreeNode {
  name: string
  path?: string
  kind: 'dir' | 'file' | string
  language?: string
  layer?: string
  role?: string
  change_kind?: string
  changed_files_count?: number
  finding_count?: number
  runtime_finding_count?: number
  risk_points?: number
  children?: TreeNode[]
}

export interface TreeReport {
  schema_version: string
  root: TreeNode
}

export interface RuntimeSummary {
  change_count: number
  added_count: number
  removed_count: number
  changed_count: number
  high_count: number
  medium_count: number
  low_count: number
}

export interface Evidence {
  file?: string
  line_start?: number
  line_end?: number
  snippet?: string
  message?: string
}

export interface RuntimeChange {
  kind: string
  id: string
  before_severity?: string
  after_severity?: string
  before_confidence?: string
  after_confidence?: string
  title?: string
  risk?: string
  suggestion?: string
  before_evidence_count?: number
  after_evidence_count?: number
  evidence?: Evidence[]
}

export interface RuntimeReport {
  schema_version: string
  summary: RuntimeSummary
  changes: RuntimeChange[]
}

export interface SectionSummary {
  [key: string]: number
}

export interface FindingChange {
  kind: string
  id: string
  before?: {
    id: string
    kind?: string
    severity?: string
    confidence?: string
    title?: string
    risk?: string
    suggestion?: string
    evidence?: Evidence[]
  }
  after?: {
    id: string
    kind?: string
    severity?: string
    confidence?: string
    title?: string
    risk?: string
    suggestion?: string
    evidence?: Evidence[]
  }
  before_evidence_count?: number
  after_evidence_count?: number
  added_evidence?: Evidence[]
  removed_evidence?: Evidence[]
}

export interface FindingsReport {
  schema_version: string
  summary: SectionSummary
  changes: FindingChange[]
}

export interface ContractChange {
  kind: string
  symbol_key: string
  before?: {
    file?: string
    signature?: string
  }
  after?: {
    file?: string
    signature?: string
  }
}

export interface ContractImpact {
  symbol_key: string
  change_kind: string
  impact: string
  location?: string
  confidence?: string
  tests_changed?: boolean
  delivery_impacted?: boolean
}

export interface ContractsReport {
  schema_version: string
  summary: SectionSummary
  changes: ContractChange[]
  impacts?: ContractImpact[]
}

export interface DependencyChange {
  kind: string
  key: string
  before?: {
    from_file?: string
    to_file?: string
    from_layer?: string
    to_layer?: string
    target?: string
    kind?: string
  }
  after?: {
    from_file?: string
    to_file?: string
    from_layer?: string
    to_layer?: string
    target?: string
    kind?: string
  }
}

export interface LayerEdgeChange {
  kind: string
  from_layer: string
  to_layer: string
  before_count?: number
  after_count?: number
}

export interface DependenciesReport {
  schema_version: string
  summary: SectionSummary
  dependency_changes: DependencyChange[]
  layer_edge_changes: LayerEdgeChange[]
}

export interface GitStatus {
  root: string
  branch: string
  head: string
  short_head: string
  has_worktree_changes: boolean
  ahead?: number
  behind?: number
  upstream?: string
}

export interface GitCommit {
  hash: string
  short_hash: string
  parents?: string[]
  refs?: string[]
  author: string
  date: string
  message: string
}

export interface GitCommitsResponse {
  root: string
  limit: number
  commits: GitCommit[]
}

export interface CreateReviewRequest {
  base: string
  head?: string
  worktree?: boolean
  config_path?: string
}

export interface CreateReviewResponse {
  id: string
  bundle_dir: string
  artifacts: Record<string, string>
  risk: {
    points: number
    level: string
    reasons?: RiskReason[]
  }
  summary: ReviewSummary
}

export interface GitBranch {
  name: string
  kind: string
  head: string
  short_head: string
  current?: boolean
  upstream?: string
  ahead?: number
  behind?: number
}

export interface GitBranchesResponse {
  root: string
  current: string
  branches: GitBranch[]
}

export interface GitRef {
  name: string
  kind: string
  target: string
  short_hash: string
}

export interface GitRefsResponse {
  root: string
  refs: GitRef[]
}

export interface GitGraphCommit extends GitCommit {
  children?: string[] | null
  lane: number
  parent_lanes?: number[] | null
  child_lanes?: number[] | null
  is_merge?: boolean | null
  is_branch_point?: boolean | null
}

export interface GitGraphResponse {
  schema_version: string
  root: string
  ref?: string
  all?: boolean
  limit: number
  commits: GitGraphCommit[]
  layout?: GitGraphLayout
}

export interface GitGraphLayout {
  row_height: number
  lane_width: number
  segments?: GitGraphSegment[]
  edges?: GitGraphEdge[]
}

export interface GitGraphSegment {
  lane: number
  from_row: number
  to_row: number
}

export interface GitGraphEdge {
  from_lane: number
  to_lane: number
  from_row: number
  to_row: number
  kind: 'branch' | 'merge' | 'parent' | string
}

export interface GitGraphLayout {
  row_height: number
  lane_width: number
  segments?: GitGraphSegment[]
  edges?: GitGraphEdge[]
}

export interface GitGraphSegment {
  lane: number
  from_row: number
  to_row: number
}

export interface GitGraphEdge {
  from_lane: number
  to_lane: number
  from_row: number
  to_row: number
  kind: 'branch' | 'merge' | 'parent' | string
}
