package freightplan

import "sort"

// parcel is the mutable search state of one open parcel.
type parcel struct {
	offerIdx int
	weight   int64
	danger   *DangerClass // nil: no dangerous goods yet
	units    []int        // indexes into the sorted unit list
}

// Plan computes a globally optimal packing for the order: total cost is
// minimized first, then the number of parcels. Remaining ties are broken
// arbitrarily. ok is false when no feasible packing exists.
func Plan(order []Unit, offers []Offer) (Result, bool) {
	if err := validate(order, offers); err != nil {
		return Result{}, false
	}
	if len(order) == 0 {
		return Result{Parcels: nil, TotalCostCents: 0}, true
	}

	accepts := make([]map[DangerClass]bool, len(offers))
	for i, o := range offers {
		accepts[i] = make(map[DangerClass]bool, len(o.Dangerous))
		for _, d := range o.Dangerous {
			accepts[i][d] = true
		}
	}

	// Sort units by weight descending for better pruning; keep original IDs.
	idx := make([]int, len(order))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool {
		return order[idx[a]].WeightG > order[idx[b]].WeightG
	})
	units := make([]Unit, len(order))
	for i, j := range idx {
		units[i] = order[j]
	}

	// Remaining-weight suffix sums for the lower bound.
	remWeight := make([]int64, len(units)+1)
	for i := len(units) - 1; i >= 0; i-- {
		remWeight[i] = remWeight[i+1] + units[i].WeightG
	}
	var maxCap int64
	var minFee int64 = -1
	for _, o := range offers {
		if o.MaxWeightG > maxCap {
			maxCap = o.MaxWeightG
		}
		if minFee < 0 || o.FeeCents < minFee {
			minFee = o.FeeCents
		}
	}

	bestCost := int64(-1)
	bestCount := 0
	var best []parcel
	parcels := make([]parcel, 0, len(units))

	var search func(pos int, cost int64)
	search = func(pos int, cost int64) {
		if bestCost >= 0 {
			if cost > bestCost {
				return
			}
			// Lower bound: weight that does not fit into the free
			// capacity of existing parcels needs additional parcels
			// at the cheapest fee.
			var free int64
			for i := range parcels {
				free += offers[parcels[i].offerIdx].MaxWeightG - parcels[i].weight
			}
			need := remWeight[pos] - free
			var extra int64
			if need > 0 {
				extra = need / maxCap
				if need%maxCap != 0 {
					extra++
				}
			}
			lower := cost + extra*minFee
			if lower > bestCost {
				return
			}
			if lower == bestCost && len(parcels)+int(extra) >= bestCount {
				return
			}
		}
		if pos == len(units) {
			if bestCost < 0 || cost < bestCost || (cost == bestCost && len(parcels) < bestCount) {
				bestCost = cost
				bestCount = len(parcels)
				best = make([]parcel, len(parcels))
				for i, p := range parcels {
					cp := p
					cp.units = append([]int(nil), p.units...)
					best[i] = cp
				}
			}
			return
		}

		u := units[pos]
		// Try existing parcels.
		for i := range parcels {
			p := &parcels[i]
			if p.weight+u.WeightG > offers[p.offerIdx].MaxWeightG {
				continue
			}
			if u.Danger != nil {
				if !accepts[p.offerIdx][*u.Danger] {
					continue
				}
				if p.danger != nil && *p.danger != *u.Danger {
					continue
				}
			}
			prevDanger := p.danger
			if u.Danger != nil {
				d := *u.Danger
				p.danger = &d
			}
			p.weight += u.WeightG
			p.units = append(p.units, pos)
			search(pos+1, cost)
			p.units = p.units[:len(p.units)-1]
			p.weight -= u.WeightG
			p.danger = prevDanger
		}
		// Open a new parcel with each offer.
		for oi, o := range offers {
			if u.WeightG > o.MaxWeightG {
				continue
			}
			if u.Danger != nil && !accepts[oi][*u.Danger] {
				continue
			}
			var d *DangerClass
			if u.Danger != nil {
				dc := *u.Danger
				d = &dc
			}
			parcels = append(parcels, parcel{
				offerIdx: oi,
				weight:   u.WeightG,
				danger:   d,
				units:    []int{pos},
			})
			search(pos+1, cost+o.FeeCents)
			parcels = parcels[:len(parcels)-1]
		}
	}
	search(0, 0)

	if bestCost < 0 {
		return Result{}, false
	}
	res := Result{TotalCostCents: bestCost}
	for _, p := range best {
		ids := make([]string, len(p.units))
		for i, u := range p.units {
			ids[i] = units[u].ID
		}
		res.Parcels = append(res.Parcels, Parcel{
			OfferID: offers[p.offerIdx].ID,
			UnitIDs: ids,
		})
	}
	return res, true
}
