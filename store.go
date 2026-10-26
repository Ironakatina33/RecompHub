//go:build windows

package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Project struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Repo           string `json:"repo"`
	Type           string `json:"type"`
	Description    string `json:"description"`
	Accent         string `json:"accent"`
	NeedsAssets    bool   `json:"needsAssets"`
	Installable    bool   `json:"installable"`
	ExecutableHint string `json:"-"`
	Tag            string `json:"tag,omitempty"`
	Asset          string `json:"asset,omitempty"`
	Size           int64  `json:"size,omitempty"`
	Installed      bool   `json:"installed"`
	Available      bool   `json:"available"`
	Error          string `json:"error,omitempty"`
}
type gitRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
		Size int64  `json:"size"`
	} `json:"assets"`
}
type storeTask struct {
	Project   string `json:"project"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Done      int64  `json:"done"`
	Total     int64  `json:"total"`
	GameAdded bool   `json:"gameAdded"`
	Error     string `json:"error,omitempty"`
}

var catalog = []Project{
	{ID: "z64", Name: "Zelda 64: Recompiled", Repo: "Zelda64Recomp/Zelda64Recomp", Type: "Recompilation", Description: "Majora's Mask recompilé avec graphismes modernes, haute fréquence et mods.", Accent: "#6192ff", NeedsAssets: true, Installable: true, ExecutableHint: "Zelda64Recompiled"},
	{ID: "soh", Name: "Ship of Harkinian", Repo: "HarbourMasters/Shipwright", Type: "Port PC", Description: "Ocarina of Time sur PC avec options modernes et randomizer.", Accent: "#e1bd6d", NeedsAssets: true, Installable: true, ExecutableHint: "soh"},
	{ID: "2ship", Name: "2 Ship 2 Harkinian", Repo: "2ship2harkinian/2ship2harkinian", Type: "Port PC", Description: "Majora's Mask en port natif et personnalisable.", Accent: "#bd79d5", NeedsAssets: true, Installable: true, ExecutableHint: "2ship"},
	{ID: "sm64", Name: "SM64 Coop Deluxe", Repo: "coop-deluxe/sm64coopdx", Type: "Port PC", Description: "Port multijoueur de Super Mario 64 et système de mods.", Accent: "#ed7469", NeedsAssets: true, Installable: true, ExecutableHint: "sm64coopdx"},
	{ID: "zelda3", Name: "Zelda3 (A Link to the Past)", Repo: "snesrev/zelda3", Type: "Décompilation", Description: "Projet open source : compilation et ressources à préparer manuellement.", Accent: "#77b88d", NeedsAssets: true, Installable: false},
}
var storeState struct {
	sync.Mutex
	list    []Project
	fetched time.Time
	task    storeTask
	active  bool
}

func storeClient() *http.Client {
	return &http.Client{Timeout: 35 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 8 || req.URL.Scheme != "https" {
			return errors.New("Redirection non autorisée")
		}
		host := strings.ToLower(req.URL.Hostname())
		if host != "github.com" && host != "api.github.com" && !strings.HasSuffix(host, ".githubusercontent.com") {
			return errors.New("Hôte non autorisé: " + host)
		}
		return nil
	}}
}
func assetFor(p Project, release gitRelease) (string, int64) {
	if !p.Installable {
		return "", 0
	}
	for _, a := range release.Assets {
		n := strings.ToLower(a.Name)
		if !strings.HasSuffix(n, ".zip") || strings.Contains(n, "source") || strings.Contains(n, "debug") {
			continue
		}
		if !(strings.Contains(n, "windows") || strings.Contains(n, "win64")) {
			continue
		}
		if a.Size <= 0 || a.Size > 350<<20 {
			continue
		}
		parsed, err := url.Parse(a.URL)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "github.com" || !strings.HasPrefix(parsed.Path, "/"+p.Repo+"/releases/download/") {
			continue
		}
		return a.URL, a.Size
	}
	return "", 0
}
func releaseFor(p Project) (gitRelease, error) {
	var rel gitRelease
	urlString := "https://api.github.com/repos/" + p.Repo + "/releases/latest"
	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		return rel, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "RecompHub/3.0")
	res, err := storeClient().Do(req)
	if err != nil {
		return rel, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return rel, fmt.Errorf("GitHub HTTP %d", res.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(&rel); err != nil {
		return rel, err
	}
	if rel.TagName == "" {
		return rel, errors.New("Aucune version disponible")
	}
	return rel, nil
}
func installedProject(id string) bool {
	root := filepath.Join(state.dir, "store", id, "releases")
	dirs, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, d := range dirs {
		if d.IsDir() {
			return true
		}
	}
	return false
}
func installedVersion(id, tag string) bool {
	if tag == "" {
		return installedProject(id)
	}
	clean := safeTag.ReplaceAllString(tag, "_")
	s, err := os.Stat(filepath.Join(state.dir, "store", id, "releases", clean))
	return err == nil && s.IsDir()
}
func getCatalog(force bool) []Project {
	storeState.Lock()
	if !force && len(storeState.list) > 0 && time.Since(storeState.fetched) < 10*time.Minute {
		r := append([]Project(nil), storeState.list...)
		storeState.Unlock()
		for i := range r {
			r[i].Installed = installedVersion(r[i].ID, r[i].Tag)
		}
		return r
	}
	storeState.Unlock()
	out := append([]Project(nil), catalog...)
	var wg sync.WaitGroup
	for i := range out {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p := &out[i]
			if !p.Installable {
				p.Error = "Source uniquement"
				return
			}
			rel, err := releaseFor(*p)
			if err != nil {
				p.Error = err.Error()
				return
			}
			p.Tag = rel.TagName
			_, size := assetFor(*p, rel)
			p.Size = size
			p.Available = size > 0
			if !p.Available {
				p.Error = "Aucun ZIP Windows détecté"
			}
		}(i)
	}
	wg.Wait()
	for i := range out {
		out[i].Installed = installedVersion(out[i].ID, out[i].Tag)
	}
	storeState.Lock()
	storeState.list = append([]Project(nil), out...)
	storeState.fetched = time.Now()
	storeState.Unlock()
	return out
}
func setTask(fn func(*storeTask)) { storeState.Lock(); fn(&storeState.task); storeState.Unlock() }
func apiStore(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/store/discover":
		if r.Method != "GET" {
			fail(w, 405, "GET requis")
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(q) < 2 || len(q) > 60 {
			fail(w, 400, "Recherche de 2 à 60 caractères requise")
			return
		}
		repos, err := discoverRepositories(q)
		if err != nil {
			fail(w, 502, err.Error())
			return
		}
		jsonOut(w, map[string]any{"ok": true, "repositories": repos})
	case "/api/store/catalog":
		if r.Method != "GET" {
			fail(w, 405, "GET requis")
			return
		}
		jsonOut(w, map[string]any{"ok": true, "projects": getCatalog(r.URL.Query().Get("refresh") == "1")})
	case "/api/store/progress":
		storeState.Lock()
		task := storeState.task
		storeState.Unlock()
		jsonOut(w, map[string]any{"ok": true, "task": task})
	case "/api/store/install":
		if r.Method != "POST" {
			fail(w, 405, "POST requis")
			return
		}
		var data struct {
			ID string `json:"id"`
		}
		if err := reqBody(r, &data); err != nil {
			fail(w, 400, "Requête invalide")
			return
		}
		var project *Project
		for i := range catalog {
			if catalog[i].ID == data.ID {
				project = &catalog[i]
			}
		}
		if project == nil || !project.Installable {
			fail(w, 400, "Ce projet n'est pas installable automatiquement")
			return
		}
		storeState.Lock()
		if storeState.active {
			storeState.Unlock()
			fail(w, 409, "Un téléchargement est déjà en cours")
			return
		}
		storeState.active = true
		storeState.task = storeTask{Project: data.ID, Status: "loading", Message: "Recherche de la dernière version..."}
		storeState.Unlock()
		go func(p Project) {
			err := installProject(p)
			storeState.Lock()
			storeState.active = false
			if err != nil {
				storeState.task.Status = "error"
				storeState.task.Error = err.Error()
				storeState.task.Message = "Installation échouée"
			} else {
				storeState.task.Status = "complete"
				storeState.task.Message = "Téléchargement et extraction terminés"
			}
			storeState.Unlock()
		}(*project)
		jsonOut(w, map[string]any{"ok": true, "started": true})
	default:
		fail(w, 404, "Introuvable")
	}
}

type discoveredRepo struct {
	Name        string `json:"name"`
	Repo        string `json:"repo"`
	Description string `json:"description"`
	Stars       int    `json:"stars"`
	URL         string `json:"url"`
}

func discoverRepositories(query string) ([]discoveredRepo, error) {
	values := url.Values{}
	values.Set("q", query+" in:name,description")
	values.Set("sort", "stars")
	values.Set("per_page", "12")
	req, err := http.NewRequest("GET", "https://api.github.com/search/repositories?"+values.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "RecompHub/3.0")
	res, err := storeClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub Search HTTP %d (limite de recherches possible)", res.StatusCode)
	}
	var result struct {
		Items []struct {
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			Stars       int    `json:"stargazers_count"`
			HTML        string `json:"html_url"`
			Archived    bool   `json:"archived"`
		} `json:"items"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&result); err != nil {
		return nil, err
	}
	out := []discoveredRepo{}
	for _, r := range result.Items {
		parsed, err := url.Parse(r.HTML)
		if err != nil || parsed.Scheme != "https" || parsed.Hostname() != "github.com" || r.Archived {
			continue
		}
		if len(r.Description) > 300 {
			r.Description = r.Description[:300]
		}
		out = append(out, discoveredRepo{Name: r.FullName, Repo: r.FullName, Description: r.Description, Stars: r.Stars, URL: r.HTML})
	}
	return out, nil
}

var safeTag = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func installProject(p Project) error {
	rel, err := releaseFor(p)
	if err != nil {
		return err
	}
	asset, size := assetFor(p, rel)
	if asset == "" {
		return errors.New("Pas d'archive Windows officielle pour cette version")
	}
	tag := safeTag.ReplaceAllString(rel.TagName, "_")
	if tag == "" || tag == "." || tag == ".." {
		return errors.New("Tag invalide")
	}
	root := filepath.Join(state.dir, "store", p.ID, "releases")
	dest := filepath.Join(root, tag)
	if _, err := os.Stat(dest); err == nil {
		return errors.New("Version déjà téléchargée : ouvre le dossier dans ta bibliothèque")
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(root, ".recomphub-stage-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	zipPath := filepath.Join(stage, "package.zip")
	req, _ := http.NewRequest("GET", asset, nil)
	req.Header.Set("User-Agent", "RecompHub/3.0")
	c := storeClient()
	c.Timeout = 20 * time.Minute
	res, err := c.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("Téléchargement GitHub : HTTP %d", res.StatusCode)
	}
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	setTask(func(t *storeTask) {
		t.Status = "download"
		t.Message = "Téléchargement depuis GitHub..."
		t.Total = size
	})
	buf := make([]byte, 64*1024)
	var written int64
	for {
		n, e := res.Body.Read(buf)
		if n > 0 {
			written += int64(n)
			if written > 350<<20 {
				_ = f.Close()
				return errors.New("Archive supérieure à 350 Mo")
			}
			if _, werr := f.Write(buf[:n]); werr != nil {
				_ = f.Close()
				return werr
			}
			setTask(func(t *storeTask) { t.Done = written })
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			_ = f.Close()
			return e
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	setTask(func(t *storeTask) { t.Status = "extract"; t.Message = "Extraction de l'archive..." })
	unpack := filepath.Join(stage, "app")
	if err := os.MkdirAll(unpack, 0700); err != nil {
		return err
	}
	if err := extractSafeZip(zipPath, unpack); err != nil {
		return err
	}
	exe := findStoreExecutable(unpack, p.ExecutableHint)
	if err := os.Rename(unpack, dest); err != nil {
		return err
	}
	if exe != "" {
		final := filepath.Join(dest, strings.TrimPrefix(exe, unpack+string(os.PathSeparator)))
		state.Lock()
		added := addGameLocked(final)
		if added {
			g := findGameLocked(state.lib.Games[len(state.lib.Games)-1].ID)
			if g != nil {
				g.Name = p.Name
				g.Category = p.Type
				if g.Category == "Décompilation" {
					g.Category = "Port PC"
				}
				g.Notes = "Installé depuis GitHub : " + p.Repo + " (" + rel.TagName + "). Données du jeu non incluses."
			}
		}
		err = saveLocked()
		state.Unlock()
		if err != nil {
			return err
		}
		setTask(func(t *storeTask) { t.GameAdded = added })
	}
	if exe == "" {
		setTask(func(t *storeTask) {
			t.Message = "Archives extraites, aucun exécutable reconnu : ajoute-le manuellement"
		})
	}
	return nil
}
func extractSafeZip(archive, destination string) error {
	f, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	if len(f.File) > 4000 {
		return errors.New("Archive trop volumineuse en nombre de fichiers")
	}
	var total int64
	var actualTotal int64
	for _, file := range f.File {
		if file.Mode()&os.ModeSymlink != 0 {
			return errors.New("Archive contenant un lien symbolique non autorisé")
		}
		if file.FileInfo().Mode()&os.ModeType != 0 && !file.FileInfo().IsDir() {
			return errors.New("Type de fichier non autorisé")
		}
		name := strings.ReplaceAll(file.Name, "\\", "/")
		if strings.HasPrefix(name, "/") || strings.Contains(name, ":") || strings.HasPrefix(name, "../") || strings.Contains(name, "/../") || name == ".." {
			return errors.New("Chemin d'archive suspect")
		}
		target := filepath.Join(destination, filepath.FromSlash(name))
		rel, err := filepath.Rel(destination, target)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return errors.New("Tentative de sortie du répertoire")
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0700); err != nil {
				return err
			}
			continue
		}
		if file.UncompressedSize64 > 600<<20 {
			return errors.New("Un fichier décompressé dépasse 600 Mo")
		}
		total += int64(file.UncompressedSize64)
		if total > 1100<<20 {
			return errors.New("Archive décompressée supérieure à 1,1 Go")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
		if err != nil {
			in.Close()
			return err
		}
		n, copyErr := io.Copy(out, io.LimitReader(in, 600<<20+1))
		closeErr := out.Close()
		in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if n > 600<<20 {
			return errors.New("Fichier extrait trop grand")
		}
		actualTotal += n
		if actualTotal > 1100<<20 {
			return errors.New("Archive trop volumineuse après extraction")
		}
	}
	return nil
}
func findStoreExecutable(root, hint string) string {
	best := ""
	score := -100
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".exe") {
			return nil
		}
		name := strings.ToLower(filepath.Base(path))
		if strings.Contains(name, "unins") || strings.Contains(name, "setup") || strings.Contains(name, "updater") || strings.Contains(name, "crash") {
			return nil
		}
		s := 0
		if strings.Contains(name, strings.ToLower(hint)) {
			s += 100
		}
		if strings.Contains(name, "launcher") {
			s += 5
		}
		if len(strings.Split(strings.TrimPrefix(path, root), string(os.PathSeparator))) < 4 {
			s += 3
		}
		if s > score {
			score = s
			best = path
		}
		return nil
	})
	return best
}
