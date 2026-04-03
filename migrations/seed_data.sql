INSERT INTO users (id, email, password_hash, role, created_at) VALUES 
    (gen_random_uuid(), 'alice@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MrAJqU5q8KJqU5q8KJqU5q8KJqU5q8KJq', 'user', NOW()),
    (gen_random_uuid(), 'bob@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MrAJqU5q8KJqU5q8KJqU5q8KJqU5q8KJq', 'user', NOW()),
    (gen_random_uuid(), 'charlie@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MrAJqU5q8KJqU5q8KJqU5q8KJqU5q8KJq', 'user', NOW())
ON CONFLICT (email) DO NOTHING;

INSERT INTO rooms (id, name, description, capacity, created_at) VALUES 
    (gen_random_uuid(), 'Конференц-зал "Москва"', 
     'Большой конференц-зал с проектором, экраном, видеоконференцсвязью. Идеально подходит для презентаций и совещаний с большим количеством участников.', 
     20, NOW()),
    
    (gen_random_uuid(), 'Переговорная "Санкт-Петербург"', 
     'Уютная переговорная для командных встреч. Оснащена интерактивной доской и маркерной доской.', 
     10, NOW()),
    
    (gen_random_uuid(), 'Meeting Room "London"', 
     'Modern meeting room with video conferencing equipment, 4K display, and comfortable seating.', 
     12, NOW()),
    
    (gen_random_uuid(), 'Raum "Berlin"', 
     'Großer Konferenzraum mit Whiteboard, Beamer und ergonomischen Stühlen.', 
     15, NOW()),
    
    (gen_random_uuid(), 'Sala "Madrid"', 
     'Sala de reuniones con pizarra digital, conexión HDMI y sistema de audio.', 
     8, NOW()),
    
    (gen_random_uuid(), 'Meeting Room "Tokyo"', 
     '静かで集中できる会議室。プロジェクター、ホワイトボード完備。', 
     6, NOW()),
    
    (gen_random_uuid(), 'Conference Hall "New York"', 
     'Spacious conference hall with stage, multiple screens, and professional audio system.', 
     50, NOW()),
    
    (gen_random_uuid(), 'Комната для переговоров "Казань"', 
     'Компактная комната для переговоров 1-2-1 и небольших встреч.', 
     4, NOW())
ON CONFLICT DO NOTHING;

WITH room AS (SELECT id FROM rooms WHERE name LIKE '%Москва%' LIMIT 1)
INSERT INTO schedules (id, room_id, days_of_week, start_time, end_time, created_at)
SELECT 
    gen_random_uuid(),
    id,
    ARRAY[1,2,3,4,5],  
    '09:00',
    '18:00',
    NOW()
FROM room
ON CONFLICT (room_id) DO NOTHING;

WITH room AS (SELECT id FROM rooms WHERE name LIKE '%Санкт-Петербург%' LIMIT 1)
INSERT INTO schedules (id, room_id, days_of_week, start_time, end_time, created_at)
SELECT 
    gen_random_uuid(),
    id,
    ARRAY[1,2,3,4,5,6],  
    '10:00',
    '19:00',
    NOW()
FROM room
ON CONFLICT (room_id) DO NOTHING;

WITH room AS (SELECT id FROM rooms WHERE name LIKE '%London%' LIMIT 1)
INSERT INTO schedules (id, room_id, days_of_week, start_time, end_time, created_at)
SELECT 
    gen_random_uuid(),
    id,
    ARRAY[1,2,3,4,5],
    '08:00',
    '20:00',
    NOW()
FROM room
ON CONFLICT (room_id) DO NOTHING;

WITH room AS (SELECT id FROM rooms WHERE name LIKE '%Berlin%' LIMIT 1)
INSERT INTO schedules (id, room_id, days_of_week, start_time, end_time, created_at)
SELECT 
    gen_random_uuid(),
    id,
    ARRAY[1,2,3,4,5,6,7],  
    '09:00',
    '22:00',
    NOW()
FROM room
ON CONFLICT (room_id) DO NOTHING;

WITH room AS (SELECT id FROM rooms WHERE name LIKE '%Madrid%' LIMIT 1)
INSERT INTO schedules (id, room_id, days_of_week, start_time, end_time, created_at)
SELECT 
    gen_random_uuid(),
    id,
    ARRAY[1,2,3,4,5],
    '08:00',
    '12:00',
    NOW()
FROM room
ON CONFLICT (room_id) DO NOTHING;

WITH room AS (SELECT id FROM rooms WHERE name LIKE '%Tokyo%' LIMIT 1)
INSERT INTO schedules (id, room_id, days_of_week, start_time, end_time, created_at)
SELECT 
    gen_random_uuid(),
    id,
    ARRAY[1,2,3,4,5],
    '15:00',
    '20:00',
    NOW()
FROM room
ON CONFLICT (room_id) DO NOTHING;

DO $$
DECLARE
    room_record RECORD;
    schedule_record RECORD;
    current_date DATE;
    start_time TIME;
    end_time TIME;
    slot_start TIMESTAMP;
    slot_end TIMESTAMP;
    slot_duration INTERVAL := '30 minutes';
    slot_count INTEGER := 0;
BEGIN
    FOR room_record IN SELECT id FROM rooms LOOP
        
        SELECT * INTO schedule_record FROM schedules WHERE room_id = room_record.id;
        
        IF FOUND THEN
            
            FOR day_offset IN 0..13 LOOP
                current_date := CURRENT_DATE + day_offset;
                
                
                IF schedule_record.days_of_week @> ARRAY[EXTRACT(ISODOW FROM current_date)::INT] THEN
                    start_time := schedule_record.start_time;
                    end_time := schedule_record.end_time;
                    
                    slot_start := current_date + start_time;
                    slot_end := current_date + end_time;
                    
                    WHILE slot_start < slot_end LOOP
                        
                        INSERT INTO slots (id, room_id, start_at, end_at)
                        VALUES (gen_random_uuid(), room_record.id, slot_start, slot_start + slot_duration)
                        ON CONFLICT DO NOTHING;
                        
                        slot_start := slot_start + slot_duration;
                        slot_count := slot_count + 1;
                    END LOOP;
                END IF;
            END LOOP;
        END IF;
    END LOOP;
    
    RAISE NOTICE 'Generated % slots for all rooms', slot_count;
END $$;

DO $$
DECLARE
    alice_id UUID;
    bob_id UUID;
    charlie_id UUID;
    slot_record RECORD;
BEGIN
    SELECT id INTO alice_id FROM users WHERE email = 'alice@example.com';
    SELECT id INTO bob_id FROM users WHERE email = 'bob@example.com';
    SELECT id INTO charlie_id FROM users WHERE email = 'charlie@example.com';
    
    
    FOR slot_record IN 
        SELECT s.id, s.start_at, r.name as room_name
        FROM slots s
        JOIN rooms r ON r.id = s.room_id
        WHERE s.start_at > NOW() 
          AND s.start_at < NOW() + INTERVAL '7 days'
        ORDER BY s.start_at
        LIMIT 10
    LOOP
        
        IF slot_record.start_at::DATE % 3 = 0 THEN
            INSERT INTO bookings (id, slot_id, user_id, status, conference_link, created_at)
            VALUES (gen_random_uuid(), slot_record.id, alice_id, 'active', 
                   CONCAT('https://meet.example.com/', REPLACE(slot_record.id::TEXT, '-', ''), '-alice'),
                   NOW())
            ON CONFLICT DO NOTHING;
        ELSIF slot_record.start_at::DATE % 3 = 1 THEN
            INSERT INTO bookings (id, slot_id, user_id, status, conference_link, created_at)
            VALUES (gen_random_uuid(), slot_record.id, bob_id, 'active',
                   CONCAT('https://meet.example.com/', REPLACE(slot_record.id::TEXT, '-', ''), '-bob'),
                   NOW())
            ON CONFLICT DO NOTHING;
        ELSE
            INSERT INTO bookings (id, slot_id, user_id, status, created_at)
            VALUES (gen_random_uuid(), slot_record.id, charlie_id, 'active', NOW())
            ON CONFLICT DO NOTHING;
        END IF;
    END LOOP;
        
    FOR slot_record IN 
        SELECT s.id, s.start_at
        FROM slots s
        WHERE s.start_at < NOW() 
          AND s.start_at > NOW() - INTERVAL '7 days'
        LIMIT 3
    LOOP
        INSERT INTO bookings (id, slot_id, user_id, status, created_at)
        VALUES (gen_random_uuid(), slot_record.id, alice_id, 'cancelled', NOW() - INTERVAL '1 day')
        ON CONFLICT DO NOTHING;
    END LOOP;
    
    RAISE NOTICE 'Created test bookings for users';
END $$;

DO $$
DECLARE
    users_count INTEGER;
    rooms_count INTEGER;
    schedules_count INTEGER;
    slots_count INTEGER;
    bookings_count INTEGER;
    active_bookings INTEGER;
BEGIN
    SELECT COUNT(*) INTO users_count FROM users;
    SELECT COUNT(*) INTO rooms_count FROM rooms;
    SELECT COUNT(*) INTO schedules_count FROM schedules;
    SELECT COUNT(*) INTO slots_count FROM slots;
    SELECT COUNT(*) INTO bookings_count FROM bookings;
    SELECT COUNT(*) INTO active_bookings FROM bookings WHERE status = 'active';
    
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Database Statistics:';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Users: %', users_count;
    RAISE NOTICE 'Rooms: %', rooms_count;
    RAISE NOTICE 'Schedules: %', schedules_count;
    RAISE NOTICE 'Slots: %', slots_count;
    RAISE NOTICE 'Bookings: % (active: %)', bookings_count, active_bookings;
    RAISE NOTICE '========================================';
END $$;