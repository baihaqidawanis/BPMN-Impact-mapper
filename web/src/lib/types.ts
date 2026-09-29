export type AnalysisMethod = "hybrid" | "bm25" | "configuration";

export interface ElementRef {
  id: string;
  name?: string;
  type: string;
  actor?: string;
  documentation?: string;
}

export interface Evidence {
  type: string;
  location: string;
  excerpt: string;
}

export interface ScoreBreakdown {
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
}

export interface Warning {
  code: string;
  message: string;
}

export interface Report {
  analysis_id: string;
  input_fingerprint: string;
  method: AnalysisMethod;
  repository: { name: string; revision: string };
  changes: Change[];
  warnings: Warning[];
}

export interface APIErrorPayload {
  error?: { code?: string; message?: string };
}
