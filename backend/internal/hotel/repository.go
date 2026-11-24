package hotel

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Hotel struct {
	ID          int64
	Name        string
	City        string
	Address     string
	Description string
	Facilities  []string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const hotelColumns = `id, name, city, address, description, facilities, status, created_at, updated_at`

func scanHotel(row pgx.Row) (Hotel, error) {
	var h Hotel
	err := row.Scan(&h.ID, &h.Name, &h.City, &h.Address, &h.Description,
		&h.Facilities, &h.Status, &h.CreatedAt, &h.UpdatedAt)
	return h, err
}

func (r *Repository) ListPublic(ctx context.Context, city, keyword string, limit, offset int) ([]Hotel, int, error) {
	where := `status = 'active'`
	args := []any{}
	if city != "" {
		args = append(args, city)
		where += fmt.Sprintf(" AND city ILIKE $%d", len(args))
	}
	if keyword != "" {
		args = append(args, "%"+keyword+"%")
		where += fmt.Sprintf(" AND name ILIKE $%d", len(args))
	}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*)::int FROM hotels WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.db.Query(ctx,
		`SELECT `+hotelColumns+` FROM hotels WHERE `+where+
			fmt.Sprintf(` ORDER BY id DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2),
		listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	hotels := make([]Hotel, 0)
	for rows.Next() {
		h, err := scanHotel(rows)
		if err != nil {
			return nil, 0, err
		}
		hotels = append(hotels, h)
	}
	return hotels, total, rows.Err()
}

func (r *Repository) GetPublic(ctx context.Context, id int64) (Hotel, error) {
	return scanHotel(r.db.QueryRow(ctx,
		`SELECT `+hotelColumns+` FROM hotels WHERE id = $1 AND status = 'active'`, id))
}

func (r *Repository) Get(ctx context.Context, id int64) (Hotel, error) {
	return scanHotel(r.db.QueryRow(ctx,
		`SELECT `+hotelColumns+` FROM hotels WHERE id = $1`, id))
}

func (r *Repository) Create(ctx context.Context, h Hotel, managerID int64) (Hotel, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Hotel{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	created, err := scanHotel(tx.QueryRow(ctx,
		`INSERT INTO hotels (name, city, address, description, facilities)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+hotelColumns,
		h.Name, h.City, h.Address, h.Description, h.Facilities))
	if err != nil {
		return Hotel{}, err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO hotel_managers (hotel_id, user_id) VALUES ($1, $2)`,
		created.ID, managerID); err != nil {
		return Hotel{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Hotel{}, err
	}
	return created, nil
}

type updateFields struct {
	Name        *string
	City        *string
	Address     *string
	Description *string
	Facilities  *[]string
}

func (r *Repository) Update(ctx context.Context, id int64, in updateFields) (Hotel, error) {
	var facilities any
	if in.Facilities != nil {
		facilities = *in.Facilities
	}
	return scanHotel(r.db.QueryRow(ctx,
		`UPDATE hotels
		 SET name = COALESCE($2, name),
		     city = COALESCE($3, city),
		     address = COALESCE($4, address),
		     description = COALESCE($5, description),
		     facilities = COALESCE($6::text[], facilities),
		     updated_at = now()
		 WHERE id = $1
		 RETURNING `+hotelColumns,
		id, in.Name, in.City, in.Address, in.Description, facilities))
}

func (r *Repository) SetStatus(ctx context.Context, id int64, status string) (Hotel, error) {
	return scanHotel(r.db.QueryRow(ctx,
		`UPDATE hotels SET status = $2, updated_at = now() WHERE id = $1 RETURNING `+hotelColumns,
		id, status))
}

func (r *Repository) ListManaged(ctx context.Context, userID int64) ([]Hotel, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+prefixedColumns("h")+`
		 FROM hotels h
		 JOIN hotel_managers hm ON hm.hotel_id = h.id
		 WHERE hm.user_id = $1
		 ORDER BY h.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hotels := make([]Hotel, 0)
	for rows.Next() {
		h, err := scanHotel(rows)
		if err != nil {
			return nil, err
		}
		hotels = append(hotels, h)
	}
	return hotels, rows.Err()
}

func (r *Repository) IsManager(ctx context.Context, hotelID, userID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM hotel_managers WHERE hotel_id = $1 AND user_id = $2)`,
		hotelID, userID).Scan(&ok)
	return ok, err
}

func prefixedColumns(alias string) string {
	return alias + `.id, ` + alias + `.name, ` + alias + `.city, ` + alias + `.address, ` +
		alias + `.description, ` + alias + `.facilities, ` + alias + `.status, ` +
		alias + `.created_at, ` + alias + `.updated_at`
}
