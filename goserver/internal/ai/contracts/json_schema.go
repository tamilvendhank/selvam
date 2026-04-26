package contracts

const InvestingReviewOutputJSONSchemaText = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://selvam.local/schemas/investing-review-output.v1.schema.json",
  "title": "Investing Review Output Envelope V1",
  "type": "object",
  "additionalProperties": false,
  "required": [
    "schema_version",
    "prompt_version",
    "output_schema_version",
    "item_correlation_id",
    "workflow_run_id",
    "company_id",
    "symbol",
    "book_type",
    "review_type",
    "payload"
  ],
  "properties": {
    "schema_version": { "const": "ai-review-output-envelope.v1" },
    "prompt_version": { "type": "string", "minLength": 1 },
    "output_schema_version": { "const": "investing-review-output.v1" },
    "item_correlation_id": { "type": "string", "minLength": 1 },
    "workflow_run_id": { "type": "string", "minLength": 1 },
    "batch_job_id": { "type": "string" },
    "batch_item_id": { "type": "string" },
    "company_id": { "type": "string", "minLength": 1 },
    "symbol": { "type": "string", "minLength": 1 },
    "book_type": { "const": "investing" },
    "review_type": { "const": "investing_company_review" },
    "model_name": { "type": "string" },
    "generated_at": { "type": "string", "format": "date-time" },
    "payload": { "$ref": "#/$defs/investing_review_output_payload" }
  },
  "$defs": {
    "score_1_to_10": { "type": "number", "minimum": 1, "maximum": 10 },
    "score_0_to_10": { "type": "number", "minimum": 0, "maximum": 10 },
    "unit_interval": { "type": "number", "minimum": 0, "maximum": 1 },
    "percentage": { "type": "number", "minimum": 0, "maximum": 100 },
    "string_array": {
      "type": "array",
      "items": { "type": "string", "minLength": 1 }
    },
    "section_name": {
      "enum": [
        "investability",
        "business_traction",
        "profit_conversion",
        "capital_efficiency_financial_strength",
        "structural_sector_attractiveness",
        "runway_industry_positioning",
        "management_governance",
        "market_confirmation",
        "valuation_entry_attractiveness"
      ]
    },
    "sub_score_name": {
      "enum": [
        "liquidity",
        "data_quality_completeness",
        "basic_investability_suitability",
        "listing_operating_history_sufficiency",
        "revenue_growth_strength",
        "revenue_growth_consistency",
        "recent_12q_acceleration_deterioration",
        "evidence_of_expanding_demand",
        "operating_margin_quality_trend",
        "profit_growth_strength",
        "cash_conversion_quality",
        "recent_operating_leverage_margin_direction",
        "roce_roic_quality",
        "balance_sheet_strength",
        "working_capital_efficiency",
        "dilution_capital_allocation_discipline",
        "demand_tailwind_strength",
        "industry_economics_quality",
        "policy_formalization_support",
        "cyclicality_risk",
        "market_opportunity_size",
        "share_gain_potential",
        "expansion_optionality",
        "competitive_positioning_strength",
        "capital_allocation_quality",
        "execution_consistency",
        "shareholder_alignment_trustworthiness",
        "disclosure_quality",
        "relative_strength",
        "trend_quality",
        "drawdown_resilience_behavior",
        "reaction_to_results_news",
        "historical_valuation_attractiveness",
        "valuation_support_vs_current_quality",
        "overvaluation_risk",
        "entry_timing_suitability"
      ]
    },
    "investing_action": {
      "enum": ["buy", "watch", "hold", "trim", "sell", "reject"]
    },
    "watchlist_bucket": {
      "enum": ["research", "watch", "buy_ready", "hold", "exit_review"]
    },
    "investing_mode": {
      "enum": ["early_hunter", "balanced", "confirmed_compounder"]
    },
    "section_action_cap": {
      "enum": ["cannot_buy", "watch_only", "exit_review_only", "none"]
    },
    "trend_direction": {
      "enum": ["improving", "stable", "weakening", "mixed"]
    },
    "evidence_strength": {
      "enum": ["low", "medium", "high"]
    },
    "metric_basis": {
      "enum": ["quant", "text", "hybrid"]
    },
    "evidence_source_type": {
      "enum": [
        "annual_report",
        "concall",
        "investor_presentation",
        "exchange_filing",
        "financial_data",
        "price_data",
        "manual_note"
      ]
    },
    "evidence_direction": {
      "enum": ["positive", "negative", "neutral"]
    },
    "recommended_tranche_style": {
      "enum": ["start", "add", "pause", "reduce", "exit"]
    },
    "investing_review_output_payload": {
      "type": "object",
      "additionalProperties": false,
      "required": [
        "company_id",
        "symbol",
        "review_date",
        "mode",
        "weighted_total_score",
        "confidence_score",
        "hard_gate_failed",
        "sections",
        "suggested_action",
        "suggested_bucket",
        "action_rationale_summary",
        "capital_eligible"
      ],
      "properties": {
        "company_id": { "type": "string", "minLength": 1 },
        "symbol": { "type": "string", "minLength": 1 },
        "review_date": { "type": "string", "format": "date-time" },
        "mode": { "$ref": "#/$defs/investing_mode" },
        "weighted_total_score": { "$ref": "#/$defs/score_0_to_10" },
        "confidence_score": { "$ref": "#/$defs/unit_interval" },
        "hard_gate_failed": { "type": "boolean" },
        "hard_gate_failure_reasons": { "$ref": "#/$defs/string_array" },
        "sections": {
          "type": "array",
          "minItems": 9,
          "maxItems": 9,
          "items": { "$ref": "#/$defs/section_score" }
        },
        "suggested_action": { "$ref": "#/$defs/investing_action" },
        "suggested_bucket": { "$ref": "#/$defs/watchlist_bucket" },
        "action_rationale_summary": { "type": "string", "minLength": 1 },
        "what_changed_summary": { "type": "string" },
        "capital_eligible": { "type": "boolean" },
        "capital_priority_score": { "$ref": "#/$defs/score_1_to_10" },
        "recommended_position_target_pct": { "$ref": "#/$defs/percentage" },
        "recommended_position_cap_pct": { "$ref": "#/$defs/percentage" },
        "recommended_tranche_style": { "$ref": "#/$defs/recommended_tranche_style" },
        "action_constraints": { "$ref": "#/$defs/string_array" },
        "thesis_summary_candidate": { "type": "string" },
        "why_this_business_can_compound": { "type": "string" },
        "key_growth_drivers": { "$ref": "#/$defs/string_array" },
        "key_moat_or_advantage_factors": { "$ref": "#/$defs/string_array" },
        "why_now": { "type": "string" },
        "key_risks": { "$ref": "#/$defs/string_array" },
        "disconfirming_signals": { "$ref": "#/$defs/string_array" },
        "what_would_break_the_thesis": { "$ref": "#/$defs/string_array" },
        "change_log": { "$ref": "#/$defs/change_log" },
        "missing_data_points": { "$ref": "#/$defs/string_array" },
        "low_confidence_areas": { "$ref": "#/$defs/string_array" },
        "assumptions_made": { "$ref": "#/$defs/string_array" },
        "warnings": { "$ref": "#/$defs/string_array" }
      }
    },
    "section_score": {
      "type": "object",
      "additionalProperties": false,
      "required": [
        "section_name",
        "section_weight",
        "section_score_raw",
        "section_passed_minimum_check",
        "section_summary",
        "section_strengths",
        "section_weaknesses",
        "section_risks",
        "section_confidence_score",
        "sub_scores",
        "evidence_refs"
      ],
      "properties": {
        "section_name": { "$ref": "#/$defs/section_name" },
        "section_weight": { "$ref": "#/$defs/percentage" },
        "section_score_raw": { "$ref": "#/$defs/score_1_to_10" },
        "section_score_weighted": { "$ref": "#/$defs/score_0_to_10" },
        "section_passed_minimum_check": { "type": "boolean" },
        "section_action_cap": { "$ref": "#/$defs/section_action_cap" },
        "section_summary": { "type": "string", "minLength": 1 },
        "section_strengths": { "$ref": "#/$defs/string_array" },
        "section_weaknesses": { "$ref": "#/$defs/string_array" },
        "section_risks": { "$ref": "#/$defs/string_array" },
        "section_confidence_score": { "$ref": "#/$defs/unit_interval" },
        "sub_scores": {
          "type": "array",
          "minItems": 1,
          "items": { "$ref": "#/$defs/sub_score" }
        },
        "evidence_refs": {
          "type": "array",
          "items": { "$ref": "#/$defs/evidence_ref" }
        }
      }
    },
    "sub_score": {
      "type": "object",
      "additionalProperties": false,
      "required": [
        "sub_score_name",
        "sub_score_weight",
        "sub_score_value",
        "sub_score_summary",
        "trend_direction",
        "evidence_strength",
        "metric_basis"
      ],
      "properties": {
        "sub_score_name": { "$ref": "#/$defs/sub_score_name" },
        "sub_score_weight": { "$ref": "#/$defs/percentage" },
        "sub_score_value": { "$ref": "#/$defs/score_1_to_10" },
        "sub_score_summary": { "type": "string", "minLength": 1 },
        "trend_direction": { "$ref": "#/$defs/trend_direction" },
        "evidence_strength": { "$ref": "#/$defs/evidence_strength" },
        "metric_basis": { "$ref": "#/$defs/metric_basis" },
        "notes": { "type": "string" },
        "evidence_ref_ids": {
          "type": "array",
          "items": { "type": "string", "minLength": 1 }
        }
      }
    },
    "evidence_ref": {
      "type": "object",
      "additionalProperties": false,
      "required": [
        "evidence_id",
        "source_type",
        "evidence_summary",
        "evidence_direction"
      ],
      "properties": {
        "evidence_id": { "type": "string", "minLength": 1 },
        "source_type": { "$ref": "#/$defs/evidence_source_type" },
        "source_date": { "type": "string", "format": "date-time" },
        "source_title": { "type": "string" },
        "source_period": { "type": "string" },
        "source_url_or_path": { "type": "string" },
        "excerpt_or_metric_name": { "type": "string" },
        "excerpt_or_metric_value": { "type": "string" },
        "evidence_summary": { "type": "string", "minLength": 1 },
        "evidence_direction": { "$ref": "#/$defs/evidence_direction" }
      }
    },
    "change_log": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "previous_review_id": { "type": "string" },
        "weighted_total_score_change": { "type": "number" },
        "bucket_change": { "type": "string" },
        "action_change": { "type": "string" },
        "thesis_status_change": { "type": "string" },
        "major_positive_changes": { "$ref": "#/$defs/string_array" },
        "major_negative_changes": { "$ref": "#/$defs/string_array" },
        "section_score_changes": {
          "type": "object",
          "additionalProperties": { "type": "number" }
        },
        "sub_score_changes": {
          "type": "object",
          "additionalProperties": { "type": "number" }
        },
        "valuation_state_change": { "type": "string" },
        "ownership_relevance_change": { "type": "string" },
        "requires_exit_review": { "type": "boolean" },
        "change_summary": { "type": "string" }
      }
    }
  }
}`

func InvestingReviewOutputJSONSchema() []byte {
	return []byte(InvestingReviewOutputJSONSchemaText)
}
