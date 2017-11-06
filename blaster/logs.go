package blaster

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"

	"strings"

	"github.com/leemcloughlin/gofarmhash"
	"github.com/pkg/errors"
)

func (b *Blaster) SetLog(w io.Writer) {
	if w == nil {
		b.logWriter = nil
		b.logCloser = nil
		return
	}
	b.logWriter = csv.NewWriter(w)
	if c, ok := w.(io.Closer); ok {
		b.logCloser = c
	} else {
		b.logCloser = nil
	}
}

func (b *Blaster) WriteLogHeaders() error {
	fields := []string{"hash", "result"}
	fields = append(fields, b.LogData...)
	fields = append(fields, b.LogOutput...)
	if err := b.logWriter.Write(fields); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (b *Blaster) LoadLogs(r io.Reader) error {
	logReader := csv.NewReader(r)
	if _, err := logReader.Read(); err != nil {
		// skip header
		if err == io.EOF {
			return nil
		}
		return errors.WithStack(err)
	}
	for {
		record, err := logReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.WithStack(err)
		}
		var lr logRecord
		if err := (&lr).fromCsv(record); err != nil {
			return err
		}
		if lr.result {
			b.skip[lr.hash] = struct{}{}
		}
	}
	return nil
}

func (b *Blaster) initialiseLog(log string) error {

	if log == "" {
		return nil
	}

	if b.Resume {
		if err := b.openAndLoadLogs(log); err != nil {
			return err
		}
	}

	if !b.Resume {
		_ = os.Remove(log) // ignore error
	}

	logFile, err := os.OpenFile(log, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return errors.WithStack(err)
	}

	s, err := logFile.Stat()
	if err != nil {
		return errors.WithStack(err)
	}

	b.SetLog(logFile)

	if s.Size() == 0 {
		if err := b.WriteLogHeaders(); err != nil {
			return err
		}
	} else {
		// TODO: Is this needed?
		logFile.WriteString("\n")
	}

	return nil
}

func (b *Blaster) openAndLoadLogs(log string) error {
	logFile, err := os.Open(log)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.WithStack(err)
	}
	defer logFile.Close()

	fs, err := logFile.Stat()
	if err != nil {
		return errors.WithStack(err)
	}

	if fs.Size() == 0 {
		return nil
	}

	if fs.Size() > 1<<20 {
		b.printf("Logs are %v MB, loading can take some time...\n", fs.Size()/(1<<20))
	}

	if err := b.LoadLogs(logFile); err != nil {
		return err
	}

	return nil
}

type logRecord struct {
	hash   farmhash.Uint128
	result bool
	fields []string
}

func (l logRecord) toCsv() []string {
	out := []string{
		fmt.Sprintf("%x|%x", l.hash.First, l.hash.Second),
		fmt.Sprint(l.result),
	}
	return append(out, l.fields...)
}

func (l *logRecord) fromCsv(in []string) error {
	var err error
	s := in[0]
	pos := strings.Index(s, "|")
	l.hash.First, err = strconv.ParseUint(s[:pos], 16, 64)
	if err != nil {
		return errors.WithStack(err)
	}
	l.hash.Second, err = strconv.ParseUint(s[pos+1:], 16, 64)
	if err != nil {
		return errors.WithStack(err)
	}
	l.result, err = strconv.ParseBool(in[1])
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
