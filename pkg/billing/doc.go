// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package billing is a thin façade over existing meter / quota / bill logic in
// internal/models. Payment gateways (WeChat / Alipay / Stripe) are intentionally
// not implemented here yet — prepaid top-up remains ops-managed.
package billing

// Status documents commercialization readiness for operators.
const Status = "skeleton"

// Doc points to the product plan for full SaaS packaging.
const Doc = "docs/commercialization.md"
