const express = require("express");
const promClient = require("prom-client");
const { NodeTracerProvider } = require("@opentelemetry/sdk-trace-node");
const { SimpleSpanProcessor } = require("@opentelemetry/sdk-trace-base");
const { OTLPTraceExporter } = require("@opentelemetry/exporter-trace-otlp-grpc");

const app = express();

// Prometheus counter
const counter = new promClient.Counter({
  name: "http_requests_total",
  help: "Total HTTP requests",
});

// OpenTelemetry setup — OTLP/gRPC, matching python-service
const provider = new NodeTracerProvider();
const exporter = new OTLPTraceExporter({
  url: "http://jaeger:4317",
});
provider.addSpanProcessor(new SimpleSpanProcessor(exporter));
provider.register();
const tracer = provider.getTracer("node-service");

app.get("/node", (req, res) => {
  counter.inc();
  const span = tracer.startSpan("hello-handler");
  res.send("Hello Node");
  span.end();
});

// Health route
app.get("/health", (req, res) => {
  res.status(200).send("OK");
});

// Metrics endpoint
app.get("/metrics", async (req, res) => {
  res.set("Content-Type", promClient.register.contentType);
  res.end(await promClient.register.metrics());
});

// Graceful shutdown — flush any pending spans before exit
process.on("SIGTERM", async () => {
  await provider.shutdown();
  process.exit(0);
});

app.listen(4000, () => console.log("Node.js service running on port 4000"));
