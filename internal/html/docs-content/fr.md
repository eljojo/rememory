---
title: "\U0001F9E0 Guide ReMemory"
subtitle: "Comment créer des enveloppes et récupérer des fichiers"
cli_guide_note: 'Il existe aussi un <a href="{{GITHUB_REPO}}/blob/main/docs/guide.md">guide en ligne de commande</a>.'
nav_home: "\U0001F9E0 ReMemory"
nav_home_link: "Accueil"
nav_create: "Créer des enveloppes"
nav_recover: "Récupérer"
toc_title: "Sommaire"
footer_source: "Code source"
footer_download: "Télécharger le CLI"
footer_home: "Accueil"
---

## Présentation {#overview}

ReMemory est un coffre-fort numérique à plusieurs clés. Vous choisissez qui les détient et combien doivent se réunir pour l'ouvrir.

Sous le capot, il chiffre vos fichiers avec [age](https://github.com/FiloSottile/age), répartit la clé entre des personnes de confiance et donne à chacune une enveloppe autonome pour la récupération.

Récupérer, c'est ouvrir un fichier dans un navigateur.\* Pas de serveur, pas besoin que ce site existe.

Si vous gardez une part pour quelqu'un, commencez par [Récupérer des fichiers](#recovering).

<p style="font-size: 0.8125rem; color: #8A8480;">* Les archives avec <a href="#timelock" style="color: #8A8480;">verrouillage temporel</a> nécessitent une brève connexion internet au moment de la récupération.</p>

<div class="tip">
<strong>À noter :</strong> Aucune personne seule ne peut accéder à vos données. Il faut réunir suffisamment de parts — par exemple, 3 sur 5.
</div>

## Pourquoi ReMemory {#why-rememory}

Vous avez probablement des secrets numériques qui comptent : codes de récupération de votre gestionnaire de mots de passe, clés de cryptomonnaie, documents importants, instructions pour vos proches. Que deviennent-ils si un jour vous n'êtes plus disponible ?

Imaginez un coffre qui a besoin de deux clés pour s'ouvrir — aucune personne seule n'a de quoi y accéder.

Les approches classiques ont leurs faiblesses :

- **Tout confier à une personne** — un seul point de défaillance et de confiance
- **Diviser les fichiers manuellement** — confus, sujet aux erreurs, sans chiffrement
- **Utiliser l'accès d'urgence d'un gestionnaire de mots de passe** — revient à « tout confier à une personne », et dépend en plus de la pérennité de l'entreprise
- **L'inscrire dans un testament** — devient un document public, procédure juridique lente

ReMemory prend un chemin différent :

- **Pas de point de défaillance unique** — plusieurs personnes doivent coopérer
- **Pas de confiance aveugle en une seule personne** — même votre ami le plus proche ne peut pas accéder seul à vos secrets
- **Hors ligne et autonome** — la récupération fonctionne sans internet ni serveur\*
- **Conçu pour tout le monde** — des instructions claires, pas des énigmes cryptographiques

## Créer des enveloppes {#creating}

Trois étapes. Tout se passe dans votre navigateur — vos fichiers ne quittent jamais votre appareil. Ouvrez le [créateur d'enveloppes](maker.html) pour commencer.

### Étape 1 : Ajouter des amis {#step1}

Ajoutez les personnes qui garderont une part de votre clé de récupération. Pour chacune, indiquez un nom et éventuellement des coordonnées.

<figure class="screenshot">
<img src="screenshots/fr/friends.png" alt="Ajout d'amis à l'étape 1">
<figcaption>Chaque personne ici gardera une part de la clé</figcaption>
</figure>

Choisissez ensuite combien d'entre elles doivent se réunir pour ouvrir vos fichiers.

<div class="tip">
<strong>Combien en choisir :</strong>
<ul>
<li><strong>2 personnes sur 3 :</strong> La configuration la plus simple</li>
<li><strong>3 personnes sur 5 :</strong> Un bon équilibre</li>
<li><strong>4 ou 5 personnes sur 7 :</strong> Plus sûr, plus de coordination</li>
</ul>
Assez élevé pour que quelques personnes ne puissent pas ouvrir vos fichiers sans les autres. Assez bas pour que cela fonctionne encore si une ou deux sont injoignables.
</div>

### Étape 2 : Ajouter des fichiers {#step2}

Glissez-déposez les fichiers ou le dossier que vous souhaitez protéger.

<figure class="screenshot">
<img src="screenshots/fr/files.png" alt="Ajout de fichiers à l'étape 2">
<figcaption>Ajoutez les fichiers que vous souhaitez protéger</figcaption>
</figure>

**Bons candidats :**

- Codes de récupération du gestionnaire de mots de passe
- Clés et seeds de cryptomonnaie
- Identifiants de comptes importants
- Instructions pour vos proches
- Emplacements de documents juridiques
- Combinaisons de coffre-fort

<div class="warning">
<strong>Note :</strong> Évitez les fichiers qui changent souvent. Cet outil est conçu pour des secrets que vous configurez une fois, puis laissez de côté.
</div>

### Étape 3 : Générer les enveloppes {#step3}

Cliquez sur « Créer les enveloppes » pour verrouiller vos fichiers et préparer une enveloppe pour chaque personne.

<figure class="screenshot">
<img src="screenshots/fr/bundles.png" alt="Génération des enveloppes à l'étape 3">
<figcaption>Téléchargez chaque enveloppe séparément, ou toutes à la fois</figcaption>
</figure>

Chaque enveloppe contient l'outil de récupération complet. Il fonctionne même si ce site n'existe plus.

### Distribuer aux amis {#distributing}

Envoyez à chaque personne son enveloppe comme vous le souhaitez :

- **E-mail :** Le fichier ZIP en pièce jointe
- **Stockage en ligne :** Partagez via Dropbox, Google Drive, etc.
- **Clé USB :** Remise en main propre
- **Messagerie chiffrée :** Signal, WhatsApp, etc.

Si vous envoyez les enveloppes par e-mail ou messagerie, supprimez les messages de votre dossier d'envoi une fois la réception confirmée par chaque personne. Sinon, votre compte finit par garder une copie de chaque part — ce qui annule l'intérêt de les avoir séparées.

### Après la création {#after-creating}

Une fois vos enveloppes prêtes, quelques choses valent la peine d'être faites avant de passer à autre chose :

- Demandez à chaque personne d'ouvrir `recover.html` une fois. La page affiche « Votre part est là » quand elle peut lire leur part.
- Expliquez à chacun ce que c'est, pourquoi il l'a, et qu'il doit le garder en lieu sûr. Seul, il ne peut rien en faire — il devra se coordonner avec d'autres.
- Enregistrez les fichiers verrouillés avec « Enregistrer les fichiers verrouillés (MANIFEST.age) » et gardez cette copie en lieu sûr. Elle ne contient aucune part : sans assez de parts, elle ne sert à personne.
- Sauvegardez votre `project.yml` pour pouvoir recréer les enveloppes plus tard
- Imprimez `LISEZMOI.pdf` comme sauvegarde papier avant d'envoyer l'enveloppe numérique. Le papier n'a besoin ni d'adaptateur, ni d'électricité.
- Programmez un rappel annuel — voir [Garder les enveloppes à jour](#keeping-current)
- Envisagez un [exercice de récupération](#recovery-drills) — c'est le seul moyen de savoir si votre plan fonctionne vraiment

## Récupérer des fichiers {#recovering}

Cette section s'adresse aux amis qui gardent une part. La plupart du temps, un ami n'a rien d'autre à faire que garder son enveloppe en lieu sûr. Un jour, quelqu'un lui demandera peut-être sa part, ou ce sera peut-être lui qui rassemblera les parts.

Si vous êtes ici parce qu'une personne qui vous est chère ne peut plus être jointe, rien ne presse. Rien ici n'expire. Prenez tout le temps qu'il vous faut — si vous fermez la page de récupération en cours de route, vous ajouterez à nouveau les parts à votre retour.

### Ce que les amis reçoivent {#bundle-contents}

Chaque enveloppe contient :

<div class="bundle-contents">
<div class="file">
<span class="file-name">LISEZMOI.txt</span>
<span class="file-desc">Ce dont il s'agit, qui d'autre détient une part, et votre part elle-même</span>
</div>
<div class="file">
<span class="file-name">LISEZMOI.pdf</span>
<span class="file-desc">La même chose, prête à imprimer. Votre part y figure sous forme de <strong>QR code</strong> et de mots.</span>
</div>
<div class="file">
<span class="file-name">MANIFEST.age</span>
<span class="file-desc">Les fichiers verrouillés. Seulement dans certaines enveloppes — la plupart les contiennent plutôt dans recover.html.</span>
</div>
<div class="file">
<span class="file-name">recover.html</span>
<span class="file-desc">La page de récupération. Elle s'ouvre dans n'importe quel navigateur, avec votre part déjà dedans.</span>
</div>
</div>

<p style="margin-top: 1rem;">
Chaque enveloppe est préparée pour un ami : sa part se trouve déjà dans son recover.html, avec les noms des autres et la façon de les joindre.
</p>

### Garder votre enveloppe {#keeping-bundle}

Gardez-la en lieu sûr, ainsi qu'une copie imprimée de LISEZMOI.pdf si vous le pouvez. Il n'y a rien d'autre à faire en attendant le moment venu.

Pour la vérifier, ouvrez recover.html. La page affiche « Votre part est là », la part que vous détenez, combien de parts il faut pour ouvrir les fichiers, et si les fichiers verrouillés sont là aussi.

### Si quelqu'un vous demande votre part {#sharing-piece}

Vérifiez d'abord que la demande est réelle. Si vous le pouvez, renseignez-vous auprès de la personne qui a préparé l'enveloppe. Si vous ne pouvez pas la joindre, renseignez-vous auprès d'un autre ami qui détient une part.

Envoyez-la ensuite de la manière la plus simple pour vous :

- **L'enveloppe entière** (le fichier ZIP) — le plus simple.
- **Votre LISEZMOI.txt** — il contient votre part à lui seul.
- **Vos mots de récupération**, lus à voix haute au téléphone.
- **Votre page imprimée**, par la poste — le QR code qui y figure contient votre part.

### Rassembler les parts {#recovery-bundle}

Une seule personne rassemble les parts. Ce peut être n'importe lequel des amis — celui qui s'en charge.

<div class="step-guide">
<div class="step-number">1</div>
<div class="step-content">
<h4>Ouvrir recover.html</h4>
<p>Double-cliquez sur l'enveloppe pour l'ouvrir, puis double-cliquez sur recover.html. Le fichier s'ouvre dans votre navigateur, comme une page web, avec votre propre part déjà dedans. Si vous n'avez pas d'enveloppe, n'importe quel ami peut vous envoyer son recover.html, ou vous pouvez utiliser la <a href="recover.html">page de récupération</a> de ce site.</p>
</div>
</div>

<div class="step-guide">
<div class="step-number">2</div>
<div class="step-content">
<h4>Contacter les autres</h4>
<p>Si vous avez une enveloppe, son recover.html et son LISEZMOI indiquent qui d'autre détient une part et comment joindre ces personnes — sauf si les enveloppes ont été créées sans noms. Dites-leur ce qui s'est passé et demandez-leur leur part. Certaines voudront d'abord vérifier auprès de vous que la demande est réelle — leurs instructions le leur demandent.</p>
</div>
</div>

<div class="step-guide">
<div class="step-number">3</div>
<div class="step-content">
<h4>Ajouter chaque part à son arrivée</h4>
<p>Glissez leur enveloppe ou leur LISEZMOI.txt sur la page, collez leur texte, saisissez les mots de récupération qu'ils vous lisent, ou scannez le QR code de leur page imprimée. Si des noms sont listés, une coche apparaît à côté de chacun.</p>
</div>
</div>

<div class="step-guide">
<div class="step-number">4</div>
<div class="step-content">
<h4>Les fichiers se déverrouillent d'eux-mêmes</h4>
<p>Dès qu'il y a assez de parts, la page ouvre les fichiers sans que vous ayez quoi que ce soit à faire. Téléchargez-les et conservez-les dans un endroit privé.</p>
</div>
</div>

<div class="tip">
<strong>Si la page demande les fichiers verrouillés :</strong> glissez-y <code>MANIFEST.age</code> depuis l'enveloppe d'un ami — ou, si son enveloppe n'en contient pas, son recover.html. La plupart des enveloppes les contiennent déjà, vous ne verrez donc probablement pas cette étape.
</div>

<figure class="screenshot">
<img src="screenshots/fr/recovery-1.png" alt="Page de récupération avec les parts rassemblées jusqu'ici">
<figcaption>Les parts rassemblées jusqu'ici, et qui contacter</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/fr/recovery-2.png" alt="Page de récupération une fois les fichiers déverrouillés">
<figcaption>Dès qu'il y a assez de parts, les fichiers sont déverrouillés et prêts à être téléchargés</figcaption>
</figure>

### Saisir les mots de récupération {#recovery-words}

Chaque page imprimée contient la part sous forme d'une liste de mots numérotés. Si un ami vous lit les siens au téléphone, ou si vous n'avez que le papier, saisissez les mots dans la page de récupération. Appuyez sur « Coller une part ou saisir les mots de récupération » et saisissez-les, séparés par des espaces. Seuls les mots comptent, pas les numéros.

<figure class="screenshot">
<img src="screenshots/fr/recovery-words-typing.png" alt="Saisie des mots d'une page imprimée dans la page de récupération">
<figcaption>Saisissez les mots de la page imprimée</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/fr/recovery-words-recognized.png" alt="La page de récupération après la saisie des mots, montrant que la part a été ajoutée">
<figcaption>La page reconnaît les mots et ajoute la part</figcaption>
</figure>

### Scanner un QR code {#recovery-pdf}

Sur un ordinateur ou un téléphone équipé d'une caméra, appuyez sur « Scanner QR » dans la page de récupération et tenez la page imprimée face à la caméra.

<figure class="screenshot">
<img src="screenshots/qr-camera-permission.png" alt="Le navigateur demande l'autorisation d'utiliser la caméra">
<figcaption>Votre navigateur vous demandera l'accès à la caméra</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/qr-scanning.png" alt="Scan d'un QR code depuis une page imprimée">
<figcaption>Tenez le QR code face à la caméra pour ajouter la part</figcaption>
</figure>

Scanner le QR code avec l'appareil photo d'un téléphone fonctionne aussi : cela ouvre la page de récupération de ce site, avec la part déjà remplie.

<div class="tip">
<strong>À propos de la récupération :</strong>
<ul>
<li>Fonctionne sans <span title="Les archives avec verrouillage temporel nécessitent une connexion pour vérifier la date de déverrouillage.">connexion internet*</span></li>
<li>Les parts et les fichiers restent sur votre appareil</li>
<li>Les amis peuvent être n'importe où — chacun n'a qu'à vous faire parvenir sa part</li>
</ul>
</div>

## Bonnes pratiques {#best-practices}

### Choisir ses amis

- **Durabilité :** Des personnes que vous pourrez joindre dans 5 à 10 ans
- **Répartition géographique :** Pas tous au même endroit
- **Compétences techniques :** Inutile d'être à l'aise avec la technique. Les instructions guident chacun pas à pas, et un [exercice de récupération](#recovery-drills) vous montrera où quelqu'un bloque.
- **Relations :** Accepteront-ils de coopérer entre eux ?
- **Confiance :** Une part seule ne révèle rien, mais vous leur confiez une responsabilité

### Considérations de sécurité

- Ne gardez pas toutes les enveloppes ensemble — cela irait à l'encontre du principe de répartition
- Pensez à imprimer `LISEZMOI.pdf` — le papier résiste aux catastrophes numériques
- Sauvegardez `project.yml` pour pouvoir recréer les enveloppes plus tard

### Stocker les enveloppes en sécurité {#storing-bundles}

Les enveloppes sont petites (moins de 10 Mo) et conçues pour être conservées dans des endroits ordinaires. Voici ce qui fonctionne bien :

- **L'e-mail** fonctionne bien. La plupart des gens gardent la même adresse pendant des décennies, et les enveloppes sont assez légères pour être jointes en pièce attachée. Beaucoup de fournisseurs conservent les messages indéfiniment.
- **Le stockage en ligne** (Google Drive, Dropbox, iCloud) convient bien comme copie secondaire.
- **Les clés USB** peuvent convenir, mais gardez à l'esprit que les connecteurs changent (l'USB-A cède déjà la place à l'USB-C) et que la mémoire flash peut se dégrader après des années sans alimentation. Pas idéal comme seule copie.
- **Le papier** est l'option la plus durable. Imprimer `LISEZMOI.pdf` offre à vos amis une copie de leur part qui n'a besoin ni d'adaptateur, ni d'électricité, ni d'appareil fonctionnel.

Le papier contient les parts, mais pas les fichiers verrouillés eux-mêmes — ceux-ci n'existent que dans des copies numériques. La meilleure approche est donc la redondance — e-mail et papier, ou cloud et papier. Plus d'une copie, sous plus d'une forme.

### Garder les enveloppes à jour {#keeping-current}

Programmez un rappel annuel pour prendre des nouvelles de vos amis. Confirmez qu'ils ont toujours leurs enveloppes et mettez à jour les coordonnées si quelque chose a changé.

Quand vos fichiers changent, créez de nouvelles enveloppes et envoyez-les. Les anciennes enveloppes n'ouvriront pas la nouvelle archive, donc aucun risque à ce qu'elles traînent — mais demandez à vos amis de remplacer les leurs par les nouvelles.

Quand les contacts changent — quelqu'un déménage, change de numéro, ou vous souhaitez ajouter ou retirer quelqu'un — même chose : nouvelles enveloppes, demandez de supprimer les anciennes.

Entre les mises à jour, conservez vos fichiers sources dans un coffre chiffré — des outils comme [Cryptomator](https://cryptomator.org) ou [VeraCrypt](https://veracrypt.fr) font l'affaire. Ne laissez pas de copies en clair dans un dossier ordinaire.

Pensez-y comme la mise à jour de vos contacts d'urgence. Bref, régulier, utile.

### Exercice de récupération {#recovery-drills}

Personne ne teste son plan de secours. Un exercice de récupération change ça — et ça se joue un peu comme un escape room.

Créez une enveloppe de test avec un secret anodin (une photo, un message, une recette). Distribuez les parts à vos amis. Ajoutez un [verrouillage temporel](#timelock) si vous voulez un vrai compte à rebours. Puis prenez du recul et laissez-les se débrouiller : se coordonner, rassembler les parts, déverrouiller les fichiers — sans votre aide.

Ce que vous apprendrez :

- Vos amis retrouvent-ils leurs enveloppes quand ils en ont besoin ?
- Comprennent-ils les instructions, ou restent-ils bloqués ?
- Peuvent-ils se contacter entre eux sans que vous coordonniez ?

Un exercice qui se passe bien signifie que votre vrai plan fonctionnera aussi. Un exercice qui se passe mal est un cadeau — vous avez trouvé le problème avant qu'il ne compte.

### Révoquer l'accès {#revoking-access}

Une fois qu'une part a été distribuée, elle ne peut pas être révoquée. C'est voulu — il n'y a pas de serveur, pas d'autorité centrale.

Si vous devez changer qui détient des parts :

1. **Créez de nouvelles enveloppes** avec un nouveau groupe d'amis et une nouvelle clé
1. **Envoyez les nouvelles enveloppes** aux amis en qui vous avez toujours confiance
1. **Demandez à chaque ami de supprimer son ancienne enveloppe** et de la remplacer par la nouvelle

<div class="warning">
<strong>Important :</strong> Les anciennes parts fonctionnent toujours avec les anciennes archives. Quand vous envoyez une nouvelle enveloppe, soyez clair : <strong>supprimer l'ancienne</strong>, ne garder que la nouvelle. Pas d'historique de versions, pas de « au cas où. »
</div>

La même chose s'applique quand les secrets changent. De nouvelles enveloppes signifient une nouvelle clé et de nouvelles parts. Les anciennes parts n'ouvriront pas la nouvelle archive, mais elles fonctionnent toujours avec l'ancienne. Assurez-vous que vos amis ne conservent pas d'anciennes copies.

### À propos de project.yml {#project-file}

Quand vous créez des enveloppes, votre projet est sauvegardé dans un fichier `project.yml`. Ce fichier contient :

- Les noms et coordonnées des amis
- Le seuil choisi (par exemple 3 sur 5)
- Un hash de vérification pour confirmer que les enveloppes correspondent
- Des sommes de contrôle des parts pour vérifier l'intégrité des enveloppes

Il ne contient **aucun** secret — pas de mot de passe, pas de matériel cryptographique, pas de contenu de fichier. Vous pouvez le garder sans risque avec vos autres documents.

Avec `project.yml`, vous pouvez recréer les enveloppes, vérifier celles qui existent, et consulter l'état de votre configuration.

## Comprendre la sécurité {#security}

ReMemory combine des outils cryptographiques éprouvés au lieu d'inventer les siens. Voici ce que cela signifie concrètement.

### Ce qui protège vos données {#cryptography}

Vos fichiers sont verrouillés avec un outil de chiffrement moderne ([age](https://github.com/FiloSottile/age)) — largement audité, sans faille connue.

La clé qui les verrouille fait 256 bits, générée par le générateur de nombres aléatoires de votre système d'exploitation. Pour donner un ordre de grandeur : la deviner prendrait plus de temps que l'univers n'a existé.

Même si quelqu'un essayait tous les mots de passe possibles, scrypt rend chaque tentative volontairement lente — des millions de fois plus lente qu'une tentative directe.

La clé est ensuite divisée avec le partage de secret de Shamir. **Toute quantité de parts inférieure au seuil contient zéro information sur la clé.** Pas « très peu. » Mathématiquement zéro.

Chaque enveloppe liste aussi les sommes de contrôle de ses fichiers, pour que `rememory verify-bundle` puisse vous dire si l'un d'eux a été abîmé en chemin. Les sommes de contrôle détectent les accidents, pas les modifications délibérées : quiconque modifie un fichier peut aussi mettre à jour sa somme. L'archive chiffrée est protégée à part — si elle est modifiée, elle ne s'ouvrira pas.

### Ce qui pourrait mal tourner {#what-could-go-wrong}

<div class="bundle-contents">
<div class="file">
<span class="file-name">Un ami perd son enveloppe</span>
<span class="file-desc">Pas de problème, tant que suffisamment d'autres gardent la leur. C'est pour cela que vous fixez le seuil en dessous du total.</span>
</div>
<div class="file">
<span class="file-name">Un ami rend sa part publique</span>
<span class="file-desc">Une part seule est inutile sans les autres. Il faudrait encore le seuil moins une part supplémentaire pour faire quoi que ce soit.</span>
</div>
<div class="file">
<span class="file-name">Certains amis sont injoignables</span>
<span class="file-desc">C'est pour cela que le seuil est inférieur au nombre total d'amis. Si vous avez choisi 3 sur 5, n'importe lesquels trois suffisent.</span>
</div>
<div class="file">
<span class="file-name">ReMemory disparaît dans 10 ans</span>
<span class="file-desc"><code>recover.html</code> fonctionne toujours — il est autonome. Pas de serveur, pas de téléchargement, aucune dépendance à ce projet.</span>
</div>
<div class="file">
<span class="file-name">Les navigateurs changent radicalement</span>
<span class="file-desc">L'outil de récupération utilise du JavaScript standard et la Web Crypto API — des fondamentaux du navigateur, pas des modes.</span>
</div>
<div class="file">
<span class="file-name">Vous oubliez comment tout cela fonctionne</span>
<span class="file-desc">Le LISEZMOI.txt de chaque enveloppe explique tout. Vos amis n'ont rien à retenir — tout est écrit pour eux.</span>
</div>
</div>

Ce qui *doit* être vrai : votre appareil est fiable au moment de créer les enveloppes, et le navigateur utilisé pour la récupération n'est pas compromis. Ce sont les mêmes hypothèses que vous faites chaque fois que vous utilisez un ordinateur pour quelque chose d'important.

Pour une évaluation technique détaillée, consultez l'[auto-évaluation de sécurité]({{GITHUB_REPO}}/blob/main/docs/security-review.md).

## Comparaison {#comparison}

ReMemory n'est pas le premier outil à utiliser le partage de secret de Shamir. Il en existe beaucoup d'autres, des outils en ligne de commande aux applications web. Voici ce qui distingue ReMemory :

- **Gère des fichiers, pas seulement du texte.** La plupart des outils Shamir ne divisent que des mots de passe ou du texte court. ReMemory chiffre des fichiers et des dossiers entiers.
- **Outil de récupération autonome.** Chaque ami reçoit `recover.html` — un outil complet qui fonctionne dans n'importe quel navigateur, hors ligne.\* Pas d'installation, pas de ligne de commande.
- **Coordonnées incluses.** Chaque enveloppe contient une liste des autres amis et comment les joindre, pour que la coordination ne dépende pas de votre disponibilité.
- **Aucune dépendance à un serveur.** Tout s'exécute localement. Pas de service auquel s'inscrire, pas de compte à maintenir, rien qui doive rester en ligne.

Pour une comparaison détaillée avec d'autres outils, consultez le [tableau comparatif sur GitHub]({{GITHUB_REPO}}#other-similar-tools).

## Alternative en ligne de commande {#cli}

Il existe aussi un outil en ligne de commande pour ceux qui préfèrent le terminal ou qui veulent automatiser la création d'enveloppes.

<a href="{{GITHUB_REPO}}/blob/main/docs/guide.md" class="btn btn-secondary">Lire le guide du CLI</a>

<p style="margin-top: 1rem;">
Le CLI offre les mêmes fonctionnalités, plus le traitement par lots et le scripting.
</p>

## Avancé : Mode anonyme {#anonymous}

Quand les participants ne doivent pas connaître l'identité des autres, utilisez le **mode anonyme** :

- Les personnes sont identifiées comme « Part 1 », « Part 2 », etc.
- Aucune coordonnée n'est collectée ni stockée
- Les README omettent la section « Autres détenteurs »
- Les noms des enveloppes utilisent des numéros au lieu de noms

### Quand utiliser le mode anonyme

C'est utile quand :

- Les participants ne doivent pas savoir qui sont les autres
- Vous faites un test rapide sans saisir de noms
- Vous avez un autre moyen de coordonner la récupération
- La confidentialité prime sur la facilité de coordination

### Comment l'activer

Dans le [créateur d'enveloppes](maker.html), activez le bouton **Anonyme** dans la section Amis :

- La liste d'amis est remplacée par un nombre de parts
- Définissez le nombre de parts et le seuil
- Les enveloppes sont nommées `bundle-share-1.zip`, `bundle-share-2.zip`, etc.

### Récupération en mode anonyme

La récupération fonctionne de la même manière, mais sans la liste de contacts. Les participants voient des étiquettes génériques comme « Part 1 » au lieu de noms.

<div class="warning">
<strong>Important :</strong> Sans liste de contacts intégrée, assurez-vous que les participants savent comment se joindre quand la récupération sera nécessaire.
</div>

## Avancé : Enveloppes multilingues {#multilingual}

Chaque personne peut recevoir son enveloppe dans la langue de son choix. L'outil est disponible en sept langues : anglais, espagnol, allemand, français, slovène, portugais et chinois (Taïwan).

### Comment ça fonctionne

- Chaque entrée d'ami a un menu déroulant **Langue du paquet**
- « Par défaut » utilise la langue actuelle de l'interface
- Personnalisable par personne pour mélanger les langues
- recover.html s'ouvre dans la langue choisie
- Chacun peut changer de langue à tout moment

<figure class="screenshot">
<img src="screenshots/fr/multilingual-language-dropdown.png" alt="Entrée d'ami montrant le menu déroulant de langue">
<figcaption>Chaque ami a un menu déroulant pour choisir la langue de son enveloppe</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/demo-pdf-es/page-1.png" alt="recover.pdf ouvert en espagnol">
<figcaption>L'outil de récupération s'ouvre dans la langue choisie pour l'ami</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/demo-pdf-es/page-2.png" alt="recover.pdf ouvert en espagnol">
<figcaption>Les listes de mots sont aussi traduites (les deux langues fonctionnent)</figcaption>
</figure>

## Avancé : Récupération différée {#timelock}

Vous pouvez définir un délai d'attente lors de la création des enveloppes. Même si vos amis réunissent leurs parts en avance, les fichiers restent verrouillés jusqu'à la date choisie — 30 jours, 6 mois, une date précise.

### Comment l'activer

Dans le [créateur d'enveloppes](maker.html), passez en mode **Avancé** et cochez **Ajouter un verrou temporel**. Choisissez combien de temps les fichiers doivent rester verrouillés — jusqu'à deux ans.

La limite de deux ans est délibérée. Les verrous temporels dépendent du fonctionnement continu de la League of Entropy, et nous ne pensons pas qu'il soit responsable de verrouiller vos fichiers sur un pari plus long concernant une infrastructure externe. Si vous avez besoin d'une durée plus longue et comprenez le compromis, la CLI n'a pas de limite : `rememory seal --timelock 5y`.

<figure class="screenshot">
<img src="screenshots/fr/tlock-setup.png" alt="Configuration du verrouillage temporel en mode Avancé">
<figcaption>L'option de verrouillage temporel apparaît en mode Avancé</figcaption>
</figure>

### Récupération

Quand quelqu'un ouvre une enveloppe avec verrouillage temporel avant la date prévue, l'outil de récupération affiche un message d'attente. Une fois le délai écoulé, la récupération reprend normalement.

<figure class="screenshot">
<img src="screenshots/fr/tlock-waiting.png" alt="Outil de récupération affichant un message de verrouillage temporel">
<figcaption>L'outil de récupération attend que le verrouillage temporel expire</figcaption>
</figure>

Ouvrir une archive avec verrouillage temporel nécessite une brève connexion internet. Vos fichiers ne sont envoyés nulle part — la connexion sert à vérifier que suffisamment de temps s'est écoulé. Sans verrouillage temporel, la récupération est entièrement hors ligne.

<div class="warning">
<strong>Expérimental.</strong> La récupération différée dépend de la <a href="https://www.cloudflare.com/en-ca/leagueofentropy/" target="_blank">League of Entropy</a>, un réseau distribué opéré par des organisations sérieuses à travers le monde. Si ce réseau cesse de fonctionner avant l'expiration d'un verrou temporel, l'archive concernée devient irrécupérable. Les enveloppes sans verrou temporel ne sont pas affectées.
</div>

### Comment ça fonctionne {#timelock-technical}

La League of Entropy produit une nouvelle valeur cryptographique toutes les 3 secondes. Chaque valeur est numérotée. On peut prédire quel numéro correspond à un instant donné, mais la valeur de ce numéro ne peut pas être produite en avance — par personne, y compris les opérateurs du réseau.

Quand vous créez une enveloppe avec verrouillage temporel, l'archive est chiffrée avec une valeur future spécifique. La clé pour l'ouvrir n'existe pas encore. Elle viendra du réseau quand le moment sera venu.

Pour en savoir plus sur la cryptographie derrière ce mécanisme, consultez la [documentation de drand sur le timelock encryption](https://docs.drand.love/docs/timelock-encryption/).
