-- Insert default admin user and sample data for Bookwork API

-- Insert default admin user (password: admin123)
INSERT INTO users (
    id, 
    name, 
    email, 
    password_hash, 
    role,
    is_active
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'System Administrator',
    'admin@bookwork.com',
    '$2a$10$vI8aWBnW3fID.ZQ4/zo1G.q1lRps.9cGLcZEiGDMVr5yUP1KUOYTa',
    'admin',
    true
) ON CONFLICT (id) DO NOTHING;

-- Insert sample users for testing
INSERT INTO users (name, email, password_hash, role) VALUES 
    ('Jane Smith', 'jane.smith@example.com', '$2a$10$vI8aWBnW3fID.ZQ4/zo1G.q1lRps.9cGLcZEiGDMVr5yUP1KUOYTa', 'member'),
    ('Bob Johnson', 'bob.johnson@example.com', '$2a$10$vI8aWBnW3fID.ZQ4/zo1G.q1lRps.9cGLcZEiGDMVr5yUP1KUOYTa', 'member'),
    ('Alice Wilson', 'alice.wilson@example.com', '$2a$10$vI8aWBnW3fID.ZQ4/zo1G.q1lRps.9cGLcZEiGDMVr5yUP1KUOYTa', 'moderator')
ON CONFLICT (email) DO NOTHING;

-- Create sample book club
INSERT INTO clubs (
    id,
    name,
    description,
    owner_id,
    is_public,
    meeting_frequency,
    location,
    tags
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Classic Literature Club',
    'A book club focused on discussing classic literature from around the world. We meet monthly to explore timeless works and their contemporary relevance.',
    '00000000-0000-0000-0000-000000000001',
    true,
    'monthly',
    'Downtown Library - Conference Room A',
    ARRAY['classics', 'literature', 'discussion', 'monthly']
) ON CONFLICT (id) DO NOTHING;

-- Add sample club members
DO $$
DECLARE
    user_rec RECORD;
    club_id UUID := '00000000-0000-0000-0000-000000000001';
BEGIN
    -- Add all sample users to the club with different roles
    FOR user_rec IN 
        SELECT id, email FROM users 
        WHERE email IN ('jane.smith@example.com', 'bob.johnson@example.com', 'alice.wilson@example.com')
    LOOP
        INSERT INTO club_members (club_id, user_id, role, books_read) 
        VALUES (
            club_id, 
            user_rec.id, 
            CASE 
                WHEN user_rec.email = 'alice.wilson@example.com' THEN 'moderator'
                ELSE 'member'
            END,
            CASE 
                WHEN user_rec.email = 'jane.smith@example.com' THEN 5
                WHEN user_rec.email = 'bob.johnson@example.com' THEN 3
                ELSE 8
            END
        ) ON CONFLICT (club_id, user_id) DO NOTHING;
    END LOOP;
END $$;

-- Create sample events
DO $$
DECLARE
    club_id UUID := '00000000-0000-0000-0000-000000000001';
    admin_id UUID := '00000000-0000-0000-0000-000000000001';
    event1_id UUID;
    event2_id UUID;
BEGIN
    -- Insert first event
    INSERT INTO events (
        club_id, title, description, event_date, event_time, location, 
        book, type, created_by
    ) VALUES (
        club_id,
        'Discussion: Pride and Prejudice',
        'Join us for an engaging discussion about Jane Austen''s timeless classic. We''ll explore themes of love, class, and social expectations in 19th century England.',
        CURRENT_DATE + INTERVAL '2 weeks',
        '19:00:00',
        'Downtown Library - Conference Room A',
        'Pride and Prejudice by Jane Austen',
        'discussion',
        admin_id
    ) RETURNING id INTO event1_id;

    -- Insert second event
    INSERT INTO events (
        club_id, title, description, event_date, event_time, location, 
        book, type, created_by
    ) VALUES (
        club_id,
        'Book Selection Meeting',
        'Help us choose our next book! We''ll review member suggestions and vote on our upcoming reads for the next quarter.',
        CURRENT_DATE + INTERVAL '1 month',
        '18:30:00',
        'Downtown Library - Conference Room A',
        NULL,
        'planning',
        admin_id
    ) RETURNING id INTO event2_id;

    -- Add event items for first event
    INSERT INTO event_items (event_id, name, category, status, created_by) VALUES
        (event1_id, 'Prepare discussion questions about character development', 'task', 'pending', admin_id),
        (event1_id, 'Research historical context of Regency England', 'material', 'pending', admin_id),
        (event1_id, 'Set up refreshments and seating', 'task', 'pending', admin_id);

    -- Add event items for second event  
    INSERT INTO event_items (event_id, name, category, status, created_by) VALUES
        (event2_id, 'Compile member book suggestions', 'task', 'pending', admin_id),
        (event2_id, 'Prepare voting ballots', 'material', 'pending', admin_id),
        (event2_id, 'Create quarterly reading calendar template', 'task', 'pending', admin_id);

END $$;
