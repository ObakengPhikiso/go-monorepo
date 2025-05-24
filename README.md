# Go Microservices Monorepo

## Local Development

- **Start all services:**  
  ```sh
  docker-compose up --build
  ```
- **Run a single service:**  
  ```sh
  cd services/users
  go run main.go
  ```

## API Gateway

- The API Gateway is available at `http://localhost:8088`.
- It proxies requests to the appropriate service:
  - `/users`, `/users/{id}` → users service
  - `/orders`, `/orders/{id}` → orders service
  - `/payments`, `/payments/{id}` → payments service
- Example usage:
  ```sh
  curl http://localhost:8088/users
  curl -X POST http://localhost:8088/orders -d '{"amount":"$300"}' -H 'Content-Type: application/json'
  ```
- **Swagger/OpenAPI docs:**
  - View interactive API docs at: [http://localhost:8088/swagger](http://localhost:8088/swagger)
  - Raw OpenAPI YAML: [http://localhost:8088/swagger.yaml](http://localhost:8088/swagger.yaml)

## Adding a New Service

1. Create a new directory under `services/yourservice`.
2. Run `go mod init github.com/ObakengPhikiso/monorepo/services/yourservice`.
3. Add to `go.work`:
    ```
    use (
      ...
      ./services/yourservice
    )
    ```
4. Add a Dockerfile and .dockerignore.
5. Update `docker-compose.yml` if needed.

## Dependency Management

- Use the shared library via `require github.com/ObakengPhikiso/monorepo/libs/shared vX.Y.Z`.
- For local dev, `replace` with relative path.
- Tag shared lib releases:  
  ```sh
  cd libs/shared
  git tag v0.2.0
  git push origin v0.2.0
  ```

## Deployment Workflow

- On push to `main`, only changed services are built and deployed.
- Docker images are pushed to GHCR.
- Railway CLI is used for deployment.
- Automatic version tags are created for traceability.

---

**Best Practices:**
- Keep each service independent.
- Use the shared library for common code.
- Use health checks and environment variables for robust orchestration.
- Use Go workspace for local development and easy dependency management.
