# freightplan

货运订单拆包规划库：联合决定每个包裹的货物组成与承运方案，
先最小化总费用，费用相同时再最小化包裹数，其余平局任选。

## 使用

```go
order := []freightplan.Unit{
    {ID: "a", WeightG: 6000},                                        // 普通货物
    {ID: "dg", WeightG: 6000, Danger: &freightplan.DangerClass{      // 危险品
        UN: "UN1263", Group: freightplan.PackingGroupII,
    }},
}
offers := []freightplan.Offer{
    {ID: "std", MaxWeightG: 10000, FeeCents: 100,
     Dangerous: []freightplan.DangerClass{{UN: "UN1263", Group: freightplan.PackingGroupII}}},
}

res, ok := freightplan.Plan(order, offers)
if !ok {
    // 无解：某单元超过所有方案上限，或危险品无人承运
}
fmt.Println(res.TotalCostCents, res.Parcels)
```

## 规则

- 每个包裹最多一种危险品二元组（UN 编号 + 包装等级），可混放任意普通货物。
- 包裹总重量不得超过所选方案的 `MaxWeightG`，危险品须在方案的 `Dangerous` 集合内。
- 所有方案均接受普通货物，包裹数量不限，费用按包裹累计（精确整数）。
- 每个单元恰好出现一次，不允许空包裹。

## 测试

```
go test ./...
```

## 实现说明

`Plan` 使用分支限界搜索：单元按重量降序分配，对每个单元尝试放入
现有包裹或用每个方案开新包裹；以「剩余重量减去现有空闲容量」除以
最大上限、乘最低费用作为费用下界剪枝，保证返回全局最优解。
