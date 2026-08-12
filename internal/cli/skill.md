# Email Manager (Gmail multi-comptes) — Mode d'emploi pour agent

Ce document est imprimé par `email-manager skill`. Il décrit le workflow complet
de gestion des emails Gmail personnels via le binaire `email-manager`, à
destination d'un agent (Claude, Cursor, etc.) qui doit assister l'utilisateur.

## Quand utiliser

Déclencheurs typiques (français) :

- "scanner / trier / gérer mes emails"
- "envoyer un email / mail"
- "cherche / recherche un mail", "trouve l'email", "lis le mail"
- "montre / affiche le mail", "lien du mail"
- "mail de X", "mail chez X" (X = prénom : Lamya, Seb, ...)
- "sur le compte de X", "dans la boîte de X"

Pour les emails **professionnels** (Employer, IBM), utiliser `o365-manager` à
la place.

## Multi-comptes : workflow OBLIGATOIRE

Le binaire gère plusieurs comptes Gmail. Avant TOUTE opération, déterminer
sur quel compte agir.

### 1. Lister les comptes

```bash
email-manager accounts
```

### 2. Identifier le compte cible

Mapper la mention utilisateur vers une adresse :

| Indice dans la demande | Compte |
|---|---|
| "chez Lamya", "mail de Lamya" | `lamya.chrif@gmail.com` |
| "chez moi", "mes emails", aucune mention, "Seb" | `seb.morand@gmail.com` (défaut) |
| Email explicite | Cet email |

Si ambigu, demander confirmation avant d'agir.

### 3. Exécuter avec `--account`

```bash
email-manager --account <email> <commande> [args...]
```

S'il n'y a qu'un seul compte authentifié, `--account` est facultatif (auto-
résolu). S'il y en a plusieurs, il est obligatoire.

### 4. Lien Gmail web

Quand l'utilisateur demande "le lien" pour un message :

```
https://mail.google.com/mail/u/<email>/#inbox/<MESSAGE_ID>
```

## Commandes disponibles

```
email-manager [--account <email>]
├── accounts                # Lister les comptes authentifiés
├── auth --account <email>  # (Re-)authentifier un compte
├── send                    # Envoyer un email
├── list                    # Lister les messages
├── get <id>                # Détails d'un message
├── search <query>          # Rechercher
├── read <id>               # Marquer lu
├── unread <id>             # Marquer non lu
├── archive <id>            # Retirer de INBOX
├── trash <id>              # Mettre à la corbeille
├── untrash <id>            # Restaurer de la corbeille
├── spam <id>               # Marquer comme spam
├── not-spam <id>           # Retirer du spam
├── download-attachments <id> [--dir DIR]
├── labels
│   ├── list
│   ├── create <name>
│   ├── apply <msg-id> <label-id>
│   └── remove <msg-id> <label-id>
├── drafts
│   ├── list
│   ├── create --to ... --subject ... --body ... [--cc ...] [--bcc ...] [--attach ...]
│   │            [--reply-to <message-id>]  # thread le brouillon en réponse à un mail existant
│   └── delete <draft-id>
├── cal                     # Agenda Google Calendar — voir section "Agenda" ci-dessous
└── skill                   # Imprime ce mode d'emploi
    └── learn --rule "..."  # Apprend une règle de tri (validation utilisateur)
```

Toute commande accepte `--help` pour voir les options détaillées.

## Corps HTML (`send` et `drafts create`)

Le corps peut être en texte ou en HTML :

- `--body "<texte>"` : corps texte (text/plain).
- `--html "<html>"` : corps HTML inline (petit HTML).
- `--html-file <chemin>` : corps HTML depuis un fichier (recommandé pour un gros
  HTML, ex. un compte rendu généré). `--html` et `--html-file` sont exclusifs.

Règles :

- Au moins un de `--body` / `--html` / `--html-file` est requis.
- Avec du HTML sans `--body`, une partie `text/plain` est dérivée
  automatiquement du HTML (multipart/alternative) pour les clients sans HTML.
- `--attach` fonctionne avec l'un ou l'autre (le corps est alors emballé dans un
  multipart/mixed avec les pièces jointes).

```bash
email-manager send --to X --subject "CR" --html-file compte-rendu.html --attach schema.png
```

## Brouillons en réponse (`drafts create --reply-to`)

Pour créer un brouillon threadé (réponse dans un fil existant), utiliser `--reply-to <message-id>` :

```bash
email-manager --account <email> drafts create \\
  --to destinataire@example.com \\
  --reply-to <MESSAGE_ID> \\
  --body "Corps de la réponse"
```

- `--reply-to` accepte l'ID Gmail du message original (ex: `19f6c544aa3b26a6`).
- `--subject` devient optionnel : si absent, le sujet est auto-dérivé du message original avec le préfixe "Re: ".
- Les headers `In-Reply-To` et `References` sont posés automatiquement (RFC 2822).
- Le brouillon est attaché au même thread Gmail — il apparaît dans le bon fil dans l'interface.
- Utile pour préparer une réponse sans envoyer : l'utilisateur ouvre le brouillon dans Gmail et envoie quand il veut.


## Tri automatique : règle par défaut

**Scanner UNIQUEMENT l'INBOX**. Les emails déjà classés dans d'autres labels
(`personal/ecole`, `personal/voyage`, `personal/commandes`, `personal/Maison`,
`pro/Employer`) ne doivent JAMAIS être re-triés. Une fois qu'un email a quitté
l'INBOX, il n'est plus touché par le tri automatique.

Exception : si l'utilisateur demande explicitement de scanner un autre label.

### Labels de destination

| Label | Contenu |
|---|---|
| `personal/ecole` | Emails scolaires (Lycée Descartes, etc.) |
| `personal/voyage` | Confirmations de voyage, billets, Uber transport |
| `personal/commandes` | Confirmations de commandes, Amazon, Fnac, Uber Eats |
| `personal/Maison` | Factures Free, Netflix, Sosh, Disney+, Crunchyroll |
| `pro/Employer` | Emails professionnels Employer |

### Workflow type pour un email d'INBOX

```bash
# 1. Lister les non-lus de l'INBOX
email-manager search "label:INBOX is:unread" --max 50

# 2. Pour chaque email, identifier le type via patterns connus
# 3. Appliquer les actions :
email-manager labels list                         # trouver l'ID du label cible
email-manager read <msg-id>
email-manager labels apply <msg-id> <label-id>
email-manager archive <msg-id>
```

### Confirmation utilisateur OBLIGATOIRE avant action

- Paiements à effectuer (factures impayées, échéances)
- Check-in vol
- Modification / annulation vol
- Documents officiels importants

### Détection mailing list / spam

Identifier les emails avec headers `List-Unsubscribe`, `List-Id`, ou liens
"unsubscribe" / "se désabonner". Proposer à l'utilisateur :

1. Ouvrir le lien de désinscription
2. Créer un filtre permanent (via `skill learn`)
3. Ignorer

### Suggestions de suppression

Pour les emails potentiellement inutiles :

- Codes expirés (>7 jours)
- Marketing ancien (>90 jours sans interaction)
- Spam évident

TOUJOURS demander confirmation avant `trash`. Aucune suppression définitive
(le binaire n'a pas de commande de hard delete — on utilise `trash`).

## Bilan de fin de scan

Format type à produire à l'utilisateur :

```
📧 Scan des emails terminé

✅ Actions effectuées :
- 8 emails archivés (confirmations Amazon, Fnac)
- 3 emails archivés (publicités Netflix, Free)
- 2 codes SafeKey envoyés à la corbeille
- 1 récap PayPal archivé après résumé

📬 Listes de distribution détectées :
- Newsletter Tech Weekly (lien unsubscribe disponible)

🗑️ Suggestions de suppression :
- 3 codes SafeKey expirés (>30 jours)

⚠️ Nécessitent confirmation :
- Air France : Check-in vol Paris-Dublin (départ 15/05)

📊 Total : 18 emails traités
```

## Apprentissage de règles : `skill learn`

Quand l'utilisateur dit *"ces mails là tu peux toujours les traiter comme ça"*
ou autre connaissance permanente, persister la règle dans
`~/.config/email-manager/regles-tri.md` :

```bash
email-manager skill learn --rule "Tous les emails de noreply@example.com -> archive + label personal/commandes"
```

Avant d'exécuter `skill learn`, **toujours demander confirmation utilisateur**
avec la formulation exacte de la règle. Le fichier produit est ensuite
concaténé automatiquement à la sortie de `email-manager skill` lors des
prochaines exécutions, ce qui permet à l'agent d'avoir un contexte évolutif.

### Fichiers de connaissance utilisateur

Trois fichiers sont attendus (créés à la demande) dans
`~/.config/email-manager/` :

- `regles-tri.md` — règles de tri par expéditeur / type
- `patterns-emails.md` — patterns regex pour identifier les types
- `filtres-evolutifs.md` — historique des filtres appris

Ces fichiers sont automatiquement inclus dans la sortie de
`email-manager skill` quand ils existent.

## Agenda (Google Calendar)

Déclencheurs typiques (français) :

- "mon agenda", "mon planning", "mes rendez-vous"
- "ajoute un rendez-vous / événement", "bloque un créneau", "planifie une réunion"
- "quand est mon prochain call/rdv", "suis-je libre le..."
- "annule le rendez-vous X", "déplace le rendez-vous X"
- "accepte / décline / réponds tentative à l'invitation Y"
- "crée un lien Google Meet"

Même règle multi-comptes que l'email : déterminer le compte cible avant
d'agir (`email-manager accounts`, puis `--account <email>`).

### Date/heure courante : OBLIGATOIRE avant toute opération temporelle

**Avant tout calcul de date relative ("aujourd'hui", "hier", "demain",
"cette semaine", "le prochain rendez-vous", "mes events" sans date précisée,
...), vérifier la date et l'heure réelles actuelles** (ex: commande shell
`date`), ne jamais les déduire d'une date d'entraînement ou d'un présupposé.

- "Aujourd'hui", "demain", "cette semaine" sont ancrés sur la date réelle du
  jour où la commande est exécutée, pas sur une date supposée par l'agent.
- Quand l'utilisateur demande "mes prochains events" / "mon agenda" /
  "suis-je libre" **sans préciser de date**, cela veut dire **à partir de
  maintenant (date/heure réelle courante) et dans le futur** — jamais une
  fenêtre calculée à partir d'une date erronée. Un "prochain rendez-vous"
  cherché avec un `--start` déjà passé par rapport à la vraie date du jour
  renverra des résultats obsolètes ou vides sans que ce soit une erreur
  d'API : c'est une erreur de raisonnement temporel de l'agent à corriger
  avant l'appel.
- Cette règle s'applique à `cal list`, `cal instances`, `cal freebusy`, et à
  tout `--start`/`--end` construit implicitement pour une demande sans date
  explicite.

### Commandes `cal`

```
email-manager cal [--calendar-id primary]
├── calendars list                         # Calendriers visibles (id, primary, accessRole)
├── list --start <RFC3339> --end <RFC3339> [--query TEXT] [--max 50] [--timezone Europe/Paris]
├── get <event-id>                         # Détails : attendees+statut, lien Meet, récurrence, rappels
├── instances <event-id> --start ... --end ...   # Occurrences d'un événement récurrent
├── add --summary/-s ... --start ... --end ...
│        [--all-day] [--timezone Europe/Paris] [--location ...] [--description ...]
│        [--attendee EMAIL ...] [--recurrence RRULE ...] [--reminder-minutes N ...]
│        [--color-id ID] [--visibility default|public|private] [--busy | --free]
│        [--meet] [--notify none|all|externalOnly]
├── update <event-id> [mêmes flags que add, tous optionnels] [--notify ...]
│        # PATCH partiel : seuls les flags explicitement passés sont modifiés
├── delete <event-id> [--notify none|all|externalOnly]   # DÉFINITIF, pas d'undo API
├── respond <event-id> --status accepted|declined|tentative [--comment TEXT] [--notify ...]
├── quick-add --text "Dinner with Sara Fri at 7pm"        # NLP Google, spécifique Calendar
└── freebusy --calendar ID [--calendar ID2 ...] --start ... --end ...   # Créneaux occupés
```

### Format des dates

- `--start`/`--end` : RFC3339 avec timezone (ex: `2026-01-05T09:00:00`, timezone
  posée séparément via `--timezone`, défaut `Europe/Paris`), ou date
  `YYYY-MM-DD` avec `--all-day`.
- `--recurrence` : chaîne RRULE brute (RFC5545), ex :
  `RRULE:FREQ=WEEKLY;BYDAY=MO;COUNT=10`. Passée telle quelle, pas de
  traduction depuis du langage naturel (sauf via `quick-add`, qui lui accepte
  du texte libre en anglais).

### Règle `--notify` : PAS d'invitation par défaut

`--notify` vaut `none` par défaut sur `add`/`update`/`delete`/`respond` :
**aucun email n'est envoyé aux participants tant que `--notify all` n'est pas
passé explicitement.** Quand une commande ajoute/modifie/retire des
participants avec `--notify none` (défaut), le binaire affiche un
avertissement stderr le rappelant.

**Comportement agent obligatoire :**

- Si des participants sont ajoutés/modifiés et que l'utilisateur attend une
  vraie invitation envoyée par email, informer l'utilisateur et proposer
  explicitement `--notify all` — ne jamais l'ajouter silencieusement.
- **Toujours demander confirmation avant `cal delete`** (suppression
  définitive, aucun undo côté API Google, contrairement à `trash` côté mail).
- **Toujours demander confirmation avant tout `--notify all`** (envoie un
  email à des tiers).

### Google Meet

`--meet` sur `cal add` (ou `cal update`) attache un lien Google Meet
généré par Google (`conferenceDataVersion=1`). Le lien apparaît dans la
sortie de `add`/`update` et dans `cal get`/`cal list` (marqueur `[meet]`).

### Scope OAuth Calendar

Si une commande `cal ...` échoue avec une erreur 403 / scope manquant sur un
compte déjà authentifié pour le mail, c'est que ce compte n'a pas encore le
scope Calendar (ajouté après sa première authentification). Relancer :

```bash
email-manager auth --account <email>
```

Ceci est un cas normal pour tout compte authentifié avant l'ajout du support
calendrier, pas une erreur à corriger dans le code.

## Authentification

- Credentials OAuth2 : `GOOGLE_CREDENTIALS_FILE` (par défaut
  `~/.credentials/google_credentials.json`)
- Tokens par compte : `~/.cache/email-manager/<email>.json`

**Reconnexion automatique** : quand un access token expire, le binaire le
rafraîchit silencieusement via le refresh token. Si le refresh token lui-
même est invalide / révoqué, le binaire ouvre **automatiquement** le
navigateur pour relancer le flow OAuth, persiste le nouveau token, puis
poursuit la commande en cours. **L'agent ne doit donc jamais demander à
l'utilisateur de lancer `email-manager auth` manuellement** ; cette
commande reste disponible uniquement pour forcer une re-authentification
volontaire (changement de scope, rotation de credentials, etc.).
