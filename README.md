# RecompHub v3

**RecompHub** — bibliothèque Windows pour ports PC et recompilations, avec des thèmes d'interface hommage aux consoles.

## Inclus

- **XMB Legacy** : catégories horizontales, navigation verticale et fond animé.
- **Channel Home** : grille de chaînes inspirée Wii, pagination et écran de démarrage de chaque jeu.
- **Blue Horizon**, **Joy Home**, **Neon Arcade** : trois autres présentations.
- Tentative de fenêtre Windows sans barre native avec commandes intégrées (et repli automatique).
- Bibliothèque locale rétrocompatible (`%LOCALAPPDATA%\RecompHub\library.json`).
- **Boutique GitHub** : liste blanche de cinq projets open source, métadonnées depuis les releases officielles, installation ZIP Windows pour quatre projets, extraction protégée contre les chemins malveillants, enregistrement du lanceur reconnu.
- Aucune ROM, image-disque ni ressource commerciale distribuée ; les données des jeux appartiennent à l'utilisateur.
- Vérification de la version officielle de RecompHub et de l'empreinte SHA256 de son exécutable lors des mises à jour.

## Compiler pour Windows

Go 1.23+ installé. À la racine du projet :

```powershell
$env:GOOS='windows'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
go build -trimpath -ldflags='-H windowsgui -s -w' -o RecompHub.exe .
```

Pas besoin de dépendances Go externes. Microsoft Edge est nécessaire pour l'interface.

## Publier automatiquement

Remplacer l'ancien `.github/workflows/windows-release.yml` par celui du projet. Le workflow compile **tous** les fichiers Go à la racine, puis publie `RecompHub.exe`, `RecompHub-Windows.zip` et `latest.json` dans Releases.

Une nouvelle version Windows peut remplacer l'ancienne tout en conservant `library.json`.

## Limites

La sélection automatique du lanceur d'une archive tierce utilise son nom ; vérifie le résultat avant de démarrer. Les ressources du jeu original ne sont pas installées. Le catalogue nécessite Internet et les API GitHub. Les téléchargements tiers ne possèdent pas de SHA256 attesté indépendamment. La barre personnalisée dépend du comportement d'Edge sur Windows.
