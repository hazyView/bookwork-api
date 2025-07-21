-- Create monitoring views for Bookwork API

-- Database health summary view
CREATE OR REPLACE VIEW db_health_summary AS
SELECT 
    'connections' as metric,
    COUNT(*) as value,
    (SELECT setting::int FROM pg_settings WHERE name = 'max_connections') as max_value,
    'Current active connections vs maximum allowed' as description
FROM pg_stat_activity
WHERE state = 'active'

UNION ALL

SELECT 
    'database_size_mb' as metric,
    pg_database_size(current_database()) / 1024 / 1024 as value,
    NULL as max_value,
    'Total database size in megabytes' as description

UNION ALL

SELECT 
    'active_users_24h' as metric,
    COUNT(DISTINCT user_id) as value,
    NULL as max_value,
    'Unique users who logged in within last 24 hours' as description
FROM users 
WHERE last_login_at > NOW() - INTERVAL '24 hours'

UNION ALL

SELECT 
    'events_this_month' as metric,
    COUNT(*) as value,
    NULL as max_value,
    'Events created in current month' as description
FROM events 
WHERE created_at >= date_trunc('month', NOW())

UNION ALL

SELECT 
    'pending_event_items' as metric,
    COUNT(*) as value,
    NULL as max_value,
    'Event items with pending status' as description
FROM event_items 
WHERE status = 'pending'

UNION ALL

SELECT 
    'active_clubs' as metric,
    COUNT(*) as value,
    NULL as max_value,
    'Total number of active clubs' as description
FROM clubs;

-- Query performance monitoring view
CREATE OR REPLACE VIEW slow_queries AS
SELECT 
    LEFT(query, 100) || '...' as query_preview,
    calls,
    total_time,
    mean_time,
    max_time,
    min_time,
    rows,
    100.0 * shared_blks_hit / nullif(shared_blks_hit + shared_blks_read, 0) AS hit_percent
FROM pg_stat_statements 
WHERE calls > 10  -- Only show queries called more than 10 times
ORDER BY mean_time DESC 
LIMIT 20;

-- Table statistics view
CREATE OR REPLACE VIEW table_stats AS
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as total_size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) as table_size,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename)) as index_size,
    n_tup_ins as inserts,
    n_tup_upd as updates,
    n_tup_del as deletes,
    n_live_tup as live_rows,
    n_dead_tup as dead_rows,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    last_autoanalyze
FROM pg_tables t
LEFT JOIN pg_stat_user_tables s ON t.tablename = s.relname AND t.schemaname = s.schemaname
WHERE t.schemaname = 'public' 
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Index usage view
CREATE OR REPLACE VIEW index_usage AS
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan as scans,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched,
    pg_size_pretty(pg_relation_size(indexrelid)) as size,
    CASE 
        WHEN idx_scan = 0 THEN 'UNUSED'
        WHEN idx_scan < 100 THEN 'LOW_USAGE'
        ELSE 'ACTIVE'
    END as usage_category
FROM pg_stat_user_indexes 
ORDER BY idx_scan DESC;

-- Connection activity view
CREATE OR REPLACE VIEW connection_activity AS
SELECT 
    state,
    COUNT(*) as connection_count,
    AVG(EXTRACT(epoch FROM (NOW() - query_start))) as avg_duration_seconds,
    MAX(EXTRACT(epoch FROM (NOW() - query_start))) as max_duration_seconds,
    MIN(EXTRACT(epoch FROM (NOW() - query_start))) as min_duration_seconds
FROM pg_stat_activity 
WHERE state IS NOT NULL
GROUP BY state
ORDER BY connection_count DESC;

-- Lock monitoring view
CREATE OR REPLACE VIEW lock_monitoring AS
SELECT 
    l.locktype,
    l.database,
    l.relation::regclass as relation,
    l.page,
    l.tuple,
    l.virtualxid,
    l.transactionid,
    l.classid,
    l.objid,
    l.objsubid,
    l.virtualtransaction,
    l.pid,
    l.mode,
    l.granted,
    a.usename,
    a.query,
    a.query_start,
    age(now(), a.query_start) AS "age"
FROM pg_locks l
LEFT OUTER JOIN pg_stat_activity a ON l.pid = a.pid
ORDER BY a.query_start;

-- User activity summary view
CREATE OR REPLACE VIEW user_activity_summary AS
SELECT 
    u.id,
    u.name,
    u.email,
    u.role,
    u.is_active,
    u.last_login_at,
    COUNT(DISTINCT cm.club_id) as clubs_joined,
    COUNT(DISTINCT e.id) as events_created,
    COUNT(DISTINCT ei.id) as event_items_assigned,
    COUNT(DISTINCT a.event_id) as events_with_availability
FROM users u
LEFT JOIN club_members cm ON u.id = cm.user_id AND cm.is_active = true
LEFT JOIN events e ON u.id = e.created_by
LEFT JOIN event_items ei ON u.id = ei.assigned_to
LEFT JOIN availability a ON u.id = a.user_id
WHERE u.is_active = true
GROUP BY u.id, u.name, u.email, u.role, u.is_active, u.last_login_at
ORDER BY u.last_login_at DESC NULLS LAST;
