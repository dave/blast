package gcsworker

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSend(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Unexpected method: %s", r.Method)
		}
	}))
	defer ts.Close()

	payload := map[string]interface{}{
		"method": "GET",
		"url":    ts.URL,
	}
	w := New()
	w.(*Worker).client = http.DefaultClient
	response, err := w.Send(context.Background(), payload)
	expected := map[string]interface{}{
		"status": 200,
	}
	if err != nil {
		log.Fatal(err)
	}
	if !reflect.DeepEqual(response, expected) {
		t.Fatalf("Unexpected: %#v", response)
	}
}

func TestPost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Unexpected method: %s", r.Method)
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(b) != "abc" {
			t.Errorf("Unexpected body: %s", string(b))
		}
	}))
	defer ts.Close()

	payload := map[string]interface{}{
		"method": "POST",
		"url":    ts.URL,
		"body":   "abc",
	}
	w := New()
	w.(*Worker).client = http.DefaultClient
	response, err := w.Send(context.Background(), payload)
	expected := map[string]interface{}{
		"status": 200,
	}
	if err != nil {
		log.Fatal(err)
	}
	if !reflect.DeepEqual(response, expected) {
		t.Fatalf("Unexpected: %#v", response)
	}
}

func TestHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Unexpected method: %s", r.Method)
		}
		if r.Header.Get("a") != "b" || r.Header.Get("c") != "d" {
			t.Errorf("Unexpected: a=%s, c=%s", r.Header.Get("a"), r.Header.Get("c"))
		}
	}))
	defer ts.Close()

	payload := map[string]interface{}{
		"method": "GET",
		"url":    ts.URL,
		"headers": map[string]string{
			"a": "b",
			"c": "d",
		},
	}
	w := New()
	w.(*Worker).client = http.DefaultClient
	response, err := w.Send(context.Background(), payload)
	expected := map[string]interface{}{
		"status": 200,
	}
	if err != nil {
		log.Fatal(err)
	}
	if !reflect.DeepEqual(response, expected) {
		t.Fatalf("Unexpected: %#v", response)
	}
}

func TestError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - error"))
	}))
	defer ts.Close()

	payload := map[string]interface{}{
		"method": "GET",
		"url":    ts.URL,
	}
	w := New()
	w.(*Worker).client = http.DefaultClient
	response, err := w.Send(context.Background(), payload)
	expected := map[string]interface{}{
		"status": 500,
	}
	if err == nil || err.Error() != "non 200 status" {
		log.Fatalf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(response, expected) {
		t.Fatalf("Unexpected: %#v", response)
	}
}

func TestErrorDecodingPayload(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	payload := map[string]interface{}{
		"method": 1,
	}
	w := New()
	w.(*Worker).client = http.DefaultClient
	response, err := w.Send(context.Background(), payload)
	expected := map[string]interface{}{
		"status": "Error decoding payload",
	}
	if err == nil || !strings.Contains(err.Error(), "'method' expected type 'string', got unconvertible type 'int'") {
		log.Fatalf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(response, expected) {
		t.Fatalf("Unexpected: %#v", response)
	}
}

func TestErrorCreatingRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	payload := map[string]interface{}{
		"method": " ",
	}
	w := New()
	w.(*Worker).client = http.DefaultClient
	response, err := w.Send(context.Background(), payload)
	expected := map[string]interface{}{
		"status": "Error creating request",
	}
	if err == nil || !strings.Contains(err.Error(), "invalid method") {
		log.Fatalf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(response, expected) {
		t.Fatalf("Unexpected: %#v", response)
	}
}

func TestUrlError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	payload := map[string]interface{}{
		"method": "a",
	}
	w := New()
	w.(*Worker).client = http.DefaultClient
	response, err := w.Send(context.Background(), payload)
	expected := map[string]interface{}{
		"status": "unsupported protocol scheme \"\"",
	}
	if err == nil || !strings.Contains(err.Error(), "a : unsupported protocol scheme") {
		log.Fatalf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(response, expected) {
		t.Fatalf("Unexpected: %#v", response)
	}
}

func TestErrorTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-time.After(time.Second)
	}))
	defer ts.Close()

	payload := map[string]interface{}{
		"method": "GET",
		"url":    ts.URL,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	w := New()
	w.(*Worker).client = http.DefaultClient
	response, err := w.Send(ctx, payload)
	expected := map[string]interface{}{
		"status": "Timeout",
	}
	if err == nil || !strings.HasSuffix(err.Error(), "context deadline exceeded") {
		log.Fatalf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(response, expected) {
		t.Fatalf("Unexpected: %#v", response)
	}
	cancel()
}

func TestErrorCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-time.After(time.Second)
	}))
	defer ts.Close()

	payload := map[string]interface{}{
		"method": "GET",
		"url":    ts.URL,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	go func() {
		cancel()
	}()
	w := New()
	w.(*Worker).client = http.DefaultClient
	response, err := w.Send(ctx, payload)
	expected := map[string]interface{}{
		"status": "Cancelled",
	}
	if err == nil || !strings.HasSuffix(err.Error(), "context canceled") {
		log.Fatalf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(response, expected) {
		t.Fatalf("Unexpected: %#v", response)
	}
}
