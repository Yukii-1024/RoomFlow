package availability

import (
	"net/http"
	"strings"

	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type roomTypeAvailabilityDTO struct {
	TypeID         int64    `json:"type_id"`
	Name           string   `json:"name"`
	AvailableCount int      `json:"available_count"`
	MaxGuests      int      `json:"max_guests"`
	BasePrice      string   `json:"base_price"`
	Features       []string `json:"features"`
}

type availabilityResponse struct {
	HotelID   int64                     `json:"hotel_id"`
	CheckIn   string                    `json:"check_in"`
	CheckOut  string                    `json:"check_out"`
	Nights    int                       `json:"nights"`
	RoomTypes []roomTypeAvailabilityDTO `json:"room_types"`
}

func (h *Handler) Availability(w http.ResponseWriter, r *http.Request) {
	hotelID, err := httpx.PathInt(r, "hotelId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	checkIn, err := httpx.QueryDate(r, "check_in")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	checkOut, err := httpx.QueryDate(r, "check_out")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	guests, err := httpx.QueryInt(r, "guests", 0)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	result, err := h.svc.Availability(r.Context(), hotelID, checkIn, checkOut, guests)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	types := make([]roomTypeAvailabilityDTO, 0, len(result.RoomTypes))
	for _, rt := range result.RoomTypes {
		features := rt.Features
		if features == nil {
			features = []string{}
		}
		types = append(types, roomTypeAvailabilityDTO{
			TypeID:         rt.TypeID,
			Name:           rt.Name,
			AvailableCount: rt.AvailableCount,
			MaxGuests:      rt.MaxGuests,
			BasePrice:      rt.BasePrice,
			Features:       features,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, availabilityResponse{
		HotelID:   result.HotelID,
		CheckIn:   httpx.FormatDate(result.CheckIn),
		CheckOut:  httpx.FormatDate(result.CheckOut),
		Nights:    result.Nights,
		RoomTypes: types,
	})
}

type quoteRequest struct {
	HotelID    int64  `json:"hotel_id"`
	RoomTypeID int64  `json:"room_type_id"`
	CheckIn    string `json:"check_in"`
	CheckOut   string `json:"check_out"`
	Guests     int    `json:"guests"`
}

type quoteResponse struct {
	Currency      string `json:"currency"`
	Nights        int    `json:"nights"`
	RoomRateTotal string `json:"room_rate_total"`
	Fees          string `json:"fees"`
	Total         string `json:"total"`
}

func (h *Handler) Quote(w http.ResponseWriter, r *http.Request) {
	var req quoteRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.HotelID <= 0 {
		httpx.WriteError(w, httpx.BadRequest("hotel_id 不能为空"))
		return
	}
	if req.RoomTypeID <= 0 {
		httpx.WriteError(w, httpx.BadRequest("room_type_id 不能为空"))
		return
	}
	rawCheckIn := strings.TrimSpace(req.CheckIn)
	if rawCheckIn == "" {
		httpx.WriteError(w, httpx.BadRequest("入住日期不能为空"))
		return
	}
	checkIn, err := httpx.ParseDate(rawCheckIn)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	rawCheckOut := strings.TrimSpace(req.CheckOut)
	if rawCheckOut == "" {
		httpx.WriteError(w, httpx.BadRequest("离店日期不能为空"))
		return
	}
	checkOut, err := httpx.ParseDate(rawCheckOut)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	result, err := h.svc.Quote(r.Context(), QuoteInput{
		HotelID:    req.HotelID,
		RoomTypeID: req.RoomTypeID,
		CheckIn:    checkIn,
		CheckOut:   checkOut,
		Guests:     req.Guests,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, quoteResponse{
		Currency:      result.Currency,
		Nights:        result.Nights,
		RoomRateTotal: result.RoomRateTotal,
		Fees:          result.Fees,
		Total:         result.Total,
	})
}
