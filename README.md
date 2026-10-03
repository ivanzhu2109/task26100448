# freightplanner

货运订单拆包规划库：联合决定每个包裹的货物组成与承运方案，
先最小化总费用，费用相同时最小化包裹数，其余平局任选。

## 业务规则

- 订单由不可拆分的包装单元组成，各有唯一 ID、正整数克重量，
  并标记为普通货物或危险品（UN 编号 + 包装等级 I/II/III 二元组）。
- 承运方案含唯一 ID、单包重量上限、单包固定费用（美分，精确整数累计）、
  可承运的危险品二元组集合；所有方案均接受普通货物，包裹数量不限。
- 一个包裹内最多一种危险品二元组，可同时放任意普通货物；
  总重量不得超过所选方案上限；每个单元恰好出现一次，不允许空包裹。

## 使用

```go
import "freightplanner"

order := freightplanner.Order{Units: []freightplanner.Unit{
    {ID: "a", WeightGrams: 6000},
    {ID: "b", WeightGrams: 5000, Hazardous: true,
        Hazard: freightplanner.HazardClass{UNNumber: "UN1234", PackingGroup: freightplanner.PackingGroupII}},
}}
offers := []freightplanner.Offer{
    {ID: "std", MaxWeightGrams: 10000, FeeCents: 100,
        AcceptedHazards: []freightplanner.HazardClass{{UNNumber: "UN1234", PackingGroup: freightplanner.PackingGroupII}}},
}

parcels, ok := freightplanner.Plan(order, offers)
if !ok {
    // 无解：例如某危险品不被任何方案承运，或某单元超过所有方案上限
}
// parcels[i].UnitIDs 为包裹内单元 ID，parcels[i].OfferID 为所选方案
```

## 算法与复杂度

枚举所有单元子集，判定其能否作为合法包裹并记录费用最低的承运方案，
再在子集掩码上做精确划分 DP，按 (总费用, 包裹数) 字典序取全局最优。
时间复杂度约 O(3^n)，适用于中小规模订单（n 约 20 以内）。

## 测试

```sh
go test ./...
```
