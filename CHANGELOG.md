# Changelog

All notable changes to ReMemory are documented here.

## Unreleased

- **Forged pieces are refused** — A specially crafted piece could make recovery open files chosen by whoever made it, instead of failing. Both the recovery tool and `rememory recover` now reject such pieces. Real pieces are never affected, and existing bundles work as before. Thanks @lyrixx for reporting this!
- **Every friend gets their own files** — Friends whose names use no Latin letters (小明, Олег) used to share one file name in `rememory seal`, so each piece overwrote the last and the secret could become unrecoverable. Their files are now numbered instead (`bundle-2.zip`). Names that differ only by capitalization or accents, like José and Jose, are now refused with a clear message, in the CLI and in the web maker. If you sealed a project with such names, please seal it again and send out the new bundles. Thanks @lyrixx for reporting this!
- **The web maker keeps the number of friends you chose** — Removing a friend, changing the page language or editing the number of pieces could quietly reset "how many friends must agree" to 2. The maker now keeps your choice, lowering it only when there are fewer friends than that, and restoring it when there are enough again. Importing a `project.yml` whose number is higher than its list of friends is now refused instead of being lowered. If you made bundles in the web maker, check the README of one of them: it says how many pieces are needed. If that's fewer than you intended, make new bundles. Thanks @lyrixx for reporting this!
- **Recovery works whatever language a bundle asks for** — A bundle set to a language the recovery tool doesn't have, such as `pt-BR` typed into `project.yml`, opened a page that showed nothing and couldn't recover. The page now uses the closest language it has (Portuguese here, or English). Each bundle carries its own copy of the recovery tool, so bundles made earlier keep the old one; if one of them shows an empty page, its pieces work with the recovery tool from a current release or the website. `rememory seal`, `rememory bundle` and importing into the web maker now refuse unsupported languages and list the ones you can use. Thanks @lyrixx for reporting this!
- **Anonymous bundles accept typed words** — In bundles made in anonymous mode, a piece typed in as recovery words, scanned from a QR code or pasted in its short form was silently ignored, so reading the words over the phone didn't work. These pieces are now added like any other. Each bundle carries its own copy of the recovery tool, so bundles made earlier keep the old one; with those, the recovery tool from a current release or the website accepts the pieces. Thanks @lyrixx for reporting this!
- **Catalan and Turkish recovery words** — Catalan and Turkish bundles print the recovery words in their own language first, but the recovery tool couldn't read them, and said a word wasn't recognized. It now understands both. Turkish words printed by v0.0.19, before the Turkish list was corrected, work too. Thanks @lyrixx for reporting this!
- **Recovery words for larger groups** — With 16 or more friends, the recovery tool refused the second piece typed as words from friend #16 onwards, saying it was already added. Each piece is now recognized by its own contents, so these are accepted. Thanks @lyrixx for reporting this!
- **Download again after recovering** — The recovery tool let go of the unlocked files after the first click on Download. If that download was cancelled or didn't arrive, clicking again did nothing, and the only way back was to start over with every piece. The files now stay available until you close the page. Thanks @lyrixx for reporting this!
- **File names shown as they are** — When recovering bundles made before v0.0.15, the recovery tool's file list mangled names with accents or other scripts (`contraseña.txt` appeared as `contrasea.txt`, `密碼.txt` as `.txt`) and cut some long names short. The downloaded files were always correct; the list now shows these names as they are, though very long folder paths can still look slightly off. Thanks @lyrixx for reporting this!
- **Files are offered even when the list can't be shown** — If a browser couldn't list what's inside the recovered archive (some older browsers can't open the format bundles before v0.0.15 used), the recovery tool showed an error and no download, even though everything had been unlocked. It now offers the download with a short note. Thanks @lyrixx for reporting this!
- **Pieces from one seal always fit together** — `rememory seal` gave each piece its own creation time. In the rare case that sealing crossed the turn of a minute, `rememory recover` and the recovery tool's file drop refused the set as coming from different seals. Every piece from a seal now carries the same time, as the web maker already did. A set whose times differ is no longer refused: you'll see a warning, and unlocking decides, so sets sealed by an unlucky earlier version recover with the current tools. Thanks @lyrixx for reporting this!
- **Files that couldn't be recovered are refused up front** — `rememory recover` extracts files of up to 100 MB, and 1 GB in total, but `rememory seal` and the web maker accepted larger ones. Recovering then stopped partway, without saying which file, and left a half-filled folder. Sealing and the maker now refuse such files right away and name them, and `rememory recover` checks every size before writing anything. Thanks @lyrixx for reporting this!
- **Bundles stay private on shared computers** — Each bundle holds a friend's piece, but `rememory seal` saved bundles so that other accounts on the same computer could read them. Bundles and their folders are now readable only by you, including ones left from earlier seals. Sealing again also replaces each file whole, so an interrupted seal no longer leaves a half-written bundle behind. Thanks @lyrixx for reporting this!
- **Clearer words about checksums** — The guide said a bundle's checksums let the recovery tool detect tampering. They don't: they sit in the same bundle, so they catch accidental damage, and only `rememory verify-bundle` checks them. The guide, README and command help now say so, and explain that the encrypted archive is protected on its own. Thanks @lyrixx for reporting this!
- **Hidden files are kept in the web maker** — The maker quietly left out files and folders whose names start with a dot, such as `.env`, `.ssh` or `.password-store`, which can be exactly what you meant to protect. They're now included, as `rememory seal` always did. If you made bundles in the maker from folders with hidden files, check the file list and make new bundles if something is missing. Thanks @lyrixx for reporting this!
- **Self-hosted server refuses requests from other websites** — A web page you visited could send requests to a `rememory serve` instance your browser can reach, for example to choose the admin password of a server that hadn't been set up yet. The server now refuses state-changing requests that come from another site. Thanks @lyrixx for reporting this!
- **The self-hosted server chooses its own admin password when it's reachable** — A new server that other machines could reach, as with the Docker image, let the first person to open it choose the admin password. When the server listens beyond `127.0.0.1`, it now generates the password at first start and prints it once (`docker logs rememory`). Set `REMEMORY_ADMIN_PASSWORD` to choose your own. Running `rememory serve` locally still asks on the first page, when opened as `localhost` on that computer; behind a reverse proxy, set `REMEMORY_ADMIN_PASSWORD`. Thanks @lyrixx for reporting this!
- **Docker examples keep the server on your machine** — The `docker run` and Docker Compose examples published the server on every network interface, so on a public host it was reachable from the internet right away. They now publish it on `127.0.0.1`; the self-hosting guide explains how to share it through a reverse proxy. Thanks @lyrixx for reporting this!
- **Creating bundles on the self-hosted server needs the admin password** — Anyone who could reach the server could upload bundles to it. Opening the maker now asks for the admin password (your browser shows the prompt; any user name works), and uploads without it are refused. Recovery pages stay open to friends. To let anyone create bundles as before, start the server with `--open-uploads` or set `REMEMORY_OPEN_UPLOADS=true`. Thanks @lyrixx for reporting this!
- **Safer first-time setup for the self-hosted server** — If two people set the admin password at the same moment, both were told it worked but only one password was kept. Now exactly one setup succeeds. The setup request is also limited in size, and setups run one at a time, so a burst of them can't exhaust the server's memory. Thanks @lyrixx for reporting this!
- **Bundle deletion on the self-hosted server can't be used to wear it down** — Each deletion attempt ran an expensive password check, even for bundles that don't exist, and many could run at once. Unknown bundles and oversized requests are now turned away before the check, and checks run one at a time, which also slows down password guessing. Thanks @lyrixx for reporting this!
- **Uploads to the self-hosted server are size-limited** — An upload could carry an unlimited amount of extra data, which the server stored and then re-read on every page view, slowing it down for good. Uploads are now limited to the archive size you configured plus a little room, bundle details must be small and sensible, and oversized details stored by earlier versions are skipped. Thanks @lyrixx for reporting this!
- **Faster maker page on the self-hosted server** — Every visit to the maker page re-compressed the whole bundle-creation engine, about a second of work each time, which also made it easy to keep the server busy. It's now prepared once. Thanks @lyrixx for reporting this!
- **The self-hosted recovery page no longer guesses between bundles** — Opening `/recover.html` without choosing a bundle loaded whichever one was uploaded last, which could be anyone's, and its failure to open blamed the friend's pieces. It now loads a bundle on its own only when the server holds exactly one; otherwise, choose the bundle from the home page. Thanks @lyrixx for reporting this!
- **The self-hosted server closes stalled connections** — Connections that never finish sending a request, or sit idle, are now closed after a while instead of being held open indefinitely. Large uploads and downloads aren't affected. Thanks @lyrixx for reporting this!
- **Large folders dragged into the web maker are read in full** — Dragging in a folder with more than about 100 files (in Chrome and similar browsers) kept only the first 100 of each folder, and dropping several files or folders at once could keep only the first. Every file is now included. Choosing files with the file picker was not affected.

## v0.0.21 — 2026-04-13

- Minor improvements to arm64 support in the Docker image.

## v0.0.20 — 2026-04-13

- **Bundle verification** — When a friend opens their bundle with just their own piece, a card now confirms everything is working: which piece they hold, how many are needed, and whether the encrypted archive is included. A small reassurance that their bundle is ready for when it matters. Thanks @tmchow!
- **Simplified Chinese support** — Added Simplified Chinese (zh-CN) across the recovery tool, maker, bundle instructions, and landing page. Thanks @gabrnavarro!
- **Improved Turkish wordlist** — Updated the Turkish BIP39 word list with corrections. Thanks @FrustT!
- **Pasted share fix** — Shares copied from PDFs now work reliably. Some PDF viewers strip a blank line that the recovery tool expected, causing pasted shares to be silently ignored. Thanks @tmchow!
- **Italian translation fixes** — Corrected typos in the Italian translations. Thanks @xushidev!

## v0.0.19 — 2026-03-18

- **Turkish language support** — Added Turkish as a supported language. Thanks @FrustT!
- **Bundles now work across operating systems** — Archives created on Windows now extract correctly everywhere, and vice versa. Thanks @vnykmshr!
- **Docker security** — The Docker image now runs as a non-root user. Thanks @vnykmshr!

## v0.0.18 — 2026-03-16

- **Italian language support** — Added Italian as a supported language. Thanks @xushidev!
- **Docker on arm64** — The Docker image now ships for both amd64 and arm64. `docker pull` picks the right one automatically.

## v0.0.17 — 2026-03-11

- **Dutch language support** — Added Dutch as a supported language for the recovery tool, maker, and bundle instructions. Thanks @idebeijer!
- **Catalan language support** — Added Catalan translations and wordlist support. Uses the Spanish BIP39 wordlist as a fallback for Catalan bundles. Thanks @xcxtxsx!
- **Chinese text in PDFs** — PDFs now render Chinese characters correctly by bundling the Noto Sans SC font. Previously, Chinese text could appear as blank boxes. Thanks @gabrnavarro!
- **Update nudge** — The CLI and web maker now gently let you know if you're running a version built more than six months ago, with a link to check for updates. No network calls — it just compares the build date to your clock.
- **Mismatched piece detection** — Adding a piece from a different set now shows a clear error right away, instead of failing later with a generic decryption message.
- **Rewritten documentation** — The guides and landing page have been rewritten to focus on what people actually need to do, in all supported languages.

## v0.0.16 — 2026-02-25

- **Time-delayed recovery** — You can now set a waiting period when creating bundles. Even if your friends combine their pieces early, the files stay locked until the date you chose. Uses the [League of Entropy](https://www.cloudflare.com/en-ca/leagueofentropy/) — a distributed randomness beacon run by organizations around the world, not a single company. This is experimental: recovery requires a brief internet connection, and relies on the beacon continuing to operate. CLI: `rememory seal --timelock 30d`. Also available in the web maker under "Advanced options."
- **Self-hosted web UI** — ReMemory can now run as a small web server on your home network. `rememory serve` starts a local instance where you can create bundles, store manifests on the server, and recover from any browser on the network. A home page shows all stored bundles at a glance, with one-click recovery for each. Designed for families or small groups who want a shared place to manage their recovery kits without relying on external services. See the [self-hosted guide](docs/selfhosted.md) for setup.
- **Static pages for easy hosting** — `rememory seal --pages` (or `bundle --pages`, `demo --pages`) now outputs a `output/pages/` directory containing `recover.html` and `MANIFEST.age`. Drop that folder on GitHub Pages, Netlify, or any static host and friends can visit the URL to recover — the page fetches the manifest automatically. No server needed.
- **Translated guide and landing page** — The guide (`docs.html`) is now available in Spanish, German, and French, each as a standalone page with its own language picker. The landing page (`about.html`) now auto-detects your browser language and switches between all seven supported languages — English, Spanish, German, French, Portuguese, Slovenian, and Traditional Chinese.
- **Offline by design, verified by default** — ReMemory has always worked without the internet, but now the test suite proves it. Every test runs with network access blocked. The handful that genuinely need a connection — like checking the time-lock beacon — declare exactly which hosts they're allowed to reach. If any part of the app tries to phone home, the test fails. Time-lock encryption now works fully offline too (recovery still requires internet).

## v0.0.15 — 2026-02-21

- **ZIP archive format** — Encrypted payloads now use ZIP instead of tar.gz. ZIP is understood by every operating system without extra tools, which makes manual inspection easier if someone ever needs it. Existing bundles created with earlier versions still work — the recovery tool detects the format automatically.
- **Recover directly from bundles** — `rememory recover` now accepts bundle ZIP files. You can pass the ZIPs your friends send you without unzipping them first, and the manifest is extracted automatically. `rememory recover bundle-alice.zip bundle-bob.zip` just works.

## v0.0.14 — 2026-02-19

- **Improved Docs**: improve guides on the website. Also updated demo bundles so it contains 5 friends instead of 3.

## v0.0.13 — 2026-02-18

- **Simpler recovery tool** — The recovery tool (`recover.html`) no longer relies on WebAssembly. It now uses plain JavaScript, making it smaller, faster to load, and more likely to work in browsers for decades to come. The CLI and bundle creation still use Go, but recovery — the part your friends will use — is now built on simpler, more universal technology.
- **Bigger files, simpler recovery** — Encrypted archives up to 10 MB (up from 5 MB) are now embedded directly in `recover.html`. More people will be able to recover by just opening the HTML file, without needing the separate `MANIFEST.age`.
- **PDF import in recovery** — PDFs from bundles now contain a machine-readable share appended to the file. You can drop a PDF directly into the recovery tool instead of typing or scanning — it just works.
- **QR scanning on more browsers** — Added a polyfill for browsers that don't support the BarcodeDetector API, so QR code scanning works in more places.
- **Tappable email addresses** — Email addresses in the contact list are now `mailto:` links, so you can tap them on a phone instead of copying and pasting. Thanks @vnykmshr!
- **Accessibility improvements** — Keyboard users can now see where they are when tabbing through the recovery tool (focus indicators). Improved contrast for muted text to meet WCAG AA. Added ARIA attributes for screen readers. Thanks @vnykmshr!
- **Safer sealing** — Running `seal` on a project where you haven't added any files to the manifest (only the template README) now gives a clear error instead of creating a useless encrypted bundle. Thanks @vnykmshr!
- **Better drop zones** — Improved the look and behavior of file drop areas in the recovery tool.
- **Language detection fix** — Regional browser locales (like `en-GB` or `es-MX`) are now detected correctly instead of falling back to English. Thanks @vnykmshr!
- **Updated Slovenian translations** — Thanks @h200101!

## v0.0.12 — 2026-02-13

- **Chinese (Traditional) support** — Added zh-TW as a seventh language for the recovery tool, maker, and bundle instructions. Thank you @JasonHK!
- **Portuguese translations** — Added Portuguese as a sixth supported language. Thanks @Kasama!
- **Language picker** — Both the recovery tool and maker now have a language switcher, so people can choose their language themselves.
- **Named and anonymous toggle in maker** — The maker UI now has a clear selector for choosing between named and anonymous bundles, instead of burying the option.
- **Security hardening** — Added Content Security Policy headers to prevent XSS. Added size limits to ZIP extraction in the browser path (thanks @vnykmshr). Applied additional safety validations from a [security review](docs/security-review.md).
- **PDF polish** — Updated colours to better match the palette, added a coloured identity bar, and tweaked the header layout.

## v0.0.11 — 2026-02-11

- **PDF redesign** — The README PDF has a clearer visual hierarchy and now shows the recovery rule ("2 of 3 required") prominently, so friends know at a glance how many people need to come together.
- **Consistent colour palette** — Standardized colours across the website, recovery tool, and PDF to use the same warm, low-contrast palette throughout.
- **PDF screenshots in README** — The project README now shows example PDF pages so people can see what friends receive before trying the tool.

## v0.0.10 — 2026-02-11

- **Embedded manifest in recovery bundles** — If your encrypted manifest is under 5 MB, it's now embedded directly inside `recover.html`. Friends can recover secrets without needing the separate `MANIFEST.age` file at all — just open the HTML file and go.
- **Per-friend language support** — Each friend can now have their own language preference. READMEs and BIP39 word lists in bundles are translated to their language using official translated word lists.
- **Intro card in the recovery tool** — A welcome card now explains what `recover.html` is and how it works, making the first-time experience less confusing for non-technical friends.
- **Improved recovery navigation** — The top navigation adapts based on whether the recovery tool is personalized (from a bundle) or generic (standalone download).
- **PDF layout fix** — QR codes no longer break across page boundaries in PDFs.
- **Guidance on revoking access** — Added documentation on what to do if you need to cut someone out of your recovery group.

## v0.0.9 — 2026-02-10

- **PDF word list fix** — Fixed a bug where BIP39 word lists in the PDF could split awkwardly across many pages, making them hard to read.

## v0.0.8 — 2026-02-10

The biggest release so far — a new protocol version, word-based shares, QR codes, and a more flexible contact system.

- **Protocol V2** — Shamir secret sharing now operates on raw bytes instead of base64-encoded strings. This produces shorter, more efficient shares. V1 bundles remain fully recoverable.
- **BIP39 word lists** — Each share can now be represented as 24 human-readable words (similar to Bitcoin seed phrases). Supports English, Spanish, French, German, and Slovenian. This makes it possible to write a share on paper, print it, or read it over the phone — no digital device required.
- **QR codes in PDFs** — Each friend's PDF now includes a QR code containing their share. Scanning the code is the fastest way to enter a share during recovery.
- **Compact share format** — A new shorter encoding for shares, used in QR codes and other space-constrained contexts.
- **Flexible contact field** — The separate "phone" and "email" fields have been replaced with a single free-text "contact" field. Put whatever you want in it — phone, email, Signal handle, physical address.
- **Recovery UX improvements** — Better messaging when only one more share is needed to reach the threshold. Improved flow after real-world testing with printed PDFs.
- **Standalone recovery tool** — A generic `recover.html` (not tied to any specific bundle) is now available for download from GitHub Releases. QR codes in PDFs link to it.
- **Backward compatibility testing** — Added golden test fixtures for V1 to ensure old bundles remain recoverable as the protocol evolves.

## v0.0.7 — 2026-02-08

- **Slovenian language support** — Added Slovenian as a fifth supported language for the recovery UI and instructions. Thanks to @h200101!
- **Unicode name fix** — Fixed an issue where non-ASCII characters in friend names (accents, umlauts, etc.) could cause problems when creating bundle folders. Thanks @h200101!
- **Path traversal protection** — Hardened tar.gz extraction against directory traversal attacks. Thanks to @vnykmshr!

## v0.0.6 — 2026-02-05

- **Release fix** — Resolved an issue with GitHub release publishing.

## v0.0.5 — 2026-02-05

- **Anonymous mode** — New option to create bundles without identifying information, for privacy-conscious setups where you don't want friend names in the bundles.
- **Redesigned website** — The homepage got a visual refresh with a cleaner layout.
- **Improved error handling** — Better error messages and guidance throughout the recovery flow.

## v0.0.4 — 2026-02-04

- **Web-based bundle creation** — You can now create recovery bundles entirely in the browser using `maker.html`, no CLI needed. Available on the project website.
- **Project website** — A proper landing page with documentation, hosted on GitHub Pages.
- **Smaller, faster recovery tool** — The WebAssembly payload in `recover.html` is now gzip-compressed and stripped down to only what's needed for recovery, making the file smaller and faster to load.
- **Bug fix: files lost when adding more** — Fixed a bug in the web UI where adding more files would cause previously added files to disappear.
- **Better translations** — Improved localization across supported languages.
- **Apache 2.0 license** — The project is now licensed under Apache 2.0.

## v0.0.3 — 2026-02-03

- **Personalized recovery bundles** — Each friend now gets a customized `recover.html` with their share pre-loaded and contact information for the other friends embedded. No more copying and pasting shares — just open the file and follow the steps.
- **Share verification** — The recovery tool now validates shares before attempting decryption, catching typos and errors early with a clear message instead of a cryptic failure.
- **Security hardening** — Tightened file permissions on generated bundles and added warnings when attempting to encrypt symlinks (which could lead to unexpected behavior).

## v0.0.2 — 2026-02-03

- **Auto-bundling** — Running `seal` now automatically generates friend bundles in one step, instead of requiring a separate `bundle` command.
- **Documentation cleanup** — Reorganized and improved the guides and README.

## v0.0.1 — 2026-02-03

The first release of ReMemory.

- **Encrypt and split** — Encrypt files with [age](https://github.com/FiloSottile/age) and split the decryption key among trusted friends using [Shamir's Secret Sharing](https://en.wikipedia.org/wiki/Shamir%27s_secret_sharing) (via HashiCorp Vault's implementation).
- **Offline browser recovery** — Each friend receives a self-contained `recover.html` that works offline in any browser. No servers, no internet, no installation — just open the file.
- **PDF instructions** — Each friend's bundle includes a PDF with clear recovery instructions and their share.
- **Multi-language support** — Recovery instructions available in English, Spanish, German, and French.
- **CLI tool** — Commands to initialize a project, seal secrets, generate bundles, and recover.
