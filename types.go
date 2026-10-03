// Package freightplanner 提供货运订单拆包规划能力：
// 联合决定每个包裹的货物组成与承运方案，使总费用最优。
package freightplanner

// PackingGroup 包装等级（危险品）。
type PackingGroup int

const (
	PackingGroupI   PackingGroup = 1
	PackingGroupII  PackingGroup = 2
	PackingGroupIII PackingGroup = 3
)

// HazardClass 危险品二元组：UN 编号 + 包装等级。
type HazardClass struct {
	UNNumber     string
	PackingGroup PackingGroup
}

// Unit 不可拆分的包装单元。
type Unit struct {
	ID          string
	WeightGrams int
	Hazardous   bool
	Hazard      HazardClass // 仅当 Hazardous 为 true 时有效
}

// Offer 承运方案。
type Offer struct {
	ID              string
	MaxWeightGrams  int
	FeeCents        int64
	AcceptedHazards []HazardClass // 可承运的危险品二元组集合；普通货物均可承运
}

// Order 待规划订单。
type Order struct {
	Units []Unit
}

// Parcel 规划结果中的一个包裹。
type Parcel struct {
	UnitIDs []string
	OfferID string
}

// Plan 对订单进行拆包规划。
// 目标：先最小化总费用（各包裹所选方案固定费用之和），
// 费用相同时最小化包裹数，其余平局任选。
// 返回包裹列表；若无解，ok 为 false。
func Plan(order Order, offers []Offer) (parcels []Parcel, ok bool) {
	return planImpl(order, offers)
}
