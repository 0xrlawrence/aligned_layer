use prometheus::{self, histogram_opts, register_histogram};
use warp::{reject::Rejection, reply::Reply, Filter};

#[derive(Clone, Debug)]
pub struct GatewayMetrics {
    pub time_elapsed_db_post: prometheus::Histogram,
}

impl GatewayMetrics {
    pub fn start(metrics_port: u16) -> anyhow::Result<Self> {
        let registry = prometheus::Registry::new();

        let time_elapsed_db_post = register_histogram!(histogram_opts!(
            "time_elapsed_db_post",
            "Time elapsed in DB posts"
        ))?;

        registry.register(Box::new(time_elapsed_db_post.clone()))?;

        let metrics_route = warp::path!("metrics")
            .and(warp::any().map(move || registry.clone()))
            .and_then(GatewayMetrics::metrics_handler);

        tokio::task::spawn(async move {
            warp::serve(metrics_route)
                .run(([0, 0, 0, 0], metrics_port))
                .await;
        });

        Ok(Self {
            time_elapsed_db_post,
        })
    }

    pub async fn metrics_handler(registry: prometheus::Registry) -> Result<impl Reply, Rejection> {
        use prometheus::Encoder;
        let encoder = prometheus::TextEncoder::new();

        let mut buffer = Vec::new();
        if let Err(e) = encoder.encode(&registry.gather(), &mut buffer) {
            tracing::error!("could not encode prometheus metrics: {e}");
        };
        let res = String::from_utf8(buffer.clone())
            .inspect_err(|e| eprintln!("prometheus metrics could not be parsed correctly: {e}"))
            .unwrap_or_default();
        buffer.clear();

        Ok(res)
    }

    pub fn register_db_response_time_post(&self, value: f64) {
        self.time_elapsed_db_post.observe(value);
    }
}
