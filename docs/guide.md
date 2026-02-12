# ReMemory User Guide

This guide walks you through using ReMemory to create encrypted recovery bundles for your trusted friends.

> **Prefer a browser?** This guide focuses on the CLI tool. If you'd rather create bundles in your browser without installing anything, see the [web-based guide](https://eljojo.github.io/rememory/docs.html).

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Creating Your First Project](#creating-your-first-project)
- [Adding Your Secrets](#adding-your-secrets)
- [Sealing the Project](#sealing-the-project)
- [Creating Distribution Bundles](#creating-distribution-bundles)
- [Distributing to Friends](#distributing-to-friends)
- [What Your Friends Receive](#what-your-friends-receive)
- [Recovery Process](#recovery-process)
- [Verifying Bundles](#verifying-bundles)
- [Best Practices](#best-practices)
- [Project Structure](#project-structure)
- [Commands Reference](#commands-reference)
- [Revoking Access](#revoking-access)
- [Advanced: Anonymous Mode](#advanced-anonymous-mode)
- [Advanced: Multilingual Bundles](#advanced-multilingual-bundles)

## Overview

ReMemory helps you:

1. Encrypt sensitive files with strong cryptography
2. Split the decryption key among trusted friends using Shamir's Secret Sharing
3. Create self-contained bundles that friends can use to recover your secrets

The key innovation is that recovery works **entirely offline in a browser**—no servers, no internet, no need for ReMemory to exist when recovery happens.

## Installation

### macOS (Homebrew)

```bash
brew install eljojo/rememory/rememory
```

### Linux

Download the binary, make it executable, and move it to your path.

**x86_64:**

```bash
curl -Lo rememory https://github.com/eljojo/rememory/releases/latest/download/rememory-linux-amd64
chmod +x rememory
sudo mv rememory /usr/local/bin/
```

**ARM64:**

```bash
curl -Lo rememory https://github.com/eljojo/rememory/releases/latest/download/rememory-linux-arm64
chmod +x rememory
sudo mv rememory /usr/local/bin/
```

Binaries for all platforms are available on the [Releases](https://github.com/eljojo/rememory/releases) page.

### Nix

Run directly without installing:

```bash
nix run github:eljojo/rememory
```

<details>
<summary>Install permanently</summary>

Add to your flake inputs:

```nix
{
  inputs.rememory.url = "github:eljojo/rememory";
  inputs.rememory.inputs.nixpkgs.follows = "nixpkgs";
}
```

Then include in your NixOS configuration:

```nix
# configuration.nix
{ inputs, ... }:
{
  environment.systemPackages = [ inputs.rememory.packages.${system}.default ];
}
```

Or in home-manager:

```nix
# home.nix
{ inputs, ... }:
{
  home.packages = [ inputs.rememory.packages.${system}.default ];
}
```

</details>

### Man pages (optional)

```bash
mkdir -p ~/.local/share/man/man1
rememory doc ~/.local/share/man/man1
```


## Creating Your First Project

Start by creating a new project:

```bash
rememory init my-recovery-2026
cd my-recovery-2026
```

You'll be prompted to configure your recovery scheme:

```
How many friends will hold shares? [5]: 5
How many shares needed to recover? [3]: 3

Friend 1:
  Name: Alice
  Contact info (optional): alice@example.com

Friend 2:
  Name: Bob
  Contact info (optional):

Friend 3:
  Name: Carol
  Contact info (optional): carol@example.com

...
```

### Choosing the Right Numbers

| Friends | Recommended Threshold | Notes |
|---------|----------------------|-------|
| 3 | 2 | Minimum viable setup |
| 5 | 3 | Good balance of security and availability |
| 7 | 4-5 | Higher security, requires more coordination |

**Rule of thumb:** Set threshold high enough that casual collusion is unlikely, but low enough that recovery is possible if 1-2 friends are unavailable.

## Adding Your Secrets

Place your sensitive files in the `manifest/` directory:

```bash
# Copy important files
cp ~/Documents/recovery-codes.txt manifest/
cp ~/Documents/crypto-seeds.txt manifest/
cp ~/Documents/important-passwords.txt manifest/

# Or create files directly
echo "The safe combination is 12-34-56" > manifest/notes.txt
echo "Bank account: 123456789" >> manifest/notes.txt
```

You can organize files in subdirectories:

```bash
mkdir -p manifest/crypto
mkdir -p manifest/accounts
cp ~/wallets/*.txt manifest/crypto/
cp ~/passwords/*.txt manifest/accounts/
```

### What to Include

Good candidates for ReMemory:
- Password manager recovery codes
- Cryptocurrency seeds/keys
- Important account credentials
- Instructions for loved ones
- Legal document locations
- Safe combinations

### What NOT to Include

- Files that change frequently (use ReMemory for static secrets)
- Extremely large files (bundles become unwieldy)
- Anything already backed up elsewhere with good recovery options

## Sealing the Project

Once your secrets are in place, seal the project:

```bash
rememory seal
```

This:
1. Generates a random 256-bit passphrase
2. Encrypts all files in `manifest/` using age encryption
3. Splits the passphrase into shares using Shamir's Secret Sharing
4. Verifies that recovery works correctly
5. Generates distribution bundles for each friend

```
Archiving manifest/ (3 files, 1.2 KB)...
Encrypting with age...
Splitting into 5 shares (threshold: 3)...
Verifying reconstruction... OK

Sealed:
  ✓ output/MANIFEST.age
  ✓ output/shares/SHARE-alice.txt
  ✓ output/shares/SHARE-bob.txt
  ✓ output/shares/SHARE-carol.txt
  ✓ output/shares/SHARE-david.txt
  ✓ output/shares/SHARE-eve.txt

Generating bundles for 5 friends...

Bundles ready to distribute:
  ✓ bundle-alice.zip (5.4 MB)
  ✓ bundle-bob.zip (5.4 MB)
  ✓ bundle-carol.zip (5.4 MB)
  ✓ bundle-david.zip (5.4 MB)
  ✓ bundle-eve.zip (5.4 MB)

Saved to: output/bundles
```

Each bundle is ~5 MB because it includes the complete recovery tool.

### Regenerating Bundles

If you need to regenerate bundles (e.g., you lost them or want to update `recover.html`):

```bash
rememory bundle
```

## Distributing to Friends

Send each friend their specific bundle. Methods:

- **Email** — Attach the ZIP file
- **Cloud storage** — Share via Dropbox, Google Drive, etc.
- **USB drive** — Physical handoff
- **Encrypted messaging** — Signal, WhatsApp, etc.

Tell your friends:
1. Keep the bundle somewhere safe (cloud backup, USB drive, etc.)
2. They cannot use it alone—they'll need to coordinate with others
3. A single share reveals nothing, but they should still keep it private

## What Your Friends Receive

Each bundle contains:

| File | Purpose |
|------|---------|
| `README.txt` | Instructions + their unique share + contact list for other holders |
| `README.pdf` | Same content, formatted for printing |
| `MANIFEST.age` | Your encrypted secrets (same in all bundles) |
| `recover.html` | **Personalized** browser-based recovery tool (~1.8 MB, self-contained) |

**What makes each bundle unique:**
- The `recover.html` is personalized for each friend:
  - Their share is pre-loaded automatically
  - Shows a contact list with other friends' info
  - If the encrypted manifest is 5 MB or less, it's also embedded in `recover.html`—so friends only need to collect shares from others to complete recovery
  - For larger manifests, they'll also need to load the separate `MANIFEST.age` file

The README.txt includes:

```
================================================================================
                          REMEMORY RECOVERY BUNDLE
                              For: Alice
================================================================================

!!  YOU CANNOT USE THIS FILE ALONE
    You will need help from other friends listed below.

!!  CONFIDENTIAL - DO NOT SHARE THIS FILE
    This document contains your secret share. Keep it safe.

    NOTA PARA HISPANOHABLANTES:
    Si no entiendes inglés, puedes usar ChatGPT u otra inteligencia artificial
    para que te ayude a entender estas instrucciones y recuperar los datos.

--------------------------------------------------------------------------------
WHAT IS THIS?
--------------------------------------------------------------------------------
This bundle allows you to help recover encrypted secrets.
You are one of 5 trusted friends who hold pieces of the recovery key.
At least 3 of you must cooperate to decrypt the contents.

--------------------------------------------------------------------------------
OTHER SHARE HOLDERS (contact to coordinate recovery)
--------------------------------------------------------------------------------
Bob - bob@example.com - 555-2345
Carol - carol@example.com
David - david@example.com - 555-4567
Eve - eve@example.com

--------------------------------------------------------------------------------
HOW TO RECOVER (PRIMARY METHOD - Browser)
--------------------------------------------------------------------------------
1. Open recover.html in any modern browser
2. Drag and drop this README.txt file
3. Collect shares from other friends (they drag their README.txt too)
4. Once you have enough shares, the tool will decrypt automatically
5. Download the recovered files

Works completely offline - no internet required!

--------------------------------------------------------------------------------
YOUR SHARE
--------------------------------------------------------------------------------
-----BEGIN REMEMORY SHARE-----
Version: 1
Index: 1
Total: 5
Threshold: 3
Holder: Alice
...
-----END REMEMORY SHARE-----
```

## Recovery Process

### Browser Recovery (Recommended)

When your friends need to recover your secrets:

1. **One friend opens `recover.html`** from their bundle in any modern browser
   - Their share is **automatically pre-loaded** (the tool is personalized!)
   - They'll see a **contact list** showing other friends who hold shares

2. **Load the encrypted manifest**
   - For small manifests (≤ 5 MB), this step is automatic—the manifest is embedded in `recover.html`
   - Otherwise, drag and drop `MANIFEST.age` from the bundle onto the manifest area, or click to browse

3. **Coordinate with other friends**
   - The contact list shows names, emails, and phone numbers
   - Reach out and ask them to send their `README.txt` file

4. **Add shares from other friends**
   - Drag and drop their `README.txt` files onto the page, OR
   - Click the 📋 clipboard button to paste share text directly
   - As each share is added, a ✓ checkmark appears next to that friend's name

5. **Recovery happens automatically**
   - Once threshold is met (e.g., 2 of 3 shares), decryption starts immediately
   - The input steps collapse to show the recovery progress
   - No need to click any buttons!

6. **Download the recovered files**

**Key points:**
- Works completely offline—no internet required
- No data leaves the browser
- Works on Chrome, Firefox, Safari, Edge
- Friends can be in different locations; they just need to share their README.txt files
- Each friend's `recover.html` is personalized with their share pre-loaded

### CLI Recovery (Fallback)

If the browser tool doesn't work:

```bash
# Download rememory from GitHub releases, then:
rememory recover \
  --shares alice-readme.txt,bob-readme.txt,carol-readme.txt \
  --manifest MANIFEST.age \
  --output recovered/
```

## Verifying Bundles

Before distributing, verify your bundles are valid:

```bash
rememory verify-bundle output/bundles/bundle-alice.zip
```

This checks:
- All required files are present
- Checksums match
- The embedded share is valid

You can also verify bundles you receive from others to ensure they haven't been corrupted.

## Best Practices

### Choosing Friends

- **Longevity** — Pick people likely to be reachable in 5-10+ years
- **Geographic diversity** — Don't put all friends in the same disaster zone
- **Technical ability** — Mix is fine; the tool is designed for everyone
- **Relationships** — Consider if they'll cooperate with each other
- **Trust** — While a single share reveals nothing, you're trusting them with responsibility

### Security Considerations

- **Keep your sealed project secure** — The passphrase is stored in project.yml after sealing
- **Delete the manifest after sealing** — Or keep it somewhere very secure
- **Don't keep all bundles together** — That defeats the purpose of splitting
- **Consider printing README.pdf** — Paper backups survive digital disasters

### Rotation

Consider creating a new project every 2-3 years:
- Friends' contact info changes
- You may want to update secrets
- Relationships change
- New cryptographic best practices emerge

You can copy friend configuration:

```bash
rememory init new-project --from old-project
```

### Revoking Access

There is no way to remotely revoke a share once it has been distributed. This is by design — the system is offline and serverless, so there is no central authority that can invalidate a share.

If you need to remove someone from your recovery group (e.g., a falling out, or you simply want to change who holds shares), the only option is:

1. **Create a new project** with a new set of friends and a fresh passphrase
2. **Send new bundles** to the friends you still trust
3. **Ask every remaining friend to delete their old bundle** and replace it with the new one

This last step is critical. Old shares can still decrypt old manifests, so friends must not keep old bundles "just in case." When you send someone a new bundle, be clear: **delete the old one, keep only the new one.** No version history, no archives — just the latest bundle.

The same applies when you update your secrets (e.g., a password changed). Sealing a new project generates a completely new passphrase and new shares. The old shares become useless for the new manifest, but they still work with the old `MANIFEST.age`. Make sure friends aren't holding on to old copies.

## Project Structure

After running all commands, your project looks like:

```
my-recovery-2026/
├── project.yml           # Configuration (friends, threshold, checksums)
├── manifest/             # Your secret files (ADD FILES HERE)
│   ├── README.md         # Default instructions file
│   ├── recovery-codes.txt
│   └── notes.txt
└── output/
    ├── MANIFEST.age      # Encrypted archive of manifest/
    ├── shares/           # Individual share files
    │   ├── SHARE-alice.txt
    │   ├── SHARE-bob.txt
    │   └── ...
    └── bundles/          # Distribution packages
        ├── bundle-alice.zip
        ├── bundle-bob.zip
        └── ...
```

## Commands Reference

| Command | Description |
|---------|-------------|
| `rememory init <name>` | Create a new project |
| `rememory demo [dir]` | Create a demo project with sample data (great for testing!) |
| `rememory seal` | Encrypt manifest, create shares, and generate bundles |
| `rememory bundle` | Regenerate bundles (if lost or need updating) |
| `rememory status` | Show project status and summary |
| `rememory verify` | Verify integrity of sealed files |
| `rememory verify-bundle <zip>` | Verify a bundle's integrity |
| `rememory recover` | Recover secrets from shares |
| `rememory doc <dir>` | Generate man pages |

For detailed help on any command:

```bash
rememory <command> --help
```

## Advanced: Anonymous Mode

For situations where you don't want shareholders to know each other's identities, ReMemory offers an **anonymous mode**. In this mode:

- Friends are labeled generically as "Share 1", "Share 2", etc.
- No contact information is collected or stored
- READMEs skip the "Other Share Holders" section
- Bundle filenames use numbers instead of names (`bundle-share-1.zip`, etc.)

### When to Use Anonymous Mode

Anonymous mode is useful when:
- You want to distribute shares to people who shouldn't know each other
- You're testing the system quickly without entering contact details
- You have a separate out-of-band method for coordinating recovery
- Privacy is a higher priority than ease of coordination

### Creating an Anonymous Project

```bash
# Create an anonymous project with 5 shares, threshold 3
rememory init my-recovery --anonymous --shares 5 --threshold 3
```

You can also run it interactively:

```bash
rememory init my-recovery --anonymous
# Prompts: How many shares? and What threshold?
```

The resulting `project.yml` will look like:

```yaml
name: my-recovery
threshold: 3
anonymous: true
friends:
  - name: Share 1
  - name: Share 2
  - name: Share 3
  - name: Share 4
  - name: Share 5
```

### Recovery in Anonymous Mode

Recovery works the same way, but:
- The contact list section won't appear in `recover.html`
- Share holders will need to coordinate through other means
- Shares show generic labels like "Share 1" instead of names

Since there's no built-in contact list, make sure share holders know how to reach each other (or you) when recovery is needed.

## Advanced: Multilingual Bundles

Each friend can receive their bundle (README.txt, README.pdf, and recover.html) in their preferred language. ReMemory supports 5 languages: English (en), Spanish (es), German (de), French (fr), and Slovenian (sl).

### CLI Usage

Set the project-level default language with `--language`:

```bash
# All bundles in Spanish
rememory init my-recovery --language es

# Per-friend language customization
rememory init my-recovery --language es \
  --friend "Alice,alice@example.com,en" \
  --friend "Roberto,roberto@example.com,es" \
  --friend "Hans,hans@example.com,de"
```

The `--friend` flag now accepts an optional third field for language: `"Name,contact,lang"`.

### project.yml Format

You can also set languages directly in `project.yml`:

```yaml
name: my-recovery-2026
threshold: 3
language: es          # default bundle language (optional, defaults to "en")
friends:
  - name: Alice
    contact: alice@example.com
    language: en      # override per friend
  - name: Roberto
    contact: roberto@example.com
    # uses project language (es)
  - name: Hans
    contact: hans@example.com
    language: de
```

### Web UI

In the web-based bundle creator (maker.html), each friend entry has a **Bundle language** dropdown. The default is the current UI language. Friends can always switch languages in recover.html regardless of the bundle default.

### What Gets Translated

- **README.txt**: All instructions, warnings, and section headings
- **README.pdf**: Same content as README.txt in PDF format
- **recover.html**: Opens in the friend's language by default (they can still switch)
