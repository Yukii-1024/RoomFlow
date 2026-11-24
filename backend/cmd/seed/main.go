// Command seed inserts demo users, a hotel, room types and rooms.
// It is idempotent and safe to run repeatedly.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/Yukii-1024/RoomFlow/backend/internal/auth"
	"github.com/Yukii-1024/RoomFlow/backend/internal/config"
	"github.com/Yukii-1024/RoomFlow/backend/internal/database"
	"github.com/Yukii-1024/RoomFlow/backend/migrations"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		return err
	}
	if err := seed(ctx, pool); err != nil {
		return err
	}

	fmt.Println("seed complete")
	fmt.Println("  manager:  manager@roomflow.local / Manager123!")
	fmt.Println("  customer: customer@roomflow.local / Customer123!")
	return nil
}

func seed(ctx context.Context, pool *pgxpool.Pool) error {
	managerID, err := ensureUser(ctx, pool, "manager@roomflow.local", "Manager123!", "酒店经理", "manager")
	if err != nil {
		return err
	}
	if _, err := ensureUser(ctx, pool, "customer@roomflow.local", "Customer123!", "张三", "customer"); err != nil {
		return err
	}

	hotelID, err := ensureHotel(ctx, pool, managerID)
	if err != nil {
		return err
	}

	standardID, err := ensureRoomType(ctx, pool, hotelID, roomType{
		Name: "标准大床房", Description: "配备 1.8 米大床，适合商务出行", Price: "328.00",
		MaxGuests: 2, Area: 28, Amenities: []string{"wifi", "air_conditioning"},
		Features: []string{"city_view"},
	})
	if err != nil {
		return err
	}
	twinID, err := ensureRoomType(ctx, pool, hotelID, roomType{
		Name: "高级双床房", Description: "两张 1.2 米单人床，适合结伴出行", Price: "388.00",
		MaxGuests: 2, Area: 32, Amenities: []string{"wifi", "air_conditioning", "breakfast"},
		Features: []string{"near_elevator"},
	})
	if err != nil {
		return err
	}
	suiteID, err := ensureRoomType(ctx, pool, hotelID, roomType{
		Name: "豪华套房", Description: "独立客厅与卧室，可观赏湖景", Price: "688.00",
		MaxGuests: 3, Area: 55, Amenities: []string{"wifi", "air_conditioning", "breakfast", "bathtub"},
		Features: []string{"lake_view", "high_floor"},
	})
	if err != nil {
		return err
	}

	rooms := []struct {
		roomTypeID int64
		roomNo     string
	}{
		{standardID, "801"}, {standardID, "802"}, {standardID, "803"}, {standardID, "804"},
		{twinID, "805"}, {twinID, "806"}, {twinID, "807"},
		{suiteID, "901"}, {suiteID, "902"},
	}
	for _, room := range rooms {
		if _, err := pool.Exec(ctx,
			`INSERT INTO rooms (hotel_id, room_type_id, room_no)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (hotel_id, room_no) DO NOTHING`,
			hotelID, room.roomTypeID, room.roomNo); err != nil {
			return fmt.Errorf("seed room %s: %w", room.roomNo, err)
		}
	}
	return nil
}

func ensureUser(ctx context.Context, pool *pgxpool.Pool, email, password, name, role string) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `SELECT id FROM users WHERE lower(email) = lower($1)`, email).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("find user %s: %w", email, err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return 0, err
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, role)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		email, hash, name, role).Scan(&id); err != nil {
		return 0, fmt.Errorf("create user %s: %w", email, err)
	}
	return id, nil
}

func ensureHotel(ctx context.Context, pool *pgxpool.Pool, managerID int64) (int64, error) {
	const name = "RoomFlow 西湖酒店"
	var id int64
	err := pool.QueryRow(ctx, `SELECT id FROM hotels WHERE name = $1 AND city = '杭州'`, name).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = pool.QueryRow(ctx,
			`INSERT INTO hotels (name, city, address, description, facilities, status)
			 VALUES ($1, '杭州', '西湖区文三路 100 号', '位于西湖商圈的示范酒店，步行可达地铁站。',
			         ARRAY['wifi', 'parking', 'breakfast'], 'active')
			 RETURNING id`, name).Scan(&id)
		if err != nil {
			return 0, fmt.Errorf("create hotel: %w", err)
		}
	} else if err != nil {
		return 0, fmt.Errorf("find hotel: %w", err)
	}

	if _, err := pool.Exec(ctx,
		`INSERT INTO hotel_managers (hotel_id, user_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`, id, managerID); err != nil {
		return 0, fmt.Errorf("link hotel manager: %w", err)
	}
	return id, nil
}

type roomType struct {
	Name        string
	Description string
	Price       string
	MaxGuests   int
	Area        float64
	Amenities   []string
	Features    []string
}

func ensureRoomType(ctx context.Context, pool *pgxpool.Pool, hotelID int64, rt roomType) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx,
		`SELECT id FROM room_types WHERE hotel_id = $1 AND name = $2`, hotelID, rt.Name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("find room type %s: %w", rt.Name, err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO room_types (hotel_id, name, description, base_price, max_guests, area, amenities, features)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id`,
		hotelID, rt.Name, rt.Description, rt.Price, rt.MaxGuests, rt.Area, rt.Amenities, rt.Features).Scan(&id); err != nil {
		return 0, fmt.Errorf("create room type %s: %w", rt.Name, err)
	}
	return id, nil
}
