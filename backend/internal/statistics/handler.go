package statistics

import (
	"net/http"

	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type dashboardDTO struct {
	HotelID          int64 `json:"hotel_id"`
	TotalRooms       int   `json:"total_rooms"`
	OccupiedRooms    int   `json:"occupied_rooms"`
	AvailableRooms   int   `json:"available_rooms"`
	UnavailableRooms int   `json:"unavailable_rooms"`
	TodayCheckIns    int   `json:"today_check_ins"`
	TodayCheckOuts   int   `json:"today_check_outs"`
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	hotelID, err := httpx.QueryInt64(r, "hotel_id")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	d, err := h.svc.Dashboard(r.Context(), p.UserID, hotelID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, dashboardDTO{
		HotelID:          d.HotelID,
		TotalRooms:       d.TotalRooms,
		OccupiedRooms:    d.OccupiedRooms,
		AvailableRooms:   d.AvailableRooms,
		UnavailableRooms: d.UnavailableRooms,
		TodayCheckIns:    d.TodayCheckIns,
		TodayCheckOuts:   d.TodayCheckOuts,
	})
}

type occupancyDTO struct {
	HotelID             int64   `json:"hotel_id"`
	StartDate           string  `json:"start_date"`
	EndDate             string  `json:"end_date"`
	TotalRooms          int     `json:"total_rooms"`
	RoomNightsAvailable int     `json:"room_nights_available"`
	RoomNightsOccupied  int     `json:"room_nights_occupied"`
	OccupancyRate       float64 `json:"occupancy_rate"`
}

func (h *Handler) Occupancy(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	hotelID, err := httpx.QueryInt64(r, "hotel_id")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	start, err := httpx.QueryDate(r, "start_date")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	end, err := httpx.QueryDate(r, "end_date")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	o, err := h.svc.Occupancy(r.Context(), p.UserID, hotelID, start, end)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, occupancyDTO{
		HotelID:             o.HotelID,
		StartDate:           httpx.FormatDate(o.StartDate),
		EndDate:             httpx.FormatDate(o.EndDate),
		TotalRooms:          o.TotalRooms,
		RoomNightsAvailable: o.RoomNightsAvailable,
		RoomNightsOccupied:  o.RoomNightsOccupied,
		OccupancyRate:       o.OccupancyRate,
	})
}
