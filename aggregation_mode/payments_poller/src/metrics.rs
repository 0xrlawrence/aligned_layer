use prometheus::{self, opts, register_gauge};
use warp::{reject::Rejection, reply::Reply, Filter};

#[derive(Clone, Debug)]
pub struct PaymentsPollerMetrics {
    pub last_processed_block: prometheus::Gauge,
}

impl PaymentsPollerMetrics {
    pub fn start(metrics_port: u16) -> anyhow::Result<Self> {
        let registry = prometheus::Registry::new();

        let last_processed_block = register_gauge!(opts!(
            "last_processed_block",
            "Last processed block by poller"
        ))?;

        registry.register(Box::new(last_processed_block.clone()))?;

        let metrics_route = warp::path!("metrics")
            .and(warp::any().map(move || registry.clone()))
            .and_then(PaymentsPollerMetrics::metrics_handler);

        tokio::task::spawn(async move {
            warp::serve(metrics_route)
                .run(([0, 0, 0, 0], metrics_port))
                .await;
        });

        Ok(Self {
            last_processed_block,
        })
    }

    pub async fn metrics_handler(registry: prometheus::Registry) -> Result<impl Reply, Rejection> {
        use prometheus::Encoder;
        let encoder = prometheus::TextEncoder::new();

        let mut buffer = Vec::new();
        if let Err(e) = encoder.encode(&registry.gather(), &mut buffer) {
            eprintln!("could not encode prometheus metrics: {}", e);
        };
        let res = String::from_utf8(buffer.clone())
            .inspect_err(|e| eprintln!("prometheus metrics could not be parsed correctly: {e}"))
            .unwrap_or_default();
        buffer.clear();

        Ok(res)
    }

    pub fn register_last_processed_block(&self, value: u64) {
        self.last_processed_block.set(value as f64);
    }
}
