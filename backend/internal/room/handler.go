package room

import (
	"net/http"
	"time"

	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type roomTypeDTO struct {
	ID          int64     `json:"id"`
	HotelID     int64     `json:"hotel_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	BasePrice   string    `json:"base_price"`
	MaxGuests   int       `json:"max_guests"`
	Area        *float64  `json:"area"`
	Amenities   []string  `json:"amenities"`
	Features    []string  `json:"features"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toRoomTypeDTO(t RoomType) roomTypeDTO {
	return roomTypeDTO{
		ID: t.ID, HotelID: t.HotelID, Name: t.Name, Description: t.Description,
		BasePrice: t.BasePrice, MaxGuests: t.MaxGuests, Area: t.Area,
		Amenities: t.Amenities, Features: t.Features, Status: t.Status,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

type roomDTO struct {
	ID           int64     `json:"id"`
	HotelID      int64     `json:"hotel_id"`
	RoomTypeID   int64     `json:"room_type_id"`
	RoomTypeName string    `json:"room_type_name"`
	RoomNo       string    `json:"room_no"`
	Features     []string  `json:"features"`
	Status       string    `json:"status"`
	StatusReason string    `json:"status_reason"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func toRoomDTO(rm Room) roomDTO {
	return roomDTO{
		ID: rm.ID, HotelID: rm.HotelID, RoomTypeID: rm.RoomTypeID,
		RoomTypeName: rm.RoomTypeName, RoomNo: rm.RoomNo, Features: rm.Features,
		Status: rm.Status, StatusReason: rm.StatusReason,
		CreatedAt: rm.CreatedAt, UpdatedAt: rm.UpdatedAt,
	}
}

func (h *Handler) ListTypes(w http.ResponseWriter, r *http.Request) {
	hotelID, err := httpx.PathInt(r, "hotelId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	types, err := h.svc.ListTypes(r.Context(), hotelID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	items := make([]roomTypeDTO, 0, len(types))
	for _, t := range types {
		items = append(items, toRoomTypeDTO(t))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) GetType(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathInt(r, "typeId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.GetType(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toRoomTypeDTO(t))
}

type createTypeRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	BasePrice   string   `json:"base_price"`
	MaxGuests   int      `json:"max_guests"`
	Area        *float64 `json:"area"`
	Amenities   []string `json:"amenities"`
	Features    []string `json:"features"`
}

func (h *Handler) CreateType(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	hotelID, err := httpx.PathInt(r, "hotelId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req createTypeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.CreateType(r.Context(), p.UserID, hotelID, CreateTypeInput{
		Name: req.Name, Description: req.Description, BasePrice: req.BasePrice,
		MaxGuests: req.MaxGuests, Area: req.Area, Amenities: req.Amenities, Features: req.Features,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toRoomTypeDTO(t))
}

type updateTypeRequest struct {
	Name        *string   `json:"name"`
	Description *string   `json:"description"`
	BasePrice   *string   `json:"base_price"`
	MaxGuests   *int      `json:"max_guests"`
	Area        *float64  `json:"area"`
	Amenities   *[]string `json:"amenities"`
	Features    *[]string `json:"features"`
}

func (h *Handler) UpdateType(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	id, err := httpx.PathInt(r, "typeId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req updateTypeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.UpdateType(r.Context(), p.UserID, id, UpdateTypeInput{
		Name: req.Name, Description: req.Description, BasePrice: req.BasePrice,
		MaxGuests: req.MaxGuests, Area: req.Area, Amenities: req.Amenities, Features: req.Features,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toRoomTypeDTO(t))
}

type typeStatusRequest struct {
	Status string `json:"status"`
}

func (h *Handler) SetTypeStatus(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	id, err := httpx.PathInt(r, "typeId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req typeStatusRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.SetTypeStatus(r.Context(), p.UserID, id, req.Status)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toRoomTypeDTO(t))
}

func (h *Handler) ListRooms(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	hotelID, err := httpx.PathInt(r, "hotelId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	rooms, err := h.svc.ListRooms(r.Context(), p.UserID, hotelID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	items := make([]roomDTO, 0, len(rooms))
	for _, rm := range rooms {
		items = append(items, toRoomDTO(rm))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) GetRoom(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	id, err := httpx.PathInt(r, "roomId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	rm, err := h.svc.GetRoom(r.Context(), p.UserID, id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toRoomDTO(rm))
}

type createRoomRequest struct {
	RoomNo     string   `json:"room_no"`
	RoomTypeID int64    `json:"room_type_id"`
	Features   []string `json:"features"`
}

func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	hotelID, err := httpx.PathInt(r, "hotelId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req createRoomRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	rm, err := h.svc.CreateRoom(r.Context(), p.UserID, hotelID, CreateRoomInput{
		RoomNo: req.RoomNo, RoomTypeID: req.RoomTypeID, Features: req.Features,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toRoomDTO(rm))
}

type updateRoomRequest struct {
	RoomNo     *string   `json:"room_no"`
	RoomTypeID *int64    `json:"room_type_id"`
	Features   *[]string `json:"features"`
}

func (h *Handler) UpdateRoom(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	id, err := httpx.PathInt(r, "roomId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req updateRoomRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	rm, err := h.svc.UpdateRoom(r.Context(), p.UserID, id, UpdateRoomInput{
		RoomNo: req.RoomNo, RoomTypeID: req.RoomTypeID, Features: req.Features,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toRoomDTO(rm))
}

type roomStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func (h *Handler) SetRoomStatus(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	id, err := httpx.PathInt(r, "roomId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req roomStatusRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	rm, err := h.svc.SetRoomStatus(r.Context(), p.UserID, id, SetRoomStatusInput{
		Status: req.Status, Reason: req.Reason,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toRoomDTO(rm))
}

func (h *Handler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	id, err := httpx.PathInt(r, "roomId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.DeleteRoom(r.Context(), p.UserID, id); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, nil)
}
