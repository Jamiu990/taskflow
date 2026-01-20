package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Task struct {
	ID       int     `json:"id"`
	Text     string  `json:"text"`
	Priority int     `json:"priority"`
	Score    float64 `json:"score"`
	Status   string  `json:"status"`
}

type store struct {
	mu    sync.Mutex
	next  int
	tasks []Task
}

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint("127.0.0.1:4317"),
	)
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", "task-api"),
			attribute.String("service.version", "0.1.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

func (s *store) listTasks(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.tasks)
}

func (s *store) createTask(workerURL string, client *http.Client) http.HandlerFunc {
	type reqBody struct {
		Text string `json:"text"`
	}
	type enrichReq struct {
		Text string `json:"text"`
	}
	type enrichResp struct {
		Priority int     `json:"priority"`
		Score    float64 `json:"score"`
		Status   string  `json:"status"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Text == "" {
			http.Error(w, "invalid json, expected {\"text\":\"...\"}", http.StatusBadRequest)
			return
		}

		enReq := enrichReq{Text: body.Text}
		b, _ := json.Marshal(enReq)

		req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, workerURL+"/enrich", bytes.NewReader(b))
		if err != nil {
			http.Error(w, "failed to build worker request", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "worker call failed", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			http.Error(w, "worker returned non-200", http.StatusBadGateway)
			return
		}

		var enResp enrichResp
		if err := json.NewDecoder(resp.Body).Decode(&enResp); err != nil {
			http.Error(w, "failed to decode worker response", http.StatusBadGateway)
			return
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		s.next++
		t := Task{
			ID:       s.next,
			Text:     body.Text,
			Priority: enResp.Priority,
			Score:    enResp.Score,
			Status:   enResp.Status,
		}
		s.tasks = append(s.tasks, t)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(t)
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func main() {
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()

	tp, err := initTracer(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	workerURL := "http://localhost:8090"

	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		Timeout:   3 * time.Second,
	}

	s := &store{}

	http.Handle("/tasks", otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			s.listTasks(w, r)
		case http.MethodPost:
			s.createTask(workerURL, client)(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}), "/tasks"))

	http.Handle("/health", otelhttp.NewHandler(http.HandlerFunc(health), "/health"))

	fmt.Println("task-api listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
