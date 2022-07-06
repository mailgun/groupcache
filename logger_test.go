package groupcache

import (
	"bytes"
	"errors"
	"github.com/sirupsen/logrus"
	"testing"
)

// This tests the compatibility of the LogrusLogger with the previous behavior.
func TestLogrusLogger(t *testing.T) {
	var buf bytes.Buffer
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	l.Out = &buf
	e := logrus.NewEntry(l)
	e = e.WithField("ContextKey", "ContextVal")
	SetLogger(e)
	logger.Error().
		WithFields(map[string]interface{}{
			"err":      errors.New("test error"),
			"key":      "keyValue",
			"category": "groupcache",
		}).Printf("error retrieving key from peer %s", "http://127.0.0.1:8080")

	interfaceOut := buf.String()
	buf.Reset()
	e.WithFields(logrus.Fields{
		"err":      errors.New("test error"),
		"key":      "keyValue",
		"category": "groupcache",
	}).Errorf("error retrieving key from peer %s", "http://127.0.0.1:8080")
	logrusOut := buf.String()
	if interfaceOut != logrusOut {
		t.Errorf("output is not the same.\ngot:\n%s\nwant:\n%s", interfaceOut, logrusOut)
	}
}

func BenchmarkLogrusLogger(b *testing.B) {
	var buf bytes.Buffer
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	l.Out = &buf
	e := logrus.NewEntry(l)
	SetLogger(e)
	for i := 0; i < b.N; i++ {
		logger.Error().
			WithFields(map[string]interface{}{
				"err":      errors.New("test error"),
				"key":      "keyValue",
				"category": "groupcache",
			}).Printf("error retrieving key from peer %s", "http://127.0.0.1:8080")
		buf.Reset()
	}
}

func BenchmarkLogrus(b *testing.B) {
	var buf bytes.Buffer
	l := logrus.New()
	l.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	l.Out = &buf
	e := logrus.NewEntry(l)
	for i := 0; i < b.N; i++ {
		e.WithFields(logrus.Fields{
			"err":      errors.New("test error"),
			"key":      "keyValue",
			"category": "groupcache",
		}).Errorf("error retrieving key from peer %s", "http://127.0.0.1:8080")
		buf.Reset()
	}
}
