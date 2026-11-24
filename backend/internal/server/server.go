// Package server wires repositories, services, handlers and the HTTP router.
package server

import (
	"log/slog"
	"net/http"

	"github.com/Yukii-1024/RoomFlow/backend/internal/auth"
	"github.com/Yukii-1024/RoomFlow/backend/internal/availability"
	"github.com/Yukii-1024/RoomFlow/backend/internal/booking"
	"github.com/Yukii-1024/RoomFlow/backend/internal/hotel"
	"github.com/Yukii-1024/RoomFlow/backend/internal/httpx"
	"github.com/Yukii-1024/RoomFlow/backend/internal/middleware"
	"github.com/Yukii-1024/RoomFlow/backend/internal/room"
	"github.com/Yukii-1024/RoomFlow/backend/internal/statistics"
	"github.com/Yukii-1024/RoomFlow/backend/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
)

func New(tokens *auth.Manager, pool *pgxpool.Pool, corsOrigin string, logger *slog.Logger) http.Handler {
	userSvc := user.NewService(user.NewRepository(pool), tokens)
	userHandler := user.NewHandler(userSvc)

	hotelSvc := hotel.NewService(hotel.NewRepository(pool))
	hotelHandler := hotel.NewHandler(hotelSvc)

	roomSvc := room.NewService(room.NewRepository(pool), hotelSvc)
	roomHandler := room.NewHandler(roomSvc)

	availabilitySvc := availability.NewService(availability.NewRepository(pool), hotelSvc)
	availabilityHandler := availability.NewHandler(availabilitySvc)

	bookingSvc := booking.NewService(booking.NewRepository(pool), hotelSvc)
	bookingHandler := booking.NewHandler(bookingSvc)

	statisticsSvc := statistics.NewService(statistics.NewRepository(pool), hotelSvc)
	statisticsHandler := statistics.NewHandler(statisticsSvc)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Auth and profile.
	mux.HandleFunc("POST /api/v1/auth/register", userHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", userHandler.Login)

	authenticated := middleware.Auth(tokens)
	managerOnly := middleware.RequireRole("manager")
	customerOnly := middleware.RequireRole("customer")

	mux.Handle("GET /api/v1/users/me", authenticated(http.HandlerFunc(userHandler.Me)))
	mux.Handle("PATCH /api/v1/users/me", authenticated(http.HandlerFunc(userHandler.UpdateMe)))
	mux.Handle("PATCH /api/v1/users/me/password", authenticated(http.HandlerFunc(userHandler.ChangePassword)))

	// Hotels.
	mux.HandleFunc("GET /api/v1/hotels", hotelHandler.List)
	mux.HandleFunc("GET /api/v1/hotels/{hotelId}", hotelHandler.Get)
	mux.Handle("POST /api/v1/hotels", authenticated(managerOnly(http.HandlerFunc(hotelHandler.Create))))
	mux.Handle("PATCH /api/v1/hotels/{hotelId}", authenticated(managerOnly(http.HandlerFunc(hotelHandler.Update))))
	mux.Handle("PATCH /api/v1/hotels/{hotelId}/status", authenticated(managerOnly(http.HandlerFunc(hotelHandler.SetStatus))))
	mux.Handle("GET /api/v1/manager/hotels", authenticated(managerOnly(http.HandlerFunc(hotelHandler.ListManaged))))

	// Room types.
	mux.HandleFunc("GET /api/v1/hotels/{hotelId}/room-types", roomHandler.ListTypes)
	mux.HandleFunc("GET /api/v1/room-types/{typeId}", roomHandler.GetType)
	mux.Handle("POST /api/v1/hotels/{hotelId}/room-types", authenticated(managerOnly(http.HandlerFunc(roomHandler.CreateType))))
	mux.Handle("PATCH /api/v1/room-types/{typeId}", authenticated(managerOnly(http.HandlerFunc(roomHandler.UpdateType))))
	mux.Handle("PATCH /api/v1/room-types/{typeId}/status", authenticated(managerOnly(http.HandlerFunc(roomHandler.SetTypeStatus))))

	// Physical rooms.
	mux.Handle("GET /api/v1/hotels/{hotelId}/rooms", authenticated(managerOnly(http.HandlerFunc(roomHandler.ListRooms))))
	mux.Handle("POST /api/v1/hotels/{hotelId}/rooms", authenticated(managerOnly(http.HandlerFunc(roomHandler.CreateRoom))))
	mux.Handle("GET /api/v1/rooms/{roomId}", authenticated(managerOnly(http.HandlerFunc(roomHandler.GetRoom))))
	mux.Handle("PATCH /api/v1/rooms/{roomId}", authenticated(managerOnly(http.HandlerFunc(roomHandler.UpdateRoom))))
	mux.Handle("PATCH /api/v1/rooms/{roomId}/status", authenticated(managerOnly(http.HandlerFunc(roomHandler.SetRoomStatus))))
	mux.Handle("DELETE /api/v1/rooms/{roomId}", authenticated(managerOnly(http.HandlerFunc(roomHandler.DeleteRoom))))

	// Public availability and quotes.
	mux.HandleFunc("GET /api/v1/hotels/{hotelId}/availability", availabilityHandler.Availability)
	mux.HandleFunc("POST /api/v1/quotes", availabilityHandler.Quote)

	// Bookings.
	mux.Handle("POST /api/v1/bookings", authenticated(customerOnly(http.HandlerFunc(bookingHandler.Create))))
	mux.Handle("GET /api/v1/bookings", authenticated(customerOnly(http.HandlerFunc(bookingHandler.ListMine))))
	mux.Handle("GET /api/v1/bookings/{bookingId}", authenticated(http.HandlerFunc(bookingHandler.Get)))
	mux.Handle("POST /api/v1/bookings/{bookingId}/cancel", authenticated(http.HandlerFunc(bookingHandler.Cancel)))
	mux.Handle("GET /api/v1/manager/bookings", authenticated(managerOnly(http.HandlerFunc(bookingHandler.ListManaged))))
	mux.Handle("POST /api/v1/bookings/{bookingId}/check-in", authenticated(managerOnly(http.HandlerFunc(bookingHandler.CheckIn))))
	mux.Handle("POST /api/v1/bookings/{bookingId}/check-out", authenticated(managerOnly(http.HandlerFunc(bookingHandler.CheckOut))))

	// Manager statistics.
	mux.Handle("GET /api/v1/manager/dashboard", authenticated(managerOnly(http.HandlerFunc(statisticsHandler.Dashboard))))
	mux.Handle("GET /api/v1/manager/statistics/occupancy", authenticated(managerOnly(http.HandlerFunc(statisticsHandler.Occupancy))))

	return middleware.Chain(mux,
		middleware.Recoverer(logger),
		middleware.Logger(logger),
		middleware.CORS(corsOrigin),
	)
}
