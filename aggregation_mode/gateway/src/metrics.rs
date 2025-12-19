use prometheus::{self, opts, register_int_counter, IntCounter};
use warp::{reject::Rejection, reply::Reply, Filter};

#[derive(Clone, Debug)]
pub struct GatewayMetrics {
    pub success_response: IntCounter,
    pub server_error_response: IntCounter,
    pub user_error_response: IntCounter,
}

impl GatewayMetrics {
    pub fn start(metrics_port: u16) -> anyhow::Result<Self> {
        let registry = prometheus::Registry::new();

        let success_response =
            register_int_counter!(opts!("success_response_count", "Success Responses"))?;

        let server_error_response =
            register_int_counter!(opts!("server_error_response_count", "Success Responses"))?;

        let user_error_response =
            register_int_counter!(opts!("user_error_response_count", "Success Responses"))?;

        registry.register(Box::new(success_response.clone()))?;
        registry.register(Box::new(server_error_response.clone()))?;
        registry.register(Box::new(user_error_response.clone()))?;

        let metrics_route = warp::path!("metrics")
            .and(warp::any().map(move || registry.clone()))
            .and_then(GatewayMetrics::metrics_handler);

        tokio::task::spawn(async move {
            warp::serve(metrics_route)
                .run(([0, 0, 0, 0], metrics_port))
                .await;
        });

        Ok(Self {
            success_response,
            server_error_response,
            user_error_response,
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

    pub fn inc_success_response(&self) {
        self.success_response.inc();
    }

    pub fn inc_server_error_response(&self) {
        self.server_error_response.inc();
    }

    pub fn inc_user_error_response(&self) {
        self.user_error_response.inc();
    }
}
