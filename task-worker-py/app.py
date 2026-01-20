from flask import Flask, request, jsonify
import random
import time

app = Flask(__name__)

@app.route('/')
def home():
    return "Welcome to the Task Enrichment Service! Go to /enrich to enrich tasks."

@app.post("/enrich")
def enrich():
    data = request.get_json(silent=True) or {}
    text = data.get("text", "")
    if not text:
        return jsonify({"error": "missing text"}), 400

    if "fail" in text.lower():
        return jsonify({"error": "forced failure"}), 500

    time.sleep(random.uniform(0.02, 0.20))

    priority = random.randint(1, 5)
    score = round(random.random(), 3)

    return jsonify({
        "priority": priority,
        "score": score,
        "status": "enriched"
    }), 200

@app.get("/health")
def health():
    return "ok", 200

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8090)

