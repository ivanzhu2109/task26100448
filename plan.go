package freightplanner

import (
	"math"
	"sort"
)

// bestCarrier 记录某个单元子集作为单个包裹时的最优承运方案。
type bestCarrier struct {
	feasible bool
	fee      int64
	offerID  string
}

// dpState 是 DP 状态值：累计费用与包裹数，按 (fee, count) 字典序比较。
type dpState struct {
	fee   int64
	count int
}

func lessState(a, b dpState) bool {
	if a.fee != b.fee {
		return a.fee < b.fee
	}
	return a.count < b.count
}

// Plan 的实现见 types.go 中的文档注释。
//
// 算法：枚举所有单元子集，判断其能否作为合法包裹（重量不超限、
// 至多一种危险品二元组且被方案接受），并记录承运该子集费用最低的方案；
// 随后在子集掩码上做精确划分 DP，按 (总费用, 包裹数) 字典序取最优。
// 时间复杂度约为 O(3^n)，适用于中小规模订单。
func planImpl(order Order, offers []Offer) ([]Parcel, bool) {
	n := len(order.Units)
	if n == 0 {
		return nil, true
	}
	if len(offers) == 0 {
		return nil, false
	}

	// 预处理每个方案接受的危险品二元组集合。
	accepted := make([]map[HazardClass]bool, len(offers))
	for i, o := range offers {
		accepted[i] = make(map[HazardClass]bool, len(o.AcceptedHazards))
		for _, h := range o.AcceptedHazards {
			accepted[i][h] = true
		}
	}

	// best[mask]：该子集作为单个包裹时的最优承运方案。
	size := 1 << n
	best := make([]bestCarrier, size)
	best[0] = bestCarrier{feasible: false}

	// 子集重量与危险品二元组状态。
	weight := make([]int, size)
	hazardKey := make([]HazardClass, size)
	hazardState := make([]int8, size) // 0=无危险品, 1=恰好一种, 2=冲突

	for mask := 1; mask < size; mask++ {
		lb := mask & (-mask)
		i := bitIndex(lb)
		prev := mask ^ lb
		weight[mask] = weight[prev] + order.Units[i].WeightGrams
		hazardState[mask] = hazardState[prev]
		hazardKey[mask] = hazardKey[prev]
		if order.Units[i].Hazardous {
			switch hazardState[prev] {
			case 0:
				hazardState[mask] = 1
				hazardKey[mask] = order.Units[i].Hazard
			case 1:
				if hazardKey[prev] != order.Units[i].Hazard {
					hazardState[mask] = 2
				}
			}
		}
	}

	for mask := 1; mask < size; mask++ {
		if hazardState[mask] == 2 {
			continue
		}
		bestFee := int64(math.MaxInt64)
		bestOffer := ""
		for i, o := range offers {
			if weight[mask] > o.MaxWeightGrams {
				continue
			}
			if hazardState[mask] == 1 && !accepted[i][hazardKey[mask]] {
				continue
			}
			if o.FeeCents < bestFee {
				bestFee = o.FeeCents
				bestOffer = o.ID
			}
		}
		if bestFee != int64(math.MaxInt64) {
			best[mask] = bestCarrier{feasible: true, fee: bestFee, offerID: bestOffer}
		}
	}

	// DP：dp[mask] 为覆盖 mask 的最优 (费用, 包裹数)。
	const infFee = int64(math.MaxInt64 / 4)
	dp := make([]dpState, size)
	choice := make([]int, size) // 记录转移时选择的子包裹掩码
	for mask := 1; mask < size; mask++ {
		dp[mask] = dpState{fee: infFee, count: math.MaxInt32}
	}
	for mask := 1; mask < size; mask++ {
		lb := mask & (-mask)
		// 枚举包含最低位的子掩码，保证每个划分只被考虑一次。
		for sub := mask; sub > 0; sub = (sub - 1) & mask {
			if sub&lb == 0 {
				continue
			}
			if !best[sub].feasible {
				continue
			}
			prev := dp[mask^sub]
			if prev.fee >= infFee {
				continue
			}
			cand := dpState{fee: prev.fee + best[sub].fee, count: prev.count + 1}
			if lessState(cand, dp[mask]) {
				dp[mask] = cand
				choice[mask] = sub
			}
		}
	}

	full := size - 1
	if dp[full].fee >= infFee {
		return nil, false
	}

	// 回溯构造包裹列表。
	var parcels []Parcel
	for mask := full; mask != 0; {
		sub := choice[mask]
		var ids []string
		for m := sub; m != 0; m &= m - 1 {
			i := bitIndex(m & (-m))
			ids = append(ids, order.Units[i].ID)
		}
		sort.Strings(ids)
		parcels = append(parcels, Parcel{UnitIDs: ids, OfferID: best[sub].offerID})
		mask ^= sub
	}
	return parcels, true
}

func bitIndex(lb int) int {
	i := 0
	for lb > 1 {
		lb >>= 1
		i++
	}
	return i
}
