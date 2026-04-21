// Package cost provides LLM inference cost tracking middleware for axon-talk.
// It computes USD cost from token counts against a rate table and aggregates
// totals across calls.
//
// Subpackage: cost.
//
// Class: domain
// UseWhen: Cost visibility for LLM calls. Wraps talk.LLMClient to track spend.
package cost
