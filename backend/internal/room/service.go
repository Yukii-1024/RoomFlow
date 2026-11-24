package room

import (
	"context"
	"errors"
	"strings"

	"github.com/Yukii-1024/RoomFlow/backend/internal/hotel"
	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
	"github.com/Yukii-1024/RoomFlow/backend/internal/money"
	"github.com/Yukii-1024/RoomFlow/backend/internal/validate"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo   *Repository
	hotels *hotel.Service
}

func NewService(repo *Repository, hotels *hotel.Service) *Service {
	return &Service{repo: repo, hotels: hotels}
}

func (s *Service) ListTypes(ctx context.Context, hotelID int64) ([]RoomType, error) {
	if _, err := s.hotels.GetPublic(ctx, hotelID); err != nil {
		return nil, err
	}
	types, err := s.repo.ListTypes(ctx, hotelID)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	return types, nil
}

func (s *Service) GetType(ctx context.Context, id int64) (RoomType, error) {
	t, err := s.repo.GetTypePublic(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoomType{}, httpx.NotFound("房型不存在或未上架")
	}
	if err != nil {
		return RoomType{}, httpx.Internal(err)
	}
	return t, nil
}

type CreateTypeInput struct {
	Name        string
	Description string
	BasePrice   string
	MaxGuests   int
	Area        *float64
	Amenities   []string
	Features    []string
}

func (s *Service) CreateType(ctx context.Context, managerID, hotelID int64, in CreateTypeInput) (RoomType, error) {
	if err := s.hotels.RequireManager(ctx, managerID, hotelID); err != nil {
		return RoomType{}, err
	}
	name, err := normalizeTypeName(in.Name)
	if err != nil {
		return RoomType{}, err
	}
	description, err := normalizeDescription(in.Description)
	if err != nil {
		return RoomType{}, err
	}
	basePrice, err := validateBasePrice(in.BasePrice)
	if err != nil {
		return RoomType{}, err
	}
	if err := validateMaxGuests(in.MaxGuests); err != nil {
		return RoomType{}, err
	}
	if err := validateArea(in.Area); err != nil {
		return RoomType{}, err
	}
	created, err := s.repo.CreateType(ctx, hotelID, createTypeFields{
		Name:        name,
		Description: description,
		BasePrice:   basePrice,
		MaxGuests:   in.MaxGuests,
		Area:        in.Area,
		Amenities:   validate.StringList(in.Amenities),
		Features:    validate.StringList(in.Features),
	})
	if err != nil {
		return RoomType{}, httpx.Internal(err)
	}
	return created, nil
}

type UpdateTypeInput struct {
	Name        *string
	Description *string
	BasePrice   *string
	MaxGuests   *int
	Area        *float64
	Amenities   *[]string
	Features    *[]string
}

func (s *Service) UpdateType(ctx context.Context, managerID, typeID int64, in UpdateTypeInput) (RoomType, error) {
	existing, err := s.repo.GetType(ctx, typeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoomType{}, httpx.NotFound("房型不存在")
	}
	if err != nil {
		return RoomType{}, httpx.Internal(err)
	}
	if err := s.hotels.RequireManager(ctx, managerID, existing.HotelID); err != nil {
		return RoomType{}, err
	}

	repoIn := updateTypeFields{}
	if in.Name != nil {
		v, err := normalizeTypeName(*in.Name)
		if err != nil {
			return RoomType{}, err
		}
		repoIn.Name = &v
	}
	if in.Description != nil {
		v, err := normalizeDescription(*in.Description)
		if err != nil {
			return RoomType{}, err
		}
		repoIn.Description = &v
	}
	if in.BasePrice != nil {
		v, err := validateBasePrice(*in.BasePrice)
		if err != nil {
			return RoomType{}, err
		}
		repoIn.BasePrice = &v
	}
	if in.MaxGuests != nil {
		if err := validateMaxGuests(*in.MaxGuests); err != nil {
			return RoomType{}, err
		}
		repoIn.MaxGuests = in.MaxGuests
	}
	if in.Area != nil {
		if err := validateArea(in.Area); err != nil {
			return RoomType{}, err
		}
		repoIn.Area = in.Area
	}
	if in.Amenities != nil {
		v := validate.StringList(*in.Amenities)
		repoIn.Amenities = &v
	}
	if in.Features != nil {
		v := validate.StringList(*in.Features)
		repoIn.Features = &v
	}
	if repoIn == (updateTypeFields{}) {
		return RoomType{}, httpx.BadRequest("至少需要提供一个待修改字段")
	}

	updated, err := s.repo.UpdateType(ctx, typeID, repoIn)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoomType{}, httpx.NotFound("房型不存在")
	}
	if err != nil {
		return RoomType{}, httpx.Internal(err)
	}
	return updated, nil
}

func (s *Service) SetTypeStatus(ctx context.Context, managerID, typeID int64, status string) (RoomType, error) {
	if status != "active" && status != "inactive" {
		return RoomType{}, httpx.BadRequest("房型状态必须为 active 或 inactive")
	}
	existing, err := s.repo.GetType(ctx, typeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoomType{}, httpx.NotFound("房型不存在")
	}
	if err != nil {
		return RoomType{}, httpx.Internal(err)
	}
	if err := s.hotels.RequireManager(ctx, managerID, existing.HotelID); err != nil {
		return RoomType{}, err
	}
	updated, err := s.repo.SetTypeStatus(ctx, typeID, status)
	if errors.Is(err, pgx.ErrNoRows) {
		return RoomType{}, httpx.NotFound("房型不存在")
	}
	if err != nil {
		return RoomType{}, httpx.Internal(err)
	}
	return updated, nil
}

func (s *Service) ListRooms(ctx context.Context, managerID, hotelID int64) ([]Room, error) {
	if err := s.hotels.RequireManager(ctx, managerID, hotelID); err != nil {
		return nil, err
	}
	rooms, err := s.repo.ListRooms(ctx, hotelID)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	return rooms, nil
}

func (s *Service) GetRoom(ctx context.Context, managerID, roomID int64) (Room, error) {
	rm, err := s.repo.GetRoom(ctx, roomID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Room{}, httpx.NotFound("房间不存在")
	}
	if err != nil {
		return Room{}, httpx.Internal(err)
	}
	if err := s.hotels.RequireManager(ctx, managerID, rm.HotelID); err != nil {
		return Room{}, err
	}
	return rm, nil
}

type CreateRoomInput struct {
	RoomNo     string
	RoomTypeID int64
	Features   []string
}

func (s *Service) CreateRoom(ctx context.Context, managerID, hotelID int64, in CreateRoomInput) (Room, error) {
	if err := s.hotels.RequireManager(ctx, managerID, hotelID); err != nil {
		return Room{}, err
	}
	roomNo, err := normalizeRoomNo(in.RoomNo)
	if err != nil {
		return Room{}, err
	}
	if in.RoomTypeID <= 0 {
		return Room{}, httpx.BadRequest("房型不能为空")
	}
	ok, err := s.repo.RoomTypeBelongsToHotel(ctx, hotelID, in.RoomTypeID)
	if err != nil {
		return Room{}, httpx.Internal(err)
	}
	if !ok {
		return Room{}, httpx.BadRequest("房型不属于该酒店")
	}
	created, err := s.repo.CreateRoom(ctx, hotelID, createRoomFields{
		RoomNo:     roomNo,
		RoomTypeID: in.RoomTypeID,
		Features:   validate.StringList(in.Features),
	})
	if err != nil {
		if isUniqueViolation(err, "rooms_hotel_id_room_no_key") {
			return Room{}, httpx.Conflict("ROOM_NO_EXISTS", "房号已存在")
		}
		return Room{}, httpx.Internal(err)
	}
	return created, nil
}

type UpdateRoomInput struct {
	RoomNo     *string
	RoomTypeID *int64
	Features   *[]string
}

func (s *Service) UpdateRoom(ctx context.Context, managerID, roomID int64, in UpdateRoomInput) (Room, error) {
	existing, err := s.repo.GetRoom(ctx, roomID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Room{}, httpx.NotFound("房间不存在")
	}
	if err != nil {
		return Room{}, httpx.Internal(err)
	}
	if err := s.hotels.RequireManager(ctx, managerID, existing.HotelID); err != nil {
		return Room{}, err
	}

	repoIn := updateRoomFields{}
	if in.RoomNo != nil {
		v, err := normalizeRoomNo(*in.RoomNo)
		if err != nil {
			return Room{}, err
		}
		repoIn.RoomNo = &v
	}
	if in.RoomTypeID != nil {
		if *in.RoomTypeID <= 0 {
			return Room{}, httpx.BadRequest("房型不能为空")
		}
		if *in.RoomTypeID != existing.RoomTypeID {
			inUse, err := s.repo.HasCheckedInBooking(ctx, roomID)
			if err != nil {
				return Room{}, httpx.Internal(err)
			}
			if inUse {
				return Room{}, httpx.Conflict("ROOM_IN_USE", "房间正在入住中，无法更换房型")
			}
			ok, err := s.repo.RoomTypeBelongsToHotel(ctx, existing.HotelID, *in.RoomTypeID)
			if err != nil {
				return Room{}, httpx.Internal(err)
			}
			if !ok {
				return Room{}, httpx.BadRequest("房型不属于该酒店")
			}
		}
		repoIn.RoomTypeID = in.RoomTypeID
	}
	if in.Features != nil {
		v := validate.StringList(*in.Features)
		repoIn.Features = &v
	}
	if repoIn == (updateRoomFields{}) {
		return Room{}, httpx.BadRequest("至少需要提供一个待修改字段")
	}

	updated, err := s.repo.UpdateRoom(ctx, roomID, repoIn)
	if errors.Is(err, pgx.ErrNoRows) {
		return Room{}, httpx.NotFound("房间不存在")
	}
	if err != nil {
		if isUniqueViolation(err, "rooms_hotel_id_room_no_key") {
			return Room{}, httpx.Conflict("ROOM_NO_EXISTS", "房号已存在")
		}
		return Room{}, httpx.Internal(err)
	}
	return updated, nil
}

type SetRoomStatusInput struct {
	Status string
	Reason string
}

func (s *Service) SetRoomStatus(ctx context.Context, managerID, roomID int64, in SetRoomStatusInput) (Room, error) {
	if in.Status != "active" && in.Status != "maintenance" && in.Status != "inactive" {
		return Room{}, httpx.BadRequest("房间状态必须为 active、maintenance 或 inactive")
	}
	existing, err := s.repo.GetRoom(ctx, roomID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Room{}, httpx.NotFound("房间不存在")
	}
	if err != nil {
		return Room{}, httpx.Internal(err)
	}
	if err := s.hotels.RequireManager(ctx, managerID, existing.HotelID); err != nil {
		return Room{}, err
	}

	reason := ""
	if in.Status != "active" {
		reason = strings.TrimSpace(in.Reason)
		if len([]rune(reason)) > 255 {
			return Room{}, httpx.BadRequest("状态原因不能超过 255 个字符")
		}
		inUse, err := s.repo.HasCheckedInBooking(ctx, roomID)
		if err != nil {
			return Room{}, httpx.Internal(err)
		}
		if inUse {
			return Room{}, httpx.Conflict("ROOM_IN_USE", "房间正在入住中，无法变更运营状态")
		}
	}

	updated, err := s.repo.SetRoomStatus(ctx, roomID, in.Status, reason)
	if errors.Is(err, pgx.ErrNoRows) {
		return Room{}, httpx.NotFound("房间不存在")
	}
	if err != nil {
		return Room{}, httpx.Internal(err)
	}
	return updated, nil
}

func (s *Service) DeleteRoom(ctx context.Context, managerID, roomID int64) error {
	existing, err := s.repo.GetRoom(ctx, roomID)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.NotFound("房间不存在")
	}
	if err != nil {
		return httpx.Internal(err)
	}
	if err := s.hotels.RequireManager(ctx, managerID, existing.HotelID); err != nil {
		return err
	}
	inUse, err := s.repo.HasCheckedInBooking(ctx, roomID)
	if err != nil {
		return httpx.Internal(err)
	}
	if inUse {
		return httpx.Conflict("ROOM_IN_USE", "房间正在入住中，无法删除")
	}
	if err := s.repo.DeleteRoom(ctx, roomID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.NotFound("房间不存在")
		}
		return httpx.Internal(err)
	}
	return nil
}

func normalizeTypeName(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" || len([]rune(v)) > 100 {
		return "", httpx.BadRequest("房型名称不能为空且不能超过 100 个字符")
	}
	return v, nil
}

func normalizeDescription(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if len([]rune(v)) > 2000 {
		return "", httpx.BadRequest("房型描述不能超过 2000 个字符")
	}
	return v, nil
}

func validateBasePrice(raw string) (string, error) {
	cents, err := money.Parse(raw)
	if err != nil || cents <= 0 {
		return "", httpx.BadRequest("基础价格必须为大于 0 的十进制金额")
	}
	return money.Format(cents), nil
}

func validateMaxGuests(v int) error {
	if v < 1 || v > 20 {
		return httpx.BadRequest("最多入住人数必须为 1 到 20")
	}
	return nil
}

func validateArea(v *float64) error {
	if v == nil {
		return nil
	}
	if *v <= 0 || *v > 9999 {
		return httpx.BadRequest("面积必须大于 0 且不超过 9999")
	}
	return nil
}

func normalizeRoomNo(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" || len([]rune(v)) > 32 {
		return "", httpx.BadRequest("房号不能为空且不能超过 32 个字符")
	}
	return v, nil
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}
