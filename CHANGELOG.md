# Changelog

## [Unreleased]

### Added
- **Search Functionality**
  - Full-text search across blog posts (title, content, category)
  - Tag-based filtering with `/api/search?tag=<tag>` endpoint
  - Search page at `/search` with live HTMX updates
  - `/api/tags` endpoint returning all unique tags
  - Unit tests for search operations

- bcrypt password hashing for admin authentication
- Rate limiting middleware (5 req/sec, burst 10) to prevent brute force attacks
- SQLite database for persistent storage with automatic YAML migration
- RSS/Atom feed support at `/rss` and `/feed` endpoints
- Unit tests for middleware, database, and config packages (40+ tests total)
- Graceful shutdown with signal handling
- Database indexes on posts (date, category) and services (status)

### Changed
- Migrated data storage from YAML files to SQLite database
- All API handlers now use database queries instead of in-memory data
- All page handlers fetch data from database
- Dockerfile now enables CGO for SQLite support
- Updated health check endpoint to return JSON
- Database schema now includes comments table with foreign key constraints

### Security
- Admin passwords now hashed with bcrypt (cost factor 10)
- Rate limiting on login and comment submission endpoints
- Session tokens use cryptographically secure random generation
- Comment email addresses not exposed in public API responses
- Comments require admin approval before display (anti-spam)

### Dependencies
- Added `github.com/mattn/go-sqlite3` v1.14.24
- Added `golang.org/x/crypto` v0.31.0
- Added `golang.org/x/time` v0.8.0
- Added `github.com/gorilla/feeds` v1.2.0
