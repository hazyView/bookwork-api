-- Create performance indexes for Bookwork API
-- These indexes optimize the most common query patterns

-- Users table indexes
CREATE UNIQUE INDEX idx_users_email_unique ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;
CREATE INDEX idx_users_last_login ON users(last_login_at);

-- Add email validation constraint
ALTER TABLE users ADD CONSTRAINT check_email_format 
    CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');

-- Clubs table indexes
CREATE INDEX idx_clubs_owner ON clubs(owner_id);
CREATE INDEX idx_clubs_public ON clubs(is_public) WHERE is_public = true;
CREATE INDEX idx_clubs_name ON clubs(name);
CREATE INDEX idx_clubs_tags ON clubs USING GIN(tags);

-- Club members table indexes
CREATE INDEX idx_club_members_club_id ON club_members(club_id);
CREATE INDEX idx_club_members_user_id ON club_members(user_id);
CREATE INDEX idx_club_members_active ON club_members(club_id, is_active) WHERE is_active = true;
CREATE INDEX idx_club_members_role ON club_members(club_id, role);

-- Events table indexes
CREATE INDEX idx_events_club_id ON events(club_id);
CREATE INDEX idx_events_date ON events(event_date);
CREATE INDEX idx_events_club_date ON events(club_id, event_date);
CREATE INDEX idx_events_type ON events(type);
CREATE INDEX idx_events_created_by ON events(created_by);
CREATE INDEX idx_events_attendees ON events USING GIN(attendees);

-- Event items table indexes
CREATE INDEX idx_event_items_event_id ON event_items(event_id);
CREATE INDEX idx_event_items_assigned_to ON event_items(assigned_to);
CREATE INDEX idx_event_items_status ON event_items(status);
CREATE INDEX idx_event_items_category ON event_items(category);
CREATE INDEX idx_event_items_event_status ON event_items(event_id, status);

-- Availability table indexes
CREATE INDEX idx_availability_event_id ON availability(event_id);
CREATE INDEX idx_availability_user_id ON availability(user_id);
CREATE INDEX idx_availability_status ON availability(status);
CREATE INDEX idx_availability_event_status ON availability(event_id, status);

-- Refresh tokens table indexes
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_active ON refresh_tokens(user_id, expires_at, is_revoked) 
    WHERE is_revoked = false;

-- Compound indexes for common query patterns
CREATE INDEX idx_club_members_active_role 
    ON club_members(club_id, is_active, role) 
    WHERE is_active = true;

CREATE INDEX idx_events_club_date_type 
    ON events(club_id, event_date, type);

CREATE INDEX idx_availability_event_summary 
    ON availability(event_id, status) 
    INCLUDE (user_id);

-- Partial indexes for common filters
CREATE INDEX idx_users_active_email 
    ON users(email) 
    WHERE is_active = true;

CREATE INDEX idx_events_future 
    ON events(event_date, club_id) 
    WHERE event_date >= CURRENT_DATE;
