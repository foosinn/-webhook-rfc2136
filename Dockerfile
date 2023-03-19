FROM golang AS builder

WORKDIR /app
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w"

# --

FROM busybox

COPY --from=builder /app/webhook-rfc2136 /server
CMD /server
