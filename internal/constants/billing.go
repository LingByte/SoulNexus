package constants

// Tenant bill lifecycle statuses.
const (
	TenantBillStatusDraft     = "draft"
	TenantBillStatusFinalized = "finalized"
	TenantBillStatusPaid      = "paid"
)

// Tenant billing modes.
const (
	TenantBillingModePrepaid  = "prepaid"
	TenantBillingModePostpaid = "postpaid"
)

// Default billing currency.
const TenantBillCurrencyCNY = "CNY"

// TenantBillingBalanceUnlimitedLabel is shown in UI when billing_unlimited is true.
const TenantBillingBalanceUnlimitedLabel = "unlimited"
