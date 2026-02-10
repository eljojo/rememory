# Recovery Manifest — {{.ProjectName}}

This folder contains your recovery information. Fill in the sections below with your critical credentials and information that your trusted friends will need to help you or your family regain access to your digital life.

**Friends who will hold shares:** {{range $i, $f := .Friends}}{{if $i}}, {{end}}{{$f.Name}}{{end}}
**Shares needed to recover:** {{.Threshold}} of {{len .Friends}}

---

## Tier 1 — The Skeleton Key (unlocks everything else)

### Password Manager
- **Provider:** (1Password / Bitwarden / etc.)
- **Email:**
- **Master Password:**
- **Secret Key:** (if applicable)
- **2FA Backup Codes:**

### Primary Email
- **Provider:**
- **Email Address:**
- **Password:**
- **2FA Backup Codes:**
- **Recovery Email/Phone:**

### Phone
- **Device:**
- **PIN/Passcode:**

---

## Tier 2 — Financial & Critical Accounts

### Bank Accounts
| Institution | Login | 2FA Method | Notes |
|-------------|-------|------------|-------|
| | | | |

### Crypto Wallets
| Wallet | Seed Phrase | PIN | Chains | Notes |
|--------|-------------|-----|--------|-------|
| | | | | |

### Cloud Storage
| Provider | Login | Notes |
|----------|-------|-------|
| | | |

### Backup Systems
| System | Location | Passphrase | Notes |
|--------|----------|------------|-------|
| | | | |

### Hardware Security Keys (Yubikeys, etc.)
| Key | Location | PIN | PUK | Notes |
|-----|----------|-----|-----|-------|
| | | | | |

---

## Tier 3 — The Map (where stuff lives)

### What is backed up where
- **Photos:**
- **Documents:**
- **Code/Projects:**
- **Music/Media:**

### Hardware locations
- **NAS/Server:**
- **External drives:**
- **Safe deposit box:**

### Recovery sequences
Describe step-by-step how to regain access to critical systems:

1. First, recover email by...
2. Then, access password manager by...
3. ...

### Legal contacts
- **Lawyer:**
- **Will location:**
- **Other important contacts:**

---

## Tier 4 — Personal Wishes (optional)

### Social media accounts
What should happen to your accounts?

### Who should have access to what
Specific instructions for specific people.

### Messages
Any messages you want to leave.

---

## What NOT to include here
- Sensitive employment / NDA-covered information
- Client/patient confidential data
- Anything that should go through a lawyer instead

**Remember:** This manifest will be encrypted and split among your trusted friends. Only add information you would trust them with in an emergency.

---

## Need help figuring out what to include?

Check out [potatoqualitee/eol-dr](https://github.com/potatoqualitee/eol-dr) — a comprehensive end-of-life [checklist](https://github.com/potatoqualitee/eol-dr/blob/main/checklist.md) covering accounts, finances, subscriptions, and devices. It's a great starting point for thinking through what your loved ones might need.
