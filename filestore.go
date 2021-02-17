package web

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"sync"
	"time"
)

// FileStore allows you to send files from an arbitrary file system.
// Additionally, it computes and caches ETags based on the file contents.
// ETags are cached by filename and updated when the modification time changes.
type FileStore struct {
	fsys       fs.FS
	etagsMutex sync.Mutex
	etags      map[string]etagInfo
}

type etagInfo struct {
	ModTime time.Time
	Tag     string
}

func NewFileStore(fsys fs.FS) *FileStore {
	return &FileStore{
		fsys:  fsys,
		etags: make(map[string]etagInfo),
	}
}

func NewFileStoreFromDirectory(dirname string) *FileStore {
	return NewFileStore(os.DirFS(dirname))
}

// SendFile answers the request by sending a file.
// If the file cannot be found or opened it returns an appropriate HTTPError.
// Otherwise it calls http.ServeContent() internally.
// Note that the filename is not sanitized in any way and passed directly to fsys.Open().
// However, if you are using http.ServeMux, it should have already sanitized
// the URL request path, so you can safely construct the filename from that.
func (store *FileStore) SendFile(w http.ResponseWriter, r *http.Request, filename string) error {
	file, err := store.fsys.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return NewHTTPError(http.StatusNotFound)
		} else {
			return NewHTTPErrorWithInternalError(http.StatusNotFound, err)
		}
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return NewHTTPErrorWithInternalError(http.StatusInternalServerError, err)
	}
	if info.IsDir() {
		return NewHTTPError(http.StatusNotFound)
	}

	reader, ok := file.(io.ReadSeeker)
	if !ok {
		// Slow path, read whole file into memory, so that we can seek.
		// Both os.File and embed.File implement the Seeker interface.
		contents := make([]byte, info.Size())
		_, err = io.ReadFull(file, contents)
		if err != nil {
			return NewHTTPErrorWithInternalError(http.StatusInternalServerError, err)
		}
		reader = bytes.NewReader(contents)
	}

	store.etagsMutex.Lock()
	tagInfo, ok := store.etags[filename]
	store.etagsMutex.Unlock()

	if !ok || !tagInfo.ModTime.Equal(info.ModTime()) {
		hasher := sha1.New()
		_, err := io.Copy(hasher, reader)
		if err != nil {
			return NewHTTPErrorWithInternalError(http.StatusInternalServerError, err)
		}

		_, err = reader.Seek(0, io.SeekStart)
		if err != nil {
			return NewHTTPErrorWithInternalError(http.StatusInternalServerError, err)
		}

		// Tag must be in double quotes.
		// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/ETag
		tagInfo.ModTime = info.ModTime()
		tagInfo.Tag = fmt.Sprintf(`"%s"`, hex.EncodeToString(hasher.Sum(nil)))

		store.etagsMutex.Lock()
		store.etags[filename] = tagInfo
		store.etagsMutex.Unlock()
	}

	// ETag is handled by ServeContent.
	w.Header().Set("ETag", tagInfo.Tag)

	http.ServeContent(w, r, info.Name(), info.ModTime(), reader)
	return nil
}
