# ReMemory Reseal Feature - Quick Reference

## What is Reseal?

The reseal feature allows you to encrypt updated data using the **same passphrase** (and thus the same shares) from your original ReMemory project. This is useful when you need to update encrypted data but don't want to redistribute new shares to your friends.

## Use Cases

1. **Regular Updates** - Update encrypted data on a regular schedule (weekly backups, monthly updates)
2. **Server Uploads** - Upload updated encrypted data to a server accessible to all friends
3. **Version Management** - Keep multiple versions of encrypted data, all decryptable with the same shares
4. **Long-term Secret Management** - Update secrets (passwords, recovery codes) without resharing

## How It Works

### Initial Setup (Day 1)
```bash
rememory init my-secrets
cd my-secrets

# Add files to manifest/
echo "password123" > manifest/passwords.txt

# Create shares and distribute to friends
rememory seal
# Share: bundle-alice.zip with Alice
# Share: bundle-bob.zip with Bob
# Share: bundle-carol.zip with Carol
```

### Update Encrypted Data (Day 30)
```bash
cd my-secrets

# Update manifest with new data
echo "password456" > manifest/passwords.txt

# Get shares from friends to recover the passphrase
# Alice gives you: SHARE-alice.txt
# Bob gives you: SHARE-bob.txt

# Reseal with the recovered passphrase
rememory reseal SHARE-alice.txt SHARE-bob.txt

# This creates new bundles with updated manifest
# Share updated bundles back to friends
```

### Recovery (Can happen anytime)
```bash
# Friends cooperate to provide shares
rememory recover SHARE-alice.txt SHARE-bob.txt carol-share.txt

# Manifests from any reseal version decrypt the same way
# The most recent version is recovered automatically
```

## Command Reference

### Reseal Command
```bash
rememory reseal share1.txt share2.txt [share3.txt ...]
```

**Options:**
- `--recovery-url URL` - Override the recovery URL in PDF (default: production URL)

**Requirements:**
- Must run in project directory (containing `project.yml`)
- Project must have been sealed with PassphraseChecksum (recent versions only)
- Must provide at least `threshold` number of shares
- Shares must be from the same original seal (same N and K values)
- Manifest directory must have new/updated content

**Output:**
- Creates versioned `MANIFEST-<timestamp>.age` file
- Updates `MANIFEST.age` for backwards compatibility
- Regenerates bundles in `output/bundles/`
- Shares in `output/shares/` remain unchanged

## Security

### What Gets Verified?

1. **Passphrase Checksum** - Recovered passphrase must match stored SHA-256 hash
2. **Share Compatibility** - All shares must be from same original seal (same N, K, version)
3. **Manifest Integrity** - Archived and encrypted with recovered passphrase

### What Stays The Same?

- Share files in `output/shares/` (never regenerated)
- Friend list and threshold (2-of-3, 3-of-5, etc.)
- Encryption algorithm (age with scrypt)
- Distribution method (friends still hold their shares)

### What's New?

- Manifest content (updated files in manifest/)
- MANIFEST.age file (new encrypted version)
- Bundles (contain new manifest)


## File Structure After Reseal

```
my-secrets/
├── project.yml                    # Updated with passphrase_checksum
├── manifest/                      # Updated content
│   ├── README.md
│   └── secrets.txt               # Updated file
├── output/
│   ├── MANIFEST.age              # New encrypted version (latest)
│   ├── MANIFEST-20260214.age      # Versioned (first seal)
│   ├── MANIFEST-20260215.age      # Versioned (second reseal)
│   ├── bundles/
│   │   ├── bundle-alice.zip      # Updated
│   │   ├── bundle-bob.zip        # Updated
│   │   └── bundle-carol.zip      # Updated
│   └── shares/
│       ├── SHARE-alice.txt       # Unchanged ✓
│       ├── SHARE-bob.txt         # Unchanged ✓
│       └── SHARE-carol.txt       # Unchanged ✓
```

## Examples

### Example 1: Monthly Backup Updates
```bash
# Month 1: Initial seal
rememory seal

# Month 2: Update and reseal
echo "$(date +%Y-%m-%d)" > manifest/last-backup.txt
rememory reseal output/shares/SHARE-alice.txt output/shares/SHARE-bob.txt

# Month 3: Another update
echo "$(date +%Y-%m-%d)" > manifest/last-backup.txt
rememory reseal output/shares/SHARE-alice.txt output/shares/SHARE-bob.txt

# Friends recover with latest version
rememory recover SHARE-alice.txt SHARE-bob.txt
```

### Example 2: Shared Recovery Codes
```bash
# Initial seal with recovery codes
echo "ABC-123" > manifest/codes.txt
rememory seal

# Later: Update recovery codes
echo "XYZ-789" > manifest/codes.txt
rememory reseal SHARE-alice.txt SHARE-bob.txt SHARE-carol.txt

# Upload to shared server
cp output/bundles/* /path/to/shared/server/
```

### Example 3: Password Manager Backup
```bash
# Get latest password export
lastpass export > manifest/passwords.csv

# Seal or reseal with friends
rememory reseal output/shares/SHARE-*.txt

# Share new bundles
```
