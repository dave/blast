package blast

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"

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
		fmt.Printf("Logs are %v MB, loading can take some time...\n", fs.Size()/(1<<20))
	}

	logReader := csv.NewReader(logFile)
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
			if b.skip == nil {
				b.skip = make(map[string]struct{})
			}
			b.skip[lr.PayloadHash] = struct{}{}
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

	var err error
	if b.logFile, err = os.OpenFile(b.config.Log, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666); err != nil {
		return errors.WithStack(err)
	}
	s, err := b.logFile.Stat()
	if err != nil {
		return errors.WithStack(err)
	}
	b.logWriter = csv.NewWriter(b.logFile)
	if s.Size() == 0 {
		if err := b.logWriter.Write([]string{"PayloadHash", "Result"}); err != nil {
			return errors.WithStack(err)
		}
	} else {
		b.logFile.WriteString("\n")
	}

	return nil
}

func (b *Blaster) flushAndCloseLog() {
	b.logWriter.Flush()
	_ = b.logFile.Close() // ignore error
}

type logRecord struct {
	PayloadHash string
	Result      bool
}

func (l logRecord) ToCsv() []string {
	return []string{
		fmt.Sprint(l.PayloadHash),
		fmt.Sprint(l.Result),
	}
}

func (l *logRecord) FromCsv(in []string) error {
	l.PayloadHash = in[0]
	result, err := strconv.ParseBool(in[1])
	if err != nil {
		return errors.WithStack(err)
	}
	l.Result = result
	return nil
}
