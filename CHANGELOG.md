# Changelog

All notable changes to the SecFlow project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Password Reset System** - Users can now reset forgotten passwords via email token-based flow
  - `POST /api/v1/auth/reset/request` - Request password reset by email
  - `POST /api/v1/auth/reset/confirm` - Confirm reset with token and new password
  - `PasswordResetToken` model for secure token storage
  - `PasswordResetTokenRepo` for token lifecycle management
  - Token validity: 15 minutes (configurable)
  - One-time use tokens (invalidated after use)
  - Audit logging for security events

- **User Repository Enhancements**
  - `GetByEmail()` method for email-based user lookup (case-insensitive)

- **Unit Test Suite** (`secflow-server/tests/`)
  - `TestUserRolePermissions` - Role-based access control tests
  - `TestInviteCodeModel` - Invite code functionality tests
  - `TestPasswordHashing` - Password hashing verification (bcrypt)
  - `TestJWTToken` - JWT generation and validation tests
  - `TestUserModel` - User model structure tests
  - `TestRegisterRequest` - Registration request validation tests
  - `TestPasswordResetToken` - Password reset token flow tests
  - `TestRoleBasedAccess` - Role hierarchy verification
  - `TestInviteCodeGeneration` - Invite code uniqueness tests
  - Integration test placeholders for full workflow testing
  - Benchmark tests for password hashing and token generation

### Changed
- `InviteCode` model now tracks `UsedByID` when code is redeemed
- Collection constant `CollPasswordResetTokens` added

### Security
- Password reset tokens are hashed before storage (SHA-256)
- Email enumeration attack prevention (always returns success)
- One-time token usage prevents replay attacks
- Audit logging for all authentication events

---

## [v0.1.0] - 2026-03-24 (Initial Release)

### Added
- **Distributed Security Information Flow Platform**
  - `secflow-server` - Go backend with Gin framework
  - `secflow-client` - Distributed crawler nodes with go-rod
  - `secflow-web` - Vue 3 + TypeScript frontend

### Core Features
- **Authentication & Authorization**
  - JWT-based authentication
  - Role-based access control (admin/editor/viewer)
  - Invite code registration system
  - First user automatically becomes admin (bootstrap protection)

- **Task Scheduling**
  - Redis-based priority queue
  - Automatic retry with exponential backoff
  - Dead letter queue for failed tasks
  - Multi-node load balancing

- **WebSocket Hub**
  - Real-time task distribution
  - Node heartbeat monitoring
  - Progress reporting
  - Result collection

- **Data Sources** (22 total)
  - 15 vulnerability sources (AVD, Seebug, CVE, NVD, etc.)
  - 7 security article sources

- **Notification Channels**
  - DingTalk (钉钉)
  - Feishu (飞书)
  - WeChat Work (企业微信)
  - Slack
  - Telegram
  - Webhook

- **Monitoring**
  - Prometheus metrics
  - Grafana dashboards
  - Structured logging (zerolog)

### Documentation
- `docs/README.md` - Project overview
- `docs/architecture/README.md` - System architecture
- `docs/backend/README.md` - Backend development guide
- `docs/deployment/README.md` - Deployment guide
- `docs/quick-start.md` - Quick start guide
- `docs/references/` - Complete API and config references

### Deployment
- Docker & docker-compose support
- Docker Compose for production (`docker-compose.prod.yml`)
- MongoDB and Redis dependencies
- Nginx configuration templates

---

## Development Notes

### Adding New Features

#### 1. New Crawler Source
```go
// 1. Create grabber in secflow-client/pkg/vulngrabber/
// 2. Implement Grabber interface
// 3. Register in secflow-client/pkg/vulngrabber/registry.go
// 4. Update defaultVulnSources in server
```

#### 2. New Push Channel
```go
// 1. Create pusher in secflow-server/pkg/pusher/
// 2. Implement Pusher interface
// 3. Register in secflow-server/pkg/pusher/factory.go
// 4. Add channel config UI in secflow-web
```

#### 3. New API Endpoint
```go
// 1. Add handler method in internal/api/handler/
// 2. Register route in internal/api/router.go
// 3. Add tests in tests/
// 4. Document in docs/
```

### Testing
```bash
# Run unit tests
cd secflow-server && go test ./tests/... -v

# Run all tests
go test ./... -v

# Run with coverage
go test ./... -coverprofile=coverage.out
```

### Code Style
- Follow [projectdiscovery/nuclei](https://github.com/projectdiscovery/nuclei) style
- Use `fmt.Errorf` for error wrapping: `"doing something: %w"`
- Context always passed as first parameter
- Godoc comments for all exported functions

---

## Migration Guides

### Upgrading from v0.1.0 to v0.2.0
- No breaking changes expected
- New password_reset_tokens collection will be created automatically
- Run `go mod tidy` to update dependencies

---

## Deprecation Notices

- `secflow-agent` has been merged into `secflow-client`
- Legacy REST authentication replaced with JWT (v0.1.0)

---

## Security Considerations

- Change JWT secret in production (`config.yaml`)
- Use strong invite codes (not guessable)
- Enable HTTPS in production
- Configure firewall rules for MongoDB/Redis
- Review audit logs regularly
- Token expiration: 72h default (configurable)

---

## Contact & Support

- Issues: https://github.com/secflow/secflow/issues
- Documentation: https://github.com/secflow/secflow/docs

---

## License

MIT License - see LICENSE file
