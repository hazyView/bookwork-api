package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type HealthHandler struct {
	db *sql.DB
}

type HealthCheck struct {
	Status   string            `json:"status"`
	Database DatabaseHealth    `json:"database"`
	System   SystemHealth      `json:"system"`
	Services map[string]string `json:"services"`
}

type DatabaseHealth struct {
	Status          string `json:"status"`
	OpenConnections int    `json:"open_connections"`
	MaxConnections  int    `json:"max_connections"`
	IdleConnections int    `json:"idle_connections"`
	WaitCount       int64  `json:"wait_count"`
	WaitDuration    string `json:"wait_duration"`
	MaxIdleTime     string `json:"max_idle_time"`
	MaxLifetime     string `json:"max_lifetime"`
}

type SystemHealth struct {
	Status     string `json:"status"`
	GoRoutines int    `json:"goroutines"`
	MemUsage   string `json:"memory_usage"`
	Uptime     string `json:"uptime"`
}

type DatabaseMetrics struct {
	TableStats     []TableStat     `json:"table_stats"`
	IndexUsage     []IndexUsage    `json:"index_usage"`
	SlowQueries    []SlowQuery     `json:"slow_queries"`
	Connections    ConnectionStats `json:"connections"`
	LockMonitoring []LockInfo      `json:"lock_monitoring"`
	UserActivity   []UserActivity  `json:"user_activity"`
	HealthSummary  HealthSummary   `json:"health_summary"`
}

type TableStat struct {
	SchemaName       string `json:"schema_name"`
	TableName        string `json:"table_name"`
	RowCount         int64  `json:"row_count"`
	TableSizeBytes   int64  `json:"table_size_bytes"`
	IndexSizeBytes   int64  `json:"index_size_bytes"`
	TotalSizeBytes   int64  `json:"total_size_bytes"`
	VacuumCount      int64  `json:"vacuum_count"`
	AutovacuumCount  int64  `json:"autovacuum_count"`
	AnalyzeCount     int64  `json:"analyze_count"`
	AutoanalyzeCount int64  `json:"autoanalyze_count"`
}

type IndexUsage struct {
	SchemaName string  `json:"schema_name"`
	TableName  string  `json:"table_name"`
	IndexName  string  `json:"index_name"`
	TimesUsed  int64   `json:"times_used"`
	TableScans int64   `json:"table_scans"`
	UsageRatio float64 `json:"usage_ratio"`
	SizeBytes  int64   `json:"size_bytes"`
}

type SlowQuery struct {
	Query     string  `json:"query"`
	Calls     int64   `json:"calls"`
	TotalTime float64 `json:"total_time"`
	MeanTime  float64 `json:"mean_time"`
	MinTime   float64 `json:"min_time"`
	MaxTime   float64 `json:"max_time"`
	Rows      int64   `json:"rows"`
}

type ConnectionStats struct {
	TotalConnections  int `json:"total_connections"`
	ActiveConnections int `json:"active_connections"`
	IdleConnections   int `json:"idle_connections"`
	WaitingQueries    int `json:"waiting_queries"`
}

type LockInfo struct {
	LockType  string `json:"lock_type"`
	Database  string `json:"database"`
	Relation  string `json:"relation"`
	Page      *int   `json:"page,omitempty"`
	Tuple     *int   `json:"tuple,omitempty"`
	Granted   bool   `json:"granted"`
	ProcessID int    `json:"process_id"`
	Query     string `json:"query"`
}

type UserActivity struct {
	Username     string `json:"username"`
	Database     string `json:"database"`
	Connections  int    `json:"connections"`
	State        string `json:"state"`
	Query        string `json:"current_query"`
	BackendStart string `json:"backend_start"`
}

type HealthSummary struct {
	DatabaseSize   string  `json:"database_size"`
	ActiveQueries  int     `json:"active_queries"`
	SlowQueries    int     `json:"slow_queries"`
	BlockedQueries int     `json:"blocked_queries"`
	CacheHitRatio  float64 `json:"cache_hit_ratio"`
	DeadlockCount  int64   `json:"deadlock_count"`
	TempFilesMB    float64 `json:"temp_files_mb"`
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// HealthCheck endpoint for monitoring
func (h *HealthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	healthCheck := HealthCheck{
		Status:   "healthy",
		Database: h.getDatabaseHealth(),
		System:   h.getSystemHealth(),
		Services: map[string]string{
			"api":      "healthy",
			"database": "healthy",
		},
	}

	// Check if database is actually accessible
	if err := h.db.Ping(); err != nil {
		healthCheck.Status = "unhealthy"
		healthCheck.Database.Status = "unhealthy"
		healthCheck.Services["database"] = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthCheck)
}

// DatabaseMetrics endpoint for detailed database monitoring
func (h *HealthHandler) DatabaseMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := DatabaseMetrics{
		TableStats:     h.getTableStats(),
		IndexUsage:     h.getIndexUsage(),
		SlowQueries:    h.getSlowQueries(),
		Connections:    h.getConnectionStats(),
		LockMonitoring: h.getLockInfo(),
		UserActivity:   h.getUserActivity(),
		HealthSummary:  h.getHealthSummary(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// Individual metric endpoints for targeted monitoring
func (h *HealthHandler) TableStats(w http.ResponseWriter, r *http.Request) {
	stats := h.getTableStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *HealthHandler) SlowQueries(w http.ResponseWriter, r *http.Request) {
	limit := 10 // default limit
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	queries := h.getSlowQueriesWithLimit(limit)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(queries)
}

func (h *HealthHandler) getDatabaseHealth() DatabaseHealth {
	stats := h.db.Stats()

	return DatabaseHealth{
		Status:          "healthy",
		OpenConnections: stats.OpenConnections,
		MaxConnections:  stats.MaxOpenConnections,
		IdleConnections: stats.Idle,
		WaitCount:       stats.WaitCount,
		WaitDuration:    stats.WaitDuration.String(),
		MaxIdleTime:     "N/A", // Not available in sql.DBStats
		MaxLifetime:     "N/A", // Not available in sql.DBStats
	}
}

func (h *HealthHandler) getSystemHealth() SystemHealth {
	// In a real implementation, you'd gather actual system metrics
	return SystemHealth{
		Status:     "healthy",
		GoRoutines: 0, // runtime.NumGoroutine()
		MemUsage:   "unknown",
		Uptime:     "unknown",
	}
}

func (h *HealthHandler) getTableStats() []TableStat {
	query := `
	SELECT 
		schemaname,
		tablename,
		n_tup_ins + n_tup_upd + n_tup_del as total_operations,
		pg_total_relation_size(schemaname||'.'||tablename) as total_size,
		pg_relation_size(schemaname||'.'||tablename) as table_size,
		pg_total_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename) as index_size,
		n_tup_ins,
		n_tup_upd,
		n_tup_del,
		vacuum_count,
		autovacuum_count,
		analyze_count,
		autoanalyze_count
	FROM pg_stat_user_tables
	ORDER BY total_size DESC
	LIMIT 20`

	rows, err := h.db.Query(query)
	if err != nil {
		return []TableStat{}
	}
	defer rows.Close()

	var stats []TableStat
	for rows.Next() {
		var stat TableStat
		var totalOps int64
		err := rows.Scan(
			&stat.SchemaName,
			&stat.TableName,
			&totalOps,
			&stat.TotalSizeBytes,
			&stat.TableSizeBytes,
			&stat.IndexSizeBytes,
			&stat.RowCount, // Using n_tup_ins as approximation
			&stat.VacuumCount,
			&stat.AutovacuumCount,
			&stat.AnalyzeCount,
			&stat.AutoanalyzeCount,
		)
		if err != nil {
			continue
		}
		stats = append(stats, stat)
	}

	return stats
}

func (h *HealthHandler) getIndexUsage() []IndexUsage {
	query := `
	SELECT 
		schemaname,
		tablename,
		indexname,
		idx_scan as times_used,
		seq_scan as table_scans,
		CASE WHEN seq_scan + idx_scan > 0 
			THEN (idx_scan::float / (seq_scan + idx_scan)::float) * 100 
			ELSE 0 
		END as usage_ratio,
		pg_relation_size(schemaname||'.'||indexname) as size_bytes
	FROM pg_stat_user_indexes
	ORDER BY times_used DESC
	LIMIT 50`

	rows, err := h.db.Query(query)
	if err != nil {
		return []IndexUsage{}
	}
	defer rows.Close()

	var usage []IndexUsage
	for rows.Next() {
		var u IndexUsage
		err := rows.Scan(
			&u.SchemaName,
			&u.TableName,
			&u.IndexName,
			&u.TimesUsed,
			&u.TableScans,
			&u.UsageRatio,
			&u.SizeBytes,
		)
		if err != nil {
			continue
		}
		usage = append(usage, u)
	}

	return usage
}

func (h *HealthHandler) getSlowQueries() []SlowQuery {
	return h.getSlowQueriesWithLimit(10)
}

func (h *HealthHandler) getSlowQueriesWithLimit(limit int) []SlowQuery {
	// This requires pg_stat_statements extension
	query := `
	SELECT 
		query,
		calls,
		total_exec_time,
		mean_exec_time,
		min_exec_time,
		max_exec_time,
		rows
	FROM pg_stat_statements
	WHERE query NOT LIKE '%pg_stat_statements%'
	ORDER BY total_exec_time DESC
	LIMIT $1`

	rows, err := h.db.Query(query, limit)
	if err != nil {
		// Extension might not be enabled, return empty slice
		return []SlowQuery{}
	}
	defer rows.Close()

	var queries []SlowQuery
	for rows.Next() {
		var q SlowQuery
		err := rows.Scan(
			&q.Query,
			&q.Calls,
			&q.TotalTime,
			&q.MeanTime,
			&q.MinTime,
			&q.MaxTime,
			&q.Rows,
		)
		if err != nil {
			continue
		}
		queries = append(queries, q)
	}

	return queries
}

func (h *HealthHandler) getConnectionStats() ConnectionStats {
	query := `
	SELECT 
		COUNT(*) as total_connections,
		COUNT(CASE WHEN state = 'active' THEN 1 END) as active_connections,
		COUNT(CASE WHEN state = 'idle' THEN 1 END) as idle_connections,
		COUNT(CASE WHEN wait_event_type IS NOT NULL THEN 1 END) as waiting_queries
	FROM pg_stat_activity
	WHERE backend_type = 'client backend'`

	var stats ConnectionStats
	row := h.db.QueryRow(query)
	err := row.Scan(
		&stats.TotalConnections,
		&stats.ActiveConnections,
		&stats.IdleConnections,
		&stats.WaitingQueries,
	)
	if err != nil {
		return ConnectionStats{}
	}

	return stats
}

func (h *HealthHandler) getLockInfo() []LockInfo {
	query := `
	SELECT 
		mode as lock_type,
		datname as database,
		COALESCE(c.relname, 'unknown') as relation,
		page,
		tuple,
		granted,
		pid as process_id,
		LEFT(query, 100) as query
	FROM pg_locks l
	LEFT JOIN pg_class c ON l.relation = c.oid
	LEFT JOIN pg_database d ON l.database = d.oid
	LEFT JOIN pg_stat_activity a ON l.pid = a.pid
	WHERE NOT granted OR l.mode IN ('AccessExclusiveLock', 'ExclusiveLock')
	ORDER BY granted, mode
	LIMIT 20`

	rows, err := h.db.Query(query)
	if err != nil {
		return []LockInfo{}
	}
	defer rows.Close()

	var locks []LockInfo
	for rows.Next() {
		var lock LockInfo
		err := rows.Scan(
			&lock.LockType,
			&lock.Database,
			&lock.Relation,
			&lock.Page,
			&lock.Tuple,
			&lock.Granted,
			&lock.ProcessID,
			&lock.Query,
		)
		if err != nil {
			continue
		}
		locks = append(locks, lock)
	}

	return locks
}

func (h *HealthHandler) getUserActivity() []UserActivity {
	query := `
	SELECT 
		usename as username,
		datname as database,
		COUNT(*) as connections,
		state,
		LEFT(query, 200) as current_query,
		backend_start::text
	FROM pg_stat_activity
	WHERE backend_type = 'client backend' 
		AND usename IS NOT NULL
	GROUP BY usename, datname, state, query, backend_start
	ORDER BY connections DESC, backend_start DESC
	LIMIT 20`

	rows, err := h.db.Query(query)
	if err != nil {
		return []UserActivity{}
	}
	defer rows.Close()

	var activities []UserActivity
	for rows.Next() {
		var activity UserActivity
		err := rows.Scan(
			&activity.Username,
			&activity.Database,
			&activity.Connections,
			&activity.State,
			&activity.Query,
			&activity.BackendStart,
		)
		if err != nil {
			continue
		}
		activities = append(activities, activity)
	}

	return activities
}

func (h *HealthHandler) getHealthSummary() HealthSummary {
	var summary HealthSummary

	// Database size
	row := h.db.QueryRow("SELECT pg_size_pretty(pg_database_size(current_database()))")
	row.Scan(&summary.DatabaseSize)

	// Active queries
	row = h.db.QueryRow("SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'active' AND backend_type = 'client backend'")
	row.Scan(&summary.ActiveQueries)

	// Blocked queries
	row = h.db.QueryRow("SELECT COUNT(*) FROM pg_stat_activity WHERE wait_event_type IS NOT NULL AND backend_type = 'client backend'")
	row.Scan(&summary.BlockedQueries)

	// Cache hit ratio
	row = h.db.QueryRow(`
		SELECT 
			CASE WHEN blks_read + blks_hit > 0 
				THEN (blks_hit::float / (blks_read + blks_hit)::float) * 100 
				ELSE 0 
			END 
		FROM pg_stat_database 
		WHERE datname = current_database()`)
	row.Scan(&summary.CacheHitRatio)

	// Deadlock count
	row = h.db.QueryRow("SELECT deadlocks FROM pg_stat_database WHERE datname = current_database()")
	row.Scan(&summary.DeadlockCount)

	// Temp files
	row = h.db.QueryRow("SELECT temp_bytes::float / 1024 / 1024 FROM pg_stat_database WHERE datname = current_database()")
	row.Scan(&summary.TempFilesMB)

	return summary
}

// RegisterHealthRoutes registers health monitoring routes
func (h *HealthHandler) RegisterRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/health", h.HealthCheck)
	r.Get("/metrics", h.DatabaseMetrics)
	r.Get("/metrics/tables", h.TableStats)
	r.Get("/metrics/slow-queries", h.SlowQueries)

	return r
}
