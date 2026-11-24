package room

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoomType struct {
	ID          int64
	HotelID     int64
	Name        string
	Description string
	BasePrice   string
	MaxGuests   int
	Area        *float64
	Amenities   []string
	Features    []string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Room struct {
	ID           int64
	HotelID      int64
	RoomTypeID   int64
	RoomTypeName string
	RoomNo       string
	Features     []string
	Status       string
	StatusReason string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const roomTypeColumns = `id, hotel_id, name, description, base_price::text, max_guests, area, amenities, features, status, created_at, updated_at`

const roomTypeColumnsAliased = `rt.id, rt.hotel_id, rt.name, rt.description, rt.base_price::text, rt.max_guests, rt.area, rt.amenities, rt.features, rt.status, rt.created_at, rt.updated_at`

const roomColumns = `r.id, r.hotel_id, r.room_type_id, rt.name, r.room_no, r.features, r.status, r.status_reason, r.created_at, r.updated_at`

func scanRoomType(row pgx.Row) (RoomType, error) {
	var t RoomType
	err := row.Scan(&t.ID, &t.HotelID, &t.Name, &t.Description, &t.BasePrice,
		&t.MaxGuests, &t.Area, &t.Amenities, &t.Features, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return RoomType{}, err
	}
	if t.Amenities == nil {
		t.Amenities = []string{}
	}
	if t.Features == nil {
		t.Features = []string{}
	}
	return t, nil
}

func scanRoom(row pgx.Row) (Room, error) {
	var rm Room
	err := row.Scan(&rm.ID, &rm.HotelID, &rm.RoomTypeID, &rm.RoomTypeName, &rm.RoomNo,
		&rm.Features, &rm.Status, &rm.StatusReason, &rm.CreatedAt, &rm.UpdatedAt)
	if err != nil {
		return Room{}, err
	}
	if rm.Features == nil {
		rm.Features = []string{}
	}
	return rm, nil
}

func (r *Repository) ListTypes(ctx context.Context, hotelID int64) ([]RoomType, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+roomTypeColumns+`
		 FROM room_types
		 WHERE hotel_id = $1 AND status = 'active'
		 ORDER BY id`, hotelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	types := make([]RoomType, 0)
	for rows.Next() {
		t, err := scanRoomType(rows)
		if err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return types, rows.Err()
}

func (r *Repository) GetTypePublic(ctx context.Context, id int64) (RoomType, error) {
	return scanRoomType(r.db.QueryRow(ctx,
		`SELECT `+roomTypeColumnsAliased+`
		 FROM room_types rt
		 JOIN hotels h ON h.id = rt.hotel_id
		 WHERE rt.id = $1 AND rt.status = 'active' AND h.status = 'active'`, id))
}

func (r *Repository) GetType(ctx context.Context, id int64) (RoomType, error) {
	return scanRoomType(r.db.QueryRow(ctx,
		`SELECT `+roomTypeColumns+` FROM room_types WHERE id = $1`, id))
}

type createTypeFields struct {
	Name        string
	Description string
	BasePrice   string
	MaxGuests   int
	Area        *float64
	Amenities   []string
	Features    []string
}

func (r *Repository) CreateType(ctx context.Context, hotelID int64, in createTypeFields) (RoomType, error) {
	return scanRoomType(r.db.QueryRow(ctx,
		`INSERT INTO room_types (hotel_id, name, description, base_price, max_guests, area, amenities, features)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING `+roomTypeColumns,
		hotelID, in.Name, in.Description, in.BasePrice, in.MaxGuests,
		in.Area, in.Amenities, in.Features))
}

type updateTypeFields struct {
	Name        *string
	Description *string
	BasePrice   *string
	MaxGuests   *int
	Area        *float64
	Amenities   *[]string
	Features    *[]string
}

func (r *Repository) UpdateType(ctx context.Context, id int64, in updateTypeFields) (RoomType, error) {
	var amenities, features any
	if in.Amenities != nil {
		amenities = *in.Amenities
	}
	if in.Features != nil {
		features = *in.Features
	}
	return scanRoomType(r.db.QueryRow(ctx,
		`UPDATE room_types
		 SET name = COALESCE($2, name),
		     description = COALESCE($3, description),
		     base_price = COALESCE($4::numeric, base_price),
		     max_guests = COALESCE($5::integer, max_guests),
		     area = COALESCE($6::numeric, area),
		     amenities = COALESCE($7::text[], amenities),
		     features = COALESCE($8::text[], features),
		     updated_at = now()
		 WHERE id = $1
		 RETURNING `+roomTypeColumns,
		id, in.Name, in.Description, in.BasePrice, in.MaxGuests,
		in.Area, amenities, features))
}

func (r *Repository) SetTypeStatus(ctx context.Context, id int64, status string) (RoomType, error) {
	return scanRoomType(r.db.QueryRow(ctx,
		`UPDATE room_types SET status = $2, updated_at = now() WHERE id = $1 RETURNING `+roomTypeColumns,
		id, status))
}

func (r *Repository) ListRooms(ctx context.Context, hotelID int64) ([]Room, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+roomColumns+`
		 FROM rooms r
		 JOIN room_types rt ON rt.id = r.room_type_id
		 WHERE r.hotel_id = $1
		 ORDER BY r.room_no`, hotelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rooms := make([]Room, 0)
	for rows.Next() {
		rm, err := scanRoom(rows)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, rm)
	}
	return rooms, rows.Err()
}

func (r *Repository) GetRoom(ctx context.Context, id int64) (Room, error) {
	return scanRoom(r.db.QueryRow(ctx,
		`SELECT `+roomColumns+`
		 FROM rooms r
		 JOIN room_types rt ON rt.id = r.room_type_id
		 WHERE r.id = $1`, id))
}

type createRoomFields struct {
	RoomNo     string
	RoomTypeID int64
	Features   []string
}

func (r *Repository) CreateRoom(ctx context.Context, hotelID int64, in createRoomFields) (Room, error) {
	return scanRoom(r.db.QueryRow(ctx,
		`WITH inserted AS (
		     INSERT INTO rooms (hotel_id, room_type_id, room_no, features)
		     VALUES ($1, $2, $3, $4)
		     RETURNING *
		 )
		 SELECT `+roomColumns+`
		 FROM inserted r
		 JOIN room_types rt ON rt.id = r.room_type_id`,
		hotelID, in.RoomTypeID, in.RoomNo, in.Features))
}

type updateRoomFields struct {
	RoomNo     *string
	RoomTypeID *int64
	Features   *[]string
}

func (r *Repository) UpdateRoom(ctx context.Context, id int64, in updateRoomFields) (Room, error) {
	var features any
	if in.Features != nil {
		features = *in.Features
	}
	return scanRoom(r.db.QueryRow(ctx,
		`WITH updated AS (
		     UPDATE rooms
		     SET room_no = COALESCE($2::text, room_no),
		         room_type_id = COALESCE($3::bigint, room_type_id),
		         features = COALESCE($4::text[], features),
		         updated_at = now()
		     WHERE id = $1
		     RETURNING *
		 )
		 SELECT `+roomColumns+`
		 FROM updated r
		 JOIN room_types rt ON rt.id = r.room_type_id`,
		id, in.RoomNo, in.RoomTypeID, features))
}

func (r *Repository) SetRoomStatus(ctx context.Context, id int64, status, reason string) (Room, error) {
	return scanRoom(r.db.QueryRow(ctx,
		`WITH updated AS (
		     UPDATE rooms
		     SET status = $2, status_reason = $3, updated_at = now()
		     WHERE id = $1
		     RETURNING *
		 )
		 SELECT `+roomColumns+`
		 FROM updated r
		 JOIN room_types rt ON rt.id = r.room_type_id`,
		id, status, reason))
}

func (r *Repository) DeleteRoom(ctx context.Context, id int64) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM rooms WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) RoomTypeBelongsToHotel(ctx context.Context, hotelID, typeID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM room_types WHERE id = $1 AND hotel_id = $2)`,
		typeID, hotelID).Scan(&ok)
	return ok, err
}

func (r *Repository) HasCheckedInBooking(ctx context.Context, roomID int64) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM bookings WHERE room_id = $1 AND status = 'checked_in')`,
		roomID).Scan(&ok)
	return ok, err
}
