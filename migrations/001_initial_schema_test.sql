
CREATE TYPE user_role AS ENUM ('admin', 'user');
CREATE TYPE booking_status AS ENUM ('active', 'cancelled');

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role user_role NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    capacity INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL UNIQUE REFERENCES rooms(id) ON DELETE CASCADE,
    days_of_week INT[] NOT NULL, 
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_slots_room_date ON slots(room_id, start_at);

CREATE TABLE IF NOT EXISTS bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slot_id UUID NOT NULL REFERENCES slots(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status booking_status NOT NULL DEFAULT 'active',
    conference_link TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_bookings_active_slot 
ON bookings(slot_id) 
WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_bookings_user ON bookings(user_id, status);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_slots_room_date_start 
ON slots(room_id, start_at);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_bookings_slot_status 
ON bookings(slot_id, status);

DROP INDEX IF EXISTS idx_bookings_active_slot;
CREATE UNIQUE INDEX idx_bookings_active_slot 
ON bookings(slot_id) 
WHERE status = 'active';