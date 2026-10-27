---
title: "\U0001F9E0 Cómo usar ReMemory"
subtitle: "Cómo crear kits y recuperar archivos"
cli_guide_note: 'También hay una <a href="{{GITHUB_REPO}}/blob/main/docs/guide.md">guía de línea de comandos</a>.'
nav_home: "\U0001F9E0 ReMemory"
nav_home_link: "Inicio"
nav_create: "Crear kits"
nav_recover: "Recuperar"
toc_title: "Contenido"
footer_source: "Código Fuente"
footer_download: "Descargar"
footer_home: "Inicio"
---

## Descripción general {#overview}

ReMemory es una caja fuerte digital con varias llaves. Tú eliges quién las guarda y cuántas personas deben juntarse para abrirla.

Por dentro, cifra tus archivos con [age](https://github.com/FiloSottile/age), divide la clave entre personas de tu confianza y entrega a cada persona un kit autónomo para la recuperación.

Recuperar es abrir un archivo en el navegador.\* Sin servidores, sin necesidad de que este sitio web exista.

Si guardas una parte para alguien, empieza en [Recuperar archivos](#recovering).

<p style="font-size: 0.8125rem; color: #8A8480;">* Los archivos con <a href="#timelock" style="color: #8A8480;">bloqueo por fecha</a> necesitan una breve conexión a internet al momento de la recuperación.</p>

<div class="tip">
<strong>Consejo:</strong> Ninguna persona puede acceder a tus datos por sí sola. Necesitan reunir suficientes partes — por ejemplo, 3 de 5.
</div>

## Por qué ReMemory {#why-rememory}

Probablemente tienes secretos digitales que importan: códigos de recuperación de tu gestor de contraseñas, semillas de criptomonedas, documentos importantes, instrucciones para tus seres queridos. ¿Qué pasa si un día no estás disponible?

Piénsalo como una caja fuerte que necesita dos llaves para abrirse — ninguna persona tiene suficiente para entrar sola.

Los enfoques tradicionales tienen debilidades:

- **Darle todo a una persona** — un único punto de fallo y de confianza
- **Dividir archivos manualmente** — confuso, propenso a errores, sin cifrado
- **Usar el acceso de emergencia de un gestor de contraseñas** — similar a "darle todo a una persona", y además depende de que la empresa siga existiendo
- **Dejarlo en un testamento** — se vuelve registro público, proceso legal lento

ReMemory toma un camino diferente:

- **Sin un único punto de fallo** — requiere la cooperación de varias personas
- **Sin confiar en una sola persona** — ni siquiera tu amigo más cercano puede acceder solo a tus secretos
- **Sin conexión y autónomo** — la recuperación funciona sin internet ni servidores\*
- **Diseñado para cualquiera** — instrucciones claras, no acertijos criptográficos

## Crear kits {#creating}

Tres pasos. Todo ocurre en tu navegador — tus archivos nunca salen de tu dispositivo. Abre el [creador de kits](maker.html) para comenzar.

### Paso 1: Agregar amigos {#step1}

Agrega a las personas que guardarán partes de tu clave de recuperación. Para cada una, indica un nombre y opcionalmente información de contacto.

<figure class="screenshot">
<img src="screenshots/es/friends.png" alt="Agregando amigos en el Paso 1">
<figcaption>Cada persona aquí guardará una parte de la clave</figcaption>
</figure>

Luego elige cuántas de ellas deben reunirse para abrir tus archivos.

<div class="tip">
<strong>Cómo elegir cuántas:</strong>
<ul>
<li><strong>2 de 3 personas:</strong> La configuración más sencilla</li>
<li><strong>3 de 5 personas:</strong> Un buen equilibrio</li>
<li><strong>4 o 5 de 7 personas:</strong> Más seguro, más coordinación</li>
</ul>
Lo suficientemente alto para que unas pocas personas no puedan abrirlo sin las demás. Lo suficientemente bajo para que siga funcionando si no se puede contactar a una o dos.
</div>

### Paso 2: Agregar archivos {#step2}

Arrastra y suelta los archivos o la carpeta que quieres proteger.

<figure class="screenshot">
<img src="screenshots/es/files.png" alt="Agregando archivos en el Paso 2">
<figcaption>Agrega los archivos que quieres proteger</figcaption>
</figure>

**Buenos candidatos:**

- Códigos de recuperación de gestor de contraseñas
- Semillas/claves de criptomonedas
- Credenciales de cuentas importantes
- Instrucciones para seres queridos
- Ubicaciones de documentos legales
- Combinaciones de cajas fuertes

<div class="warning">
<strong>Nota:</strong> Evita archivos que cambien con frecuencia. Esto está diseñado para secretos que configuras una vez y dejas.
</div>

### Paso 3: Generar kits {#step3}

Haz clic en "Generar kits" para bloquear tus archivos y preparar un kit para cada persona.

<figure class="screenshot">
<img src="screenshots/es/bundles.png" alt="Generando kits en el Paso 3">
<figcaption>Descarga cada kit, o todos a la vez</figcaption>
</figure>

Cada kit incluye la herramienta de recuperación completa. Funciona incluso si este sitio web desaparece.

### Distribuir a los amigos {#distributing}

Envía a cada persona su kit como prefieras:

- **Correo electrónico:** Adjunta el archivo ZIP
- **Almacenamiento en la nube:** Comparte por Dropbox, Google Drive, etc.
- **Memoria USB:** Entrega en mano
- **Mensajería cifrada:** Signal, WhatsApp, etc.

Si envías los kits por correo o mensajería, borra los mensajes de tu carpeta de enviados cuando cada persona confirme que los recibió. De lo contrario, tu cuenta acabará guardando una copia de cada parte — lo que anula el sentido de haberlas separado.

### Después de crear los kits {#after-creating}

Una vez que tus kits están listos, hay algunas cosas que vale la pena hacer antes de dejarlo de lado:

- Pide a cada persona que abra `recover.html` una vez. Muestra "Tu parte está aquí" cuando todo está en orden.
- Dile a cada persona qué es esto, por qué lo tiene, y que debe guardarlo en un lugar seguro. No puede usarlo solo — necesitará coordinarse con otros.
- Guarda los archivos bloqueados con "Guardar los archivos bloqueados (MANIFEST.age)" y conserva esa copia en un lugar seguro. No contiene ninguna parte, así que no le sirve a nadie que no tenga suficientes partes.
- Guarda tu `project.yml` para poder regenerar los kits más adelante
- Imprime `README.pdf` como respaldo en papel antes de enviar el kit digital. El papel no necesita adaptadores ni electricidad.
- Pon un recordatorio anual para verificar — consulta [Mantener los kits actualizados](#keeping-current)
- Considera hacer un [simulacro de recuperación](#recovery-drills) — es la única forma de saber si tu plan realmente funciona

## Recuperar archivos {#recovering}

Esta sección es para los amigos que guardan una parte. Casi siempre, lo único que hace un amigo es guardar su kit en un lugar seguro. Algún día, alguien puede pedirle su parte, o puede ser quien reúna las partes.

Si estás aquí porque ya no puedes contactar a alguien que te importa, no hay prisa. Nada de esto caduca. Puedes tomarte todo el tiempo que necesites — si cierras la página de recuperación a mitad de camino, volverás a agregar las partes cuando regreses.

### Qué reciben los amigos {#bundle-contents}

Cada kit contiene:

<div class="bundle-contents">
<div class="file">
<span class="file-name">README.txt</span>
<span class="file-desc">Qué es esto, quién más tiene una parte, y tu propia parte</span>
</div>
<div class="file">
<span class="file-name">README.pdf</span>
<span class="file-desc">Lo mismo, listo para imprimir. Tu parte aparece como <strong>código QR</strong> y como palabras.</span>
</div>
<div class="file">
<span class="file-name">MANIFEST.age</span>
<span class="file-desc">Los archivos bloqueados. Solo viene en algunos kits — la mayoría los lleva dentro de recover.html.</span>
</div>
<div class="file">
<span class="file-name">recover.html</span>
<span class="file-desc">La página de recuperación. Se abre en cualquier navegador, con tu parte ya incluida.</span>
</div>
</div>

<p style="margin-top: 1rem;">
Cada kit está hecho para un amigo: su parte ya está dentro de su recover.html, junto con los nombres de los demás y cómo contactarlos.
</p>

### Guardar tu kit {#keeping-bundle}

Guárdalo en un lugar seguro y, si puedes, también una copia impresa de README.pdf. No hay nada más que hacer hasta que llegue el momento.

Para comprobar que funciona, abre recover.html. Muestra "Tu parte está aquí", qué parte tienes y cuántas hacen falta para abrir los archivos.

### Si alguien te pide tu parte {#sharing-piece}

Primero, confirma que la petición es real. Si puedes, consúltalo con la persona que hizo el kit. Si no puedes contactarla, consúltalo con otro amigo que tenga una parte.

Luego envíala de la forma que te resulte más fácil:

- **El kit completo** (el archivo ZIP) — lo más sencillo.
- **Tu README.txt** — contiene tu parte por sí solo.
- **Tus palabras de recuperación**, leídas en voz alta por teléfono.
- **Tu página impresa**, por correo postal — el código QR de la página contiene tu parte.

### Reunir las partes {#recovery-bundle}

Una persona reúne las partes. Puede ser cualquiera de los amigos — quien se haga cargo.

<div class="step-guide">
<div class="step-number">1</div>
<div class="step-content">
<h4>Abre recover.html</h4>
<p>Haz doble clic en el kit para abrirlo, y luego doble clic en recover.html. Se abre en tu navegador, como una página web, con tu propia parte ya incluida. Si no tienes un kit, cualquier amigo puede enviarte su recover.html, o puedes usar la <a href="recover.html">página de recuperación</a> de este sitio.</p>
</div>
</div>

<div class="step-guide">
<div class="step-number">2</div>
<div class="step-content">
<h4>Contacta a los demás</h4>
<p>La página muestra quién más tiene una parte y cómo contactarlos. Cuéntales lo que pasó y pídeles su parte. Algunos querrán confirmar contigo primero que la petición es real — sus instrucciones les piden hacerlo.</p>
</div>
</div>

<div class="step-guide">
<div class="step-number">3</div>
<div class="step-content">
<h4>Agrega cada parte a medida que llegue</h4>
<p>Arrastra su kit o su README.txt a la página, pega su texto, escribe las palabras de recuperación que te lean, o escanea el código QR de su página impresa. Aparece una marca junto a cada nombre.</p>
</div>
</div>

<div class="step-guide">
<div class="step-number">4</div>
<div class="step-content">
<h4>Los archivos se desbloquean solos</h4>
<p>Cuando hay suficientes partes, la página abre los archivos sin que tengas que hacer nada. Descárgalos y guárdalos en un lugar privado.</p>
</div>
</div>

<div class="tip">
<strong>Si la página te pide los archivos bloqueados:</strong> arrastra a la página el <code>MANIFEST.age</code> de un kit, o el recover.html de cualquier amigo. La mayoría de los kits ya los llevan, así que probablemente no veas este paso.
</div>

<figure class="screenshot">
<img src="screenshots/es/recovery-1.png" alt="Página de recuperación con las partes reunidas hasta ahora">
<figcaption>Las partes reunidas hasta ahora, y a quién contactar</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/es/recovery-2.png" alt="Página de recuperación después de desbloquear los archivos">
<figcaption>Cuando hay suficientes partes, los archivos se desbloquean y quedan listos para descargar</figcaption>
</figure>

### Escribir las palabras de recuperación {#recovery-words}

Cada página impresa tiene la parte como una lista de palabras numeradas. Si un amigo te lee las suyas por teléfono, o solo tienes papel, escribe las palabras en la página de recuperación. Pulsa "Pegar una parte o escribir palabras clave" y escríbelas separadas por espacios. Solo importan las palabras, no los números.

<figure class="screenshot">
<img src="screenshots/es/recovery-words-typing.png" alt="Escribiendo las palabras de una página impresa en la página de recuperación">
<figcaption>Escribe las palabras de la página impresa</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/es/recovery-words-recognized.png" alt="Página de recuperación después de escribir las palabras, mostrando que se agregó la parte">
<figcaption>La página reconoce las palabras y agrega la parte</figcaption>
</figure>

### Escanear un código QR {#recovery-pdf}

En una computadora o un teléfono con cámara, pulsa "Escanear QR" en la página de recuperación y pon la página impresa frente a la cámara.

<figure class="screenshot">
<img src="screenshots/qr-camera-permission.png" alt="El navegador pide permiso para usar la cámara">
<figcaption>Tu navegador te pedirá permiso para usar la cámara</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/qr-scanning.png" alt="Escaneando un código QR de una página impresa">
<figcaption>Pon el código QR frente a la cámara para agregar la parte</figcaption>
</figure>

Escanear el código QR con la cámara del teléfono también funciona: abre la página de recuperación de este sitio, con la parte ya puesta.

<div class="tip">
<strong>Sobre la recuperación:</strong>
<ul>
<li>Funciona sin conexión a <span title="Los archivos con bloqueo por fecha necesitan una conexión para comprobar la fecha de apertura.">internet*</span></li>
<li>Las partes y los archivos se quedan en tu dispositivo</li>
<li>Los amigos pueden estar en cualquier lugar — cada uno solo tiene que hacerte llegar su parte</li>
</ul>
</div>

## Buenas prácticas {#best-practices}

### Elegir amigos

- **Permanencia:** Personas con quienes puedas contactarte en 5 a 10 años
- **Distribución geográfica:** Que no estén todos en el mismo lugar
- **Habilidad técnica:** No hace falta que sean personas técnicas. Sus instrucciones los guían paso a paso, y un [simulacro de recuperación](#recovery-drills) te mostrará dónde se atasca alguien.
- **Relaciones:** ¿Cooperarán entre ellos?
- **Confianza:** Una sola parte no revela nada, pero les estás confiando una responsabilidad

### Consideraciones de seguridad

- No guardes todos los kits juntos — eso anula el propósito de dividirlos
- Considera imprimir `README.pdf` — el papel sobrevive desastres digitales
- Guarda `project.yml` si quieres regenerar los kits más adelante

### Almacenar los kits de forma segura {#storing-bundles}

Los kits son pequeños (menos de 10 MB) y están diseñados para guardarse en lugares cotidianos. Esto es lo que funciona bien:

- **El correo electrónico** funciona bien. La mayoría de las personas mantiene la misma dirección de correo durante décadas, y los kits son lo suficientemente pequeños para adjuntarlos. Muchos proveedores conservan los mensajes indefinidamente.
- **El almacenamiento en la nube** (Google Drive, Dropbox, iCloud) funciona bien como copia secundaria.
- **Las memorias USB** pueden servir, pero ten en cuenta que los conectores cambian con el tiempo (el USB-A ya está dando paso al USB-C) y la memoria flash puede degradarse si pasa años sin energía. No es ideal como única copia.
- **El papel** es la opción más duradera. Imprimir `README.pdf` le da a tus amigos una copia de su parte que no necesita adaptadores, electricidad ni ningún dispositivo funcional.

El papel guarda las partes, pero no los archivos bloqueados en sí — esos solo están en los kits digitales. Por eso el mejor enfoque es la redundancia — correo más papel, o nube más papel. Más de una copia, en más de una forma.

### Mantener los kits actualizados {#keeping-current}

Pon un recordatorio anual para hablar con tus amigos. Confirma que aún tienen sus kits y actualiza los datos de contacto si algo cambió.

Cuando tus archivos cambien, crea nuevos kits y envíalos. Los kits antiguos no abrirán el nuevo archivo, así que no hay riesgo en dejarlos por ahí — pero pide a tus amigos que reemplacen los suyos para mantener el orden.

Cuando los contactos cambien — alguien se muda, cambia su número de teléfono, o quieres agregar o quitar a alguien — lo mismo: nuevos kits, pide que borren los anteriores.

Entre actualizaciones, guarda tus archivos fuente en una bóveda cifrada — herramientas como [Cryptomator](https://cryptomator.org) o [VeraCrypt](https://veracrypt.fr) funcionan bien. No dejes copias en texto plano en una carpeta normal.

Piénsalo como actualizar tus contactos de emergencia. Breve, periódico, vale la pena hacerlo.

### Simulacros de recuperación {#recovery-drills}

Nadie prueba su plan de respaldo. Un simulacro de recuperación cambia eso — y se juega un poco como un escape room.

Crea un kit de prueba con un secreto inofensivo (una foto, un mensaje, una receta). Reparte las partes a tus amigos. Pon un [bloqueo por fecha](#timelock) si quieres una cuenta regresiva real. Después aléjate y déjalos resolver: coordinarse, combinar partes, desbloquear los archivos — sin tu ayuda.

Lo que aprendes:

- ¿Pueden tus amigos encontrar sus kits cuando los necesitan?
- ¿Entienden las instrucciones, o se quedan atascados?
- ¿Pueden contactarse entre ellos sin que tú coordines?

Un simulacro que sale bien significa que tu plan real también funcionará. Un simulacro que sale mal es un regalo — encontraste el problema antes de que importara.

### Revocar acceso {#revoking-access}

Una vez que una parte ha sido distribuida, no se puede revocar. Esto es por diseño — no hay servidor, no hay autoridad central.

Si necesitas cambiar quién tiene partes:

1. **Crea nuevos kits** con un nuevo grupo de amigos y una clave nueva
1. **Envía los nuevos kits** a los amigos en quienes sigues confiando
1. **Pide a cada amigo que borre su kit anterior** y lo reemplace con el nuevo

<div class="warning">
<strong>Importante:</strong> Las partes previas siguen funcionando con los archivos que protegían. Cuando envíes un nuevo kit, sé claro: <strong>borra el anterior</strong>, quédate solo con el nuevo. Sin historial de versiones, sin "por si acaso."
</div>

Lo mismo aplica cuando los secretos cambian. Nuevos kits significan una clave nueva y partes nuevas. Las partes anteriores no abrirán el nuevo archivo, pero siguen funcionando con el anterior. Asegúrate de que tus amigos no conserven copias viejas.

### Sobre project.yml {#project-file}

Cuando creas kits, tu proyecto se guarda en un archivo `project.yml`. Este archivo almacena:

- Nombres e información de contacto de los amigos
- El umbral que elegiste (por ejemplo, 3 de 5)
- Un hash de verificación para comprobar si los kits coinciden
- Sumas de verificación de las partes para confirmar la integridad de los kits

**No** almacena ningún secreto — ni contraseña, ni material criptográfico, ni contenido de archivos. Es seguro guardarlo junto con el resto de tus documentos.

Con `project.yml`, puedes regenerar los kits, verificar los existentes y revisar el estado de tu configuración.

## Entender la seguridad {#security}

ReMemory combina herramientas criptográficas bien establecidas en lugar de inventar las suyas. Esto es lo que significa en la práctica.

### Qué protege tus datos {#cryptography}

Tus archivos se bloquean con una herramienta de cifrado moderna ([age](https://github.com/FiloSottile/age)) — ampliamente revisada, sin debilidades conocidas.

La clave que los bloquea tiene 256 bits, generada por el generador de números aleatorios de tu sistema operativo. Para dar una idea: adivinarla tomaría más tiempo de lo que lleva existiendo el universo.

Incluso si alguien intentara todas las contraseñas posibles, scrypt hace cada intento deliberadamente lento — millones de veces más lento que un intento directo.

La clave se divide usando Shamir's Secret Sharing. **Cualquier cantidad menor al umbral de partes contiene cero información sobre la clave.** No "muy poca." Matemáticamente cero.

Cada kit también lista sumas de verificación de sus archivos, para que `rememory verify-bundle` pueda decirte si alguno se dañó por el camino. Las sumas detectan accidentes, no cambios deliberados: quien modifique un archivo puede actualizar también su suma. El archivo cifrado está protegido por sí mismo — si se altera, no se abrirá.

### Qué podría salir mal {#what-could-go-wrong}

<div class="bundle-contents">
<div class="file">
<span class="file-name">Un amigo pierde su kit</span>
<span class="file-desc">No hay problema, siempre que suficientes amigos conserven el suyo. Para eso fijas el umbral por debajo del total.</span>
</div>
<div class="file">
<span class="file-name">Un amigo expone su parte públicamente</span>
<span class="file-desc">Una sola parte es inútil sin las demás. Alguien aún necesitaría el umbral menos una parte adicional para hacer algo.</span>
</div>
<div class="file">
<span class="file-name">Algunos amigos no están disponibles</span>
<span class="file-desc">Para eso fijas el umbral por debajo del total de amigos. Si elegiste 3 de 5, cualquier tres sirven.</span>
</div>
<div class="file">
<span class="file-name">ReMemory desaparece en 10 años</span>
<span class="file-desc"><code>recover.html</code> sigue funcionando — es autónomo. Sin servidores, sin descargas, sin dependencia de este proyecto.</span>
</div>
<div class="file">
<span class="file-name">Los navegadores cambian radicalmente</span>
<span class="file-desc">La herramienta de recuperación usa JavaScript estándar y la Web Crypto API — fundamentos del navegador, no tendencias.</span>
</div>
<div class="file">
<span class="file-name">Olvidas cómo funciona esto</span>
<span class="file-desc">El README.txt de cada kit lo explica todo. Tus amigos no necesitan recordar nada — todo está escrito para ellos.</span>
</div>
</div>

Lo que *sí* necesita ser verdad: que tu dispositivo sea confiable cuando creas los kits, y que el navegador usado para recuperar no esté comprometido. Son las mismas suposiciones que haces cada vez que usas una computadora para algo importante.

Para una evaluación técnica detallada, consulta la [autoevaluación de seguridad]({{GITHUB_REPO}}/blob/main/docs/security-review.md).

## Cómo se compara {#comparison}

ReMemory no es la primera herramienta en usar Shamir's Secret Sharing. Hay muchas otras, desde herramientas de línea de comandos hasta aplicaciones web. Esto es lo que distingue a ReMemory:

- **Maneja archivos, no solo texto.** La mayoría de las herramientas Shamir solo dividen contraseñas o texto corto. ReMemory cifra archivos y carpetas enteras.
- **Herramienta de recuperación autónoma.** Cada amigo recibe `recover.html` — una herramienta completa que funciona en cualquier navegador, sin conexión.\* Sin instalación, sin línea de comandos.
- **Datos de contacto incluidos.** Cada kit incluye una lista de los otros amigos y cómo contactarlos, para que la coordinación no dependa de que tú estés disponible.
- **Sin dependencia de servidores.** Todo se ejecuta localmente. No hay servicio al que registrarse, ni cuenta que mantener, ni nada que deba permanecer en línea.

Para una comparación detallada con otras herramientas, consulta la [tabla comparativa completa en GitHub]({{GITHUB_REPO}}#other-similar-tools).

## Alternativa por línea de comandos {#cli}

También hay una herramienta de línea de comandos para quienes prefieren la terminal o necesitan automatizar la creación de kits.

<a href="{{GITHUB_REPO}}/blob/main/docs/guide.md" class="btn btn-secondary">Leer la guía del CLI</a>

<p style="margin-top: 1rem;">
El CLI ofrece la misma funcionalidad, más operaciones por lotes y scripting.
</p>

## Avanzado: Modo anónimo {#anonymous}

Cuando los participantes no deben conocer la identidad de los demás, usa el **modo anónimo**:

- Las personas se identifican como "Parte 1", "Parte 2", etc.
- No se recopila ni almacena información de contacto
- Los READMEs omiten la sección "Otros participantes"
- Los nombres de los kits usan números en lugar de nombres

### Cuándo usar el modo anónimo

Es útil cuando:

- Los participantes no deben saber quiénes son los demás
- Estás haciendo una prueba rápida sin ingresar nombres
- Tienes otra forma de coordinar la recuperación
- La privacidad es más importante que la facilidad de coordinación

### Cómo activarlo

En el [creador de kits](maker.html), activa el interruptor **Anónimo** en la sección de Amigos:

- La lista de amigos se reemplaza por un conteo de partes
- Configura cuántas partes y el umbral
- Los kits se nombran `bundle-share-1.zip`, `bundle-share-2.zip`, etc.

### Recuperación en modo anónimo

La recuperación funciona de la misma forma, pero sin la lista de contactos. Los participantes ven etiquetas genéricas como "Parte 1" en lugar de nombres.

<div class="warning">
<strong>Importante:</strong> Sin una lista de contactos integrada, asegúrate de que los participantes sepan cómo contactarse entre sí cuando se necesite la recuperación.
</div>

## Avanzado: Kits multilingües {#multilingual}

Cada persona puede recibir su kit en su idioma preferido. Está disponible en siete idiomas: inglés, español, alemán, francés, esloveno, portugués y chino (Taiwán).

### Cómo funciona

- Cada entrada de amigo tiene un menú desplegable de **Idioma del kit**
- "Por defecto" usa el idioma actual de la interfaz
- Se puede personalizar por persona para mezclar idiomas
- recover.html se abre en el idioma seleccionado
- Cualquiera puede cambiar de idioma en cualquier momento

<figure class="screenshot">
<img src="screenshots/es/multilingual-language-dropdown.png" alt="Entrada de amigo mostrando el menú desplegable de idioma del kit en la interfaz web">
<figcaption>Cada amigo tiene un menú desplegable de idioma para configurar el idioma de su kit</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/demo-pdf/page-1.png" alt="README.pdf abierto en inglés">
<figcaption>La herramienta de recuperación se abre en el idioma seleccionado del amigo</figcaption>
</figure>

<figure class="screenshot">
<img src="screenshots/demo-pdf/page-2.png" alt="README.pdf abierto en inglés">
<figcaption>Las listas de palabras también están traducidas (ambos idiomas funcionan)</figcaption>
</figure>

## Avanzado: Bloqueo por fecha {#timelock}

Puedes establecer un periodo de espera al crear los kits. Incluso si tus amigos combinan sus partes antes de tiempo, los archivos permanecen bloqueados hasta la fecha que elegiste — 30 días, 6 meses, una fecha específica.

### Cómo activarlo

En el [creador de kits](maker.html), cambia al modo **Avanzado** y marca **Agregar un bloqueo por fecha**. Elige cuánto tiempo deben permanecer bloqueados los archivos — hasta dos años.

El límite de dos años es deliberado. Los bloqueos por fecha dependen de que la League of Entropy siga operando, y no nos parece responsable bloquear tus archivos con una apuesta más larga sobre infraestructura externa. Si necesitas una duración mayor y entiendes el compromiso, la CLI no tiene límite: `rememory seal --timelock 5y`.

<figure class="screenshot">
<img src="screenshots/es/tlock-setup.png" alt="Configuración del bloqueo por fecha en modo Avanzado">
<figcaption>La opción de bloqueo por fecha aparece en el modo Avanzado</figcaption>
</figure>

### Recuperación

Cuando alguien abre un kit con bloqueo por fecha antes de la fecha indicada, la herramienta de recuperación muestra un aviso de espera. Cuando el tiempo pasa, la recuperación continúa normalmente.

<figure class="screenshot">
<img src="screenshots/es/tlock-waiting.png" alt="Herramienta de recuperación mostrando un aviso de bloqueo por fecha">
<figcaption>La herramienta de recuperación espera hasta que el bloqueo por fecha expire</figcaption>
</figure>

Abrir un archivo con bloqueo por fecha requiere una breve conexión a internet. Tus archivos no se envían a ningún lado — la conexión verifica que ha pasado suficiente tiempo. Sin el bloqueo por fecha, la recuperación es completamente sin conexión.

<div class="warning">
<strong>Experimental.</strong> El bloqueo por fecha depende de la <a href="https://www.cloudflare.com/en-ca/leagueofentropy/" target="_blank">League of Entropy</a>, una red distribuida operada por organizaciones serias alrededor del mundo. Si esta red deja de funcionar antes de que expire un bloqueo por fecha, ese archivo se vuelve irrecuperable. Los kits sin bloqueo por fecha no se ven afectados.
</div>

### Cómo funciona {#timelock-technical}

La League of Entropy produce un nuevo valor criptográfico cada 3 segundos. Cada valor está numerado. Puedes predecir qué número corresponde a un momento dado, pero el valor para ese número no puede producirse antes de tiempo — por nadie, incluyendo a los operadores de la red.

Cuando creas un kit con bloqueo por fecha, el archivo se cifra con un valor futuro específico. La clave para abrirlo aún no existe. Vendrá de la red cuando llegue ese momento.

Para más detalles sobre la criptografía detrás de esto, consulta la [documentación de timelock encryption de drand](https://docs.drand.love/docs/timelock-encryption/).
