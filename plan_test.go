package freightplan

import (
	"math/rand"
	"testing"
)

// checkSolution verifies that res is a feasible packing of order under
// offers and returns the total cost and parcel count.
func checkSolution(t *testing.T, order []Unit, offers []Offer, res Result) (int64, int) {
	t.Helper()
	byID := make(map[string]Offer, len(offers))
	for _, o := range offers {
		byID[o.ID] = o
	}
	units := make(map[string]Unit, len(order))
	for _, u := range order {
		units[u.ID] = u
	}
	seen := make(map[string]bool, len(order))
	for _, p := range res.Parcels {
		if len(p.UnitIDs) == 0 {
			t.Fatalf("empty parcel with offer %q", p.OfferID)
		}
		o, ok := byID[p.OfferID]
		if !ok {
			t.Fatalf("parcel uses unknown offer %q", p.OfferID)
		}
		allowed := make(map[DangerClass]bool, len(o.Dangerous))
		for _, d := range o.Dangerous {
			allowed[d] = true
		}
		var weight int64
		var danger *DangerClass
		for _, id := range p.UnitIDs {
			u, ok := units[id]
			if !ok {
				t.Fatalf("parcel contains unknown unit %q", id)
			}
			if seen[id] {
				t.Fatalf("unit %q packed more than once", id)
			}
			seen[id] = true
			weight += u.WeightG
			if u.Danger != nil {
				if !allowed[*u.Danger] {
					t.Fatalf("offer %q cannot carry %+v", o.ID, *u.Danger)
				}
				if danger != nil && *danger != *u.Danger {
					t.Fatalf("parcel mixes danger classes %+v and %+v", *danger, *u.Danger)
				}
				d := *u.Danger
				danger = &d
			}
		}
		if weight > o.MaxWeightG {
			t.Fatalf("parcel on offer %q weighs %d > limit %d", o.ID, weight, o.MaxWeightG)
		}
	}
	if len(seen) != len(order) {
		t.Fatalf("packed %d of %d units", len(seen), len(order))
	}
	var cost int64
	for _, p := range res.Parcels {
		cost += byID[p.OfferID].FeeCents
	}
	if cost != res.TotalCostCents {
		t.Fatalf("reported cost %d, actual %d", res.TotalCostCents, cost)
	}
	return cost, len(res.Parcels)
}

func TestSingleOfferOrdinaryGoods(t *testing.T) {
	order := []Unit{
		{ID: "a", WeightG: 6000},
		{ID: "b", WeightG: 5000},
		{ID: "c", WeightG: 3000},
		{ID: "d", WeightG: 2000},
		{ID: "e", WeightG: 2000},
		{ID: "f", WeightG: 2000},
	}
	offers := []Offer{{ID: "std", MaxWeightG: 10000, FeeCents: 100}}

	res, ok := Plan(order, offers)
	if !ok {
		t.Fatal("expected feasible plan")
	}
	cost, n := checkSolution(t, order, offers, res)
	if cost != 200 || n != 2 {
		t.Fatalf("got %d parcels, %d cents; want 2 parcels, 200 cents", n, cost)
	}
}

func TestSingleOfferShuffledOrder(t *testing.T) {
	base := []Unit{
		{ID: "a", WeightG: 6000},
		{ID: "b", WeightG: 5000},
		{ID: "c", WeightG: 3000},
		{ID: "d", WeightG: 2000},
		{ID: "e", WeightG: 2000},
		{ID: "f", WeightG: 2000},
	}
	offers := []Offer{{ID: "std", MaxWeightG: 10000, FeeCents: 100}}

	rng := rand.New(rand.NewSource(42))
	for trial := 0; trial < 50; trial++ {
		order := append([]Unit(nil), base...)
		rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
		res, ok := Plan(order, offers)
		if !ok {
			t.Fatal("expected feasible plan")
		}
		cost, n := checkSolution(t, order, offers, res)
		if cost != 200 || n != 2 {
			t.Fatalf("trial %d: got %d parcels, %d cents; want 2 parcels, 200 cents", trial, n, cost)
		}
	}
}

func TestDangerousGoodsTwoOffers(t *testing.T) {
	dg1 := DangerClass{UN: "UN1263", Group: PackingGroupII}
	dg2 := DangerClass{UN: "UN1263", Group: PackingGroupIII}
	order := []Unit{
		{ID: "dg-a", WeightG: 6000, Danger: &dg1},
		{ID: "dg-b", WeightG: 6000, Danger: &dg2},
		{ID: "ord-a", WeightG: 4000},
		{ID: "ord-b", WeightG: 4000},
	}
	offers := []Offer{
		{ID: "small", MaxWeightG: 8000, FeeCents: 400, Dangerous: []DangerClass{dg1, dg2}},
		{ID: "large", MaxWeightG: 12000, FeeCents: 600, Dangerous: []DangerClass{dg1, dg2}},
	}

	res, ok := Plan(order, offers)
	if !ok {
		t.Fatal("expected feasible plan")
	}
	cost, n := checkSolution(t, order, offers, res)
	if cost != 1200 || n != 2 {
		t.Fatalf("got %d parcels, %d cents; want 2 parcels, 1200 cents", n, cost)
	}
}

func TestInfeasible(t *testing.T) {
	dg := DangerClass{UN: "UN1203", Group: PackingGroupII}
	order := []Unit{{ID: "dg", WeightG: 100, Danger: &dg}}
	offers := []Offer{{ID: "std", MaxWeightG: 1000, FeeCents: 100}}
	if _, ok := Plan(order, offers); ok {
		t.Fatal("expected infeasible: no offer carries the danger class")
	}

	order = []Unit{{ID: "heavy", WeightG: 2000}}
	if _, ok := Plan(order, offers); ok {
		t.Fatal("expected infeasible: unit exceeds weight limit")
	}
}

func TestCostBeatsParcelCount(t *testing.T) {
	// Two 6000g units: one large parcel (600) beats two small (2x400),
	// and cost dominates parcel count.
	order := []Unit{
		{ID: "a", WeightG: 6000},
		{ID: "b", WeightG: 6000},
	}
	offers := []Offer{
		{ID: "small", MaxWeightG: 8000, FeeCents: 400},
		{ID: "large", MaxWeightG: 12000, FeeCents: 600},
	}
	res, ok := Plan(order, offers)
	if !ok {
		t.Fatal("expected feasible plan")
	}
	cost, n := checkSolution(t, order, offers, res)
	if cost != 600 || n != 1 {
		t.Fatalf("got %d parcels, %d cents; want 1 parcel, 600 cents", n, cost)
	}
}
