package service

import (
	"context"
	"fmt"
	"time"

	"git.neolidy.top/neo/fuckcmdb/internal/model"
	"git.neolidy.top/neo/fuckcmdb/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthConfig 认证配置
type AuthConfig struct {
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	BcryptCost      int
}

// DefaultAuthConfig 默认认证配置
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		JWTSecret:       "your-secret-key-change-in-production",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		BcryptCost:      10,
	}
}

// AuthService 认证服务
type AuthService struct {
	store  store.Store
	config *AuthConfig
}

// NewAuthService 创建认证服务
func NewAuthService(store store.Store, config *AuthConfig) *AuthService {
	if config == nil {
		config = DefaultAuthConfig()
	}
	return &AuthService{
		store:  store,
		config: config,
	}
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.LoginResponse, error) {
	// 获取用户
	user, err := s.store.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	// 生成Token
	accessToken, err := s.generateToken(user.ID, user.Username, model.UserRole(user.Role), "access")
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateToken(user.ID, user.Username, model.UserRole(user.Role), "refresh")
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.config.AccessTokenTTL.Seconds()),
		User: &model.UserInfo{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      model.UserRole(user.Role),
			CreatedAt: parseTime(user.CreatedAt),
		},
	}, nil
}

// RefreshToken 刷新Access Token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*model.LoginResponse, error) {
	// 验证Refresh Token
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("invalid token type")
	}

	// 获取用户信息
	user, err := s.store.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// 生成新Token
	newAccessToken, err := s.generateToken(user.ID, user.Username, model.UserRole(user.Role), "access")
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.generateToken(user.ID, user.Username, model.UserRole(user.Role), "refresh")
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &model.LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.config.AccessTokenTTL.Seconds()),
		User: &model.UserInfo{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      model.UserRole(user.Role),
			CreatedAt: parseTime(user.CreatedAt),
		},
	}, nil
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, req *model.RegisterRequest) (*model.UserInfo, error) {
	// 检查用户名是否已存在
	_, err := s.store.GetUserByUsername(ctx, req.Username)
	if err == nil {
		return nil, fmt.Errorf("username already exists")
	}

	// 加密密码
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.config.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 设置默认角色
	role := string(model.UserRoleUser)
	if req.Role != "" {
		role = string(req.Role)
	}

	// 创建用户
	userCreate := &store.UserCreate{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(passwordHash),
		Role:         role,
	}

	id, err := s.store.CreateUser(ctx, userCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 获取创建的用户
	user, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get created user: %w", err)
	}

	return &model.UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      model.UserRole(user.Role),
		CreatedAt: parseTime(user.CreatedAt),
	}, nil
}

// GetUserByID 根据ID获取用户
func (s *AuthService) GetUserByID(ctx context.Context, id string) (*model.UserInfo, error) {
	user, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &model.UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      model.UserRole(user.Role),
		CreatedAt: parseTime(user.CreatedAt),
	}, nil
}

// generateToken 生成JWT Token
func (s *AuthService) generateToken(userID, username string, role model.UserRole, tokenType string) (string, error) {
	now := time.Now()
	var expiry time.Time
	if tokenType == "access" {
		expiry = now.Add(s.config.AccessTokenTTL)
	} else {
		expiry = now.Add(s.config.RefreshTokenTTL)
	}

	claims := jwt.MapClaims{
		"user_id":    userID,
		"username":   username,
		"role":       string(role),
		"token_type": tokenType,
		"exp":        expiry.Unix(),
		"iat":        now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

// ParseToken 解析JWT Token
func (s *AuthService) ParseToken(tokenString string) (*model.Claims, error) {
	return s.parseToken(tokenString)
}

// parseToken 解析JWT Token（内部方法）
func (s *AuthService) parseToken(tokenString string) (*model.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	return &model.Claims{
		UserID:    getStringClaim(claims, "user_id"),
		Username:  getStringClaim(claims, "username"),
		Role:      model.UserRole(getStringClaim(claims, "role")),
		TokenType: getStringClaim(claims, "token_type"),
	}, nil
}

// getStringClaim 安全地获取字符串claim
func getStringClaim(claims jwt.MapClaims, key string) string {
	val, ok := claims[key].(string)
	if !ok {
		return ""
	}
	return val
}

// parseTime 解析时间字符串
func parseTime(timeStr string) time.Time {
	t, _ := time.Parse(time.RFC3339, timeStr)
	return t
}
