-- RoomFlow initial schema.
-- Money is NUMERIC(12,2); dates are DATE; timestamps are TIMESTAMPTZ.

CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    name          TEXT        NOT NULL,
    phone         TEXT        NOT NULL DEFAULT '',
    role          TEXT        NOT NULL DEFAULT 'customer' CHECK (role IN ('customer', 'manager')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_email_lower_key ON users (lower(email));

CREATE TABLE hotels (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT        NOT NULL,
    city        TEXT        NOT NULL,
    address     TEXT        NOT NULL DEFAULT '',
    description TEXT        NOT NULL DEFAULT '',
    facilities  TEXT[]      NOT NULL DEFAULT '{}',
    status      TEXT        NOT NULL DEFAULT 'inactive' CHECK (status IN ('active', 'inactive')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX hotels_city_idx ON hotels (city);
CREATE INDEX hotels_status_idx ON hotels (status);

CREATE TABLE hotel_managers (
    hotel_id   BIGINT      NOT NULL REFERENCES hotels (id) ON DELETE CASCADE,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (hotel_id, user_id)
);

CREATE INDEX hotel_managers_user_idx ON hotel_managers (user_id);

CREATE TABLE room_types (
    id          BIGSERIAL PRIMARY KEY,
    hotel_id    BIGINT        NOT NULL REFERENCES hotels (id) ON DELETE CASCADE,
    name        TEXT          NOT NULL,
    description TEXT          NOT NULL DEFAULT '',
    base_price  NUMERIC(12, 2) NOT NULL CHECK (base_price >= 0),
    max_guests  INTEGER       NOT NULL CHECK (max_guests > 0),
    area        NUMERIC(6, 1) CHECK (area > 0),
    amenities   TEXT[]        NOT NULL DEFAULT '{}',
    features    TEXT[]        NOT NULL DEFAULT '{}',
    status      TEXT          NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    UNIQUE (hotel_id, id)
);

CREATE INDEX room_types_hotel_idx ON room_types (hotel_id);

CREATE TABLE rooms (
    id            BIGSERIAL PRIMARY KEY,
    hotel_id      BIGINT      NOT NULL REFERENCES hotels (id) ON DELETE CASCADE,
    room_type_id  BIGINT      NOT NULL,
    room_no       TEXT        NOT NULL,
    features      TEXT[]      NOT NULL DEFAULT '{}',
    status        TEXT        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'maintenance', 'inactive')),
    status_reason TEXT        NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (hotel_id, room_no),
    UNIQUE (hotel_id, id),
    FOREIGN KEY (hotel_id, room_type_id) REFERENCES room_types (hotel_id, id) ON DELETE RESTRICT
);

CREATE INDEX rooms_room_type_idx ON rooms (room_type_id);
CREATE INDEX rooms_status_idx ON rooms (hotel_id, status);

CREATE TABLE bookings (
    id               BIGSERIAL PRIMARY KEY,
    booking_no       TEXT UNIQUE,
    user_id          BIGINT        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    hotel_id         BIGINT        NOT NULL REFERENCES hotels (id) ON DELETE RESTRICT,
    room_type_id     BIGINT        NOT NULL,
    room_id          BIGINT        REFERENCES rooms (id) ON DELETE SET NULL,
    check_in         DATE          NOT NULL,
    check_out        DATE          NOT NULL,
    guests           INTEGER       NOT NULL CHECK (guests > 0),
    contact_name     TEXT          NOT NULL,
    contact_phone    TEXT          NOT NULL,
    special_requests TEXT          NOT NULL DEFAULT '',
    status           TEXT          NOT NULL DEFAULT 'confirmed'
        CHECK (status IN ('confirmed', 'checked_in', 'checked_out', 'cancelled')),
    currency         TEXT          NOT NULL DEFAULT 'CNY',
    total_amount     NUMERIC(12, 2) NOT NULL CHECK (total_amount >= 0),
    cancelled_at     TIMESTAMPTZ,
    checked_in_at    TIMESTAMPTZ,
    checked_out_at   TIMESTAMPTZ,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    CHECK (check_out > check_in),
    FOREIGN KEY (hotel_id, room_type_id) REFERENCES room_types (hotel_id, id) ON DELETE RESTRICT
);

CREATE INDEX bookings_user_idx ON bookings (user_id, created_at DESC);
CREATE INDEX bookings_hotel_idx ON bookings (hotel_id, created_at DESC);
CREATE INDEX bookings_overlap_idx ON bookings (room_type_id, check_in, check_out)
    WHERE status IN ('confirmed', 'checked_in');
CREATE INDEX bookings_room_idx ON bookings (room_id) WHERE room_id IS NOT NULL;
CREATE INDEX bookings_checkin_idx ON bookings (hotel_id, check_in);
CREATE INDEX bookings_checkout_idx ON bookings (hotel_id, check_out);
