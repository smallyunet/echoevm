FROM rust:1.95.0-bookworm AS build
WORKDIR /src
COPY Cargo.toml Cargo.lock rust-toolchain.toml ./
COPY crates ./crates
RUN cargo build --locked --release -p echoevm

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates wget && rm -rf /var/lib/apt/lists/*
COPY --from=build /src/target/release/echoevm /usr/local/bin/echoevm
COPY deploy/docker-compose.yml deploy/Caddyfile deploy/deploy-image.sh /usr/local/share/echoevm/deploy/
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/echoevm"]
CMD ["web", "--addr", "0.0.0.0:8080", "--code", "00"]
