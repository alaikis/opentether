package agent

import "testing"

func TestHasConfiguredFullDataAccess(t *testing.T) {
	cfg := map[string]interface{}{"full_access_groups": []interface{}{"group-1", "sales_leads"}}
	if !hasConfiguredFullDataAccess(&UserContext{Role: "admin"}, map[string]interface{}{}) {
		t.Fatal("admin should have full access")
	}
	if !hasConfiguredFullDataAccess(&UserContext{Role: "user", Groups: []GroupContext{{Code: "sales_leads"}}}, cfg) {
		t.Fatal("configured group should have full access")
	}
	if hasConfiguredFullDataAccess(&UserContext{Role: "user", Groups: []GroupContext{{Code: "sales_manager"}}}, map[string]interface{}{}) {
		t.Fatal("unconfigured group name must not grant full access")
	}
}
