package main

import (
	"log"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Skill struct {
	ID              string `gorm:"primaryKey"`
	Name            string
	SkillType       string
	Description     string
	Keywords        string
	Category        string
	Enabled         bool
	Config          string
	PromptTemplate  string
	AllowedGroups   string
	DataScope       string
	RequireApproval bool
}

func main() {
	db, err := gorm.Open(sqlite.Open("data/opentether.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	dsID := "1cce62ad-7c32-4179-8b2b-581f0edd1e87" // nw-mysql
	dsName := "nw-mysql"

	skills := []Skill{
		{
			ID:          uuid.New().String(),
			Name:        "销售业绩查询",
			SkillType:   "text2sql",
			Description: "查询销售业绩：按员工、部门、产品、时间段统计销售额、数量、利润。数据源：" + dsName,
			Keywords:    "销售,业绩,销售额,订单,利润,销售排名,销售趋势,销售统计",
			Category:    "nw-mysql-业务",
			Enabled:     true,
			Config:      `{"tool":"text2sql","data_source_id":"` + dsID + `","data_source_name":"` + dsName + `","policy":{"allowed_tables":["t_order","t_order_item","t_order_amount","t_order_sales_profit","t_variants","t_item","t_profile"],"max_rows":500}}`,
		},
		{
			ID:          uuid.New().String(),
			Name:        "成本分析查询",
			SkillType:   "text2sql",
			Description: "查询采购成本、库存成本、单品成本，支持按产品/供应商/仓库/时间分析成本趋势。数据源：" + dsName,
			Keywords:    "成本,采购,库存成本,成本分析,成本趋势,成本对比",
			Category:    "nw-mysql-业务",
			Enabled:     true,
			Config:      `{"tool":"text2sql","data_source_id":"` + dsID + `","data_source_name":"` + dsName + `","policy":{"allowed_tables":["t_inventory_cost_summary","t_inventory_cost_log","t_purchase","t_purchase_item","t_variants","t_item","t_warehouse"],"max_rows":500}}`,
		},
		{
			ID:          uuid.New().String(),
			Name:        "库存查询",
			SkillType:   "text2sql",
			Description: "查询当前库存：各仓库各产品的实时库存数量、库存价值、可用库存。支持按仓库/产品/SKU 过滤。数据源：" + dsName,
			Keywords:    "库存,仓库,库存数量,库存价值,可用库存,库存状态",
			Category:    "nw-mysql-业务",
			Enabled:     true,
			Config:      `{"tool":"text2sql","data_source_id":"` + dsID + `","data_source_name":"` + dsName + `","policy":{"allowed_tables":["t_inventory","t_variants","t_item","t_warehouse","t_inventory_cost_summary"],"max_rows":500}}`,
		},
		{
			ID:          uuid.New().String(),
			Name:        "产品目录查询",
			SkillType:   "text2sql",
			Description: "查询产品信息：产品名称、规格、价格、成本、供应商、仓库等信息。支持按分类/品牌/SKU 筛选。数据源：" + dsName,
			Keywords:    "产品,商品,SKU,价格,规格,品牌,分类,产品信息",
			Category:    "nw-mysql-业务",
			Enabled:     true,
			Config:      `{"tool":"text2sql","data_source_id":"` + dsID + `","data_source_name":"` + dsName + `","policy":{"allowed_tables":["t_item","t_variants","t_item_type","t_brand","t_supplier","t_resource_categories"],"max_rows":500}}`,
		},
		{
			ID:          uuid.New().String(),
			Name:        "采购管理查询",
			SkillType:   "text2sql",
			Description: "查询采购记录：采购订单、供应商、采购数量、采购金额、采购状态等。数据源：" + dsName,
			Keywords:    "采购,供应商,采购订单,采购金额,采购状态,进货",
			Category:    "nw-mysql-业务",
			Enabled:     true,
			Config:      `{"tool":"text2sql","data_source_id":"` + dsID + `","data_source_name":"` + dsName + `","policy":{"allowed_tables":["t_purchase","t_purchase_item","t_supplier","t_purchase_pecking","t_purchase_item_related_order_item_mapping"],"max_rows":500}}`,
		},
		{
			ID:          uuid.New().String(),
			Name:        "员工销售分析",
			SkillType:   "text2sql",
			Description: "按员工分析销售业绩：每个员工的销售额、订单数、利润，支持排名和趋势。数据源：" + dsName,
			Keywords:    "员工,销售员,销售分析,个人业绩,员工排名,销售对比",
			Category:    "nw-mysql-业务",
			Enabled:     true,
			Config:      `{"tool":"text2sql","data_source_id":"` + dsID + `","data_source_name":"` + dsName + `","policy":{"allowed_tables":["t_order","t_order_item","t_order_sales_profit","t_profile","t_depart"],"max_rows":500,"data_scope":"specified"}}`,
		},
		{
			ID:          uuid.New().String(),
			Name:        "部门销售分析",
			SkillType:   "text2sql",
			Description: "按部门分析销售数据：各部门销售额、利润、订单数，支持部门间对比和趋势分析。数据源：" + dsName,
			Keywords:    "部门,部门销售,部门业绩,部门排名,部门对比",
			Category:    "nw-mysql-业务",
			Enabled:     true,
			Config:      `{"tool":"text2sql","data_source_id":"` + dsID + `","data_source_name":"` + dsName + `","policy":{"allowed_tables":["t_order","t_order_item","t_order_sales_profit","t_depart","t_profile"],"max_rows":500,"data_scope":"department"}}`,
		},
		{
			ID:          uuid.New().String(),
			Name:        "销售利润分析",
			SkillType:   "text2sql",
			Description: "分析销售利润：售价、成本价、采购价对比，计算毛利率。按产品/客户/员工/时间维度。数据源：" + dsName,
			Keywords:    "利润,毛利率,销售利润,利润分析,利润排名",
			Category:    "nw-mysql-业务",
			Enabled:     true,
			Config:      `{"tool":"text2sql","data_source_id":"` + dsID + `","data_source_name":"` + dsName + `","policy":{"allowed_tables":["t_order_sales_profit","t_order","t_order_item","t_variants","t_item","t_member"],"max_rows":500}}`,
		},
		{
			ID:          uuid.New().String(),
			Name:        "客户管理查询",
			SkillType:   "text2sql",
			Description: "查询客户信息：客户名称、公司、联系人、地址、客户组、关联销售员。数据源：" + dsName,
			Keywords:    "客户,客户信息,客户群,会员,客户地址",
			Category:    "nw-mysql-业务",
			Enabled:     true,
			Config:      `{"tool":"text2sql","data_source_id":"` + dsID + `","data_source_name":"` + dsName + `","policy":{"allowed_tables":["t_member","t_member_communication","t_address","t_marketing_event","t_special_price_rule"],"max_rows":500}}`,
		},
		{
			ID:          uuid.New().String(),
			Name:        "物流追踪查询",
			SkillType:   "text2sql",
			Description: "查询物流信息：包裹、追踪号、物流状态、仓库发货记录等。数据源：" + dsName,
			Keywords:    "物流,快递,包裹,追踪号,发货,物流状态",
			Category:    "nw-mysql-业务",
			Enabled:     true,
			Config:      `{"tool":"text2sql","data_source_id":"` + dsID + `","data_source_name":"` + dsName + `","policy":{"allowed_tables":["t_order_packages","t_logistics_tracking_events","t_warehouse","t_pickup_task","t_pickup_task_packages"],"max_rows":500}}`,
		},
		{
			ID:          uuid.New().String(),
			Name:        "广告效果分析",
			SkillType:   "text2sql",
			Description: "查询广告投入产出：广告花费、展示量、转化率，按广告组/员工/时间段分析。数据源：" + dsName,
			Keywords:    "广告,推广,广告花费,转化,广告效果,ROI",
			Category:    "nw-mysql-业务",
			Enabled:     true,
			Config:      `{"tool":"text2sql","data_source_id":"` + dsID + `","data_source_name":"` + dsName + `","policy":{"allowed_tables":["t_adv_result","t_adv","t_adv_space","t_profile"],"max_rows":500}}`,
		},
	}

	for _, sk := range skills {
		if err := db.Create(&sk).Error; err != nil {
			log.Printf("创建 Skill 失败 %s: %v", sk.Name, err)
		} else {
			log.Printf("创建 Skill: %s (%s)", sk.Name, sk.ID)
		}
	}
}
