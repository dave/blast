package blast

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

func (b *Blaster) loadPreviousLogs() error {
	logFile, err := os.Open(b.config.Log)
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
		fmt.Fprintf(b.out, "Logs are %v MB, loading can take some time...\n", fs.Size()/(1<<20))
	}

	if err := b.loadPreviousLogsFromReader(logFile); err != nil {
		return err
	}

	return nil
}

func (b *Blaster) loadPreviousLogsFromReader(r io.Reader) error {
	logReader := csv.NewReader(r)
	// skip header
	if _, err := logReader.Read(); err != nil {
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
		if err := (&lr).FromCsv(record); err != nil {
			return err
		}
		if lr.Result {
			b.skip[lr.Hash] = struct{}{}
		}
	}
	return nil
}

func (b *Blaster) openLogAndInit() error {

	if b.config.Resume {
		if err := b.loadPreviousLogs(); err != nil {
			return err
		}
	}

	if !b.config.Resume {
		_ = os.Remove(b.config.Log) // ignore error
	}

	logFile, err := os.OpenFile(b.config.Log, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return errors.WithStack(err)
	}
	s, err := logFile.Stat()
	if err != nil {
		return errors.WithStack(err)
	}
	b.logWriter = csv.NewWriter(logFile)
	b.logCloser = logFile
	if s.Size() == 0 {
		fields := []string{"hash", "result"}
		fields = append(fields, b.config.LogData...)
		fields = append(fields, b.config.LogOutput...)
		if err := b.logWriter.Write(fields); err != nil {
			return errors.WithStack(err)
		}
	} else {
		logFile.WriteString("\n")
	}

	return nil
}

func (b *Blaster) flushAndCloseLog() {
	b.logWriter.Flush()
	_ = b.logCloser.Close() // ignore error
}

type logRecord struct {
	Hash   farmhash.Uint128
	Result bool
	Fields []string
}

func (l logRecord) ToCsv() []string {
	out := []string{
		fmt.Sprintf("%x|%x", l.Hash.First, l.Hash.Second),
		fmt.Sprint(l.Result),
	}
	return append(out, l.Fields...)
}

func (l *logRecord) FromCsv(in []string) error {
	var err error
	s := in[0]
	pos := strings.Index(s, "|")
	l.Hash.First, err = strconv.ParseUint(s[:pos], 16, 64)
	if err != nil {
		return errors.WithStack(err)
	}
	l.Hash.Second, err = strconv.ParseUint(s[pos+1:], 16, 64)
	if err != nil {
		return errors.WithStack(err)
	}
	l.Result, err = strconv.ParseBool(in[1])
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
