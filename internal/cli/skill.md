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

Pour les emails **professionnels** (L'*******, IBM), utiliser `o365-manager` à
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
│   └── delete <draft-id>
└── skill                   # Imprime ce mode d'emploi
    └── learn --rule "..."  # Apprend une règle de tri (validation utilisateur)
```

Toute commande accepte `--help` pour voir les options détaillées.

## Tri automatique : règle par défaut

**Scanner UNIQUEMENT l'INBOX**. Les emails déjà classés dans d'autres labels
(`personal/ecole`, `personal/voyage`, `personal/commandes`, `personal/Maison`,
`pro/L'*******`) ne doivent JAMAIS être re-triés. Une fois qu'un email a quitté
l'INBOX, il n'est plus touché par le tri automatique.

Exception : si l'utilisateur demande explicitement de scanner un autre label.

### Labels de destination

| Label | Contenu |
|---|---|
| `personal/ecole` | Emails scolaires (Lycée Descartes, etc.) |
| `personal/voyage` | Confirmations de voyage, billets, Uber transport |
| `personal/commandes` | Confirmations de commandes, Amazon, Fnac, Uber Eats |
| `personal/Maison` | Factures Free, Netflix, Sosh, Disney+, Crunchyroll |
| `pro/L'*******` | Emails professionnels L'******* |

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

## Authentification

- Credentials OAuth2 : `GOOGLE_CREDENTIALS_FILE` (par défaut
  `~/.credentials/google_credentials.json`)
- Tokens par compte : `~/.cache/email-manager/<email>.json`
- Pour (re)-authentifier après expiration :
  ```bash
  email-manager auth --account <email>
  ```
