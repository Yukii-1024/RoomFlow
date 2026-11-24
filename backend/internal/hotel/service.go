package hotel

import (
	"context"
	"errors"
	"strings"

	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
	"github.com/Yukii-1024/RoomFlow/backend/internal/validate"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

type CreateInput struct {
	Name        string
	City        string
	Address     string
	Description string
	Facilities  []string
}

func (s *Service) List(ctx context.Context, city, keyword string, page, pageSize int) ([]Hotel, int, error) {
	city = strings.TrimSpace(city)
	keyword = strings.TrimSpace(keyword)
	if len([]rune(city)) > 100 || len([]rune(keyword)) > 100 {
		return nil, 0, httpx.BadRequest("筛选条件过长")
	}
	hotels, total, err := s.repo.ListPublic(ctx, city, keyword, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, httpx.Internal(err)
	}
	return hotels, total, nil
}

func (s *Service) GetPublic(ctx context.Context, id int64) (Hotel, error) {
	h, err := s.repo.GetPublic(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Hotel{}, httpx.NotFound("酒店不存在或未上架")
	}
	if err != nil {
		return Hotel{}, httpx.Internal(err)
	}
	return h, nil
}

func (s *Service) Create(ctx context.Context, managerID int64, in CreateInput) (Hotel, error) {
	h := Hotel{
		Name:        strings.TrimSpace(in.Name),
		City:        strings.TrimSpace(in.City),
		Address:     strings.TrimSpace(in.Address),
		Description: strings.TrimSpace(in.Description),
		Facilities:  validate.StringList(in.Facilities),
	}
	if err := validateHotel(h); err != nil {
		return Hotel{}, err
	}
	created, err := s.repo.Create(ctx, h, managerID)
	if err != nil {
		return Hotel{}, httpx.Internal(err)
	}
	return created, nil
}

type UpdateInput struct {
	Name        *string
	City        *string
	Address     *string
	Description *string
	Facilities  *[]string
}

func (s *Service) Update(ctx context.Context, managerID, hotelID int64, in UpdateInput) (Hotel, error) {
	if err := s.RequireManager(ctx, managerID, hotelID); err != nil {
		return Hotel{}, err
	}

	repoIn := updateFields{}
	if in.Name != nil {
		v := strings.TrimSpace(*in.Name)
		if v == "" || len([]rune(v)) > 150 {
			return Hotel{}, httpx.BadRequest("酒店名称不能为空且不能超过 150 个字符")
		}
		repoIn.Name = &v
	}
	if in.City != nil {
		v := strings.TrimSpace(*in.City)
		if v == "" || len([]rune(v)) > 100 {
			return Hotel{}, httpx.BadRequest("城市不能为空且不能超过 100 个字符")
		}
		repoIn.City = &v
	}
	if in.Address != nil {
		v := strings.TrimSpace(*in.Address)
		if len([]rune(v)) > 255 {
			return Hotel{}, httpx.BadRequest("地址不能超过 255 个字符")
		}
		repoIn.Address = &v
	}
	if in.Description != nil {
		v := strings.TrimSpace(*in.Description)
		if len([]rune(v)) > 5000 {
			return Hotel{}, httpx.BadRequest("酒店介绍不能超过 5000 个字符")
		}
		repoIn.Description = &v
	}
	if in.Facilities != nil {
		v := validate.StringList(*in.Facilities)
		repoIn.Facilities = &v
	}
	if repoIn == (updateFields{}) {
		return Hotel{}, httpx.BadRequest("至少需要提供一个待修改字段")
	}

	h, err := s.repo.Update(ctx, hotelID, repoIn)
	if errors.Is(err, pgx.ErrNoRows) {
		return Hotel{}, httpx.NotFound("酒店不存在")
	}
	if err != nil {
		return Hotel{}, httpx.Internal(err)
	}
	return h, nil
}

func (s *Service) ListManaged(ctx context.Context, managerID int64) ([]Hotel, error) {
	hotels, err := s.repo.ListManaged(ctx, managerID)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	return hotels, nil
}

func (s *Service) SetStatus(ctx context.Context, managerID, hotelID int64, status string) (Hotel, error) {
	if status != "active" && status != "inactive" {
		return Hotel{}, httpx.BadRequest("酒店状态必须为 active 或 inactive")
	}
	if err := s.RequireManager(ctx, managerID, hotelID); err != nil {
		return Hotel{}, err
	}
	h, err := s.repo.SetStatus(ctx, hotelID, status)
	if errors.Is(err, pgx.ErrNoRows) {
		return Hotel{}, httpx.NotFound("酒店不存在")
	}
	if err != nil {
		return Hotel{}, httpx.Internal(err)
	}
	return h, nil
}

// RequireManager returns 403 unless the user is associated with the hotel.
func (s *Service) RequireManager(ctx context.Context, userID, hotelID int64) error {
	ok, err := s.repo.IsManager(ctx, hotelID, userID)
	if err != nil {
		return httpx.Internal(err)
	}
	if !ok {
		return httpx.Forbidden("无权管理该酒店")
	}
	return nil
}

func validateHotel(h Hotel) error {
	if h.Name == "" || len([]rune(h.Name)) > 150 {
		return httpx.BadRequest("酒店名称不能为空且不能超过 150 个字符")
	}
	if h.City == "" || len([]rune(h.City)) > 100 {
		return httpx.BadRequest("城市不能为空且不能超过 100 个字符")
	}
	if len([]rune(h.Address)) > 255 {
		return httpx.BadRequest("地址不能超过 255 个字符")
	}
	if len([]rune(h.Description)) > 5000 {
		return httpx.BadRequest("酒店介绍不能超过 5000 个字符")
	}
	return nil
}
