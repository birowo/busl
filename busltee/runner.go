package busltee

import (
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// Config holds the runner configuration
type Config struct {
	Insecure      bool
	Timeout       float64
	Retry         int
	StreamRetry   int
	SleepDuration time.Duration
	URL           string
	Args          []string
	LogPrefix     string
	LogFile       string
	RequestID     string
}

// Run creates the stdin listener and forwards logs to URI
func Run(url string, args []string, conf *Config) (exitCode int) {
	defer monitor("busltee.busltee", time.Now())

	reader, writer := io.Pipe()
	done := post(url, reader, conf)

	if err := run(args, writer, writer); err != nil {
		log.Printf("count#busltee.exec.error=1 error=%v", err.Error())
		exitCode = exitStatus(err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		log.Printf("count#busltee.exec.upload.timeout=1")
	}

	return exitCode
}

func monitor(subject string, ts time.Time) {
	log.Printf("%s.time time=%f", subject, time.Now().Sub(ts).Seconds())
}

func post(url string, reader io.Reader, conf *Config) chan struct{} {
	done := make(chan struct{})

	go func() {
		if err := stream(url, reader, conf); err != nil {
			log.Printf("count#busltee.stream.error=1 error=%v", err.Error())
			// Prevent writes from blocking.
			io.Copy(ioutil.Discard, reader)
		} else {
			log.Printf("count#busltee.stream.success=1")
		}
		close(done)
	}()

	return done
}

func stream(url string, stdin io.Reader, conf *Config) (err error) {
	for retries := conf.Retry; retries >= 0; retries-- {
		if err = streamNoRetry(url, stdin, conf); !isTimeout(err) {
			return err
		}
		log.Printf("count#busltee.stream.retry")
	}
	return err
}

var errMissingURL = errors.New("Missing URL")

func streamNoRetry(url string, stdin io.Reader, conf *Config) error {
	defer monitor("busltee.stream", time.Now())

	if url == "" {
		log.Printf("count#busltee.stream.missingurl")
		return errMissingURL
	}

	client := &http.Client{Transport: newTransport(conf)}

	// In the event that the `busl` connection doesn't work,
	// we still need to proceed with the command's execution.
	// For this reason, we wrap `stdin` in NopCloser to prevent
	// it from being closed prematurely (and thus allowing writes
	// on the other end of the pipe to work).
	req, err := http.NewRequest("POST", url, ioutil.NopCloser(stdin))
	if conf.RequestID != "" {
		req.Header.Set("Request-Id", conf.RequestID)
	}

	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	return err
}

func newTransport(conf *Config) http.RoundTripper {
	tr := &http.Transport{}

	if conf.Timeout > 0 {
		tr.Dial = (&net.Dialer{
			KeepAlive: 30 * time.Second,
			Timeout:   time.Duration(conf.Timeout) * time.Second,
		}).Dial
	}

	if conf.Insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &Transport{
		Transport:     tr,
		MaxRetries:    uint(conf.StreamRetry),
		SleepDuration: conf.SleepDuration,
	}
}

func run(args []string, stdout, stderr io.WriteCloser) error {
	defer stdout.Close()
	defer stderr.Close()
	defer monitor("busltee.run", time.Now())

	cmd := exec.Command(args[0], args[1:]...)

	errCh, err := attachCmd(cmd, io.MultiWriter(stdout, os.Stdout), io.MultiWriter(stderr, os.Stderr))
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Catch any signals sent to busltee, and pass those along.
	deliverSignals(cmd)

	state, err := cmd.Process.Wait()

	var copyErr error
	select {
	case copyErr = <-errCh:
	}

	if err != nil {
		return err
	} else if !state.Success() {
		return &exec.ExitError{ProcessState: state}
	}

	return copyErr
}

func attachCmd(cmd *exec.Cmd, stdout, stderr io.Writer) (<-chan error, error) {
	ch := make(chan error)
	errCh := make(chan error, 2)

	copyFunc := func(w io.Writer, r io.ReadCloser) {
		_, err := io.Copy(w, r)
		r.Close()
		errCh <- err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	go copyFunc(stdout, stdoutPipe)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	go copyFunc(stderr, stderrPipe)

	go func() {
		var copyErr error
		for i := 0; i < 2; i++ {
			if err := <-errCh; err != nil && copyErr == nil {
				copyErr = err
			}
		}
		if copyErr != nil {
			ch <- copyErr
		}
		close(ch)
	}()

	return ch, nil
}

func deliverSignals(cmd *exec.Cmd) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc)
	go func() {
		s := <-sigc
		cmd.Process.Signal(s)
	}()
}

func isTimeout(err error) bool {
	e, ok := err.(net.Error)
	return ok && e.Timeout()
}

func exitStatus(err error) int {
	if exit, ok := err.(*exec.ExitError); ok {
		if status, ok := exit.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}
	// Default to exit status 1 if we can't type assert the error.
	return 1
}
