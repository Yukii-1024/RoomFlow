package user

import (
	"net/http"
	"time"

	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type userDTO struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func toUserDTO(u User) userDTO {
	return userDTO{
		ID: u.ID, Email: u.Email, Name: u.Name, Phone: u.Phone,
		Role: u.Role, CreatedAt: u.CreatedAt,
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	u, err := h.svc.Register(r.Context(), RegisterInput{
		Email: req.Email, Password: req.Password, Name: req.Name, Phone: req.Phone,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toUserDTO(u))
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	User        struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Role string `json:"role"`
	} `json:"user"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	token, expiresIn, u, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var resp loginResponse
	resp.AccessToken = token
	resp.TokenType = "Bearer"
	resp.ExpiresIn = expiresIn
	resp.User.ID = u.ID
	resp.User.Name = u.Name
	resp.User.Role = u.Role
	httpx.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	u, err := h.svc.Me(r.Context(), p.UserID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toUserDTO(u))
}

type updateMeRequest struct {
	Name  *string `json:"name"`
	Phone *string `json:"phone"`
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	var req updateMeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	u, err := h.svc.UpdateProfile(r.Context(), p.UserID, req.Name, req.Phone)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toUserDTO(u))
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	p := httpx.MustPrincipal(r.Context())
	var req changePasswordRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.svc.ChangePassword(r.Context(), p.UserID, req.CurrentPassword, req.NewPassword); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, nil)
}
