-- Create maintenance and utility functions for Bookwork API

-- Function to cleanup expired refresh tokens
CREATE OR REPLACE FUNCTION cleanup_expired_tokens()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM refresh_tokens 
    WHERE expires_at < CURRENT_TIMESTAMP OR is_revoked = true;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Log the cleanup operation
    RAISE NOTICE 'Cleaned up % expired/revoked refresh tokens', deleted_count;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to perform routine maintenance
CREATE OR REPLACE FUNCTION perform_maintenance()
RETURNS TEXT AS $$
DECLARE
    result TEXT := '';
    rec RECORD;
    cleanup_count INTEGER;
BEGIN
    -- Update table statistics for all user tables
    FOR rec IN SELECT schemaname, tablename FROM pg_tables 
               WHERE schemaname = 'public' 
               AND tablename NOT LIKE 'pg_%'
               AND tablename NOT LIKE 'information_schema%' LOOP
        
        EXECUTE 'ANALYZE ' || quote_ident(rec.schemaname) || '.' || quote_ident(rec.tablename);
        result := result || 'Analyzed ' || rec.tablename || E'\n';
    END LOOP;
    
    -- Clean up expired refresh tokens
    SELECT cleanup_expired_tokens() INTO cleanup_count;
    result := result || 'Cleaned up ' || cleanup_count || ' expired refresh tokens' || E'\n';
    
    -- Vacuum analyze main tables for optimal performance
    VACUUM ANALYZE users;
    result := result || 'Vacuumed users table' || E'\n';
    
    VACUUM ANALYZE clubs;
    result := result || 'Vacuumed clubs table' || E'\n';
    
    VACUUM ANALYZE events;
    result := result || 'Vacuumed events table' || E'\n';
    
    VACUUM ANALYZE club_members;
    result := result || 'Vacuumed club_members table' || E'\n';
    
    VACUUM ANALYZE availability;
    result := result || 'Vacuumed availability table' || E'\n';
    
    -- Add timestamp to result
    result := result || 'Maintenance completed at: ' || CURRENT_TIMESTAMP || E'\n';
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Function to get database health metrics
CREATE OR REPLACE FUNCTION get_db_health_metrics()
RETURNS TABLE(
    metric_name TEXT,
    metric_value NUMERIC,
    max_value NUMERIC,
    unit TEXT,
    status TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        'active_connections'::TEXT,
        COUNT(*)::NUMERIC,
        (SELECT setting::NUMERIC FROM pg_settings WHERE name = 'max_connections'),
        'connections'::TEXT,
        CASE 
            WHEN COUNT(*) > (SELECT setting::NUMERIC FROM pg_settings WHERE name = 'max_connections') * 0.8 
            THEN 'WARNING'::TEXT
            ELSE 'OK'::TEXT
        END
    FROM pg_stat_activity
    WHERE state = 'active'
    
    UNION ALL
    
    SELECT 
        'database_size_mb'::TEXT,
        (pg_database_size(current_database()) / 1024 / 1024)::NUMERIC,
        NULL::NUMERIC,
        'MB'::TEXT,
        'INFO'::TEXT
    
    UNION ALL
    
    SELECT 
        'active_users_24h'::TEXT,
        COUNT(DISTINCT user_id)::NUMERIC,
        NULL::NUMERIC,
        'users'::TEXT,
        'INFO'::TEXT
    FROM users 
    WHERE last_login_at > NOW() - INTERVAL '24 hours'
    
    UNION ALL
    
    SELECT 
        'events_this_month'::TEXT,
        COUNT(*)::NUMERIC,
        NULL::NUMERIC,
        'events'::TEXT,
        'INFO'::TEXT
    FROM events 
    WHERE created_at >= date_trunc('month', NOW())
    
    UNION ALL
    
    SELECT 
        'total_clubs'::TEXT,
        COUNT(*)::NUMERIC,
        NULL::NUMERIC,
        'clubs'::TEXT,
        'INFO'::TEXT
    FROM clubs
    WHERE created_at IS NOT NULL;
END;
$$ LANGUAGE plpgsql;

-- Function to validate data integrity
CREATE OR REPLACE FUNCTION check_data_integrity()
RETURNS TABLE(
    check_name TEXT,
    status TEXT,
    details TEXT
) AS $$
BEGIN
    -- Check for orphaned club members
    RETURN QUERY
    SELECT 
        'orphaned_club_members'::TEXT,
        CASE WHEN COUNT(*) = 0 THEN 'OK'::TEXT ELSE 'ERROR'::TEXT END,
        'Found ' || COUNT(*) || ' orphaned club members'
    FROM club_members cm
    LEFT JOIN clubs c ON cm.club_id = c.id
    WHERE c.id IS NULL;
    
    -- Check for orphaned event items
    RETURN QUERY
    SELECT 
        'orphaned_event_items'::TEXT,
        CASE WHEN COUNT(*) = 0 THEN 'OK'::TEXT ELSE 'ERROR'::TEXT END,
        'Found ' || COUNT(*) || ' orphaned event items'
    FROM event_items ei
    LEFT JOIN events e ON ei.event_id = e.id
    WHERE e.id IS NULL;
    
    -- Check for orphaned availability records
    RETURN QUERY
    SELECT 
        'orphaned_availability'::TEXT,
        CASE WHEN COUNT(*) = 0 THEN 'OK'::TEXT ELSE 'ERROR'::TEXT END,
        'Found ' || COUNT(*) || ' orphaned availability records'
    FROM availability a
    LEFT JOIN events e ON a.event_id = e.id
    WHERE e.id IS NULL;
    
    -- Check for expired but not revoked refresh tokens
    RETURN QUERY
    SELECT 
        'expired_tokens'::TEXT,
        CASE WHEN COUNT(*) = 0 THEN 'OK'::TEXT ELSE 'WARNING'::TEXT END,
        'Found ' || COUNT(*) || ' expired but not revoked tokens'
    FROM refresh_tokens
    WHERE expires_at < CURRENT_TIMESTAMP AND is_revoked = false;
END;
$$ LANGUAGE plpgsql;
