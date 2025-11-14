# Security Policy

## Supported Versions

We actively support the following versions of Sentra Lab:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

**Note:** Only the latest minor version within each major version receives security updates.

---

## Reporting a Vulnerability

**Please DO NOT report security vulnerabilities through public GitHub issues.**

If you discover a security vulnerability, please follow these steps:

### 1. Report via Email

Send a detailed report to: **security@sentra.dev**

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if applicable)
- Your contact information

### 2. Expected Response Time

- **Initial response:** Within 48 hours
- **Status update:** Within 7 days
- **Fix timeline:** Depends on severity (see below)

### 3. Vulnerability Severity

We use the CVSS 3.1 scoring system:

| Severity | CVSS Score | Response Time |
|----------|------------|---------------|
| Critical | 9.0 - 10.0 | 1-3 days      |
| High     | 7.0 - 8.9  | 7-14 days     |
| Medium   | 4.0 - 6.9  | 30 days       |
| Low      | 0.1 - 3.9  | 90 days       |

### 4. Disclosure Policy

We follow **Responsible Disclosure**:

1. You report the issue privately
2. We confirm receipt within 48 hours
3. We investigate and develop a fix
4. We release a patch
5. We publicly disclose the issue (with credit to you, if desired)

**Public disclosure timeline:**
- After patch is released
- No sooner than 90 days after initial report
- Sooner if exploit is actively being used

---

## Security Best Practices

### For Users

**When running Sentra Lab:**

1. **Keep updated:** Always use the latest version
   ```bash
   brew upgrade sentra/tap/lab
   ```

2. **Isolate environments:** Run mocks in isolated Docker networks
   ```bash
   sentra lab start --network isolated
   ```

3. **No real credentials:** Never use production API keys in Sentra Lab
   - Use mock keys instead
   - If you must test with real keys, use dedicated test accounts

4. **Review recordings:** Recordings may contain sensitive data
   ```bash
   # Sanitize recordings before sharing
   sentra lab export --sanitize
   ```

5. **Secure local storage:** Protect your local Sentra Lab data
   ```bash
   # Encrypt recordings
   sentra lab config set encryption.enabled true
   ```

### For Contributors

**When contributing code:**

1. **No secrets in code:** Never commit API keys, passwords, or certificates
   ```bash
   # Check before committing
   git secrets --scan
   ```

2. **Validate inputs:** Always validate and sanitize user inputs
   ```go
   // ✅ GOOD
   if err := validateScenarioID(id); err != nil {
       return nil, fmt.Errorf("invalid scenario ID: %w", err)
   }
   
   // ❌ BAD
   scenario := loadScenario(id) // No validation
   ```

3. **Handle errors:** Never ignore errors, especially in security-critical code
   ```go
   // ✅ GOOD
   hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
   if err != nil {
       return fmt.Errorf("failed to hash password: %w", err)
   }
   
   // ❌ BAD
   hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
   ```

4. **Dependency audits:** Run security audits before submitting PRs
   ```bash
   make security-audit
   ```

5. **Code review:** All PRs require security review

---

## Known Vulnerabilities

### CVE Database

We maintain a list of known vulnerabilities at:
https://github.com/sentra-dev/sentra-lab/security/advisories

### Dependency Vulnerabilities

We use automated tools to detect vulnerable dependencies:

- **Go:** `govulncheck`
- **Rust:** `cargo audit`
- **Node.js:** `npm audit`
- **Python:** `safety`

Check for vulnerabilities:
```bash
make security-audit
```

---

## Security Features

### 1. Sandboxing

Agent code runs in isolated containers:
- Network isolation (only mock APIs accessible)
- File system isolation (read-only mounts)
- Resource limits (CPU, memory, disk)
- No access to host system

### 2. Credential Management

Mock services use fake credentials:
- OpenAI: `mock_sk_test_...`
- Stripe: `mock_pk_test_...`
- No real API calls made

If you need to test with real APIs:
```bash
# Use separate test environment
sentra lab start --real-apis --test-keys-only
```

### 3. Data Privacy

Recordings may contain sensitive data:
- **Local-only by default:** Recordings stay on your machine
- **Optional cloud sync:** Encrypted in transit (TLS 1.3)
- **Sanitization:** Remove sensitive data before sharing
  ```bash
  sentra lab export --sanitize --redact-pii
  ```

### 4. Network Security

All communication is encrypted:
- gRPC uses TLS 1.3
- HTTP mocks support HTTPS
- Optional mTLS for production deployments

---

## Compliance

### GDPR (European Union)

- **Data minimization:** We only collect necessary data
- **Right to deletion:** `sentra lab data delete`
- **Data portability:** Export in standard formats
- **Privacy by design:** Local-first architecture

### CCPA (California)

- **Transparency:** Clear data usage policies
- **Control:** Users control their data
- **No selling:** We never sell user data

### SOC 2 Type II (Cloud Platform)

For cloud features:
- Annual audits
- Penetration testing
- Incident response plan

---

## Security Contacts

- **General security inquiries:** security@sentra.dev
- **Vulnerability reports:** security@sentra.dev (encrypted: [PGP Key](https://sentra.dev/pgp-key.asc))
- **Bug bounty program:** https://sentra.dev/security/bounty

---

## Bug Bounty Program

We offer rewards for security vulnerabilities:

| Severity | Reward      |
|----------|-------------|
| Critical | $5,000+     |
| High     | $1,000-5,000|
| Medium   | $500-1,000  |
| Low      | $100-500    |

**Eligibility:**
- First to report the vulnerability
- Provide a clear reproduction
- Follow responsible disclosure
- Do not publicly disclose before patch

**Learn more:** https://sentra.dev/security/bounty

---

## Security Audits

### Internal Audits

- **Frequency:** Quarterly
- **Scope:** All packages
- **Tools:** Automated + manual review

### External Audits

- **Frequency:** Annually
- **Scope:** Core engine + cloud platform
- **Auditors:** Independent security firms

**Latest audit reports:** https://sentra.dev/security/audits

---

## Incident Response

If a security incident occurs:

1. **Detection:** Automated alerts + user reports
2. **Assessment:** Severity determination (< 2 hours)
3. **Containment:** Isolate affected systems
4. **Eradication:** Remove vulnerability
5. **Recovery:** Deploy patch
6. **Post-incident:** Root cause analysis

**Incident status:** https://status.sentra.dev

---

## Security Updates

Subscribe to security announcements:

- **Email:** https://sentra.dev/security/subscribe
- **RSS:** https://sentra.dev/security.rss
- **GitHub:** Watch releases

---

## Questions?

Security questions? Contact us:
- Email: security@sentra.dev
- Discord: #security channel
- GitHub Discussions: [Security category](https://github.com/sentra-dev/sentra-lab/discussions/categories/security)

---

**Last updated:** November 14, 2025
