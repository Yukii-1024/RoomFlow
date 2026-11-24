package hotel

import (
	"net/http"
	"time"

	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type hotelDTO struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	City        string    `json:"city"`
	Address     string    `json:"address"`
	Description string    `json:"description"`
	Facilities  []string  `json:"facilities"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toHotelDTO(h Hotel) hotelDTO {
	return hotelDTO{
		ID: h.ID, Name: h.Name, City: h.City, Address: h.Address,
		Description: h.Description, Facilities: h.Facilities, Status: h.Status,
		CreatedAt: h.CreatedAt, UpdatedAt: h.UpdatedAt,
	}
}

type listResponse struct {
	Items    []hotelDTO `json:"items"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
	Total    int        `json:"total"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, pageSize, err := httpx.Pagination(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	hotels, total, err := h.svc.List(r.Context(),
		r.URL.Query().Get("city"), r.URL.Query().Get("keyword"), page, pageSize)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	items := make([]hotelDTO, 0, len(hotels))
	for _, hotel := range hotels {
		items = append(items, toHotelDTO(hotel))
	}
	httpx.WriteJSON(w, http.StatusOK, listResponse{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathInt(r, "hotelId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	hotel, err := h.svc.GetPublic(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toHotelDTO(hotel))
}

type createRequest struct {
	Name        string   `json:"name"`
	City        string   `json:"city"`
	Address     string   `json:"address"`
	Description string   `json:"description"`
	Facilities  []string `json:"facilities"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	var req createRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	hotel, err := h.svc.Create(r.Context(), p.UserID, CreateInput{
		Name: req.Name, City: req.City, Address: req.Address,
		Description: req.Description, Facilities: req.Facilities,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toHotelDTO(hotel))
}

type updateRequest struct {
	Name        *string   `json:"name"`
	City        *string   `json:"city"`
	Address     *string   `json:"address"`
	Description *string   `json:"description"`
	Facilities  *[]string `json:"facilities"`
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	id, err := httpx.PathInt(r, "hotelId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req updateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	hotel, err := h.svc.Update(r.Context(), p.UserID, id, UpdateInput{
		Name: req.Name, City: req.City, Address: req.Address,
		Description: req.Description, Facilities: req.Facilities,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toHotelDTO(hotel))
}

func (h *Handler) ListManaged(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	hotels, err := h.svc.ListManaged(r.Context(), p.UserID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	items := make([]hotelDTO, 0, len(hotels))
	for _, hotel := range hotels {
		items = append(items, toHotelDTO(hotel))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

type statusRequest struct {
	Status string `json:"status"`
}

func (h *Handler) SetStatus(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	id, err := httpx.PathInt(r, "hotelId")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req statusRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	hotel, err := h.svc.SetStatus(r.Context(), p.UserID, id, req.Status)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toHotelDTO(hotel))
}
