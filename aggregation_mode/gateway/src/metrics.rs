use actix_web::{web, App, HttpResponse, HttpServer, Responder};
use prometheus::{self, histogram_opts, Encoder, Histogram, Registry, TextEncoder};
use std::sync::Arc;

#[derive(Clone, Debug)]
pub struct GatewayMetrics {
    pub registry: Registry,
    pub time_elapsed_db_post: Histogram,
}

impl GatewayMetrics {
    pub fn start(metrics_port: u16) -> Result<Self, prometheus::Error> {
        let registry = Registry::new();

        let time_elapsed_db_post = Histogram::with_opts(histogram_opts!(
            "time_elapsed_db_post",
            "Time elapsed in DB posts"
        ))?;

        registry.register(Box::new(time_elapsed_db_post.clone()))?;

        // Arc is used because metrics are a shared resource accessed by both the background and metrics HTTP
        // server and the application code, across multiple Actix worker threads. The server outlives start(),
        // so the data must be static and safely shared between threads.
        let metrics = Arc::new(Self {
            registry,
            time_elapsed_db_post,
        });

        let server_metrics = metrics.clone();
        tokio::spawn(async move {
            let _ = HttpServer::new(move || {
                App::new()
                    .app_data(web::Data::new(server_metrics.clone()))
                    .route("/metrics", web::get().to(GatewayMetrics::metrics_handler))
            })
            .bind(("0.0.0.0", metrics_port))
            .expect("failed to bind metrics server")
            .run()
            .await;
        });

        Ok(Arc::try_unwrap(metrics).unwrap_or_else(|arc| (*arc).clone()))
    }

    async fn metrics_handler(metrics: web::Data<Arc<GatewayMetrics>>) -> impl Responder {
        let encoder = TextEncoder::new();
        let metric_families = metrics.registry.gather();

        let mut buffer = Vec::new();
        if let Err(e) = encoder.encode(&metric_families, &mut buffer) {
            tracing::error!("could not encode prometheus metrics: {e}");
        }

        HttpResponse::Ok()
            .insert_header(("Content-Type", encoder.format_type()))
            .body(buffer)
    }

    pub fn register_db_response_time_post(&self, value: f64) {
        self.time_elapsed_db_post.observe(value);
    }
}
