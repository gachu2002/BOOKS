package store

import (
	"database/sql"
	"encoding/json"
	"time"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Product struct {
	ID          string          `json:"id"`
	Slug        string          `json:"slug"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	ProductType string          `json:"product_type"`
	PriceCents  int             `json:"price_cents"`
	Currency    string          `json:"currency"`
	Published   bool            `json:"published"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ProductCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

type ProductSearchResult struct {
	Product Product `json:"product"`
	Rank    float64 `json:"rank"`
}

type Cohort struct {
	ID                 string     `json:"id"`
	ProductID          string     `json:"product_id"`
	Slug               string     `json:"slug"`
	Title              string     `json:"title"`
	Capacity           int        `json:"capacity"`
	Status             string     `json:"status"`
	StartsAt           time.Time  `json:"starts_at"`
	EndsAt             time.Time  `json:"ends_at"`
	EnrollmentOpensAt  *time.Time `json:"enrollment_opens_at,omitempty"`
	EnrollmentClosesAt *time.Time `json:"enrollment_closes_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type PromoCode struct {
	ID             string     `json:"id"`
	Code           string     `json:"code"`
	DiscountType   string     `json:"discount_type"`
	DiscountValue  int        `json:"discount_value"`
	MaxRedemptions *int       `json:"max_redemptions,omitempty"`
	StartsAt       *time.Time `json:"starts_at,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Order struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	Status         string     `json:"status"`
	Currency       string     `json:"currency"`
	SubtotalCents  int        `json:"subtotal_cents"`
	DiscountCents  int        `json:"discount_cents"`
	TotalCents     int        `json:"total_cents"`
	IdempotencyKey *string    `json:"idempotency_key,omitempty"`
	PlacedAt       *time.Time `json:"placed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Payment struct {
	ID                string    `json:"id"`
	OrderID           string    `json:"order_id"`
	Provider          string    `json:"provider"`
	ProviderPaymentID *string   `json:"provider_payment_id,omitempty"`
	Status            string    `json:"status"`
	AmountCents       int       `json:"amount_cents"`
	Currency          string    `json:"currency"`
	IdempotencyKey    string    `json:"idempotency_key"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Entitlement struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	ProductID string     `json:"product_id"`
	OrderID   string     `json:"order_id"`
	CohortID  *string    `json:"cohort_id,omitempty"`
	Status    string     `json:"status"`
	GrantedAt *time.Time `json:"granted_at,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type EntitlementWithProduct struct {
	Entitlement Entitlement `json:"entitlement"`
	Product     Product     `json:"product"`
}

type Pagination struct {
	Limit  int
	Offset int
}

type UserCreateParams struct {
	Email    string
	FullName string
	Role     string
}

type UserUpdateParams struct {
	Email    *string
	FullName *string
	Role     *string
}

type ProductCreateParams struct {
	Slug        string
	Title       string
	Description string
	ProductType string
	PriceCents  int
	Currency    string
	Published   bool
	Metadata    json.RawMessage
}

type ProductUpdateParams struct {
	Slug        *string
	Title       *string
	Description *string
	ProductType *string
	PriceCents  *int
	Currency    *string
	Published   *bool
	Metadata    json.RawMessage
	UpdateMeta  bool
}

type CohortCreateParams struct {
	ProductID          string
	Slug               string
	Title              string
	Capacity           int
	Status             string
	StartsAt           time.Time
	EndsAt             time.Time
	EnrollmentOpensAt  *time.Time
	EnrollmentClosesAt *time.Time
}

type CohortUpdateParams struct {
	ProductID          *string
	Slug               *string
	Title              *string
	Capacity           *int
	Status             *string
	StartsAt           *time.Time
	EndsAt             *time.Time
	EnrollmentOpensAt  sql.NullTime
	UpdateEnrollOpen   bool
	EnrollmentClosesAt sql.NullTime
	UpdateEnrollClose  bool
}

type PromoCodeCreateParams struct {
	Code           string
	DiscountType   string
	DiscountValue  int
	MaxRedemptions *int
	StartsAt       *time.Time
	ExpiresAt      *time.Time
}

type PromoCodeUpdateParams struct {
	Code                 *string
	DiscountType         *string
	DiscountValue        *int
	MaxRedemptions       sql.NullInt32
	UpdateMaxRedemptions bool
	StartsAt             sql.NullTime
	UpdateStartsAt       bool
	ExpiresAt            sql.NullTime
	UpdateExpiresAt      bool
}

type CheckoutParams struct {
	UserID            string
	CohortID          string
	PromoCode         *string
	PaymentProvider   string
	IdempotencyKey    string
	ProviderPaymentID *string
}

type CheckoutResult struct {
	IdempotentReplay bool              `json:"idempotent_replay"`
	Order            Order             `json:"order"`
	Payment          Payment           `json:"payment"`
	Entitlement      Entitlement       `json:"entitlement"`
	Product          Product           `json:"product"`
	Cohort           Cohort            `json:"cohort"`
	Placement        CheckoutPlacement `json:"placement"`
	AppliedPromoCode *PromoCode        `json:"applied_promo_code,omitempty"`
	SeatsRemaining   int               `json:"seats_remaining"`
}
