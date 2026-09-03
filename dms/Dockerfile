# Build a single static binary, ship it in a minimal distroless image.
FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /inh-dms .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /inh-dms /inh-dms
# /data holds config.json, envelope.asc (GPG ciphertext) and state.json.
VOLUME ["/data"]
ENTRYPOINT ["/inh-dms", "--config", "/data/config.json"]
