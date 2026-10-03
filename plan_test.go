package freightplanner

import (
	"math/rand"
	"testing"
)

// totalFee 汇总规划结果的总费用，并校验每个包裹引用的方案存在。
func totalFee(t *testing.T, parcels []Parcel, offers []Offer) int64 {
	t.Helper()
	fees := make(map[string]int64)
	for _, o := range offers {
		fees[o.ID] = o.FeeCents
	}
	var sum int64
	for _, p := range parcels {
		f, ok := fees[p.OfferID]
		if !ok {
			t.Fatalf("包裹引用了不存在的方案 %q", p.OfferID)
		}
		sum += f
	}
	return sum
}

// checkValid 校验规划结果：每个单元恰好出现一次、无空包裹、
// 重量不超限、危险品约束满足。
func checkValid(t *testing.T, order Order, offers []Offer, parcels []Parcel) {
	t.Helper()
	unitByID := make(map[string]Unit)
	for _, u := range order.Units {
		unitByID[u.ID] = u
	}
	offerByID := make(map[string]Offer)
	for _, o := range offers {
		offerByID[o.ID] = o
	}
	seen := make(map[string]int)
	for _, p := range parcels {
		if len(p.UnitIDs) == 0 {
			t.Fatalf("存在空包裹")
		}
		o := offerByID[p.OfferID]
		accepted := make(map[HazardClass]bool)
		for _, h := range o.AcceptedHazards {
			accepted[h] = true
		}
		weight := 0
		hazards := make(map[HazardClass]bool)
		for _, id := range p.UnitIDs {
			u, ok := unitByID[id]
			if !ok {
				t.Fatalf("包裹包含未知单元 %q", id)
			}
			seen[id]++
			weight += u.WeightGrams
			if u.Hazardous {
				hazards[u.Hazard] = true
			}
		}
		if weight > o.MaxWeightGrams {
			t.Fatalf("包裹超重: %d > %d", weight, o.MaxWeightGrams)
		}
		if len(hazards) > 1 {
			t.Fatalf("包裹含多种危险品二元组: %v", hazards)
		}
		for h := range hazards {
			if !accepted[h] {
				t.Fatalf("方案 %q 不可承运危险品 %v", o.ID, h)
			}
		}
	}
	if len(seen) != len(order.Units) {
		t.Fatalf("覆盖单元数 %d != 订单单元数 %d", len(seen), len(order.Units))
	}
	for id, c := range seen {
		if c != 1 {
			t.Fatalf("单元 %q 出现 %d 次", id, c)
		}
	}
}

// 场景一：单一方案，上限 10000 克、费用 100 美分，
// 六件普通货物 6000/5000/3000/2000/2000/2000，最优两包、200 美分。
func TestSingleOfferGeneralCargo(t *testing.T) {
	weights := []int{6000, 5000, 3000, 2000, 2000, 2000}
	offers := []Offer{{ID: "std", MaxWeightGrams: 10000, FeeCents: 100}}

	build := func(ws []int) Order {
		var o Order
		for i, w := range ws {
			o.Units = append(o.Units, Unit{ID: string(rune('a' + i)), WeightGrams: w})
		}
		return o
	}

	parcels, ok := Plan(build(weights), offers)
	if !ok {
		t.Fatal("应有解")
	}
	if len(parcels) != 2 {
		t.Fatalf("包裹数 = %d, 期望 2", len(parcels))
	}
	if fee := totalFee(t, parcels, offers); fee != 200 {
		t.Fatalf("总费用 = %d, 期望 200", fee)
	}
	checkValid(t, build(weights), offers, parcels)

	// 打乱输入顺序不应改变最优值。
	rng := rand.New(rand.NewSource(42))
	for trial := 0; trial < 20; trial++ {
		ws := append([]int(nil), weights...)
		rng.Shuffle(len(ws), func(i, j int) { ws[i], ws[j] = ws[j], ws[i] })
		order := build(ws)
		parcels, ok := Plan(order, offers)
		if !ok {
			t.Fatal("应有解")
		}
		if len(parcels) != 2 {
			t.Fatalf("打乱后包裹数 = %d, 期望 2", len(parcels))
		}
		if fee := totalFee(t, parcels, offers); fee != 200 {
			t.Fatalf("打乱后总费用 = %d, 期望 200", fee)
		}
		checkValid(t, order, offers, parcels)
	}
}

// 场景二：两件同 UN 不同包装等级的危险品各 6000 克，两件普通货物各 4000 克。
// 方案 A：上限 8000 克、400 美分；方案 B：上限 12000 克、600 美分。
// 两种危险品不能同包，最优两包、1200 美分（各走方案 B）。
func TestHazmatDifferentPackingGroups(t *testing.T) {
	hazI := HazardClass{UNNumber: "UN1234", PackingGroup: PackingGroupI}
	hazII := HazardClass{UNNumber: "UN1234", PackingGroup: PackingGroupII}
	order := Order{Units: []Unit{
		{ID: "h1", WeightGrams: 6000, Hazardous: true, Hazard: hazI},
		{ID: "h2", WeightGrams: 6000, Hazardous: true, Hazard: hazII},
		{ID: "g1", WeightGrams: 4000},
		{ID: "g2", WeightGrams: 4000},
	}}
	offers := []Offer{
		{ID: "A", MaxWeightGrams: 8000, FeeCents: 400, AcceptedHazards: []HazardClass{hazI, hazII}},
		{ID: "B", MaxWeightGrams: 12000, FeeCents: 600, AcceptedHazards: []HazardClass{hazI, hazII}},
	}

	parcels, ok := Plan(order, offers)
	if !ok {
		t.Fatal("应有解")
	}
	if len(parcels) != 2 {
		t.Fatalf("包裹数 = %d, 期望 2", len(parcels))
	}
	if fee := totalFee(t, parcels, offers); fee != 1200 {
		t.Fatalf("总费用 = %d, 期望 1200", fee)
	}
	checkValid(t, order, offers, parcels)
}

// 无解：危险品不被任何方案接受。
func TestUnsolvable(t *testing.T) {
	order := Order{Units: []Unit{
		{ID: "h1", WeightGrams: 100, Hazardous: true,
			Hazard: HazardClass{UNNumber: "UN9999", PackingGroup: PackingGroupI}},
	}}
	offers := []Offer{{ID: "std", MaxWeightGrams: 10000, FeeCents: 100}}
	if _, ok := Plan(order, offers); ok {
		t.Fatal("应返回无解")
	}
}

// 平局时优先更少包裹：两个方案费用相同，应选能装下全部的大上限方案（一包）。
func TestTieBreakFewerParcels(t *testing.T) {
	order := Order{Units: []Unit{
		{ID: "a", WeightGrams: 6000},
		{ID: "b", WeightGrams: 6000},
	}}
	offers := []Offer{
		{ID: "small", MaxWeightGrams: 8000, FeeCents: 100},
		{ID: "large", MaxWeightGrams: 12000, FeeCents: 200},
	}
	parcels, ok := Plan(order, offers)
	if !ok {
		t.Fatal("应有解")
	}
	if len(parcels) != 1 || parcels[0].OfferID != "large" {
		t.Fatalf("期望单包走 large, 得到 %+v", parcels)
	}
	checkValid(t, order, offers, parcels)
}
