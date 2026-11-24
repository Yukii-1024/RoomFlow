package availability

import (
	"context"
	"errors"
	"time"

	"github.com/Yukii-1024/RoomFlow/backend/internal/hotel"
	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
	"github.com/Yukii-1024/RoomFlow/backend/internal/money"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo   *Repository
	hotels *hotel.Service
}

func NewService(repo *Repository, hotels *hotel.Service) *Service {
	return &Service{repo: repo, hotels: hotels}
}

type AvailabilityResult struct {
	HotelID   int64
	CheckIn   time.Time
	CheckOut  time.Time
	Nights    int
	RoomTypes []RoomTypeAvailability
}

func (s *Service) Availability(ctx context.Context, hotelID int64, checkIn, checkOut time.Time, guests int) (AvailabilityResult, error) {
	if !checkOut.After(checkIn) {
		return AvailabilityResult{}, httpx.BadRequest("离店日期必须晚于入住日期")
	}
	if guests < 0 {
		return AvailabilityResult{}, httpx.BadRequest("入住人数不能为负数")
	}
	if _, err := s.hotels.GetPublic(ctx, hotelID); err != nil {
		return AvailabilityResult{}, err
	}
	types, err := s.repo.ListAvailable(ctx, hotelID, checkIn, checkOut, guests)
	if err != nil {
		return AvailabilityResult{}, httpx.Internal(err)
	}
	return AvailabilityResult{
		HotelID:   hotelID,
		CheckIn:   checkIn,
		CheckOut:  checkOut,
		Nights:    httpx.Nights(checkIn, checkOut),
		RoomTypes: types,
	}, nil
}

type QuoteInput struct {
	HotelID    int64
	RoomTypeID int64
	CheckIn    time.Time
	CheckOut   time.Time
	Guests     int
}

type QuoteResult struct {
	Currency      string
	Nights        int
	RoomRateTotal string
	Fees          string
	Total         string
}

// Quote prices a stay. It intentionally does not check inventory; booking
// creation re-validates both price and stock.
func (s *Service) Quote(ctx context.Context, in QuoteInput) (QuoteResult, error) {
	if !in.CheckOut.After(in.CheckIn) {
		return QuoteResult{}, httpx.BadRequest("离店日期必须晚于入住日期")
	}
	if in.Guests < 1 {
		return QuoteResult{}, httpx.BadRequest("入住人数必须大于等于 1")
	}
	if _, err := s.hotels.GetPublic(ctx, in.HotelID); err != nil {
		return QuoteResult{}, err
	}

	rt, err := s.repo.GetRoomType(ctx, in.RoomTypeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return QuoteResult{}, httpx.NotFound("房型不存在或与酒店不匹配")
	}
	if err != nil {
		return QuoteResult{}, httpx.Internal(err)
	}
	if rt.HotelID != in.HotelID {
		return QuoteResult{}, httpx.NotFound("房型不存在或与酒店不匹配")
	}
	if rt.Status != "active" {
		return QuoteResult{}, httpx.NotFound("房型已停用")
	}
	if in.Guests > rt.MaxGuests {
		return QuoteResult{}, httpx.BadRequest("入住人数超过房型上限")
	}

	baseCents, err := money.Parse(rt.BasePrice)
	if err != nil {
		return QuoteResult{}, httpx.Internal(err)
	}
	nights := httpx.Nights(in.CheckIn, in.CheckOut)
	total := money.Format(baseCents * int64(nights))
	return QuoteResult{
		Currency:      "CNY",
		Nights:        nights,
		RoomRateTotal: total,
		Fees:          "0.00",
		Total:         total,
	}, nil
}
