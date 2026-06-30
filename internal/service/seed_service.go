package service

import (
	"encoding/json"
	"time"

	"github.com/alaikis/opentether/internal/models"
	"gorm.io/gorm"
)

func SeedInitialData(db *gorm.DB) error {
	var count int64
	db.Model(&models.SkillRuntimeMemory{}).Where("type = ? AND source = ?", "text2sql_template", "admin").Count(&count)
	if count > 0 {
		return nil
	}

	var skill models.Skill
	err := db.Where("skill_type = ? AND enabled = ?", "text2sql", true).First(&skill).Error
	if err != nil {
		return nil
	}

	var ds models.DataSource
	err = db.Where("enabled = ?", true).First(&ds).Error
	if err != nil {
		return nil
	}

	templates := map[string]string{
		"employee_monthly_order_count":  `{"intent":"卖多少单,多少单,出了多少单,出单量,订单数,单量,出单,出了多少","chart_type":"bar","SQLTemplate":"SELECT DATE_FORMAT(t_order.delivery_time, '%Y-%m') AS month, COUNT(DISTINCT t_order.id) AS order_count FROM t_order JOIN t_profile ON t_order.sale_staff_id=t_profile.user_id WHERE t_profile.real_name={{employee_name}} AND t_order.delivery_time >= {{start_date}} AND t_order.delivery_time < {{end_date}} AND t_order.delivery_time IS NOT NULL AND t_order.order_status != 0 GROUP BY month ORDER BY month"}`,
		"employee_monthly_sales_amount": `{"intent":"销售额,卖多少钱,销售金额,业绩,卖了多少钱,累计多少钱,卖了多少,销售","chart_type":"bar","SQLTemplate":"SELECT DATE_FORMAT(t_order.delivery_time, '%Y-%m') AS month, SUM(t_order_amount.amount) AS sales_amount FROM t_order JOIN t_profile ON t_order.sale_staff_id=t_profile.user_id JOIN t_order_amount ON t_order_amount.order_id=t_order.id WHERE t_profile.real_name={{employee_name}} AND t_order.delivery_time >= {{start_date}} AND t_order.delivery_time < {{end_date}} AND t_order.delivery_time IS NOT NULL AND t_order.order_status != 0 AND t_order_amount.fee_code='sub_total' GROUP BY month ORDER BY month"}`,
		"employee_monthly_profit":       `{"intent":"利润,毛利,赚了多少,利润多少,利润量","chart_type":"bar","SQLTemplate":"SELECT DATE_FORMAT(t_order.delivery_time, '%Y-%m') AS month, SUM(t_order_sales_profit.qty*(t_order_sales_profit.sale_price-t_order_sales_profit.purchase_price)) AS profit FROM t_order JOIN t_profile ON t_order.sale_staff_id=t_profile.user_id JOIN t_order_sales_profit ON t_order_sales_profit.order_id=t_order.id WHERE t_profile.real_name={{employee_name}} AND t_order.delivery_time >= {{start_date}} AND t_order.delivery_time < {{end_date}} AND t_order.delivery_time IS NOT NULL AND t_order.order_status != 0 GROUP BY month ORDER BY month"}`,
		"country_sales_distribution":    `{"intent":"国家,国家分布,哪个国家,卖到哪些国家,国家排行","chart_type":"pie","SQLTemplate":"SELECT COALESCE(t_address.country_code,'未知') AS country, COUNT(DISTINCT t_order.id) AS order_count, SUM(t_order_amount.amount) AS sales_amount FROM t_order LEFT JOIN t_address ON t_order.shipping_address=t_address.id JOIN t_order_amount ON t_order_amount.order_id=t_order.id WHERE t_order.delivery_time >= {{start_date}} AND t_order.delivery_time < {{end_date}} AND t_order.delivery_time IS NOT NULL AND t_order.order_status != 0 AND t_order_amount.fee_code='sub_total' GROUP BY country ORDER BY sales_amount DESC"}`,
		"employee_sales_rank":           `{"intent":"排行,排名,销售排行,谁卖最多,排序","chart_type":"bar","SQLTemplate":"SELECT t_profile.real_name AS employee, COUNT(DISTINCT t_order.id) AS order_count, SUM(t_order_amount.amount) AS sales_amount FROM t_order JOIN t_profile ON t_order.sale_staff_id=t_profile.user_id JOIN t_order_amount ON t_order_amount.order_id=t_order.id WHERE t_order.delivery_time >= {{start_date}} AND t_order.delivery_time < {{end_date}} AND t_order.delivery_time IS NOT NULL AND t_order.order_status != 0 AND t_order_amount.fee_code='sub_total' GROUP BY employee ORDER BY sales_amount DESC"}`,
	}
	for key, content := range templates {
		db.Create(&models.SkillRuntimeMemory{
			SkillID:      skill.ID,
			DataSourceID: ds.ID,
			Type:         "text2sql_template",
			Key:          key,
			Content:      content,
			Confidence:   0.97,
			Source:       "admin",
			Status:       "active",
			LastUsedAt:   time.Now(),
		})
	}
	return nil
}

var _ = json.Marshal
