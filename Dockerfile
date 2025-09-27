FROM golang:1.23-alpine
WORKDIR /app/backend

COPY backend/ .
RUN go build -o /app/cronnect

COPY frontend/index.html /app/frontend/index.html

EXPOSE 8080
CMD ["/app/cronnect"]
