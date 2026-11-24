package booking

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Yukii-1024/RoomFlow/backend/internal/hotel"
	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
)

type Service struct {
	repo   *Repository
	hotels *hotel.Service
}

func NewService(repo *Repository, hotels *hotel.Service) *Service {
	return &Service{repo: repo, hotels: hotels}
}

type CreateInput struct {
	HotelID         int64
	RoomTypeID      int64
	CheckIn         time.Time
	CheckOut        time.Time
	Guests          int
	ContactName     string
	ContactPhone    string
	SpecialRequests string
}

func (s *Service) Create(ctx context.Context, userID int64, in CreateInput) (Booking, error) {
	in.ContactName = strings.TrimSpace(in.ContactName)
	in.ContactPhone = strings.TrimSpace(in.ContactPhone)
	in.SpecialRequests = strings.TrimSpace(in.SpecialRequests)

	if in.HotelID <= 0 || in.RoomTypeID <= 0 {
		return Booking{}, httpx.BadRequest("酒店和房型不能为空")
	}
	if in.CheckIn.Before(httpx.Today()) {
		return Booking{}, httpx.BadRequest("入住日期不能早于今天")
	}
	if !in.CheckOut.After(in.CheckIn) {
		return Booking{}, httpx.BadRequest("退房日期必须晚于入住日期")
	}
	if in.Guests < 1 {
		return Booking{}, httpx.BadRequest("入住人数至少为 1")
	}
	if in.ContactName == "" || len([]rune(in.ContactName)) > 100 {
		return Booking{}, httpx.BadRequest("联系人姓名不能为空且不能超过 100 个字符")
	}
	if in.ContactPhone == "" || len([]rune(in.ContactPhone)) > 32 {
		return Booking{}, httpx.BadRequest("联系电话不能为空且不能超过 32 个字符")
	}
	if len([]rune(in.SpecialRequests)) > 1000 {
		return Booking{}, httpx.BadRequest("特殊要求不能超过 1000 个字符")
	}

	b, err := s.repo.Create(ctx, createParams{
		UserID:          userID,
		HotelID:         in.HotelID,
		RoomTypeID:      in.RoomTypeID,
		CheckIn:         in.CheckIn,
		CheckOut:        in.CheckOut,
		Guests:          in.Guests,
		ContactName:     in.ContactName,
		ContactPhone:    in.ContactPhone,
		SpecialRequests: in.SpecialRequests,
	})
	if err != nil {
		return Booking{}, mapRepositoryError(err)
	}
	return b, nil
}

func (s *Service) ListMine(ctx context.Context, userID int64, page, pageSize int) ([]Booking, int, error) {
	bookings, total, err := s.repo.ListByUser(ctx, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, httpx.Internal(err)
	}
	return bookings, total, nil
}

func (s *Service) Get(ctx context.Context, bookingID, userID int64, role string) (Booking, error) {
	b, err := s.repo.GetByID(ctx, bookingID)
	if err != nil {
		return Booking{}, mapRepositoryError(err)
	}
	if err := s.authorize(ctx, b, userID, role); err != nil {
		return Booking{}, err
	}
	return b, nil
}

func (s *Service) Cancel(ctx context.Context, bookingID, userID int64, role string) (Booking, error) {
	b, err := s.repo.GetByID(ctx, bookingID)
	if err != nil {
		return Booking{}, mapRepositoryError(err)
	}
	if err := s.authorize(ctx, b, userID, role); err != nil {
		return Booking{}, err
	}
	cancelled, err := s.repo.Cancel(ctx, bookingID)
	if errors.Is(err, ErrInvalidStatus) {
		return Booking{}, httpx.Conflict("INVALID_BOOKING_STATUS", "当前订单状态不可取消")
	}
	if err != nil {
		return Booking{}, mapRepositoryError(err)
	}
	return cancelled, nil
}

func (s *Service) ListManaged(ctx context.Context, userID, hotelID int64, status string, page, pageSize int) ([]Booking, int, error) {
	status = strings.TrimSpace(status)
	switch status {
	case "", "confirmed", "checked_in", "checked_out", "cancelled":
	default:
		return nil, 0, httpx.BadRequest("订单状态必须为 confirmed、checked_in、checked_out 或 cancelled")
	}
	if err := s.hotels.RequireManager(ctx, userID, hotelID); err != nil {
		return nil, 0, err
	}
	bookings, total, err := s.repo.ListByHotel(ctx, hotelID, status, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, httpx.Internal(err)
	}
	return bookings, total, nil
}

func (s *Service) CheckIn(ctx context.Context, bookingID, userID int64, roomID *int64) (Booking, error) {
	b, err := s.repo.GetByID(ctx, bookingID)
	if err != nil {
		return Booking{}, mapRepositoryError(err)
	}
	if err := s.hotels.RequireManager(ctx, userID, b.HotelID); err != nil {
		return Booking{}, err
	}
	if b.Status != "confirmed" {
		return Booking{}, httpx.Conflict("INVALID_BOOKING_STATUS", "当前订单状态不可入住")
	}
	if roomID != nil && *roomID <= 0 {
		return Booking{}, httpx.BadRequest("房间 ID 无效")
	}
	checkedIn, err := s.repo.CheckIn(ctx, bookingID, roomID)
	if errors.Is(err, ErrInvalidStatus) {
		return Booking{}, httpx.Conflict("INVALID_BOOKING_STATUS", "当前订单状态不可入住")
	}
	if err != nil {
		return Booking{}, mapRepositoryError(err)
	}
	return checkedIn, nil
}

func (s *Service) CheckOut(ctx context.Context, bookingID, userID int64) (Booking, error) {
	b, err := s.repo.GetByID(ctx, bookingID)
	if err != nil {
		return Booking{}, mapRepositoryError(err)
	}
	if err := s.hotels.RequireManager(ctx, userID, b.HotelID); err != nil {
		return Booking{}, err
	}
	if b.Status != "checked_in" {
		return Booking{}, httpx.Conflict("INVALID_BOOKING_STATUS", "当前订单状态不可退房")
	}
	checkedOut, err := s.repo.CheckOut(ctx, bookingID)
	if errors.Is(err, ErrInvalidStatus) {
		return Booking{}, httpx.Conflict("INVALID_BOOKING_STATUS", "当前订单状态不可退房")
	}
	if err != nil {
		return Booking{}, mapRepositoryError(err)
	}
	return checkedOut, nil
}

// authorize allows the owning customer or a manager of the booking's hotel.
func (s *Service) authorize(ctx context.Context, b Booking, userID int64, role string) error {
	if b.UserID == userID {
		return nil
	}
	if role == "manager" {
		return s.hotels.RequireManager(ctx, userID, b.HotelID)
	}
	return httpx.Forbidden("无权访问该订单")
}

func mapRepositoryError(err error) error {
	var httpErr *httpx.HTTPError
	if errors.As(err, &httpErr) {
		return err
	}
	switch {
	case errors.Is(err, ErrBookingNotFound):
		return httpx.NotFound("订单不存在")
	case errors.Is(err, ErrRoomTypeNotFound):
		return httpx.NotFound("酒店或房型不存在")
	case errors.Is(err, ErrHotelInactive):
		return httpx.BadRequest("酒店未上架，无法预订")
	case errors.Is(err, ErrRoomTypeInactive):
		return httpx.BadRequest("房型已停用，无法预订")
	case errors.Is(err, ErrGuestLimit):
		return httpx.BadRequest("入住人数超过房型上限")
	case errors.Is(err, ErrRoomNotAvailable):
		return httpx.Conflict("ROOM_NOT_AVAILABLE", "所选日期内房型库存不足")
	case errors.Is(err, ErrNoRoomAvailable):
		return httpx.Conflict("ROOM_NOT_AVAILABLE", "没有可用房间可分配")
	case errors.Is(err, ErrRoomUnavailable):
		return httpx.BadRequest("所选房间不可用")
	case errors.Is(err, ErrRoomOccupied):
		return httpx.Conflict("ROOM_NOT_AVAILABLE", "所选房间在入住日期内已被占用")
	default:
		return httpx.Internal(err)
	}
}
