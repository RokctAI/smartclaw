# Frappe API Integration Strategy (RCore Platform)

This document outlines the planned architecture for bridging the GoClaw native Coding Engine (Go) with the RCore Frappe backend (Python/MariaDB).

## The Goal
To give the GoClaw agent native access to the business logic, database, and background workers of the Frappe application without manually defining every tool in Go. We will utilize Frappe's dynamic generation of `ai_tools.json` (baked via `manager.py`).

## The Architecture

### 1. The Frappe Side (RCore Platform)
- The existing `manager.py` uses `bake_assets()` to scan the Frappe application for `@frappe.whitelist()` functions.
- It dynamically generates `ai_tools.json` containing the names, descriptions, and required JSON schema properties of all available commands.
- It exposes a unified execution gateway in `api.py` (`/api/method/rcore.platform.api.tenant`). This gateway authenticates, validates against the manifest, and routes the request securely to the native Python method via `frappe.get_attr()`.

### 2. The GoClaw Side (SmartClaw Engine)
- **The Ingester Module (`internal/coding/frappe_provider.go`)**: 
  - On initialization, GoClaw reads `ai_tools.json` from the specified RCore platform directory.
  - It iterates over the JSON array and dynamically registers each command (e.g., `paas:create_tenant`) into the `Registry` as a **"Ghost Tool"**.
  - **Token Optimization Rule**: Frappe tools MUST NOT be injected into the default system prompt array. This prevents prompt bloat. The LLM must discover Frappe capabilities dynamically using the `tool_search` mechanism.
- **The Universal Executor Tool (`internal/tools/frappe.go`)**:
  - We build a single, universal tool struct implementation in Go.
  - When the LLM natively calls a discovered Frappe tool, it is caught by this universal executor.
  - The executor serializes the arguments into a JSON payload and performs an HTTP POST request against the RCore Gateway (`/api/method/rcore.platform.api.tenant`).
  - It receives the response and feeds the structured JSON back into the LLM's context window. 
  - If the Frappe API returns 3 consecutive errors (e.g., hallucinated JSON schema), it automatically triggers GoClaw's internal 3-Error Tripwire to force a reload from `self_correction_sop.md`.

## Execution Requirements
- GoClaw must have knowledge of the RCore server URL and API Keys (or utilize a specialized admin authorization token) to communicate securely with the `tenant` and `control` gateways.
- `ai_tools.json` location must be provided strictly via environment variables or loaded through the `.rokct` global protocol rules.
