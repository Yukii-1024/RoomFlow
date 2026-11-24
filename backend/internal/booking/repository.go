package booking

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
	"github.com/Yukii-1024/RoomFlow/backend/internal/money"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Booking is the full detail view of a booking with hotel and room type names.
type Booking struct {
	ID              int64
	BookingNo       string
	UserID          int64
	HotelID         int64
	RoomTypeID      int64
	RoomID          *int64
	RoomNo          *string
	HotelName       string
	RoomTypeName    string
	CheckIn         time.Time
	CheckOut        time.Time
	Guests          int
	ContactName     string
	ContactPhone    string
	SpecialRequests string
	Status          string
	Currency        string
	TotalAmount     string
	CreatedAt       time.Time
	CancelledAt     *time.Time
	CheckedInAt     *time.Time
	CheckedOutAt    *time.Time
}

var (
	ErrBookingNotFound  = errors.New("booking not found")
	ErrRoomTypeNotFound = errors.New("room type not found")
	ErrHotelInactive    = errors.New("hotel inactive")
	ErrRoomTypeInactive = errors.New("room type inactive")
	ErrGuestLimit       = errors.New("guest limit exceeded")
	ErrRoomNotAvailable = errors.New("room type sold out")
	ErrNoRoomAvailable  = errors.New("no assignable room")
	ErrRoomUnavailable  = errors.New("selected room unavailable")
	ErrRoomOccupied     = errors.New("selected room occupied")
	ErrInvalidStatus    = errors.New("invalid booking status")
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const bookingSelect = `SELECT b.id, COALESCE(b.booking_no, ''), b.user_id, b.hotel_id, b.room_type_id,
		       COALESCE(b.room_id, 0), COALESCE(rm.room_no, ''), h.name, rt.name,
		       b.check_in, b.check_out, b.guests, b.contact_name, b.contact_phone,
		       b.special_requests, b.status, b.currency, b.total_amount::text, b.created_at,
		       b.cancelled_at, b.checked_in_at, b.checked_out_at
		FROM bookings b
		JOIN hotels h ON h.id = b.hotel_id
		JOIN room_types rt ON rt.id = b.room_type_id
		LEFT JOIN rooms rm ON rm.id = b.room_id`

func scanBooking(row pgx.Row) (Booking, error) {
	var (
		b      Booking
		roomID int64
		roomNo string
	)
	err := row.Scan(&b.ID, &b.BookingNo, &b.UserID, &b.HotelID, &b.RoomTypeID,
		&roomID, &roomNo, &b.HotelName, &b.RoomTypeName,
		&b.CheckIn, &b.CheckOut, &b.Guests, &b.ContactName, &b.ContactPhone,
		&b.SpecialRequests, &b.Status, &b.Currency, &b.TotalAmount, &b.CreatedAt,
		&b.CancelledAt, &b.CheckedInAt, &b.CheckedOutAt)
	if err != nil {
		return Booking{}, err
	}
	if roomID != 0 {
		b.RoomID = &roomID
	}
	if roomNo != "" {
		b.RoomNo = &roomNo
	}
	return b, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Booking, error) {
	b, err := scanBooking(r.db.QueryRow(ctx, bookingSelect+` WHERE b.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Booking{}, ErrBookingNotFound
	}
	return b, err
}

func (r *Repository) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]Booking, int, error) {
	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT count(*)::int FROM bookings WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx,
		bookingSelect+` WHERE b.user_id = $1 ORDER BY b.created_at DESC, b.id DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	bookings, err := collectBookings(rows)
	if err != nil {
		return nil, 0, err
	}
	return bookings, total, nil
}

func (r *Repository) ListByHotel(ctx context.Context, hotelID int64, status string, limit, offset int) ([]Booking, int, error) {
	where := `hotel_id = $1`
	listWhere := `b.hotel_id = $1`
	args := []any{hotelID}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(` AND status = $%d`, len(args))
		listWhere += fmt.Sprintf(` AND b.status = $%d`, len(args))
	}

	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*)::int FROM bookings WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.db.Query(ctx,
		bookingSelect+` WHERE `+listWhere+
			fmt.Sprintf(` ORDER BY b.created_at DESC, b.id DESC LIMIT $%d OFFSET $%d`,
				len(args)+1, len(args)+2),
		listArgs...)
	if err != nil {
		return nil, 0, err
	}
	bookings, err := collectBookings(rows)
	if err != nil {
		return nil, 0, err
	}
	return bookings, total, nil
}

func collectBookings(rows pgx.Rows) ([]Booking, error) {
	defer rows.Close()
	bookings := make([]Booking, 0)
	for rows.Next() {
		b, err := scanBooking(rows)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return bookings, nil
}

type createParams struct {
	UserID          int64
	HotelID         int64
	RoomTypeID      int64
	CheckIn         time.Time
	CheckOut        time.Time
	Guests          int
	ContactName     string
	ContactPhone    string
	SpecialRequests string
}

// Create reserves inventory in a transaction, assigns the booking number and
// returns the full detail record.
func (r *Repository) Create(ctx context.Context, in createParams) (Booking, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Booking{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var roomTypeStatus, hotelStatus, basePrice string
	var maxGuests int
	err = tx.QueryRow(ctx,
		`SELECT rt.status, h.status, rt.base_price::text, rt.max_guests
		 FROM room_types rt
		 JOIN hotels h ON h.id = rt.hotel_id
		 WHERE rt.id = $1 AND rt.hotel_id = $2
		 FOR UPDATE OF rt`,
		in.RoomTypeID, in.HotelID).Scan(&roomTypeStatus, &hotelStatus, &basePrice, &maxGuests)
	if errors.Is(err, pgx.ErrNoRows) {
		return Booking{}, ErrRoomTypeNotFound
	}
	if err != nil {
		return Booking{}, err
	}
	if hotelStatus != "active" {
		return Booking{}, ErrHotelInactive
	}
	if roomTypeStatus != "active" {
		return Booking{}, ErrRoomTypeInactive
	}
	if in.Guests > maxGuests {
		return Booking{}, ErrGuestLimit
	}

	priceCents, err := money.Parse(basePrice)
	if err != nil {
		return Booking{}, err
	}
	nights := httpx.Nights(in.CheckIn, in.CheckOut)
	if nights <= 0 {
		return Booking{}, httpx.BadRequest("退房日期必须晚于入住日期")
	}
	totalCents := priceCents * int64(nights)

	var available int
	if err := tx.QueryRow(ctx,
		`SELECT
		   (SELECT count(*)::int FROM rooms WHERE room_type_id = $1 AND status = 'active')
		   - (SELECT count(*)::int FROM bookings
		      WHERE room_type_id = $1
		        AND status IN ('confirmed', 'checked_in')
		        AND check_in < $3 AND check_out > $2)`,
		in.RoomTypeID, in.CheckIn, in.CheckOut).Scan(&available); err != nil {
		return Booking{}, err
	}
	if available <= 0 {
		return Booking{}, ErrRoomNotAvailable
	}

	var id int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO bookings
		   (user_id, hotel_id, room_type_id, check_in, check_out, guests,
		    contact_name, contact_phone, special_requests, total_amount)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::numeric / 100)
		 RETURNING id`,
		in.UserID, in.HotelID, in.RoomTypeID, in.CheckIn, in.CheckOut, in.Guests,
		in.ContactName, in.ContactPhone, in.SpecialRequests, totalCents).Scan(&id); err != nil {
		return Booking{}, err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE bookings
		 SET booking_no = 'RF' || to_char(check_in, 'YYYYMMDD') || lpad(id::text, 4, '0')
		 WHERE id = $1`, id); err != nil {
		return Booking{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Booking{}, err
	}
	return r.GetByID(ctx, id)
}

// Cancel marks a confirmed booking as cancelled.
func (r *Repository) Cancel(ctx context.Context, id int64) (Booking, error) {
	tag, err := r.db.Exec(ctx,
		`UPDATE bookings SET status = 'cancelled', cancelled_at = now(), updated_at = now()
		 WHERE id = $1 AND status = 'confirmed'`, id)
	if err != nil {
		return Booking{}, err
	}
	if tag.RowsAffected() == 0 {
		return Booking{}, r.mutationMiss(ctx, id)
	}
	return r.GetByID(ctx, id)
}

// CheckIn locks the booking, resolves an assignable room and marks the booking
// as checked in.
func (r *Repository) CheckIn(ctx context.Context, id int64, roomID *int64) (Booking, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Booking{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var hotelID, roomTypeID int64
	var checkIn, checkOut time.Time
	var status string
	err = tx.QueryRow(ctx,
		`SELECT hotel_id, room_type_id, check_in, check_out, status
		 FROM bookings WHERE id = $1 FOR UPDATE`, id).
		Scan(&hotelID, &roomTypeID, &checkIn, &checkOut, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return Booking{}, ErrBookingNotFound
	}
	if err != nil {
		return Booking{}, err
	}
	if status != "confirmed" {
		return Booking{}, ErrInvalidStatus
	}

	var assigned int64
	if roomID != nil {
		err = tx.QueryRow(ctx,
			`SELECT id FROM rooms
			 WHERE id = $1 AND hotel_id = $2 AND room_type_id = $3 AND status = 'active'
			 FOR UPDATE`, *roomID, hotelID, roomTypeID).Scan(&assigned)
		if errors.Is(err, pgx.ErrNoRows) {
			return Booking{}, ErrRoomUnavailable
		}
		if err != nil {
			return Booking{}, err
		}
		var occupied bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (
			   SELECT 1 FROM bookings
			   WHERE room_id = $1 AND id <> $2 AND status = 'checked_in'
			     AND check_in < $4 AND check_out > $3
			 )`, assigned, id, checkIn, checkOut).Scan(&occupied); err != nil {
			return Booking{}, err
		}
		if occupied {
			return Booking{}, ErrRoomOccupied
		}
	} else {
		err = tx.QueryRow(ctx,
			`SELECT r.id
			 FROM rooms r
			 WHERE r.hotel_id = $1 AND r.room_type_id = $2 AND r.status = 'active'
			   AND NOT EXISTS (
			     SELECT 1 FROM bookings ob
			     WHERE ob.room_id = r.id AND ob.status = 'checked_in'
			       AND ob.check_in < $4 AND ob.check_out > $3
			   )
			 ORDER BY r.room_no
			 LIMIT 1
			 FOR UPDATE SKIP LOCKED`,
			hotelID, roomTypeID, checkIn, checkOut).Scan(&assigned)
		if errors.Is(err, pgx.ErrNoRows) {
			return Booking{}, ErrNoRoomAvailable
		}
		if err != nil {
			return Booking{}, err
		}
	}

	if _, err := tx.Exec(ctx,
		`UPDATE bookings
		 SET room_id = $2, status = 'checked_in', checked_in_at = now(), updated_at = now()
		 WHERE id = $1 AND status = 'confirmed'`, id, assigned); err != nil {
		return Booking{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Booking{}, err
	}
	return r.GetByID(ctx, id)
}

// CheckOut marks a checked in booking as checked out.
func (r *Repository) CheckOut(ctx context.Context, id int64) (Booking, error) {
	tag, err := r.db.Exec(ctx,
		`UPDATE bookings SET status = 'checked_out', checked_out_at = now(), updated_at = now()
		 WHERE id = $1 AND status = 'checked_in'`, id)
	if err != nil {
		return Booking{}, err
	}
	if tag.RowsAffected() == 0 {
		return Booking{}, r.mutationMiss(ctx, id)
	}
	return r.GetByID(ctx, id)
}

// mutationMiss distinguishes a missing booking from a status conflict.
func (r *Repository) mutationMiss(ctx context.Context, id int64) error {
	var exists bool
	if err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM bookings WHERE id = $1)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrBookingNotFound
	}
	return ErrInvalidStatus
}
