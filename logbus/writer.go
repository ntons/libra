package main

import (
	"bytes"
	"context"
	"path/filepath"

	trf "github.com/ntons/log-go/appenders/timedrollingfile"
)

type Writer interface {
	Write(ctx context.Context, v ...string) (err error)
}

func NewWriter(key string) Writer {
	// multiple writer can be combined to writerarray here
	var a WriterArray
	a = append(a, NewFileWriter(&cfg.TimedRollingFile, key))
	return a
}

// combine multiple writers as one
type WriterArray []Writer

func (a WriterArray) Write(ctx context.Context, v ...string) (err error) {
	for _, w := range a {
		if err = w.Write(ctx, v...); err != nil {
			return
		}
	}
	return
}

// FileWriter write logs into timed rolling files
type FileWriter struct{ a *trf.Appender }

func NewFileWriter(cfg *TimedRollingFileConfig, key string) Writer {
	return FileWriter{
		a: &trf.Appender{
			MaxSize:            cfg.MaxSize,
			MaxBackups:         cfg.MaxBackups,
			LocalTime:          cfg.LocalTime,
			Compress:           cfg.Compress,
			BackupTimeFormat:   cfg.BackupTimeFormat,
			FilenameTimeFormat: filepath.Join(cfg.Datastore, key, cfg.FilenameTimeFormat),
			DisableMutex:       true,
			LogTime:            GetTimeFromLog,
		},
	}
}

func (w FileWriter) Write(ctx context.Context, v ...string) (err error) {
	buflen := 0
	for _, s := range v {
		buflen += len(s) + 1
	}
	buf := bytes.NewBuffer(make([]byte, 0, buflen))
	for _, s := range v {
		buf.WriteString(s)
		buf.WriteString("\n")
	}
	if _, err = w.a.Write(buf.Bytes()); err != nil {
		return
	}
	return
}
