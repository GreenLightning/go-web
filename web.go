package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func SendFile(w http.ResponseWriter, r *http.Request, filename string) error {
	file, err := os.Open(filename)
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
	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	return nil
}

func SendRedirect(w http.ResponseWriter, r *http.Request, statusCode int, url string) error {
	if statusCode < 300 || statusCode > 399 {
		return fmt.Errorf("redirect status code should be in 3xx range, but was %d", statusCode)
	}
	http.Redirect(w, r, url, statusCode)
	return nil
}

func SendTemplate(w http.ResponseWriter, r *http.Request, statusCode int, renderer *Renderer, name string, data interface{}) error {
	buffer := new(bytes.Buffer)
	err := renderer.Render(buffer, name, data)
	if err != nil {
		return err
	}
	return SendBLOB(w, r, statusCode, "text/html; charset=UTF-8", buffer.Bytes())
}

func SendJSON(w http.ResponseWriter, r *http.Request, statusCode int, value interface{}) error {
	pretty := false // @Todo: Set on debug.

	var data []byte
	var err error
	if pretty {
		data, err = json.MarshalIndent(value, "", "\t")
	} else {
		data, err = json.Marshal(value)
	}
	if err != nil {
		return err
	}

	return SendBLOB(w, r, statusCode, "application/json; charset=UTF-8", data)
}

func SendBLOB(w http.ResponseWriter, r *http.Request, statusCode int, contentType string, data []byte) error {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)
	_, err := w.Write(data)
	if err != nil {
		// Do not return this error, as we have already committed the response.
		log.Println("[web][error] failed to write data:", err)
	}
	return nil
}
