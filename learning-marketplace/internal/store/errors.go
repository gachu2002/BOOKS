package store

import "errors"

var ErrNotFound = errors.New("not found")

var ErrSoldOut = errors.New("cohort sold out")

var ErrCohortNotOpen = errors.New("cohort not open")

var ErrPromoCodeInvalid = errors.New("promo code invalid")

var ErrPromoCodeExhausted = errors.New("promo code exhausted")
