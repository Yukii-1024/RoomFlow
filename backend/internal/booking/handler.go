package booking

import (
	"net/http"
	"time"

	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type bookingDTO struct {
	ID              int64      `json:"id"`
	BookingNo       string     `json:"booking_no"`
	Status          string     `json:"status"`
	HotelID         int64      `json:"hotel_id"`
	HotelName       string     `json:"hotel_name"`
	RoomTypeID      int64      `json:"room_type_id"`
	RoomTypeName    string     `json:"room_type_name"`
	RoomID          *int64     `json:"room_id"`
	RoomNo          *string    `json:"room_no"`
	CheckIn         string     `json:"check_in"`
	CheckOut        string     `json:"check_out"`
	Nights          int        `json:"nights"`
	Guests          int        `json:"guests"`
	Currency        string     `json:"currency"`
	TotalAmount     string     `json:"total_amount"`
	ContactName     string     `json:"contact_name"`
	ContactPhone    string     `json:"contact_phone"`
	SpecialRequests string     `json:"special_requests"`
	CreatedAt       time.Time  `json:"created_at"`
	CancelledAt     *time.Time `json:"cancelled_at"`
	CheckedInAt     *time.Time `json:"checked_in_at"`
	CheckedOutAt    *time.Time `json:"checked_out_at"`
}

func toBookingDTO(b Booking) bookingDTO {
	return bookingDTO{
		ID:              b.ID,
		BookingNo:       b.BookingNo,
		Status:          b.Status,
		HotelID:         b.HotelID,
		HotelName:       b.HotelName,
		RoomTypeID:      b.RoomTypeID,
		RoomTypeName:    b.RoomTypeName,
		RoomID:          b.RoomID,
		RoomNo:          b.RoomNo,
		CheckIn:         httpx.FormatDate(b.CheckIn),
		CheckOut:        httpx.FormatDate(b.CheckOut),
		Nights:          httpx.Nights(b.CheckIn, b.CheckOut),
		Guests:          b.Guests,
		Currency:        b.Currency,
		TotalAmount:     b.TotalAmount,
		ContactName:     b.ContactName,
		ContactPhone:    b.ContactPhone,
		SpecialRequests: b.SpecialRequests,
		CreatedAt:       b.CreatedAt,
		CancelledAt:     b.CancelledAt,
		CheckedInAt:     b.CheckedInAt,
		CheckedOutAt:    b.CheckedOutAt,
	}
}

type listResponse struct {
	Items    []bookingDTO `json:"items"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
	Total    int          `json:"total"`
}

type createRequest struct {
	HotelID         int64  `json:"hotel_id"`
	RoomTypeID      int64  `json:"room_type_id"`
	CheckIn         string `json:"check_in"`
	CheckOut        string `json:"check_out"`
	Guests          int    `json:"guests"`
	ContactName     string `json:"contact_name"`
	ContactPhone    string `json:"contact_phone"`
	SpecialRequests string `json:"special_requests"`
}

type createResponse struct {
	ID          int64  `json:"id"`
	BookingNo   string `json:"booking_no"`
	Status      string `json:"status"`
	TotalAmount string `json:"total_amount"`
	RoomID      *int64 `json:"room_id"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	var req createRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	checkIn, err := httpx.ParseDate(req.CheckIn)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	checkOut, err := httpx.ParseDate(req.CheckOut)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	b, err := h.svc.Create(r.Context(), p.UserID, CreateInput{
		HotelID:         req.HotelID,
		RoomTypeID:      req.RoomTypeID,
		CheckIn:         checkIn,
		CheckOut:        checkOut,
		Guests:          req.Guests,
		ContactName:     req.ContactName,
		ContactPhone:    req.ContactPhone,
		SpecialRequests: req.SpecialRequests,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, createResponse{
		ID:          b.ID,
		BookingNo:   b.BookingNo,
		Status:      b.Status,
		TotalAmount: b.TotalAmount,
		RoomID:      b.RoomID,
	})
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	page, pageSize, err := httpx.Pagination(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	bookings, total, err := h.svc.ListMine(r.Context(), p.UserID, page, pageSize)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toListResponse(bookings, page, pageSize, total))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	id, err := httpx.PathInt(r, "bookingId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	b, err := h.svc.Get(r.Context(), id, p.UserID, p.Role)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBookingDTO(b))
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	id, err := httpx.PathInt(r, "bookingId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	b, err := h.svc.Cancel(r.Context(), id, p.UserID, p.Role)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBookingDTO(b))
}

func (h *Handler) ListManaged(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	hotelID, err := httpx.QueryInt64(r, "hotel_id")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	page, pageSize, err := httpx.Pagination(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	bookings, total, err := h.svc.ListManaged(r.Context(), p.UserID, hotelID,
		r.URL.Query().Get("status"), page, pageSize)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toListResponse(bookings, page, pageSize, total))
}

type checkInRequest struct {
	RoomID *int64 `json:"room_id"`
}

func (h *Handler) CheckIn(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	id, err := httpx.PathInt(r, "bookingId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req checkInRequest
	if err := httpx.DecodeOptional(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	b, err := h.svc.CheckIn(r.Context(), id, p.UserID, req.RoomID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBookingDTO(b))
}

func (h *Handler) CheckOut(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	id, err := httpx.PathInt(r, "bookingId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	b, err := h.svc.CheckOut(r.Context(), id, p.UserID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toBookingDTO(b))
}

func toListResponse(bookings []Booking, page, pageSize, total int) listResponse {
	items := make([]bookingDTO, 0, len(bookings))
	for _, b := range bookings {
		items = append(items, toBookingDTO(b))
	}
	return listResponse{Items: items, Page: page, PageSize: pageSize, Total: total}
}
