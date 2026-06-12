package service

import (
	"testing"

	"github.com/alaikis/opentether/internal/config"
	"github.com/alaikis/opentether/internal/models"
	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func newTestAuthService(t *testing.T) (*AuthService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}

	return NewAuthService(db, &config.Config{
		Security: config.SecurityConfig{
			JWT: config.JWTConfig{
				Secret: "test-secret",
				Expire: "24h",
			},
		},
	}), db
}

func TestUserServiceUpdatePassword(t *testing.T) {
	svc, db := newTestAuthService(t)

	// Create user with known password
	hash, err := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash old password: %v", err)
	}
	user := models.User{
		GlobalUserID: "testuser",
		Name:         "测试用户",
		Email:        "test@example.com",
		Role:         models.RoleUser,
		Status:       "active",
		PasswordHash: string(hash),
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Old password should work
	_, _, err = svc.Login("testuser", "oldpass")
	if err != nil {
		t.Fatalf("login with old password: %v", err)
	}

	// Update password via UserService
	userSvc := NewUserService(db)
	if _, err := userSvc.Update(user.ID, UpdateUserInput{
		Name:     user.Name,
		Email:    user.Email,
		Role:     user.Role,
		Status:   user.Status,
		Password: "newpass",
	}); err != nil {
		t.Fatalf("update password: %v", err)
	}

	// Old password should be rejected
	_, _, err = svc.Login("testuser", "oldpass")
	if err == nil {
		t.Fatal("expected old password to be rejected after update")
	}

	// New password should work
	_, _, err = svc.Login("testuser", "newpass")
	if err != nil {
		t.Fatalf("login with new password: %v", err)
	}
}

func TestAuthServiceLoginWithDefaultAdmin(t *testing.T) {
	svc, db := newTestAuthService(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := models.User{
		GlobalUserID: "admin",
		Name:         "管理员",
		Email:        "admin@example.com",
		Role:         models.RoleAdmin,
		Status:       "active",
		PasswordHash: string(hash),
		CreatedBy:    "system",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	token, got, err := svc.Login("admin", "admin123")
	if err != nil {
		t.Fatalf("login default admin: %v", err)
	}
	if token == "" {
		t.Fatal("expected token")
	}
	if got.GlobalUserID != "admin" || got.Role != models.RoleAdmin {
		t.Fatalf("unexpected user: global_user_id=%q role=%q", got.GlobalUserID, got.Role)
	}
}
