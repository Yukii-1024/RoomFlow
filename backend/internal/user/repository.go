package user

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID        int64
	Email     string
	Name      string
	Phone     string
	Role      string
	CreatedAt time.Time
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const userColumns = `id, email, name, phone, role, created_at`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Phone, &u.Role, &u.CreatedAt)
	return u, err
}

func (r *Repository) Create(ctx context.Context, email, passwordHash, name, phone, role string) (User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, phone, role)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+userColumns,
		email, passwordHash, name, phone, role,
	))
}

// ByEmail returns the public user and the stored password hash.
func (r *Repository) ByEmail(ctx context.Context, email string) (User, string, error) {
	var u User
	var hash string
	err := r.db.QueryRow(ctx,
		`SELECT `+userColumns+`, password_hash FROM users WHERE lower(email) = lower($1)`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Phone, &u.Role, &u.CreatedAt, &hash)
	return u, hash, err
}

func (r *Repository) ByID(ctx context.Context, id int64) (User, error) {
	return scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

func (r *Repository) UpdateProfile(ctx context.Context, id int64, name, phone *string) (User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`UPDATE users
		 SET name = COALESCE($2, name),
		     phone = COALESCE($3, phone),
		     updated_at = now()
		 WHERE id = $1
		 RETURNING `+userColumns,
		id, name, phone,
	))
}

func (r *Repository) PasswordHash(ctx context.Context, id int64) (string, error) {
	var hash string
	err := r.db.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, id).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", pgx.ErrNoRows
	}
	return hash, err
}

func (r *Repository) UpdatePassword(ctx context.Context, id int64, hash string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1`, id, hash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
