-- Database schema for Bookwork API
-- Created: July 18, 2025

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users Table
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  phone VARCHAR(20),
  avatar VARCHAR(500),
  role VARCHAR(20) DEFAULT 'member',
  is_active BOOLEAN DEFAULT true,
  last_login_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Clubs Table
CREATE TABLE clubs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(255) NOT NULL,
  description TEXT,
  owner_id UUID REFERENCES users(id),
  is_public BOOLEAN DEFAULT false,
  max_members INTEGER,
  meeting_frequency VARCHAR(50),
  current_book VARCHAR(255),
  location VARCHAR(255),
  tags TEXT[],
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Club Members Table
CREATE TABLE club_members (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID REFERENCES clubs(id) ON DELETE CASCADE,
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  role VARCHAR(20) DEFAULT 'member',
  joined_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  books_read INTEGER DEFAULT 0,
  is_active BOOLEAN DEFAULT true,
  UNIQUE(club_id, user_id)
);

-- Events Table  
CREATE TABLE events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  club_id UUID REFERENCES clubs(id) ON DELETE CASCADE,
  title VARCHAR(255) NOT NULL,
  description TEXT,
  event_date DATE NOT NULL,
  event_time TIME NOT NULL,
  location VARCHAR(255) NOT NULL,
  book VARCHAR(255),
  type VARCHAR(50) DEFAULT 'discussion',
  max_attendees INTEGER,
  is_public BOOLEAN DEFAULT false,
  created_by UUID REFERENCES users(id),
  attendees UUID[],
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Event Items Table
CREATE TABLE event_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID REFERENCES events(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  category VARCHAR(50) DEFAULT 'other',
  assigned_to UUID REFERENCES users(id),
  status VARCHAR(20) DEFAULT 'pending',
  notes TEXT,
  created_by UUID REFERENCES users(id),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Availability Table
CREATE TABLE availability (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID REFERENCES events(id) ON DELETE CASCADE,
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  status VARCHAR(20) NOT NULL,
  notes TEXT,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(event_id, user_id)
);

-- Refresh Tokens Table (for JWT token management)
CREATE TABLE refresh_tokens (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  token_hash VARCHAR(255) NOT NULL UNIQUE,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  is_revoked BOOLEAN DEFAULT false
);

-- Create indexes for performance
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_club_members_club_id ON club_members(club_id);
CREATE INDEX idx_club_members_user_id ON club_members(user_id);
CREATE INDEX idx_events_club_id ON events(club_id);
CREATE INDEX idx_events_date ON events(event_date);
CREATE INDEX idx_event_items_event_id ON event_items(event_id);
CREATE INDEX idx_availability_event_id ON availability(event_id);
CREATE INDEX idx_availability_user_id ON availability(user_id);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

-- Insert sample admin user for testing
INSERT INTO users (id, name, email, password_hash, role) VALUES (
  '00000000-0000-0000-0000-000000000001',
  'Admin User',
  'admin@bookwork.com',
  '$2a$10$vI8aWBnW3fID.ZQ4/zo1G.q1lRps.9cGLcZEiGDMVr5yUP1KUOYTa', -- password: admin123
  'admin'
);
