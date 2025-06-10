FROM ghcr.io/yetanotherco/aligned_layer/aligned_base:latest AS base

RUN apt update -y && apt install -y gcc

# Install SP1 toolchain
RUN curl -L https://sp1up.succinct.xyz | bash -s -- -y
ENV PATH="/root/.sp1/bin:${PATH}"
RUN sp1up

# Install Risc0 toolchain
RUN curl -L https://risczero.com/install | bash
ENV PATH="/root/.risc0/bin:${PATH}"
RUN rzup install

COPY crates /aligned_layer/crates/
COPY aggregation_mode /aligned_layer/aggregation_mode/
WORKDIR /aligned_layer

RUN IN_DOCKER=true cargo build --manifest-path ./aggregation_mode/Cargo.toml --features prove --release --bin proof_aggregator_cpu

FROM debian:bookworm-slim AS final

COPY --from=base /aligned_layer/aggregation_mode/target/release/proof_aggregator_cpu /aligned_layer/proof_aggregator_cpu
COPY ./config-files/config-proof-aggregator-docker.yaml ./config-files/
COPY ./config-files/anvil.proof-aggregator.ecdsa.key.json ./config-files/

RUN apt update -y && apt install -y libssl-dev ca-certificates
