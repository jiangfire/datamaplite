package model

import "time"

// UserRole 用户角色
type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

// User 用户信息
type User struct {
	ID           string    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	Role         UserRole  `json:"role" db:"role"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int64     `json:"expires_in"`
	User         *UserInfo `json:"user"`
}

// RefreshTokenRequest 刷新Token请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string   `json:"username" binding:"required,min=3,max=32"`
	Password string   `json:"password" binding:"required,min=6,max=128"`
	Email    string   `json:"email" binding:"required,email"`
	Role     UserRole `json:"role" binding:"omitempty,oneof=admin user"`
}

// UpdateUserRequest 更新用户请求（管理员使用）
type UpdateUserRequest struct {
	Email    *string   `json:"email,omitempty" binding:"omitempty,email"`
	Password *string   `json:"password,omitempty" binding:"omitempty,min=6,max=128"`
	Role     *UserRole `json:"role,omitempty" binding:"omitempty,oneof=admin user"`
}

// UserInfo 用户信息（不含敏感字段）
type UserInfo struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      UserRole  `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// ToUserInfo 转换为UserInfo
func (u *User) ToUserInfo() *UserInfo {
	return &UserInfo{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}
}

// Claims JWT Claims
type Claims struct {
	UserID    string   `json:"user_id"`
	Username  string   `json:"username"`
	Role      UserRole `json:"role"`
	TokenType string   `json:"token_type"`
}

// AuthContext 认证上下文（存储在gin.Context中）
type AuthContext struct {
	UserID   string
	Username string
	Role     UserRole
}
