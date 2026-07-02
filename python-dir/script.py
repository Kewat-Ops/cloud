from flask import Flask
import prometheus_client
from prometheus_client import Counter
from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
import atexit

app = Flask(__name__)
FlaskInstrumentor().instrument_app(app)

# Prometheus counter
http_requests_total = Counter("http_requests_total", "Total HTTP requests")

# OpenTelemetry setup
provider = TracerProvider()
trace.set_tracer_provider(provider)
tracer = trace.get_tracer(__name__)

# OTLP exporter (Jaeger accepts OTLP/gRPC on 4317)
otlp_exporter = OTLPSpanExporter(endpoint="http://jaeger:4317", insecure=True)
provider.add_span_processor(SimpleSpanProcessor(otlp_exporter))

# Flush any pending spans on shutdown
atexit.register(lambda: provider.shutdown())


@app.route("/python")
def python():
    http_requests_total.inc()
    with tracer.start_as_current_span("hello-handler"):
        return "Hello Python"


@app.route("/metrics")
def metrics():
    return prometheus_client.generate_latest(), 200, {"Content-Type": prometheus_client.CONTENT_TYPE_LATEST}


@app.route("/health")
def health():
    return "OK", 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000)
