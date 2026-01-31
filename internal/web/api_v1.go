package web

import (
	"archive/zip"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type apiError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type okResponse struct {
	OK bool `json:"ok"`
}

type cartridgeInfoResponse struct {
	Present    bool     `json:"present"`
	Mounted    bool     `json:"mounted"`
	IsRetroPie bool     `json:"isRetroPie"`
	Systems    []string `json:"systems"`
	Busy       bool     `json:"busy"`
}

func apiV1Router(ejectFunc func(ctx context.Context) error, flashFunc func(ctx context.Context, reader io.Reader) error) http.Handler {
	// Backwards-compatible defaults: keep the existing device behavior
	// (mount via scripts, roms under /cartridge/...) unless an entrypoint
	// registers routes with explicit deps.
	deps := NewDeviceAPIV1Deps(nil).withDefaults()
	return apiV1RouterWithDeps(APIV1Handlers{EjectFunc: ejectFunc, FlashFunc: flashFunc}, deps)
}

func apiV1RouterWithDeps(handlers APIV1Handlers, deps APIV1Deps) http.Handler {
	deps = deps.withDefaults()
	mux := http.NewServeMux()
	mux.HandleFunc("/cartridgeinfo", func(w http.ResponseWriter, r *http.Request) { handleCartridgeInfo(w, r, deps) })
	mux.HandleFunc("/retropie", func(w http.ResponseWriter, r *http.Request) { handleRetroPie(w, r, deps) })
	mux.HandleFunc("/retropie/", func(w http.ResponseWriter, r *http.Request) { handleRetroPie(w, r, deps) })
	mux.HandleFunc("/eject", func(w http.ResponseWriter, r *http.Request) {
		handleEject(w, r, deps, handlers.EjectFunc)
	})
	mux.HandleFunc("/flash", func(w http.ResponseWriter, r *http.Request) {
		handleFlash(w, r, deps, handlers.FlashFunc)
	})
	return mux
}

func handleEject(w http.ResponseWriter, r *http.Request, deps APIV1Deps, ejectFunc func(ctx context.Context) error) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if ejectFunc == nil {
		writeAPIError(w, http.StatusNotImplemented, "not_implemented", "eject not configured")
		return
	}

	snap := deps.Cartridge.Snapshot()
	if snap.Busy {
		writeAPIError(w, http.StatusConflict, "cartridge_busy", "cartridge is busy")
		return
	}
	if !snap.Present {
		writeAPIError(w, http.StatusConflict, "no_cartridge", "no cartridge present")
		return
	}

	if err := ejectFunc(r.Context()); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "eject_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, okResponse{OK: true})
}

func handleFlash(w http.ResponseWriter, r *http.Request, deps APIV1Deps, flashFunc func(ctx context.Context, reader io.Reader) error) {
	if r.Method != http.MethodPost {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if flashFunc == nil {
		writeAPIError(w, http.StatusNotImplemented, "not_implemented", "flash not configured")
		return
	}
	if err := requireContentLength(r); err != nil {
		writeAPIError(w, http.StatusLengthRequired, "length_required", err.Error())
		return
	}

	snap := deps.Cartridge.Snapshot()
	if snap.Busy {
		writeAPIError(w, http.StatusConflict, "cartridge_busy", "cartridge is busy")
		return
	}
	if !snap.Present {
		writeAPIError(w, http.StatusConflict, "no_cartridge", "no cartridge present")
		return
	}

	// Stream the body directly into the flashing pipeline.
	limitedBody := io.LimitReader(r.Body, r.ContentLength)
	if err := flashFunc(r.Context(), limitedBody); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "flash_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, okResponse{OK: true})
}

func handleRetroPie(w http.ResponseWriter, r *http.Request, deps APIV1Deps) {
	// Step 3: GET /retropie -> systems list (from CartridgeInfo snapshot)
	// Step 4: GET /retropie/{system} -> game list (requires mounted cartridge)
	// Step 5: GET /retropie/{system}/{game} -> download game bytes (zip folder if needed)
	// Step 6: POST /retropie/{system}/{game} -> upload a game (unzip if {game} ends with .zip)
	path := r.URL.Path
	if !strings.HasPrefix(path, "/retropie") {
		writeAPIError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	snap := deps.Cartridge.Snapshot()
	if snap.Busy {
		writeAPIError(w, http.StatusConflict, "cartridge_busy", "cartridge is busy")
		return
	}
	if !snap.Present {
		writeAPIError(w, http.StatusConflict, "no_cartridge", "no cartridge present")
		return
	}
	if !snap.IsRetroPie {
		writeAPIError(w, http.StatusConflict, "not_retropie", "cartridge is not a RetroPie cartridge")
		return
	}

	// /retropie or /retropie/
	if path == "/retropie" || path == "/retropie/" {
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		writeJSON(w, http.StatusOK, snap.Systems)
		return
	}

	// /retropie/{system} (only)
	rel := strings.TrimPrefix(path, "/retropie/")
	rel = strings.Trim(rel, "/")
	if rel == "" {
		writeJSON(w, http.StatusOK, snap.Systems)
		return
	}

	parts := strings.Split(rel, "/")
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		systemName := parts[0]
		if !containsString(snap.Systems, systemName) {
			writeAPIError(w, http.StatusNotFound, "system_not_found", "system not found")
			return
		}

		if err := deps.Mounter.EnsureMounted(r.Context()); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "mount_failed", err.Error())
			return
		}

		games, err := deps.RetroPie.ListGames(r.Context(), systemName)
		if err != nil {
			if errorsIsNotExist(err) {
				writeAPIError(w, http.StatusNotFound, "system_not_found", "system not found")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "list_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, games)
		return
	}

	if len(parts) == 2 {
		systemName := parts[0]
		gameName := parts[1]
		if !containsString(snap.Systems, systemName) {
			writeAPIError(w, http.StatusNotFound, "system_not_found", "system not found")
			return
		}
		if strings.TrimSpace(gameName) == "" || gameName == "." || gameName == ".." {
			writeAPIError(w, http.StatusBadRequest, "invalid_game", "invalid game")
			return
		}

		if err := deps.Mounter.EnsureMounted(r.Context()); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "mount_failed", err.Error())
			return
		}

		switch r.Method {
		case http.MethodGet:
			if err := deps.RetroPie.DownloadGame(r.Context(), w, r, systemName, gameName); err != nil {
				if errorsIsNotExist(err) {
					writeAPIError(w, http.StatusNotFound, "game_not_found", "game not found")
					return
				}
				writeAPIError(w, http.StatusInternalServerError, "download_failed", err.Error())
				return
			}
			return
		case http.MethodPost:
			if err := requireContentLength(r); err != nil {
				writeAPIError(w, http.StatusLengthRequired, "length_required", err.Error())
				return
			}
			if err := deps.RetroPie.UploadGame(r.Context(), systemName, gameName, r.Body, r.ContentLength); err != nil {
				if errorsIsNotExist(err) {
					writeAPIError(w, http.StatusNotFound, "system_not_found", "system not found")
					return
				}
				writeAPIError(w, http.StatusInternalServerError, "upload_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, okResponse{OK: true})
			return
		case http.MethodDelete:
			if err := deps.RetroPie.DeleteGame(r.Context(), systemName, gameName); err != nil {
				if errorsIsNotExist(err) {
					writeAPIError(w, http.StatusNotFound, "game_not_found", "game not found")
					return
				}
				writeAPIError(w, http.StatusInternalServerError, "delete_failed", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, okResponse{OK: true})
			return
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
	}

	// Reserved for later steps.
	writeAPIError(w, http.StatusNotFound, "not_found", "not found")
}

func listGamesForSystem(romsRoot, systemName string) ([]string, error) {
	romDir := filepath.Join(romsRoot, systemName)
	entries, err := os.ReadDir(romDir)
	if err != nil {
		return nil, err
	}
	games := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		if strings.HasPrefix(name, ".") {
			continue
		}
		games = append(games, name)
	}
	sort.Strings(games)
	return games, nil
}

func downloadGame(romsRoot string, w http.ResponseWriter, r *http.Request, systemName, gameName string) error {
	gamePath := filepath.Join(romsRoot, systemName, gameName)
	info, err := os.Stat(gamePath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		downloadName := gameName
		if !strings.HasSuffix(strings.ToLower(downloadName), ".zip") {
			downloadName += ".zip"
		}
		setDownloadHeaders(w, downloadName, "application/zip")
		return streamZipDir(w, gamePath, filepath.Base(gamePath))
	}

	// Serve file directly.
	f, err := os.Open(gamePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	setDownloadHeaders(w, gameName, "application/octet-stream")
	http.ServeContent(w, r, gameName, info.ModTime(), f)
	return nil
}

func setDownloadHeaders(w http.ResponseWriter, filename, contentType string) {
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	cd := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	w.Header().Set("Content-Disposition", cd)
}

func streamZipDir(w io.Writer, dirPath, baseFolder string) error {
	zipWriter := zip.NewWriter(w)
	defer func() { _ = zipWriter.Close() }()

	parent := filepath.Dir(dirPath)
	return filepath.WalkDir(dirPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		relToParent, err := filepath.Rel(parent, path)
		if err != nil {
			return err
		}
		relToParent = filepath.ToSlash(relToParent)
		// Ensure the zip has a stable top-level folder.
		if baseFolder != "" {
			// If the rel path already includes baseFolder (it should), keep it.
			// This check just avoids accidental double-prefixing.
			if !strings.HasPrefix(relToParent, baseFolder+"/") {
				relToParent = filepath.ToSlash(filepath.Join(baseFolder, relToParent))
			}
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = relToParent
		hdr.Method = zip.Deflate

		zw, err := zipWriter.CreateHeader(hdr)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { _ = src.Close() }()
		_, err = io.Copy(zw, src)
		return err
	})
}

func requireContentLength(r *http.Request) error {
	// Reject chunked/unknown length. We need Content-Length to avoid buffering.
	if r.ContentLength <= 0 {
		return errLengthRequired
	}
	return nil
}

var errLengthRequired = &apiSimpleError{Message: "Content-Length header is required"}

type apiSimpleError struct{ Message string }

func (e *apiSimpleError) Error() string { return e.Message }

func uploadGame(romsRoot, systemName, gameName string, body io.Reader, contentLength int64) error {
	romSystemDir := filepath.Join(romsRoot, systemName)
	if _, err := os.Stat(romSystemDir); err != nil {
		return err
	}

	isZip := strings.HasSuffix(strings.ToLower(gameName), ".zip")
	if !isZip {
		targetPath := filepath.Join(romSystemDir, gameName)
		_ = os.RemoveAll(targetPath)
		return writeStreamToFile(targetPath, body, contentLength)
	}

	baseName := gameName[:len(gameName)-4]
	baseName = strings.TrimSpace(baseName)
	if baseName == "" {
		baseName = "game"
	}

	// Store the uploaded zip temporarily on the cartridge (not in RAM).
	tmpName := ".upload-" + strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + sanitizeFilename(gameName)
	tmpZipPath := filepath.Join(romSystemDir, tmpName)
	if err := writeStreamToFile(tmpZipPath, body, contentLength); err != nil {
		_ = os.Remove(tmpZipPath)
		return err
	}
	defer func() { _ = os.Remove(tmpZipPath) }()

	destDir := filepath.Join(romSystemDir, baseName)
	_ = os.RemoveAll(destDir)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	if err := unzipToDir(tmpZipPath, destDir); err != nil {
		return err
	}
	return flattenSingleTopLevelDir(destDir)
}

func writeStreamToFile(targetPath string, src io.Reader, expectedBytes int64) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	// Limit to Content-Length bytes so we never read beyond what the client declared.
	written, err := io.Copy(f, io.LimitReader(src, expectedBytes))
	if err != nil {
		return err
	}
	if written != expectedBytes {
		return fmtUnexpectedLength(written, expectedBytes)
	}
	return nil
}

func fmtUnexpectedLength(written, expected int64) error {
	return &apiLengthError{Written: written, Expected: expected}
}

type apiLengthError struct {
	Written  int64
	Expected int64
}

func (e *apiLengthError) Error() string {
	return "unexpected request body length (written=" + strconv.FormatInt(e.Written, 10) + ", expected=" + strconv.FormatInt(e.Expected, 10) + ")"
}

func unzipToDir(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	for _, f := range r.File {
		name := filepath.ToSlash(f.Name)
		name = strings.TrimPrefix(name, "/")
		if name == "" {
			continue
		}
		// Zip Slip protection.
		clean := filepath.ToSlash(filepath.Clean(name))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "..") {
			return os.ErrInvalid
		}
		if filepath.IsAbs(filepath.FromSlash(clean)) {
			return os.ErrInvalid
		}

		targetPath := filepath.Join(destDir, filepath.FromSlash(clean))
		rel, err := filepath.Rel(destDir, targetPath)
		if err != nil {
			return err
		}
		if strings.HasPrefix(rel, "..") {
			return os.ErrInvalid
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		src, err := f.Open()
		if err != nil {
			return err
		}
		dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			_ = src.Close()
			return err
		}
		_, copyErr := io.Copy(dst, src)
		_ = dst.Close()
		_ = src.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func flattenSingleTopLevelDir(destDir string) error {
	entries, err := os.ReadDir(destDir)
	if err != nil {
		return err
	}
	var nonHidden []os.DirEntry
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		nonHidden = append(nonHidden, e)
	}
	if len(nonHidden) != 1 {
		return nil
	}
	if !nonHidden[0].IsDir() {
		return nil
	}

	subDir := filepath.Join(destDir, nonHidden[0].Name())
	subEntries, err := os.ReadDir(subDir)
	if err != nil {
		return err
	}
	for _, se := range subEntries {
		oldPath := filepath.Join(subDir, se.Name())
		newPath := filepath.Join(destDir, se.Name())
		_ = os.RemoveAll(newPath)
		if err := os.Rename(oldPath, newPath); err != nil {
			return err
		}
	}
	return os.Remove(subDir)
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "upload"
	}
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "..", "_")
	return name
}

func deleteGame(romsRoot, systemName, gameName string) error {
	gamePath := filepath.Join(romsRoot, systemName, gameName)
	info, err := os.Stat(gamePath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return os.RemoveAll(gamePath)
	}
	return os.Remove(gamePath)
}

func containsString(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func errorsIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}

func handleCartridgeInfo(w http.ResponseWriter, r *http.Request, deps APIV1Deps) {
	if r.Method != http.MethodGet {
		writeAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	snap := deps.Cartridge.Snapshot()
	resp := cartridgeInfoResponse{
		Present:    snap.Present,
		Mounted:    snap.Mounted,
		IsRetroPie: snap.IsRetroPie,
		Systems:    snap.Systems,
		Busy:       snap.Busy,
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, apiError{Error: code, Message: message})
}
