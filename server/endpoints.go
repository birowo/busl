package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/heroku/busl/broker"
	"github.com/heroku/busl/util"
	"github.com/heroku/rollbar"
)

func (s *Server) put(w http.ResponseWriter, r *http.Request) {
	registrar := broker.NewRedisRegistrar()

	if err := registrar.Register(key(r)); err != nil {
		http.Error(w, "Unable to create stream. Please try again.", http.StatusServiceUnavailable)
		rollbar.Error(rollbar.ERR, fmt.Errorf("unable to register stream: %#v", err))
		util.CountWithData("put.create.fail", 1, "error=%s", err)
		return
	}
	util.Count("put.create.success")
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func (s *Server) pub(w http.ResponseWriter, r *http.Request) {
	if !util.StringInSlice(r.TransferEncoding, "chunked") {
		http.Error(w, "A chunked Transfer-Encoding header is required.", http.StatusBadRequest)
		return
	}

	writer, err := broker.NewWriter(key(r))
	if err != nil {
		handleError(w, r, err)
		return
	}
	defer writer.Close()

	body := bufio.NewReader(r.Body)
	defer r.Body.Close()

	_, err = io.Copy(writer, body)

	if err == io.ErrUnexpectedEOF {
		util.CountWithData("server.pub.read.eoferror", 1, "msg=\"%v\"", err.Error())
		return
	}

	netErr, ok := err.(net.Error)
	if ok && netErr.Timeout() {
		util.CountWithData("server.pub.read.timeout", 1, "msg=\"%v\"", err.Error())
		handleError(w, r, netErr)
		return
	}

	if err != nil {
		log.Printf("%#v", err)
		http.Error(w, "Unhandled error, please try again.", http.StatusInternalServerError)
		rollbar.Error(rollbar.ERR, fmt.Errorf("unhandled error: %#v", err))
		return
	}

	// Asynchronously upload the output to our defined storage backend.
	go storeOutput(key(r), requestURI(r), s.StorageBaseURL)
}

func (s *Server) sub(w http.ResponseWriter, r *http.Request) {
	if _, ok := w.(http.Flusher); !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	rd, err := s.newReader(w, r)
	if rd != nil {
		defer rd.Close()
	}
	if err != nil {
		handleError(w, r, err)
		return
	}
	_, err = io.Copy(newWriteFlusher(w), rd)

	netErr, ok := err.(net.Error)
	if ok && netErr.Timeout() {
		util.CountWithData("server.sub.read.timeout", 1, "msg=\"%v\"", err.Error())
		return
	}

	if err != nil {
		rollbar.Error(rollbar.ERR, fmt.Errorf("unhandled error: %#v", err))
	}
}
