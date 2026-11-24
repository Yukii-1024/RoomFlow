package user

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/Yukii-1024/RoomFlow/backend/internal/auth"
	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type Service struct {
	repo   *Repository
	tokens *auth.Manager
}

func NewService(repo *Repository, tokens *auth.Manager) *Service {
	return &Service{repo: repo, tokens: tokens}
}

type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Phone    string
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (User, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	name := strings.TrimSpace(in.Name)
	phone := strings.TrimSpace(in.Phone)
	if !emailPattern.MatchString(email) || len(email) > 254 {
		return User{}, httpx.BadRequest("邮箱格式不正确")
	}
	if err := auth.ValidatePassword(in.Password); err != nil {
		return User{}, httpx.BadRequest(err.Error())
	}
	if name == "" || len([]rune(name)) > 100 {
		return User{}, httpx.BadRequest("姓名不能为空且不能超过 100 个字符")
	}
	if len([]rune(phone)) > 32 {
		return User{}, httpx.BadRequest("电话号码不能超过 32 个字符")
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return User{}, httpx.Internal(err)
	}
	u, err := s.repo.Create(ctx, email, hash, name, phone, "customer")
	if err != nil {
		if isUniqueViolation(err, "users_email_lower_key") {
			return User{}, httpx.Conflict("EMAIL_ALREADY_EXISTS", "该邮箱已注册")
		}
		return User{}, httpx.Internal(err)
	}
	return u, nil
}

// Login returns a signed token, its lifetime and the authenticated user.
func (s *Service) Login(ctx context.Context, email, password string) (token string, expiresIn int, u User, err error) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return "", 0, User{}, httpx.BadRequest("邮箱和密码不能为空")
	}
	u, hash, err := s.repo.ByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", 0, User{}, httpx.NewError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "邮箱或密码错误")
		}
		return "", 0, User{}, httpx.Internal(err)
	}
	if !auth.CheckPassword(hash, password) {
		return "", 0, User{}, httpx.NewError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "邮箱或密码错误")
	}
	token, expiresIn, err = s.tokens.Issue(u.ID, u.Role)
	if err != nil {
		return "", 0, User{}, httpx.Internal(err)
	}
	return token, expiresIn, u, nil
}

func (s *Service) Me(ctx context.Context, userID int64) (User, error) {
	u, err := s.repo.ByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, httpx.NotFound("用户不存在")
	}
	if err != nil {
		return User{}, httpx.Internal(err)
	}
	return u, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID int64, name, phone *string) (User, error) {
	if name == nil && phone == nil {
		return User{}, httpx.BadRequest("至少需要提供一个待修改字段")
	}
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" || len([]rune(trimmed)) > 100 {
			return User{}, httpx.BadRequest("姓名不能为空且不能超过 100 个字符")
		}
		name = &trimmed
	}
	if phone != nil {
		trimmed := strings.TrimSpace(*phone)
		if len([]rune(trimmed)) > 32 {
			return User{}, httpx.BadRequest("电话号码不能超过 32 个字符")
		}
		phone = &trimmed
	}
	u, err := s.repo.UpdateProfile(ctx, userID, name, phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, httpx.NotFound("用户不存在")
	}
	if err != nil {
		return User{}, httpx.Internal(err)
	}
	return u, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID int64, current, next string) error {
	if current == "" || next == "" {
		return httpx.BadRequest("当前密码和新密码不能为空")
	}
	if err := auth.ValidatePassword(next); err != nil {
		return httpx.BadRequest(err.Error())
	}
	hash, err := s.repo.PasswordHash(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.NotFound("用户不存在")
	}
	if err != nil {
		return httpx.Internal(err)
	}
	if !auth.CheckPassword(hash, current) {
		return httpx.NewError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "当前密码错误")
	}
	newHash, err := auth.HashPassword(next)
	if err != nil {
		return httpx.Internal(err)
	}
	if err := s.repo.UpdatePassword(ctx, userID, newHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.NotFound("用户不存在")
		}
		return httpx.Internal(err)
	}
	return nil
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}
