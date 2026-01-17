FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy prebuilt ARM64 binary from CI
COPY dist/stocks-service /app/stocks-service

# Copy migrations if needed at runtime
COPY db/migrations /app/db/migrations

EXPOSE 8081

USER nonroot:nonroot

ENTRYPOINT ["/app/stocks-service"]
