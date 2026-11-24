// Package availability implements the public inventory and quote endpoints.
package availability

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RoomTypeAvailability is one active room type with its remaining inventory.
type RoomTypeAvailability struct {
	TypeID         int64
	Name           string
	AvailableCount int
	MaxGuests      int
	BasePrice      string
	Features       []string
}

// RoomType is the subset of a room type needed to price a quote.
type RoomType struct {
	ID        int64
	HotelID   int64
	Name      string
	BasePrice string
	MaxGuests int
	Status    string
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// ListAvailable returns the hotel's active room types that still have at least
// one free room for [checkIn, checkOut). When guests is greater than zero only
// types that can host that party size are returned. Bookings count as occupied
// while they overlap the requested range under the left-closed/right-open
// convention.
func (r *Repository) ListAvailable(ctx context.Context, hotelID int64, checkIn, checkOut time.Time, guests int) ([]RoomTypeAvailability, error) {
	rows, err := r.db.Query(ctx, `
		SELECT type_id, name, max_guests, base_price, features, available_count
		FROM (
			SELECT rt.id AS type_id, rt.name, rt.max_guests,
			       rt.base_price::text AS base_price, rt.features,
			       (SELECT count(*)::int FROM rooms rm
			         WHERE rm.room_type_id = rt.id AND rm.status = 'active')
			     - (SELECT count(*)::int FROM bookings b
			         WHERE b.room_type_id = rt.id
			           AND b.status IN ('confirmed', 'checked_in')
			           AND b.check_in < $3 AND b.check_out > $2) AS available_count
			FROM room_types rt
			WHERE rt.hotel_id = $1 AND rt.status = 'active'
			  AND ($4 = 0 OR rt.max_guests >= $4)
		) t
		WHERE available_count > 0
		ORDER BY type_id`, hotelID, checkIn, checkOut, guests)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]RoomTypeAvailability, 0)
	for rows.Next() {
		var it RoomTypeAvailability
		if err := rows.Scan(&it.TypeID, &it.Name, &it.MaxGuests, &it.BasePrice,
			&it.Features, &it.AvailableCount); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// GetRoomType loads a room type regardless of its hotel or status so the
// service can distinguish "not found or wrong hotel" from "disabled".
func (r *Repository) GetRoomType(ctx context.Context, id int64) (RoomType, error) {
	var rt RoomType
	err := r.db.QueryRow(ctx, `
		SELECT id, hotel_id, name, base_price::text, max_guests, status
		FROM room_types
		WHERE id = $1`, id).
		Scan(&rt.ID, &rt.HotelID, &rt.Name, &rt.BasePrice, &rt.MaxGuests, &rt.Status)
	return rt, err
}
