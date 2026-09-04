package router

import (
	"testing"
	"github.com/go-think/think/contract"
)

func TestGroupInheritance(t *testing.T) {
	r := New()
	r.Prefix("/admin").Name("admin.").Group(func(group contract.Router) {
		group.Get("/profile", func() {}).Name("profile")
		group.Prefix("/api").Name("api.").Group(func(api contract.Router) {
			api.Get("/users", func() {}).Name("users")
		})
	})
	
	r.Register()
	
	url1 := r.Url("admin.profile", nil)
	if url1 != "/admin/profile" {
		t.Errorf("Expected /admin/profile, got %s", url1)
	}
	
	url2 := r.Url("admin.api.users", nil)
	if url2 != "/admin/api/users" {
		t.Errorf("Expected /admin/api/users, got %s", url2)
	}
}
