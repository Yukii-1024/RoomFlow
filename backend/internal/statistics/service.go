package statistics

import (
	"context"
	"time"

	"github.com/Yukii-1024/RoomFlow/backend/internal/hotel"
	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
	"github.com/Yukii-1024/RoomFlow/backend/internal/money"
)

type Service struct {
	repo   *Repository
	hotels *hotel.Service
}

func NewService(repo *Repository, hotels *hotel.Service) *Service {
	return &Service{repo: repo, hotels: hotels}
}

type Dashboard struct {
	HotelID          int64
	TotalRooms       int
	OccupiedRooms    int
	AvailableRooms   int
	UnavailableRooms int
	TodayCheckIns    int
	TodayCheckOuts   int
}

func (s *Service) Dashboard(ctx context.Context, managerID, hotelID int64) (Dashboard, error) {
	if err := s.hotels.RequireManager(ctx, managerID, hotelID); err != nil {
		return Dashboard{}, err
	}
	counts, err := s.repo.Dashboard(ctx, hotelID, httpx.Today())
	if err != nil {
		return Dashboard{}, httpx.Internal(err)
	}
	return Dashboard{
		HotelID:          hotelID,
		TotalRooms:       counts.TotalRooms,
		OccupiedRooms:    counts.OccupiedRooms,
		AvailableRooms:   counts.AvailableRooms,
		UnavailableRooms: counts.UnavailableRooms,
		TodayCheckIns:    counts.TodayCheckIns,
		TodayCheckOuts:   counts.TodayCheckOuts,
	}, nil
}

type Occupancy struct {
	HotelID             int64
	StartDate           time.Time
	EndDate             time.Time
	TotalRooms          int
	RoomNightsAvailable int
	RoomNightsOccupied  int
	OccupancyRate       float64
}

func (s *Service) Occupancy(ctx context.Context, managerID, hotelID int64, start, end time.Time) (Occupancy, error) {
	days := httpx.Nights(start, end)
	if days <= 0 {
		return Occupancy{}, httpx.BadRequest("结束日期必须晚于开始日期")
	}
	if err := s.hotels.RequireManager(ctx, managerID, hotelID); err != nil {
		return Occupancy{}, err
	}
	counts, err := s.repo.Occupancy(ctx, hotelID, start, end)
	if err != nil {
		return Occupancy{}, httpx.Internal(err)
	}
	available := counts.ActiveRooms * days
	var rate float64
	if available > 0 {
		rate = money.FormatRate(float64(counts.OccupiedRoomNights) / float64(available))
	}
	return Occupancy{
		HotelID:             hotelID,
		StartDate:           start,
		EndDate:             end,
		TotalRooms:          counts.ActiveRooms,
		RoomNightsAvailable: available,
		RoomNightsOccupied:  counts.OccupiedRoomNights,
		OccupancyRate:       rate,
	}, nil
}
