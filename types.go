// Package freightplan jointly optimizes parcel composition and carrier
// selection for freight orders.
package freightplan

// PackingGroup is the UN packing group of a dangerous good.
type PackingGroup int

const (
	PackingGroupI PackingGroup = iota + 1
	PackingGroupII
	PackingGroupIII
)

// DangerClass identifies a dangerous good by UN number and packing group.
type DangerClass struct {
	UN    string
	Group PackingGroup
}

// Unit is an indivisible packing unit of an order.
type Unit struct {
	ID      string
	WeightG int64
	Danger  *DangerClass // nil means ordinary goods
}

// Offer is a carrier plan that can be used for any number of parcels.
type Offer struct {
	ID         string
	MaxWeightG int64
	FeeCents   int64
	// Dangerous lists the danger classes this offer can carry; empty means
	// ordinary goods only. All offers accept ordinary goods.
	Dangerous []DangerClass
}

// Parcel is one packed parcel with its chosen carrier offer.
type Parcel struct {
	OfferID string
	UnitIDs []string
}

// Result is the outcome of planning: the parcels plus totals.
type Result struct {
	Parcels        []Parcel
	TotalCostCents int64
}
