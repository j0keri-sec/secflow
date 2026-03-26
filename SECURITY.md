# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | ✅ Supported |

## Reporting a Vulnerability

If you discover a security vulnerability within SecFlow, please follow our responsible disclosure process:

### Private Disclosure (Recommended)

Please report security vulnerabilities privately to the maintainers before disclosing them publicly. This gives us time to develop and test fixes before the vulnerability is widely known.

**How to report:**
1. Email: Please contact the repository maintainers directly
2. Include in your report:
   - Description of the vulnerability
   - Steps to reproduce the issue
   - Potential impact assessment
   - Any suggested fixes (optional)

### Public Disclosure

If you cannot use private disclosure, you may submit the vulnerability through standard public channels. However, we encourage private disclosure to protect users.

### Scope

SecFlow is a vulnerability monitoring and threat intelligence platform. Security issues in dependent services (MongoDB, Redis, etc.) should be reported to their respective maintainers.

## Security Best Practices

When deploying SecFlow in production:

### Authentication
- [ ] Change default JWT secret (`jwt.secret` in config)
- [ ] Change default node token (`node.token_key` in config)
- [ ] Use strong, unique passwords for all admin accounts
- [ ] Enable TLS/SSL for all connections

### Network Security
- [ ] Run MongoDB and Redis on internal networks
- [ ] Use firewall rules to restrict access to SecFlow ports
- [ ] Enable CORS origins restriction in production

### Configuration
- [ ] Set `server.mode: "release"` in production
- [ ] Configure rate limiting for auth endpoints
- [ ] Enable email notifications for critical alerts

### Monitoring
- [ ] Monitor SecFlow logs for suspicious activity
- [ ] Set up Prometheus/Grafana alerts
- [ ] Configure Dead Letter Queue monitoring

## Security Features

SecFlow includes the following security features:

| Feature | Description |
|---------|-------------|
| JWT Authentication | Token-based auth for all API endpoints |
| RBAC | Role-based access control (admin/editor/viewer) |
| Rate Limiting | Protection against brute force attacks |
| SSRF Protection | Blocks requests to private/internal networks |
| TLS Support | Encrypted SMTP and HTTPS connections |
| Input Validation | Sanitization of user inputs |
| Error Message Sanitization | Internal errors not exposed to clients |
| WebSocket Origin Validation | Prevents unauthorized WS connections |

## Security Updates

Security updates will be released as patch versions (e.g., v1.2.1) and announced through:
- GitHub Security Advisories
- CHANGELOG.md

We recommend watching the repository for new releases.
