export type AnalysisMethod = "hybrid" | "fusion" | "bm25" | "semantic" | "configuration";

export interface ElementRef {
  id: string;
  name?: string;
  type: string;
  actor?: string;
  documentation?: string;
	process_id?: string;
	parent_id?: string;
	attached_to_ref?: string;
	gateway_direction?: string;
	default_flow?: string;
	event_definitions?: string[];
}

export interface ChangeContext {
	pattern: "INSERT_TASK" | "REMOVE_TASK_WITH_REWIRE" | "ADD_GATEWAY_BRANCH" | string;
	predecessors?: ElementRef[];
	successors?: ElementRef[];
	related_flow_ids?: string[];
}

export interface Evidence {
  type: string;
  location: string;
  excerpt: string;
}

export interface ScoreBreakdown {
  dependency?: number;
  semantic?: number;
  rrf?: number;
  trace_id: number;
  symbol: number;
  path: number;
  lexical: number;
  structured: number;
  bm25: number;
  total: number;
}

export interface Impact {
  module: string;
  layer: string;
  file: string;
  symbol?: string;
  required_action: string;
  evidence: Evidence[];
  score_breakdown: ScoreBreakdown;
  confidence: number;
}

export interface Change {
  change_type: string;
  old_element: ElementRef | null;
  new_element: ElementRef | null;
  impacts: Impact[];
  unresolved_impacts: string[];
  reason?: string;
	context?: ChangeContext;
}

export interface Warning {
  code: string;
  message: string;
}

export interface Report {
  source_analysis?: { file: string; language: string; adapter: string; capabilities: string[]; fallback?: boolean }[];
  analysis_id: string;
  input_fingerprint: string;
  method: AnalysisMethod;
  method_metadata?: Record<string, string>;
  repository: { name: string; revision: string };
  changes: Change[];
  warnings: Warning[];
}

export interface ReportSummary {
  analysis_id: string;
  input_fingerprint: string;
  method: AnalysisMethod;
  repository: { name: string; revision: string };
  changes: number;
  candidates: number;
  unresolved: number;
  warnings: number;
}

export interface APIErrorPayload {
  error?: { code?: string; message?: string };
}

export interface ReviewerExpectedImpact {
  file: string;
  symbol?: string;
  module?: string;
  layer?: string;
  required_action: string;
}

export interface ReviewerCase {
  change_type: string;
  element_id: string;
  expected: ReviewerExpectedImpact[];
}

export interface ReviewerAnnotation {
  version: "1";
  reviewer_id: string;
  cases: ReviewerCase[];
}
