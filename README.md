# RecompHub v2

Bibliothèque de jeux Windows avec cinq interfaces originales inspirées des grandes consoles : **XMB Legacy**, **Blue Horizon**, **Channel Studio**, **Joy Home** et **Neon Arcade**.

![RecompHub](assets/icon.svg)

## Lancer l'application

Télécharger `RecompHub-Windows.zip` dans l'onglet [Releases](https://github.com/Ironakatina33/RecompHub/releases), extraire et ouvrir `RecompHub.exe`.

**Prérequis :** Windows 10/11 (64 bits) avec Microsoft Edge. La fenêtre est ouverte par Edge en mode application (sans barre d'adresse), tandis que le programme Go tourne localement. L'interface ne nécessite pas de connexion Internet, sauf pour rechercher les mises à jour.

## Fonctionnalités

- Cinq thèmes sélectionnables instantanément, sauvegardés dans la bibliothèque.
- Ajouter des jeux `.exe`, `.bat`, `.cmd`, `.lnk` et `.jar`.
- Scanner un dossier, classer par catégorie, chercher et trier.
- Jouer, marquer des favoris, modifier le nom, les arguments, les notes et la couverture.
- Compatible avec la bibliothèque des précédentes versions (`%LOCALAPPDATA%\RecompHub\library.json`).
- Vérifier les mises à jour sur GitHub et installer une nouvelle version après confirmation et vérification SHA-256.

## Compilation

Depuis le dossier contenant `go.mod` :

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-H windowsgui -s -w -X main.releaseRepo=Ironakatina33/RecompHub -X main.version=2.0.0" -o RecompHub.exe .
```

Pas de dépendance Go externe. Tous les fichiers web sont embarqués dans le binaire grâce à `go:embed`.

## Publication GitHub

Le fichier `.github/workflows/windows-release.yml` compile une version Windows et publie `RecompHub.exe`, `RecompHub-Windows.zip`, et `latest.json` sur GitHub Releases après une modification sur la branche `main`.

Les interfaces sont des créations originales **inspirées** des conventions de navigation des consoles, sans reproduire les logos, marques ni ressources graphiques propriétaires.
