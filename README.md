# RecompHub v4

RecompHub regroupe les recompilations, ports PC et launchers installés sur Windows.

## Les expériences console

- **XMB** : navigation horizontale entre catégories, déplacement vertical parmi les jeux, fond à vagues léger, sans tableau de bord.
- **Chaînes** : menu 4 × 3, navigation à la croix directionnelle, ouverture d'une chaîne et écran de démarrage.
- **Horizon**, **Simple** et **Nuit** : trois thèmes secondaires conservés.

Les références sont des hommages **originaux**, pas des copies des ressources de consoles commerciales.

## Utilisation

1. Extraire le fichier ZIP Windows.
2. Lancer `RecompHub.exe` (Windows 10/11 64 bits, Microsoft Edge requis).
3. Ajouter un jeu par son `.exe`, `.bat`, `.cmd`, `.lnk` ou `.jar`, ou scanner un dossier.
4. Changer de thème depuis les options. Les préférences et la bibliothèque sont conservées dans `%LOCALAPPDATA%\RecompHub\library.json`.
5. Utiliser les flèches, Entrée et Échap; `Ctrl+K` ouvre la recherche.

## Boutique

La sélection provient des releases GitHub de projets libres publiées par leurs équipes. L'installation télécharge un ZIP Windows et le décompresse dans le dossier de RecompHub, puis tente de reconnaître le bon exécutable. **Aucune ROM ou ressource commerciale n'est incluse.** Certains ports exigent des données de jeu personnelles légalement obtenues. Les programmes tiers ne sont jamais lancés sans intervention de l'utilisateur.

## Fenêtre et mises à jour

Le lanceur tente de supprimer la bordure de la fenêtre Edge en modifiant son style Windows à l'exécution. Ce comportement n'est **pas garanti sur toutes les éditions/versions d'Edge**. Si le détachement échoue, les contrôles Windows natifs restent accessibles. La bibliothèque ne dépend pas de cette personnalisation.

Mise à jour : menu **Options > Rechercher une mise à jour**. RecompHub ne propose une mise à jour automatique que si une release existe avec un manifeste `latest.json` signé par SHA-256 (vérification de l'intégrité, **pas** signature de l'éditeur). Les prochaines versions sont produites avec `.github/workflows/windows-release.yml` après publication du code et activation de GitHub Actions sur le dépôt.

## Publication de la v4 sur GitHub

Dépôt actuel : https://github.com/Ironakatina33/RecompHub

Déposer le contenu de l'archive source **à la racine du dépôt** (pas dans un répertoire supplémentaire). Le dossier `.github/workflows` peut déjà exister : il suffit alors d'éditer `.github/workflows/windows-release.yml` pour coller le contenu du nouveau fichier. GitHub Actions compile automatiquement le projet en Go pour Windows et crée une release v4.0.x.

Ne pas committer l'exécutable compilé ou `%LOCALAPPDATA%\RecompHub` dans le dépôt public.

## Développement

Le backend est écrit en Go standard library, sans dépendance tierce. Compilation Windows depuis Linux :

```
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-H windowsgui -s -w" -o RecompHub.exe .
```

Le frontend se trouve dans `assets/`. L'application le sert uniquement sur une adresse loopback aléatoire, avec une clé d'accès par session.
