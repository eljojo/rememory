---
title: "\U0001F9E0 ReMemory Guide"
subtitle: "How to create bundles and recover files"
cli_guide_note: 'There is also a <a href="{{GITHUB_REPO}}/blob/main/docs/guide.md">CLI guide</a>.'
nav_home: "\U0001F9E0 ReMemory"
nav_home_link: "Home"
nav_create: "Create Bundles"
nav_recover: "Recover"
toc_title: "Contents"
footer_source: "Source Code"
footer_download: "Download"
footer_home: "Home"
---

## Overview {#overview}

ReMemory is a digital safe with multiple keys. You choose who holds them and how many must come together to open it.

Under the hood, it encrypts your files with [age](https://github.com/FiloSottile/age), splits the key among people you trust, and gives each person a self-contained bundle for recovery.

Recovery is opening a file in a browser.\* No servers, no need for this website to exist.

If you hold a piece for someone, start at [Recovering Files](#recovering).

<p style="font-size: 0.8125rem; color: #8A8480;">* <a href="#timelock" style="color: #8A8480;">Time-locked</a> archives need a brief internet connection at recovery time.</p>

<div class="tip">
<strong>Tip:</strong> No one person can access your data. They need to bring together enough pieces — say, 3 of 5.
</div>

## Why ReMemory {#why-rememory}

You probably have digital secrets that matter: password manager recovery codes, cryptocurrency seeds, important documents, instructions for loved ones. What happens to these if you're suddenly unavailable?

Think of it like a safe deposit box that needs two keys to open — no single person holds enough to get in alone.

Traditional approaches have weaknesses:

- **Give one person everything** — a single point of failure and trust
- **Split files manually** — confusing, error-prone, no encryption
- **Use a password manager's emergency access** — similar to "give one person everything", also relies on the company existing
- **Write it in a will** — becomes public record, slow legal process

ReMemory takes a different approach:

- **No single point of failure** — requires multiple people to cooperate
- **No trust in any one person** — even your most trusted friend can't access your secrets alone
- **Offline and self-contained** — recovery works without internet or servers\*
- **Designed for anyone** — clear instructions, not cryptographic puzzles

## Creating Bundles {#creating}

Three steps. Everything happens in your browser — your files never leave your device. Open the [bundle creator](maker.html) to get started.

### Step 1: Add Friends {#step1}

Add the people who will hold pieces of your recovery key. For each, provide a name and optionally contact information.

<figure class="screenshot">
<img src="screenshots/en/friends.png" alt="Adding friends in Step 1">
<figcaption>Each person here will hold one piece of the key</figcaption>
</figure>

Then choose how many of them must come together to open your files.

<div class="tip">
<strong>Choosing how many:</strong>
<ul>
<li><strong>2 of 3 people:</strong> The simplest setup</li>
<li><strong>3 of 5 people:</strong> A good balance</li>
<li><strong>4 or 5 of 7 people:</strong> More secure, more coordination</li>
</ul>
High enough that a few people can't open it without the others. Low enough that it still works if one or two can't be reached.
</div>

### Step 2: Add Files {#step2}

Drag and drop the files or folder you want to protect.

<figure class="screenshot">
<img src="screenshots/en/files.png" alt="Adding files in Step 2">
<figcaption>Add the files you want to protect</figcaption>
</figure>

**Good candidates:**

- Password manager recovery codes
- Cryptocurrency seeds/keys
- Important account credentials
- Instructions for loved ones
- Legal document locations
- Safe combinations

<div class="warning">
<strong>Note:</strong> Avoid files that change often. This is designed for secrets you set once and leave.
</div>

### Step 3: Generate Bundles {#step3}

Click "Generate Bundles" to lock your files and make a bundle for each person.

<figure class="screenshot">
<img src="screenshots/en/bundles.png" alt="Generating bundles in Step 3">
<figcaption>Download each bundle, or all at once</figcaption>
</figure>

Each bundle includes the full recovery tool. It works even if this website is gone.

### Distributing to Friends {#distributing}

Send each person their bundle however you prefer:

- **Email:** Attach the ZIP file
- **Cloud storage:** Share via Dropbox, Google Drive, etc.
- **USB drive:** Physical handoff
- **Encrypted messaging:** Signal, WhatsApp, etc.

If you send bundles by email or messaging, delete the messages from your sent folder once each friend confirms receipt. Otherwise your account ends up holding a copy of every piece — defeating the point of splitting them.

### After Creating Bundles {#after-creating}

Once your bundles are ready, there are a few things worth doing before you put this out of your mind:

- Ask each person to open `recover.html` once. It shows "Your piece is here" when it can read their piece.
- Tell each person what this is, why they have it, and that they should keep it safe. They cannot use it alone — they will need to coordinate with others.
- Save the locked files with "Save the locked files (MANIFEST.age)" and keep that copy somewhere safe. It holds no piece, so it's useless to anyone without enough of them.
- Save your `project.yml` so you can regenerate bundles later
- Print `README.pdf` as a paper backup before sending the digital bundle. Paper doesn't need adapters or power.
- Set a yearly reminder to check in — see [Keeping Bundles Current](#keeping-current)
- Consider running a [recovery drill](#recovery-drills) — it's the only way to know your plan actually works

## Recovering Files {#recovering}

This part is for the friends who hold a piece. Most of the time, all a friend does is keep their bundle safe. One day, someone may ask them for their piece, or they may be the one who brings the pieces together.

If you're here because someone you care about can't be reached anymore, there's no hurry. Nothing here expires. You can take as long as you need — if you close the recovery page partway, you'll add the pieces again when you come back.

### What's a Piece? {#what-is-a-piece}

A piece is one part of the key that locks the files. On its own, it opens nothing and reveals nothing.

Your README holds your piece as a block of text and as 25 words. The printed README.pdf also has it as a QR code. These are all the same piece — any one of them is enough to pass on.

### What Friends Receive {#bundle-contents}

Each bundle contains:

<div class="bundle-contents">
<div class="file">
<span class="file-name">README.txt</span>
<span class="file-desc">What this is, who else holds a piece, and your piece itself</span>
</div>
<div class="file">
<span class="file-name">README.pdf</span>
<span class="file-desc">The same, ready to print. Your piece is on it as a <strong>QR code</strong> and as words.</span>
</div>
<div class="file">
<span class="file-name">MANIFEST.age</span>
<span class="file-desc">The locked files. Only in some bundles — most carry them inside recover.html instead.</span>
</div>
<div class="file">
<span class="file-name">recover.html</span>
<span class="file-desc">The recovery page. It opens in any web browser, with your piece already in it.</span>
</div>
</div>

<p style="margin-top: 1rem;">
Each bundle is made for one friend: their piece is already inside their recover.html, along with the names of the others and how to reach them.
</p>

### Keeping Your Bundle {#keeping-bundle}

Keep it somewhere safe, and a printed copy of README.pdf too if you can. There's nothing else to do until the time comes.

To check it, open recover.html. It shows "Your piece is here", which piece you hold, how many are needed to open the files, and whether the locked files are there too.

### If Someone Asks for Your Piece {#sharing-piece}

First, make sure the request is real. If you can, check with the person who made the bundle. If you can't reach them, check with another friend who holds a piece.

Then send it whichever way is easiest:

- **The whole bundle** (the ZIP file) — the simplest.
- **Your README.txt** — it holds your piece on its own.
- **Your recovery words**, read aloud over the phone.
- **Your printed page**, by post — the QR code on it holds your piece.

### Bringing the Pieces Together {#recovery-bundle}

One person gathers the pieces. It can be any of the friends — whoever takes it on.

<div class="step-guide">
<div class="step-number">1</div>
<div class="step-content">
<h4>Open recover.html</h4>
<p>Double-click the bundle to open it, then double-click recover.html. It opens in your web browser, like a web page, with your own piece already in it. If you don't have a bundle, any friend can send you their recover.html, or you can use the <a href="recover.html">recovery page</a> here.</p>
</div>
</div>

<div class="step-guide">
<div class="step-number">2</div>
<div class="step-content">
<h4>Reach out to the others</h4>
<p>If you have a bundle, its recover.html and README list who else holds a piece and how to reach them — unless the bundles were made without names. Tell them what has happened and ask for their piece. Some will want to check with you first that the request is real — their instructions ask them to.</p>
</div>
</div>

<div class="step-guide">
<div class="step-number">3</div>
<div class="step-content">
<h4>Add each piece as it arrives</h4>
<p>Drag their bundle or README.txt onto the page, paste their text, type the recovery words they read to you, or scan the QR code on their printed page. If names are listed, a checkmark appears next to each one.</p>
</div>
</div>

<div class="step-guide">
<div class="step-number">4</div>
<div class="step-content">
<h4>The files unlock on their own</h4>
<p>Once there are enough pieces, the page opens the files without you doing anything. Download them and keep them somewhere private.</p>
</div>
</div>

<div class="tip">
<strong>If the page asks for the locked files:</strong> drag <code>MANIFEST.age</code> from a friend's bundle onto it — or, if their bundle has none, their recover.html. Most bundles carry them already, so you won't see this step.
</div>

<figure class="screenshot">
<img src="screenshots/en/recovery-1.png" alt="Recovery page with pieces gathered so far">
<figcaption>The pieces gathered so far, and who to contact</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/en/recovery-2.png" alt="Recovery page after the files are unlocked">
<figcaption>Once there are enough pieces, the files are unlocked and ready to download</figcaption>
</figure>

### Typing Recovery Words {#recovery-words}

Each printed page has the piece as a list of numbered words. If a friend reads you theirs over the phone, or all you have is paper, type the words into the recovery page. Press "Paste a piece or type recovery words" and type them in, separated by spaces. Only the words matter, not the numbers.

<figure class="screenshot">
<img src="screenshots/en/recovery-words-typing.png" alt="Typing recovery words from a printed page into the recovery page">
<figcaption>Type the words from the printed page</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/en/recovery-words-recognized.png" alt="Recovery page after the words were entered, showing the piece was added">
<figcaption>The page recognizes the words and adds the piece</figcaption>
</figure>

### Scanning a QR Code {#recovery-pdf}

On a computer or phone with a camera, press "Scan QR code" on the recovery page and hold the printed page up to it.

<figure class="screenshot">
<img src="screenshots/qr-camera-permission.png" alt="Browser asking for permission to use the camera">
<figcaption>Your browser will ask to use the camera</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/qr-scanning.png" alt="Scanning a QR code from a printed page">
<figcaption>Hold the QR code up to the camera to add the piece</figcaption>
</figure>

Scanning the QR code with a phone's camera app works too: it opens the recovery page on this website, with the piece filled in.

<div class="tip">
<strong>About recovery:</strong>
<ul>
<li>Works without an internet <span title="Time-locked archives need a connection to check the unlock date.">connection*</span></li>
<li>The pieces and the files stay on your device</li>
<li>Friends can be anywhere — each one only needs to get their piece to you</li>
</ul>
</div>

## Best Practices {#best-practices}

### Choosing Friends

- **Longevity:** People likely to be reachable in 5–10 years
- **Geographic spread:** Not all in the same place
- **Technical ability:** People don't need to be technical. Their instructions walk them through it, and a [recovery drill](#recovery-drills) will show you where anyone gets stuck.
- **Relationships:** Will they cooperate with each other?
- **Trust:** A single piece reveals nothing, but you're trusting them with responsibility

### Security Considerations

- Don't keep all bundles together — that defeats the point of splitting
- Consider printing `README.pdf` — paper survives digital disasters
- Save `project.yml` if you want to regenerate bundles later

### Storing Bundles Safely {#storing-bundles}

Bundles are small (under 10 MB) and designed to be stored in everyday places. Here's what works well:

- **Email** works well. Most people keep the same email address for decades, and bundles are small enough to attach. Many email providers retain messages indefinitely.
- **Cloud storage** (Google Drive, Dropbox, iCloud) works well as a secondary copy.
- **USB drives** can work, but keep in mind that connectors change over time (USB-A is already giving way to USB-C) and flash memory can degrade if left unpowered for years. Not ideal as the only copy.
- **Paper** is the most durable option. Printing `README.pdf` gives your friends a copy of their piece that doesn't need adapters, power, or any working device.

Paper holds the pieces, but not the locked files themselves — those only live in digital copies. So the best approach is redundancy — email plus paper, or cloud plus paper. More than one copy, in more than one form.

### Keeping Bundles Current {#keeping-current}

Set a yearly reminder to check in with your friends. Confirm they still have their bundles and update contact details if anything has changed.

When your files change, create new bundles and send them. The old bundles won't open the new archive, so there's no risk in leaving them around — but ask friends to replace theirs to keep things tidy.

When contacts change — someone moves, changes their phone number, or you want to add or remove someone — same thing: new bundles, ask people to delete the old ones.

Between updates, keep your source files in an encrypted vault — tools like [Cryptomator](https://cryptomator.org) or [VeraCrypt](https://veracrypt.fr) work well. Don't leave plaintext copies sitting in a regular folder.

Think of it like updating your emergency contacts. Brief, periodic, worth doing.

### Recovery Drills {#recovery-drills}

Nobody tests their backup plan. A recovery drill changes that — and it plays a little like an escape room.

Create a test bundle with a harmless secret (a photo, a message, a recipe). Hand out pieces to your friends. Set a [timelock](#timelock) if you want a real countdown. Then step back and let them figure it out: coordinate, combine pieces, unlock the files — without your help.

What you learn:

- Can your friends actually find their bundles when they need them?
- Do they understand the instructions, or do they get stuck?
- Can they reach each other without you coordinating?

A drill that goes smoothly means your real plan will too. A drill that goes badly is a gift — you found the problem before it mattered.

### Revoking Access {#revoking-access}

Once a piece has been distributed, it cannot be revoked. This is by design — there is no server, no central authority.

If you need to change who holds pieces:

1. **Create new bundles** with a new set of friends and a fresh key
1. **Send new bundles** to the friends you still trust
1. **Ask every remaining friend to delete their old bundle** and replace it with the new one

<div class="warning">
<strong>Important:</strong> Old pieces still work with old archives. When you send a new bundle, be clear: <strong>delete the old one</strong>, keep only the new one. No version history, no "just in case."
</div>

The same applies when secrets change. New bundles mean a new key and new pieces. Old pieces won't open the new archive, but they still work with the old one. Make sure friends aren't holding on to old copies.

### About project.yml {#project-file}

When you create bundles, your project is saved in a `project.yml` file. This file stores:

- Friends' names and contact information
- Your chosen threshold (e.g., 3 of 5)
- A verification hash for checking if bundles match
- Share checksums for verifying bundle integrity

It does **not** store any secrets — no passphrase, no key material, no file contents. It's safe to keep with your other records.

With `project.yml`, you can regenerate bundles, verify existing ones, and check the status of your setup.

## Understanding the Security {#security}

ReMemory composes well-established cryptographic tools rather than inventing its own. Here's what that means in practice.

### What Protects Your Data {#cryptography}

Your files are locked with a modern encryption tool ([age](https://github.com/FiloSottile/age)) — widely reviewed, no known weaknesses.

The key that locks them is 256 bits long, generated from your operating system's random number generator. For scale: guessing it would take longer than the universe has existed.

Even if someone tried every possible password, scrypt makes each guess deliberately slow — millions of times slower than a naive attempt.

The key is then split using Shamir's Secret Sharing. **Any fewer than *threshold* pieces contain zero information about the original.** Not "very little." Mathematically zero.

Each bundle also lists checksums of its files, so `rememory verify-bundle` can tell you if one was damaged along the way. Checksums catch accidents, not deliberate changes: whoever alters a file can update its checksum too. The encrypted archive is protected on its own — if it's altered, it won't open.

### What Could Go Wrong {#what-could-go-wrong}

<div class="bundle-contents">
<div class="file">
<span class="file-name">A friend loses their bundle</span>
<span class="file-desc">Fine, as long as enough other friends still have theirs. That's why you set the threshold below the total.</span>
</div>
<div class="file">
<span class="file-name">A friend leaks their piece publicly</span>
<span class="file-desc">A single piece is useless without the others. Someone would still need threshold-1 more pieces to do anything.</span>
</div>
<div class="file">
<span class="file-name">Some friends can't be reached</span>
<span class="file-desc">That's why you set the threshold below the total number of friends. If you chose 3-of-5, any three will do.</span>
</div>
<div class="file">
<span class="file-name">ReMemory disappears in 10 years</span>
<span class="file-desc"><code>recover.html</code> still works — it's self-contained. No servers, no downloads, no dependencies on this project.</span>
</div>
<div class="file">
<span class="file-name">Browsers change dramatically</span>
<span class="file-desc">The recovery tool uses standard JavaScript and the Web Crypto API — browser fundamentals, not trends.</span>
</div>
<div class="file">
<span class="file-name">You forget how this works</span>
<span class="file-desc">Each bundle's README.txt explains everything. Your friends don't need to remember anything — it's all written down for them.</span>
</div>
</div>

The things that *do* need to be true: your device is trusted when you create bundles, and the browser used for recovery isn't compromised. These are the same assumptions you make any time you use a computer for something important.

For a detailed technical evaluation, see the [security self-review]({{GITHUB_REPO}}/blob/main/docs/security-review.md).

## How It Compares {#comparison}

ReMemory isn't the first tool to use Shamir's Secret Sharing. There are many others, from command-line tools to web apps. Here's what sets ReMemory apart:

- **Handles files, not just text.** Most Shamir tools only split passwords or short text. ReMemory encrypts entire files and folders.
- **Self-contained recovery tool.** Each friend receives `recover.html` — a complete recovery tool that runs in any browser, offline.\* No installation, no CLI needed.
- **Contact details included.** Each bundle includes a list of other friends and how to reach them, so coordination doesn't depend on you being available.
- **No server dependency.** Everything runs locally. There's no service to sign up for, no account to maintain, nothing that needs to stay online.

For a detailed comparison with other tools, see the [full comparison table on GitHub]({{GITHUB_REPO}}#other-similar-tools).

## CLI Alternative {#cli}

There is also a command-line tool for those who prefer a terminal or need to automate bundle creation.

<a href="{{GITHUB_REPO}}/blob/main/docs/guide.md" class="btn btn-secondary">Read the CLI Guide</a>

<p style="margin-top: 1rem;">
The CLI provides the same functionality, plus batch operations and scripting.
</p>

## Advanced: Anonymous Mode {#anonymous}

When holders should not know each other's identities, use **anonymous mode**:

- People are labeled as "Share 1", "Share 2", etc.
- No contact information is collected or stored
- READMEs skip the "Other Share Holders" section
- Bundle filenames use numbers instead of names

### When to Use Anonymous Mode

This is useful when:

- Holders should not know each other
- You're testing quickly without entering names
- You have another way to coordinate recovery
- Privacy is a higher priority than ease of coordination

### How to Enable

In the [bundle creator](maker.html), enable the **Anonymous** toggle in the Friends section:

- The friend list is replaced by a share count
- Set how many shares and the threshold
- Bundles are named `bundle-share-1.zip`, `bundle-share-2.zip`, etc.

### Recovery in Anonymous Mode

Recovery works the same way, but without the contact list. Holders see generic labels like "Share 1" instead of names.

<div class="warning">
<strong>Important:</strong> Without a built-in contact list, make sure holders know how to reach each other when recovery is needed.
</div>

## Advanced: Multilingual Bundles {#multilingual}

Each person can receive their bundle in their preferred language. Seven languages are supported: English, Spanish, German, French, Slovenian, Portuguese and Chinese (Taiwan).

### How It Works

- Each friend entry has a **Bundle language** dropdown
- "Default" uses the current UI language
- Override per person to mix languages
- recover.html opens in the selected language
- Anyone can switch languages at any time

<figure class="screenshot">
<img src="screenshots/en/multilingual-language-dropdown.png" alt="Friend entry showing the bundle language dropdown in the web UI">
<figcaption>Each friend has a language dropdown to set their bundle language</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/demo-pdf-es/page-1.png" alt="recover.pdf opened in Spanish">
<figcaption>The recovery tool opens in the friend's selected language</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/demo-pdf-es/page-2.png" alt="recover.pdf opened in Spanish">
<figcaption>Word lists are translated too (both languages work)</figcaption>
</figure>

## Advanced: Time-Delayed Recovery {#timelock}

You can set a waiting period when creating bundles. Even if your friends combine their pieces early, the files stay locked until the date you chose — 30 days, 6 months, a specific date.

### How to Enable

In the [bundle creator](maker.html), switch to **Advanced** mode and check **Add a time lock**. Choose how long the files should stay locked — up to two years.

The two-year limit is deliberate. Time locks depend on the League of Entropy continuing to operate, and we don't think it's responsible to lock your files behind a longer bet on external infrastructure. If you need a longer duration and understand the trade-off, the CLI has no cap: `rememory seal --timelock 5y`.

<figure class="screenshot">
<img src="screenshots/en/tlock-setup.png" alt="Time lock setup in Advanced mode">
<figcaption>The time lock option appears in Advanced mode</figcaption>
</figure>

### Recovery

When someone opens a time-locked bundle before the date, the recovery tool shows a waiting notice. Once the time passes, recovery proceeds normally.

<figure class="screenshot">
<img src="screenshots/en/tlock-waiting.png" alt="Recovery tool showing a time lock waiting notice">
<figcaption>The recovery tool waits until the time lock expires</figcaption>
</figure>

Opening a time-locked archive requires a brief internet connection. Your files aren't sent anywhere — the connection verifies that enough time has passed. Without the time lock, recovery is fully offline.

<div class="warning">
<strong>Experimental.</strong> Time-delayed recovery depends on the <a href="https://www.cloudflare.com/en-ca/leagueofentropy/" target="_blank">League of Entropy</a>, a distributed network operated by serious organizations around the world. If this network stops operating before a time lock expires, that archive becomes unrecoverable. Bundles without a time lock are not affected.
</div>

### How It Works {#timelock-technical}

The League of Entropy produces a new cryptographic value every 3 seconds. Each value is numbered. You can predict which number corresponds to a given time, but the value for that number can't be produced early — not by anyone, including the network operators.

When you create a time-locked bundle, the archive is encrypted to a specific future value. The key to open it doesn't exist yet. It will come from the network when that moment arrives.

For a deeper look at the cryptography behind this, see the [drand timelock encryption documentation](https://docs.drand.love/docs/timelock-encryption/).

## Self-Hosting {#selfhosted}

ReMemory can also run as a web app on your own server using `rememory serve`. The server provides the same creation and recovery tools through a browser. Friends only need their share — the encrypted archive is served automatically.

This is an advanced option for people who already run a homelab or want a shared web UI. The offline bundles remain the primary way to use ReMemory and work without any server. See the [self-hosting guide](https://github.com/eljojo/rememory/blob/main/docs/selfhosted.md) on GitHub for details.
