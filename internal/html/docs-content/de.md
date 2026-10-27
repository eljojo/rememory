---
title: "\U0001F9E0 ReMemory Anleitung"
subtitle: "Pakete erstellen und Dateien wiederherstellen"
cli_guide_note: 'Es gibt auch eine <a href="{{GITHUB_REPO}}/blob/main/docs/guide.md">Anleitung für die Kommandozeile</a>.'
nav_home: "\U0001F9E0 ReMemory"
nav_home_link: "Startseite"
nav_create: "Pakete erstellen"
nav_recover: "Wiederherstellen"
toc_title: "Inhalt"
footer_source: "Quellcode"
footer_download: "CLI herunterladen"
footer_home: "Startseite"
---

## Überblick {#overview}

ReMemory ist ein digitaler Tresor mit mehreren Schlüsseln. Du entscheidest, wer sie hält und wie viele zusammenkommen müssen, um ihn zu öffnen.

Unter der Haube verschlüsselt es deine Dateien mit [age](https://github.com/FiloSottile/age), teilt den Schlüssel unter Vertrauenspersonen auf und gibt jeder Person ein eigenständiges Paket zur Wiederherstellung.

Wiederherstellung heißt: eine Datei im Browser öffnen.\* Keine Server, die Website muss dafür nicht existieren.

Wenn du für jemanden einen Teil aufbewahrst, fang bei [Dateien wiederherstellen](#recovering) an.

<p style="font-size: 0.8125rem; color: #8A8480;">* Archive mit <a href="#timelock" style="color: #8A8480;">Zeitsperre</a> brauchen bei der Wiederherstellung eine kurze Internetverbindung.</p>

<div class="tip">
<strong>Hinweis:</strong> Keine einzelne Person kann auf deine Daten zugreifen. Dazu müssen genügend Teile zusammenkommen — zum Beispiel 3 von 5.
</div>

## Warum ReMemory {#why-rememory}

Du hast wahrscheinlich digitale Informationen, die wichtig sind: Wiederherstellungscodes deines Passwort-Managers, Kryptowährungs-Seeds, wichtige Dokumente, Anweisungen für deine Liebsten. Was passiert damit, wenn du eines Tages nicht mehr erreichbar bist?

Stell dir einen Tresor vor, der zwei Schlüssel braucht — keine einzelne Person hat genug, um allein hineinzukommen.

Herkömmliche Ansätze haben Schwächen:

- **Einer Person alles geben** — ein einzelner Ausfallpunkt und Vertrauenspunkt
- **Dateien manuell aufteilen** — verwirrend, fehleranfällig, ohne Verschlüsselung
- **Notfallzugang eines Passwort-Managers nutzen** — im Grunde auch „einer Person alles geben", und die Firma muss weiter bestehen
- **Im Testament hinterlegen** — wird öffentlich, langsamer Rechtsweg

ReMemory geht einen anderen Weg:

- **Kein einzelner Ausfallpunkt** — mehrere Personen müssen zusammenarbeiten
- **Kein blindes Vertrauen in eine Person** — selbst dein engster Freund kommt allein nicht an deine Geheimnisse
- **Offline und eigenständig** — die Wiederherstellung funktioniert ohne Internet oder Server\*
- **Für alle gemacht** — klare Anleitungen, keine kryptografischen Fachkenntnisse nötig

## Pakete erstellen {#creating}

Drei Schritte. Alles passiert in deinem Browser — deine Dateien verlassen dein Gerät nicht. Öffne den [Paket-Ersteller](maker.html), um loszulegen.

### Schritt 1: Freunde hinzufügen {#step1}

Füge die Personen hinzu, die Teile deines Wiederherstellungsschlüssels aufbewahren sollen. Für jede Person gibst du einen Namen und optional Kontaktdaten an.

<figure class="screenshot">
<img src="screenshots/de/friends.png" alt="Freunde hinzufügen in Schritt 1">
<figcaption>Jede Person hier bewahrt einen Teil des Schlüssels auf</figcaption>
</figure>

Dann wählst du, wie viele von ihnen zusammenkommen müssen, um deine Dateien zu öffnen.

<div class="tip">
<strong>Wie viele es sein sollten:</strong>
<ul>
<li><strong>2 von 3 Personen:</strong> Die einfachste Konfiguration</li>
<li><strong>3 von 5 Personen:</strong> Eine gute Balance</li>
<li><strong>4 oder 5 von 7 Personen:</strong> Sicherer, mehr Koordination</li>
</ul>
Hoch genug, dass einige wenige die Dateien nicht ohne die anderen öffnen können. Niedrig genug, dass es noch klappt, wenn ein oder zwei nicht erreichbar sind.
</div>

### Schritt 2: Dateien hinzufügen {#step2}

Ziehe die Dateien oder den Ordner, den du schützen willst, per Drag & Drop in den Bereich.

<figure class="screenshot">
<img src="screenshots/de/files.png" alt="Dateien hinzufügen in Schritt 2">
<figcaption>Füge die Dateien hinzu, die du schützen willst</figcaption>
</figure>

**Gute Kandidaten:**

- Wiederherstellungscodes des Passwort-Managers
- Kryptowährungs-Seeds und -Schlüssel
- Wichtige Zugangsdaten
- Anweisungen für Angehörige
- Standorte wichtiger Dokumente
- Tresor-Kombinationen

<div class="warning">
<strong>Hinweis:</strong> Vermeide Dateien, die sich häufig ändern. Das hier ist für Geheimnisse gedacht, die du einmal festlegst und dann liegen lässt.
</div>

### Schritt 3: Pakete erstellen {#step3}

Klicke auf „Umschläge erstellen", um deine Dateien zu verschließen und für jede Person ein Paket zu erzeugen.

<figure class="screenshot">
<img src="screenshots/de/bundles.png" alt="Pakete erstellen in Schritt 3">
<figcaption>Lade jedes Paket einzeln herunter, oder alle auf einmal</figcaption>
</figure>

Jedes Paket enthält das vollständige Wiederherstellungstool. Es funktioniert selbst dann, wenn diese Website nicht mehr existiert.

### An Freunde verteilen {#distributing}

Schicke jeder Person ihr Paket, wie du willst:

- **E-Mail:** Die ZIP-Datei als Anhang
- **Cloud-Speicher:** Über Dropbox, Google Drive usw. teilen
- **USB-Stick:** Persönliche Übergabe
- **Verschlüsselter Messenger:** Signal, WhatsApp usw.

Wenn du Pakete per E-Mail oder Messenger verschickst, lösche die Nachrichten aus deinem Gesendet-Ordner, sobald die jeweilige Person den Empfang bestätigt hat. Sonst behält dein Konto eine Kopie jedes Teils — und genau das wollte das Aufteilen verhindern.

### Nach dem Erstellen {#after-creating}

Sobald deine Pakete fertig sind, gibt es ein paar Dinge, die sich lohnen, bevor du das Thema beiseitelegst:

- Bitte jede Person, `recover.html` einmal zu öffnen. Wenn die Seite ihren Teil lesen kann, steht dort „Dein Teil ist da".
- Erkläre jeder Person, was das ist, warum sie es hat, und dass sie es sicher aufbewahren soll. Allein kann sie nichts damit anfangen — sie muss sich mit anderen abstimmen.
- Sichere die verschlossenen Dateien mit „Die verschlossenen Dateien speichern (MANIFEST.age)" und bewahre diese Kopie an einem sicheren Ort auf. Sie enthält keinen Teil, also ist sie ohne genügend Teile für niemanden von Nutzen.
- Sichere deine `project.yml`, um die Pakete später neu erstellen zu können
- Drucke `LIESMICH.pdf` als Papier-Backup aus, bevor du das digitale Paket verschickst. Papier braucht keine Adapter und keinen Strom.
- Setze eine jährliche Erinnerung — siehe [Pakete aktuell halten](#keeping-current)
- Erwäge eine [Wiederherstellungsübung](#recovery-drills) — nur so weißt du, ob dein Plan wirklich funktioniert

## Dateien wiederherstellen {#recovering}

Dieser Abschnitt ist für die Freunde, die einen Teil aufbewahren. Meistens tut ein Freund nichts weiter, als sein Paket sicher aufzubewahren. Eines Tages bittet ihn vielleicht jemand um seinen Teil, oder er ist derjenige, der die Teile zusammenbringt.

Wenn du hier bist, weil jemand, der dir wichtig ist, nicht mehr erreichbar ist: Es eilt nicht. Nichts hier läuft ab. Nimm dir so viel Zeit, wie du brauchst — wenn du die Wiederherstellungsseite zwischendurch schließt, fügst du die Teile wieder hinzu, sobald du zurückkommst.

### Was Freunde erhalten {#bundle-contents}

Jedes Paket enthält:

<div class="bundle-contents">
<div class="file">
<span class="file-name">LIESMICH.txt</span>
<span class="file-desc">Worum es geht, wer sonst noch einen Teil hat, und dein Teil selbst</span>
</div>
<div class="file">
<span class="file-name">LIESMICH.pdf</span>
<span class="file-desc">Dasselbe, bereit zum Ausdrucken. Dein Teil steht darauf als <strong>QR-Code</strong> und als Wörter.</span>
</div>
<div class="file">
<span class="file-name">MANIFEST.age</span>
<span class="file-desc">Die verschlossenen Dateien. Nur in manchen Paketen — die meisten tragen sie stattdessen in recover.html.</span>
</div>
<div class="file">
<span class="file-name">recover.html</span>
<span class="file-desc">Die Wiederherstellungsseite. Sie öffnet sich in jedem Webbrowser, mit deinem Teil schon darin.</span>
</div>
</div>

<p style="margin-top: 1rem;">
Jedes Paket ist für einen bestimmten Freund gemacht: Sein Teil steckt bereits in seiner recover.html, zusammen mit den Namen der anderen und wie man sie erreicht.
</p>

### Dein Paket aufbewahren {#keeping-bundle}

Bewahre es an einem sicheren Ort auf, und wenn möglich auch einen Ausdruck von LIESMICH.pdf. Bis es so weit ist, gibt es nichts weiter zu tun.

Um es zu prüfen, öffne recover.html. Dort steht „Dein Teil ist da", welchen Teil du hast, wie viele nötig sind, um die Dateien zu öffnen, und ob die verschlossenen Dateien auch da sind.

### Wenn dich jemand um deinen Teil bittet {#sharing-piece}

Prüfe zuerst, ob die Anfrage echt ist. Frag, wenn möglich, bei der Person nach, die das Paket erstellt hat. Wenn du sie nicht erreichen kannst, frag bei einem anderen Freund nach, der einen Teil hat.

Dann schick ihn auf dem Weg, der für dich am einfachsten ist:

- **Das ganze Paket** (die ZIP-Datei) — am einfachsten.
- **Deine LIESMICH.txt** — sie enthält deinen Teil auch für sich allein.
- **Deine Wiederherstellungswörter**, am Telefon vorgelesen.
- **Deine ausgedruckte Seite**, per Post — der QR-Code darauf enthält deinen Teil.

### Die Teile zusammenbringen {#recovery-bundle}

Eine Person sammelt die Teile. Das kann jeder der Freunde sein — wer es eben übernimmt.

<div class="step-guide">
<div class="step-number">1</div>
<div class="step-content">
<h4>recover.html öffnen</h4>
<p>Doppelklicke auf das Paket, um es zu öffnen, dann doppelklicke auf recover.html. Es öffnet sich in deinem Webbrowser wie eine Webseite, mit deinem eigenen Teil schon darin. Wenn du kein Paket hast, kann dir jeder Freund seine recover.html schicken, oder du nutzt die <a href="recover.html">Wiederherstellungsseite</a> hier.</p>
</div>
</div>

<div class="step-guide">
<div class="step-number">2</div>
<div class="step-content">
<h4>Die anderen kontaktieren</h4>
<p>Wenn du ein Paket hast, zeigen seine recover.html und seine LIESMICH, wer sonst noch einen Teil hat und wie man diese Personen erreicht — außer die Pakete wurden ohne Namen erstellt. Erzähl ihnen, was passiert ist, und bitte sie um ihren Teil. Manche werden erst bei dir nachfragen wollen, ob die Anfrage echt ist — ihre Anleitung bittet sie darum.</p>
</div>
</div>

<div class="step-guide">
<div class="step-number">3</div>
<div class="step-content">
<h4>Jeden Teil hinzufügen, sobald er ankommt</h4>
<p>Zieh ihr Paket oder ihre LIESMICH.txt auf die Seite, füge ihren Text ein, tippe die Wiederherstellungswörter ein, die sie dir vorlesen, oder scanne den QR-Code auf ihrer ausgedruckten Seite. Wenn Namen aufgeführt sind, erscheint neben jedem ein Häkchen.</p>
</div>
</div>

<div class="step-guide">
<div class="step-number">4</div>
<div class="step-content">
<h4>Die Dateien öffnen sich von selbst</h4>
<p>Sobald genug Teile da sind, entsperrt die Seite die Dateien, ohne dass du etwas tun musst. Lade sie herunter und bewahre sie an einem privaten Ort auf.</p>
</div>
</div>

<div class="tip">
<strong>Wenn die Seite nach den verschlossenen Dateien fragt:</strong> Zieh <code>MANIFEST.age</code> aus dem Paket eines Freundes auf die Seite — oder, wenn sein Paket keine hat, seine recover.html. Die meisten Pakete enthalten sie bereits, deshalb wirst du diesen Schritt meist gar nicht sehen.
</div>

<figure class="screenshot">
<img src="screenshots/de/recovery-1.png" alt="Wiederherstellungsseite mit den bisher gesammelten Teilen">
<figcaption>Die bisher gesammelten Teile, und wen du kontaktieren kannst</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/de/recovery-2.png" alt="Wiederherstellungsseite, nachdem die Dateien entsperrt wurden">
<figcaption>Sobald genug Teile da sind, werden die Dateien entsperrt und können heruntergeladen werden</figcaption>
</figure>

### Wiederherstellungswörter eingeben {#recovery-words}

Auf jeder ausgedruckten Seite steht der Teil als Liste nummerierter Wörter. Wenn dir ein Freund seine Wörter am Telefon vorliest, oder wenn du nur Papier hast, tippe die Wörter in die Wiederherstellungsseite ein. Drücke auf „Teil einfügen oder Wiederherstellungswörter eingeben" und tippe sie ein, getrennt durch Leerzeichen. Es zählen nur die Wörter, nicht die Nummern.

<figure class="screenshot">
<img src="screenshots/de/recovery-words-typing.png" alt="Wiederherstellungswörter von einer ausgedruckten Seite in die Wiederherstellungsseite eingeben">
<figcaption>Tippe die Wörter von der ausgedruckten Seite ein</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/de/recovery-words-recognized.png" alt="Wiederherstellungsseite nach Eingabe der Wörter, der Teil wurde hinzugefügt">
<figcaption>Die Seite erkennt die Wörter und fügt den Teil hinzu</figcaption>
</figure>

### Einen QR-Code scannen {#recovery-pdf}

Drücke an einem Computer oder Handy mit Kamera auf der Wiederherstellungsseite auf „QR-Code scannen" und halte die ausgedruckte Seite vor die Kamera.

<figure class="screenshot">
<img src="screenshots/qr-camera-permission.png" alt="Browser fragt nach der Erlaubnis, die Kamera zu nutzen">
<figcaption>Dein Browser fragt, ob er die Kamera nutzen darf</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/qr-scanning.png" alt="QR-Code von einer ausgedruckten Seite scannen">
<figcaption>Halte den QR-Code vor die Kamera, um den Teil hinzuzufügen</figcaption>
</figure>

Du kannst den QR-Code auch mit der Kamera-App eines Handys scannen: Dann öffnet sich die Wiederherstellungsseite auf dieser Website, mit dem Teil schon eingetragen.

<div class="tip">
<strong>Zur Wiederherstellung:</strong>
<ul>
<li>Funktioniert ohne <span title="Archive mit Zeitsperre brauchen eine Verbindung, um das Entsperrdatum zu prüfen.">Internetverbindung*</span></li>
<li>Die Teile und die Dateien bleiben auf deinem Gerät</li>
<li>Freunde können überall sein — jeder muss dir nur seinen Teil zukommen lassen</li>
</ul>
</div>

## Empfehlungen {#best-practices}

### Freunde wählen

- **Langfristigkeit:** Personen, die du in 5 bis 10 Jahren noch erreichen kannst
- **Geografische Verteilung:** Nicht alle am gleichen Ort
- **Technisches Wissen:** Niemand muss sich mit Technik auskennen. Die Anleitung führt alle Schritt für Schritt hindurch, und eine [Wiederherstellungsübung](#recovery-drills) zeigt dir, wo jemand hängen bleibt.
- **Beziehungen:** Werden sie miteinander kooperieren?
- **Vertrauen:** Ein einzelner Teil verrät nichts, aber du vertraust ihnen eine Verantwortung an

### Sicherheitshinweise

- Bewahre nicht alle Pakete zusammen auf — das würde den Sinn des Aufteilens zunichtemachen
- Erwäge, `LIESMICH.pdf` auszudrucken — Papier übersteht digitale Katastrophen
- Sichere `project.yml`, um die Pakete später neu erstellen zu können

### Pakete sicher aufbewahren {#storing-bundles}

Pakete sind klein (unter 10 MB) und dafür gedacht, an ganz normalen Orten aufbewahrt zu werden. Das funktioniert gut:

- **E-Mail** funktioniert gut. Die meisten Menschen behalten dieselbe E-Mail-Adresse jahrzehntelang, und Pakete sind klein genug als Anhang. Viele Anbieter speichern Nachrichten unbegrenzt.
- **Cloud-Speicher** (Google Drive, Dropbox, iCloud) eignet sich gut als Zweitkopie.
- **USB-Sticks** können funktionieren, aber bedenke: Anschlüsse ändern sich (USB-A weicht bereits USB-C), und Flash-Speicher kann über Jahre ohne Strom an Qualität verlieren. Nicht ideal als einzige Kopie.
- **Papier** ist die haltbarste Option. `LIESMICH.pdf` ausdrucken gibt deinen Freunden eine Kopie ihres Teils, die keine Adapter, keinen Strom und kein funktionierendes Gerät braucht.

Auf Papier stehen die Teile, aber nicht die verschlossenen Dateien selbst — die gibt es nur in digitalen Kopien. Am besten ist deshalb Redundanz — E-Mail plus Papier, oder Cloud plus Papier. Mehr als eine Kopie, in mehr als einer Form.

### Pakete aktuell halten {#keeping-current}

Setze eine jährliche Erinnerung, um bei deinen Freunden nachzufragen. Bestätige, dass sie noch ihre Pakete haben, und aktualisiere Kontaktdaten, wenn sich etwas geändert hat.

Wenn sich deine Dateien ändern, erstelle neue Pakete und verschicke sie. Die alten Pakete öffnen das neue Archiv nicht, also schadet es nicht, wenn sie noch herumliegen — aber bitte deine Freunde, ihre durch die neuen zu ersetzen.

Wenn sich Kontakte ändern — jemand zieht um, wechselt die Telefonnummer, oder du willst jemanden hinzufügen oder entfernen — dasselbe Vorgehen: neue Pakete, alte löschen lassen.

Zwischen den Aktualisierungen bewahrst du deine Quelldateien am besten in einem verschlüsselten Tresor auf — Tools wie [Cryptomator](https://cryptomator.org) oder [VeraCrypt](https://veracrypt.fr) funktionieren gut. Lass keine unverschlüsselten Kopien in einem normalen Ordner liegen.

Denk daran wie ans Aktualisieren deiner Notfallkontakte. Kurz, regelmäßig, es lohnt sich.

### Wiederherstellungsübung {#recovery-drills}

Niemand testet seinen Notfallplan. Eine Wiederherstellungsübung ändert das — und es spielt sich ein bisschen wie ein Escape Room.

Erstelle ein Test-Paket mit einem harmlosen Geheimnis (ein Foto, eine Nachricht, ein Rezept). Verteile die Teile an deine Freunde. Setze einen [Zeitschloss](#timelock), wenn du einen echten Countdown willst. Dann tritt zurück und lass sie es herausfinden: koordinieren, Teile zusammenführen, Dateien entsperren — ohne deine Hilfe.

Was du lernst:

- Können deine Freunde ihre Pakete finden, wenn sie sie brauchen?
- Verstehen sie die Anleitung, oder kommen sie nicht weiter?
- Können sie sich untereinander erreichen, ohne dass du vermittelst?

Eine Übung, die gut läuft, bedeutet, dass dein echter Plan auch funktioniert. Eine Übung, die schiefgeht, ist ein Geschenk — du hast das Problem gefunden, bevor es wichtig war.

### Zugriff widerrufen {#revoking-access}

Sobald ein Teil verteilt ist, kann er nicht zurückgerufen werden. Das ist beabsichtigt — es gibt keinen Server, keine zentrale Instanz.

Wenn du ändern willst, wer Teile hat:

1. **Erstelle neue Pakete** mit einer neuen Gruppe von Freunden und einem neuen Schlüssel
1. **Verschicke die neuen Pakete** an die Freunde, denen du weiterhin vertraust
1. **Bitte jeden Freund, sein altes Paket zu löschen** und durch das neue zu ersetzen

<div class="warning">
<strong>Wichtig:</strong> Alte Teile funktionieren weiterhin mit alten Archiven. Wenn du ein neues Paket verschickst, mach es deutlich: <strong>Das alte löschen</strong>, nur das neue behalten. Keine Versionshistorie, kein „zur Sicherheit."
</div>

Dasselbe gilt, wenn sich Geheimnisse ändern. Neue Pakete bedeuten einen neuen Schlüssel und neue Teile. Die alten Teile öffnen das neue Archiv nicht, funktionieren aber weiterhin mit dem alten. Achte darauf, dass deine Freunde keine alten Kopien aufheben.

### Über project.yml {#project-file}

Wenn du Pakete erstellst, wird dein Projekt in einer `project.yml`-Datei gespeichert. Diese Datei enthält:

- Namen und Kontaktdaten der Freunde
- Die gewählte Schwelle (z.B. 3 von 5)
- Einen Prüf-Hash, um festzustellen, ob Pakete zusammengehören
- Prüfsummen der Teile, um die Integrität der Pakete zu bestätigen

Sie enthält **keine** Geheimnisse — kein Passwort, kein Schlüsselmaterial, keine Dateiinhalte. Du kannst sie bedenkenlos bei deinen anderen Unterlagen aufbewahren.

Mit `project.yml` kannst du Pakete neu erstellen, bestehende überprüfen und den Stand deiner Konfiguration einsehen.

## Die Sicherheit verstehen {#security}

ReMemory kombiniert bewährte kryptografische Werkzeuge, statt eigene zu erfinden. Das bedeutet in der Praxis:

### Was deine Daten schützt {#cryptography}

Deine Dateien werden mit einem modernen Verschlüsselungstool gesperrt ([age](https://github.com/FiloSottile/age)) — breit geprüft, keine bekannten Schwächen.

Der Schlüssel ist 256 Bit lang, erzeugt vom Zufallsgenerator deines Betriebssystems. Zur Einordnung: Ihn zu erraten würde länger dauern, als das Universum existiert.

Selbst wenn jemand alle möglichen Passwörter durchprobieren würde, macht scrypt jeden Versuch absichtlich langsam — millionenfach langsamer als ein direkter Versuch.

Der Schlüssel wird dann mit Shamir's Secret Sharing aufgeteilt. **Weniger als die Schwelle an Teilen enthalten null Information über den Schlüssel.** Nicht „sehr wenig." Mathematisch null.

Jedes Paket listet außerdem Prüfsummen seiner Dateien auf, damit `rememory verify-bundle` dir sagen kann, ob unterwegs etwas beschädigt wurde. Prüfsummen erkennen Versehen, keine absichtlichen Änderungen: Wer eine Datei verändert, kann auch ihre Prüfsumme anpassen. Das verschlüsselte Archiv ist für sich geschützt — wird es verändert, lässt es sich nicht mehr öffnen.

### Was schiefgehen könnte {#what-could-go-wrong}

<div class="bundle-contents">
<div class="file">
<span class="file-name">Ein Freund verliert sein Paket</span>
<span class="file-desc">Kein Problem, solange genügend andere ihres noch haben. Deshalb setzt du die Schwelle unter die Gesamtzahl.</span>
</div>
<div class="file">
<span class="file-name">Ein Freund macht seinen Teil öffentlich</span>
<span class="file-desc">Ein einzelner Teil ist ohne die anderen nutzlos. Jemand bräuchte noch Schwelle minus eins weitere Teile.</span>
</div>
<div class="file">
<span class="file-name">Einige Freunde sind nicht erreichbar</span>
<span class="file-desc">Deshalb setzt du die Schwelle unter die Gesamtzahl. Bei 3 von 5 reichen irgendwelche drei.</span>
</div>
<div class="file">
<span class="file-name">ReMemory gibt es in 10 Jahren nicht mehr</span>
<span class="file-desc"><code>recover.html</code> funktioniert weiterhin — es ist eigenständig. Keine Server, keine Downloads, keine Abhängigkeit von diesem Projekt.</span>
</div>
<div class="file">
<span class="file-name">Browser ändern sich grundlegend</span>
<span class="file-desc">Das Wiederherstellungstool nutzt Standard-JavaScript und die Web Crypto API — Grundlagen des Browsers, keine Trends.</span>
</div>
<div class="file">
<span class="file-name">Du vergisst, wie das funktioniert</span>
<span class="file-desc">Die LIESMICH.txt in jedem Paket erklärt alles. Deine Freunde müssen sich nichts merken — alles ist für sie aufgeschrieben.</span>
</div>
</div>

Was *tatsächlich* stimmen muss: Dein Gerät ist vertrauenswürdig, wenn du die Pakete erstellst, und der Browser, mit dem wiederhergestellt wird, ist nicht kompromittiert. Das sind dieselben Annahmen, die du jedes Mal triffst, wenn du einen Computer für etwas Wichtiges benutzt.

Für eine detaillierte technische Bewertung siehe die [Sicherheits-Selbstbewertung]({{GITHUB_REPO}}/blob/main/docs/security-review.md).

## Vergleich {#comparison}

ReMemory ist nicht das erste Tool, das Shamir's Secret Sharing verwendet. Es gibt viele andere, von Kommandozeilen-Werkzeugen bis zu Web-Apps. Das unterscheidet ReMemory:

- **Verarbeitet Dateien, nicht nur Text.** Die meisten Shamir-Tools teilen nur Passwörter oder kurzen Text. ReMemory verschlüsselt ganze Dateien und Ordner.
- **Eigenständiges Wiederherstellungstool.** Jeder Freund bekommt `recover.html` — ein vollständiges Tool, das in jedem Browser läuft, offline.\* Keine Installation, keine Kommandozeile nötig.
- **Kontaktdaten inklusive.** Jedes Paket enthält eine Liste der anderen Freunde und wie man sie erreicht, damit die Koordination nicht davon abhängt, ob du selbst erreichbar bist.
- **Keine Serverabhängigkeit.** Alles läuft lokal. Kein Dienst zum Anmelden, kein Konto zum Pflegen, nichts das online bleiben muss.

Einen detaillierten Vergleich mit anderen Tools findest du in der [Vergleichstabelle auf GitHub]({{GITHUB_REPO}}#other-similar-tools).

## Alternative: Kommandozeile {#cli}

Es gibt auch ein Kommandozeilen-Tool für alle, die die Terminal bevorzugen oder die Paketerstellung automatisieren wollen.

<a href="{{GITHUB_REPO}}/blob/main/docs/guide.md" class="btn btn-secondary">CLI-Anleitung lesen</a>

<p style="margin-top: 1rem;">
Das CLI bietet dieselbe Funktionalität, plus Stapelverarbeitung und Skripting.
</p>

## Erweitert: Anonymer Modus {#anonymous}

Wenn die Teilnehmer die Identität der anderen nicht kennen sollen, verwende den **anonymen Modus**:

- Personen werden als „Teil 1", „Teil 2" usw. bezeichnet
- Es werden keine Kontaktdaten erfasst oder gespeichert
- Die READMEs lassen den Abschnitt „Andere Teilnehmer" weg
- Pakete werden mit Nummern statt Namen benannt

### Wann den anonymen Modus verwenden

Nützlich wenn:

- Die Teilnehmer nicht wissen sollen, wer die anderen sind
- Du schnell etwas testen willst, ohne Namen einzugeben
- Du die Wiederherstellung auf andere Weise koordinierst
- Privatsphäre wichtiger ist als einfache Koordination

### So aktivierst du ihn

Aktiviere im [Paket-Ersteller](maker.html) den Schalter **Anonym** im Bereich Freunde:

- Die Freundesliste wird durch eine Teilanzahl ersetzt
- Lege die Anzahl der Teile und die Schwelle fest
- Die Pakete heißen `bundle-share-1.zip`, `bundle-share-2.zip` usw.

### Wiederherstellung im anonymen Modus

Die Wiederherstellung funktioniert genauso, nur ohne die Kontaktliste. Teilnehmer sehen Bezeichnungen wie „Teil 1" statt Namen.

<div class="warning">
<strong>Wichtig:</strong> Ohne eingebaute Kontaktliste musst du sicherstellen, dass die Teilnehmer wissen, wie sie einander erreichen, wenn die Wiederherstellung nötig wird.
</div>

## Erweitert: Mehrsprachige Pakete {#multilingual}

Jede Person kann ihr Paket in ihrer bevorzugten Sprache erhalten. Es ist in sieben Sprachen verfügbar: Englisch, Spanisch, Deutsch, Französisch, Slowenisch, Portugiesisch und Chinesisch (Taiwan).

### So funktioniert es

- Jeder Freundeintrag hat ein Dropdown-Menü für die **Paket-Sprache**
- „Standard" verwendet die aktuelle Sprache der Oberfläche
- Pro Person anpassbar, um Sprachen zu mischen
- recover.html öffnet sich in der gewählten Sprache
- Jeder kann jederzeit die Sprache wechseln

<figure class="screenshot">
<img src="screenshots/de/multilingual-language-dropdown.png" alt="Freundeintrag mit dem Sprach-Dropdown für das Paket">
<figcaption>Jeder Freund hat ein Dropdown-Menü, um die Sprache seines Pakets festzulegen</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/demo-pdf-es/page-1.png" alt="recover.pdf auf Spanisch geöffnet">
<figcaption>Das Wiederherstellungstool öffnet sich in der gewählten Sprache des Freundes</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/demo-pdf-es/page-2.png" alt="recover.pdf auf Spanisch geöffnet">
<figcaption>Die Wortlisten sind ebenfalls übersetzt (beide Sprachen funktionieren)</figcaption>
</figure>

## Erweitert: Zeitverzögerte Wiederherstellung {#timelock}

Du kannst beim Erstellen der Pakete eine Wartezeit festlegen. Selbst wenn deine Freunde ihre Teile vorzeitig zusammenbringen, bleiben die Dateien bis zum gewählten Datum gesperrt — 30 Tage, 6 Monate, ein bestimmtes Datum.

### So aktivierst du es

Wechsle im [Paket-Ersteller](maker.html) in den Modus **Erweitert** und aktiviere **Zeitsperre hinzufügen**. Wähle, wie lange die Dateien gesperrt bleiben sollen — bis zu zwei Jahre.

Das Limit von zwei Jahren ist bewusst gewählt. Zeitsperren hängen davon ab, dass die League of Entropy weiter betrieben wird, und wir halten es nicht für verantwortbar, deine Dateien mit einer längeren Wette auf externe Infrastruktur zu sperren. Wenn du eine längere Dauer brauchst und den Kompromiss verstehst, hat die CLI kein Limit: `rememory seal --timelock 5y`.

<figure class="screenshot">
<img src="screenshots/de/tlock-setup.png" alt="Zeitsperre-Einrichtung im erweiterten Modus">
<figcaption>Die Zeitsperre-Option erscheint im erweiterten Modus</figcaption>
</figure>

### Wiederherstellung

Wenn jemand ein Paket mit Zeitsperre vor dem Datum öffnet, zeigt das Wiederherstellungstool einen Hinweis zum Warten. Sobald die Zeit abgelaufen ist, geht die Wiederherstellung normal weiter.

<figure class="screenshot">
<img src="screenshots/de/tlock-waiting.png" alt="Wiederherstellungstool zeigt einen Zeitsperre-Hinweis">
<figcaption>Das Wiederherstellungstool wartet, bis die Zeitsperre abgelaufen ist</figcaption>
</figure>

Das Öffnen eines Archivs mit Zeitsperre erfordert eine kurze Internetverbindung. Deine Dateien werden nirgendwohin gesendet — die Verbindung prüft nur, ob genügend Zeit vergangen ist. Ohne Zeitsperre funktioniert die Wiederherstellung komplett offline.

<div class="warning">
<strong>Experimentell.</strong> Die zeitverzögerte Wiederherstellung hängt von der <a href="https://www.cloudflare.com/en-ca/leagueofentropy/" target="_blank">League of Entropy</a> ab, einem verteilten Netzwerk, das von seriösen Organisationen weltweit betrieben wird. Wenn dieses Netzwerk den Betrieb einstellt, bevor eine Zeitsperre abläuft, wird das betreffende Archiv unwiederbringlich. Pakete ohne Zeitsperre sind davon nicht betroffen.
</div>

### Wie es funktioniert {#timelock-technical}

Die League of Entropy erzeugt alle 3 Sekunden einen neuen kryptografischen Wert. Jeder Wert ist nummeriert. Man kann vorhersagen, welche Nummer einem bestimmten Zeitpunkt entspricht, aber der Wert für diese Nummer kann nicht vorzeitig erzeugt werden — von niemandem, auch nicht von den Betreibern des Netzwerks.

Wenn du ein Paket mit Zeitsperre erstellst, wird das Archiv mit einem bestimmten zukünftigen Wert verschlüsselt. Der Schlüssel zum Öffnen existiert noch nicht. Er kommt vom Netzwerk, wenn der Moment gekommen ist.

Mehr über die Kryptografie dahinter findest du in der [drand Timelock-Encryption-Dokumentation](https://docs.drand.love/docs/timelock-encryption/).
