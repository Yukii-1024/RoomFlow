package statistics

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

type DashboardCounts struct {
	TotalRooms       int
	OccupiedRooms    int
	AvailableRooms   int
	UnavailableRooms int
	TodayCheckIns    int
	TodayCheckOuts   int
}

func (r *Repository) Dashboard(ctx context.Context, hotelID int64, today time.Time) (DashboardCounts, error) {
	var c DashboardCounts
	var activeRooms int
	err := r.db.QueryRow(ctx,
		`SELECT
		     (SELECT count(*)::int FROM rooms WHERE hotel_id = $1) AS total_rooms,
		     (SELECT count(*)::int FROM rooms WHERE hotel_id = $1 AND status = 'active') AS active_rooms,
		     (SELECT count(*)::int FROM rooms WHERE hotel_id = $1 AND status <> 'active') AS unavailable_rooms,
		     (SELECT count(*)::int FROM bookings
		       WHERE hotel_id = $1 AND status IN ('confirmed', 'checked_in')
		         AND check_in <= $2 AND check_out > $2) AS booked_rooms,
		     (SELECT count(*)::int FROM bookings
		       WHERE hotel_id = $1 AND status <> 'cancelled' AND check_in = $2) AS today_check_ins,
		     (SELECT count(*)::int FROM bookings
		       WHERE hotel_id = $1 AND status IN ('checked_in', 'checked_out')
		         AND check_out = $2) AS today_check_outs`,
		hotelID, today).
		Scan(&c.TotalRooms, &activeRooms, &c.UnavailableRooms, &c.OccupiedRooms,
			&c.TodayCheckIns, &c.TodayCheckOuts)
	if err != nil {
		return DashboardCounts{}, err
	}
	if c.OccupiedRooms > activeRooms {
		c.OccupiedRooms = activeRooms
	}
	c.AvailableRooms = activeRooms - c.OccupiedRooms
	return c, nil
}

type OccupancyCounts struct {
	ActiveRooms        int
	OccupiedRoomNights int
}

func (r *Repository) Occupancy(ctx context.Context, hotelID int64, start, end time.Time) (OccupancyCounts, error) {
	var c OccupancyCounts
	err := r.db.QueryRow(ctx,
		`SELECT
		     (SELECT count(*)::int FROM rooms WHERE hotel_id = $1 AND status = 'active') AS active_rooms,
		     COALESCE((
		         SELECT SUM(LEAST(check_out, $3::date) - GREATEST(check_in, $2::date))
		         FROM bookings
		         WHERE hotel_id = $1 AND status <> 'cancelled'
		           AND check_in < $3 AND check_out > $2
		     ), 0)::int AS occupied_room_nights`,
		hotelID, start, end).
		Scan(&c.ActiveRooms, &c.OccupiedRoomNights)
	if err != nil {
		return OccupancyCounts{}, err
	}
	return c, nil
}
