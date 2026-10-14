package domain

import "time"

// UserRole описывает роль пользователя в системе команд RISC-V
type UserRole string

const (
	UserRoleUser      UserRole = "user"
	UserRoleModerator UserRole = "moderator"
)

// User описывает учётную запись пользователя
type User struct {
	ID           int64
	Login        string
	PasswordHash string
	Role         UserRole
}

// AuthenticatedUser описывает пользователя, подтверждённого JWT текущего запроса
type AuthenticatedUser struct {
	ID        int64
	Login     string
	Role      UserRole
	TokenID   string
	ExpiresAt time.Time
}

// AccessToken описывает выпущенный JWT доступа
type AccessToken struct {
	Value     string
	ExpiresIn time.Duration
	User      User
}
