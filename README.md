# Taskflow – Distributed Task Processing with OpenTelemetry

Taskflow is a small distributed system built with **Go and Python** to demonstrate **zero-code (automatic) OpenTelemetry instrumentation** using the OpenTelemetry Collector and Jaeger.

The project is intentionally simple in business logic but realistic in architecture, mirroring how observability is applied in real backend systems.

---

## Architecture Overview

Taskflow consists of three main parts:

1. **task-api-go**
   A Go HTTP API that accepts tasks from clients.

2. **task-worker-py**
   A Python service that processes and enriches tasks.

3. **Observability Stack**
   OpenTelemetry Collector and Jaeger for receiving, processing, and visualizing traces.

High-level flow:

Client
→ Go API (`task-api-go`)
→ Python Worker (`task-worker-py`)
→ OpenTelemetry Collector
→ Jaeger UI

All telemetry is exported using **OTLP**.

---

## Services

### task-api-go (Go)

What it does:

* Exposes HTTP endpoints to create and list tasks.
* Calls the Python worker to enrich tasks.
* Stores tasks in memory.

Endpoints:

* `POST /tasks` – Create a new task
* `GET /tasks` – List all tasks
* `GET /health` – Health check

Instrumentation:

* Incoming HTTP requests are automatically instrumented using `otelhttp`.
* Outgoing HTTP calls to the Python worker are automatically instrumented.
* No manual spans are created.

---

### task-worker-py (Python)

What it does:

* Receives tasks from the Go service.
* Simulates processing latency.
* Returns enrichment data such as priority and score.

Endpoints:

* `POST /enrich` – Enrich a task
* `GET /health` – Health check
* `GET /` – Basic service response

Instrumentation:

* Automatically instrumented using `opentelemetry-instrument`.
* Flask instrumentation enabled.
* OTLP exporter sends traces to the Collector.

---

### Observability Stack

Components:

* **OpenTelemetry Collector**
* **Jaeger**

Collector:

* Receives OTLP traces on ports 4317 (gRPC) and 4318 (HTTP).
* Exports traces to Jaeger.
* Logs received traces using the debug exporter.

Jaeger:

* Receives OTLP data from the Collector.
* Provides a web UI for trace visualization.

Jaeger UI:

* [http://localhost:16686](http://localhost:16686)

---

## Folder Structure

```
taskflow/
├── task-api-go/
│   ├── main.go
│   ├── go.mod
│   └── go.sum
│
├── task-worker-py/
│   ├── app.py
│   ├── requirements.txt
│   └── .venv/
│
├── observability/
│   ├── docker-compose.yml
│   └── otelcol-config.yml
│
├── .gitignore
└── README.md
```

---

## Running the System (Windows)

### 1. Start Observability Stack

From the `observability` folder:

```
docker-compose up -d
```

Verify:

* OpenTelemetry Collector is running on port 4317
* Jaeger UI is available at [http://localhost:16686](http://localhost:16686)

---

### 2. Run Python Worker

From `task-worker-py`:

Activate virtual environment:

```
.\.venv\Scripts\activate
```

Run with auto-instrumentation:

```
opentelemetry-instrument python app.py
```

Service runs on:

* [http://localhost:8090](http://localhost:8090)

---

### 3. Run Go API

From `task-api-go`:

```
go run .
```

Service runs on:

* [http://localhost:8081](http://localhost:8081)

---

## Using the System

### Create a Task

PowerShell:

```
$body = @{ text = "Learn OpenTelemetry" } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri http://localhost:8081/tasks -ContentType application/json -Body $body
```

### List Tasks

```
Invoke-RestMethod -Method Get -Uri http://localhost:8081/tasks
```

---

## Viewing Traces

1. Open Jaeger UI:
   [http://localhost:16686](http://localhost:16686)

2. Select service:

   * `task-api-go`
   * `task-worker-py`

3. Run requests again.

You will see:

* Distributed traces spanning Go and Python
* Parent-child relationships across services
* Latency introduced by enrichment

---

## Observability Goals

This project demonstrates:

* Zero-code instrumentation in Go and Python
* Cross-service distributed tracing
* OTLP-based telemetry flow
* Collector-based architecture
* Realistic debugging and troubleshooting scenarios

---

## Next Steps

Planned improvements:

* Manual spans for critical business logic
* Custom attributes on spans
* Error and retry analysis
* Metrics and logs integration
* Trace interpretation and latency analysis

---

## License

This project is for learning and experimentation purposes.
