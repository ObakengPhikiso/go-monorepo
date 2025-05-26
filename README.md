# Go Microservices Monorepo

## Local Development

- **Prerequisites:**
  - Docker and Docker Compose
  - Go 1.24 or later
  - MongoDB (for local development without Docker)

- **Start all services:**  
  ```sh
  docker compose up --build -d
  ```
- **Run a single service:**  
  ```sh
  cd services/orders
  go run main.go
  ```

## API Gateway

- The API Gateway is available at `http://localhost:8088`
- It proxies requests to the appropriate service:
  - `/auth/register`, `/auth/login`, `/auth/validate` → auth service
  - `/orders`, `/orders/{id}` → orders service
  - `/payments`, `/payments/{id}` → payments service

- Example usage:

```sh
# Register a new user
curl -X POST http://localhost:8088/auth/register -d '{"username":"john","password":"password123"}' -H 'Content-Type: application/json'

# Login
curl -X POST http://localhost:8088/auth/login -d '{"username":"john","password":"password123"}' -H 'Content-Type: application/json'

# Create an order (requires auth token)
curl -X POST http://localhost:8088/orders -d '{"amount":300,"items":[{"product_id":"123","quantity":1,"unit_price":300}]}' -H 'Content-Type: application/json' -H 'Authorization: Bearer <token>'
```

- **Authentication:**
  - Most endpoints require a valid JWT token in the Authorization header
  - Token format: `Bearer <token>`
  - Get token by registering or logging in via `/auth` endpoints

- **Health Checks:**
  - `/health` endpoint returns status of all services
  - Each service has its own `/health` endpoint

- **Swagger/OpenAPI docs:**
  - View interactive API docs at: [http://localhost:8088/swagger](http://localhost:8088/swagger)
  - Raw OpenAPI YAML: [http://localhost:8088/swagger.yaml](http://localhost:8088/swagger.yaml)

## Services

### Auth Service

- **Port:** 8084
- **Database:** MongoDB
- **Features:**
  - User registration and authentication
  - JWT token generation and validation
  - User management (CRUD operations)

### Orders Service

- **Port:** 8080
- **Database:** MongoDB
- **Features:**
  - Order creation and management
  - Order status updates
  - Order cancellation
  - Pagination and filtering support

### Payments Service

- **Port:** 8080
- **Database:** MongoDB

### Database Configuration

Each service connects to its own MongoDB database:

```sh
# Default MongoDB URLs (configurable via environment variables)
Auth Service:    MONGODB_URL=mongodb://mongo:27017/auth
Orders Service:  MONGODB_URL=mongodb://mongo:27017/orders
Payments Service: MONGODB_URL=mongodb://mongo:27017/payments
```

## Adding a New Service

1. Create a new directory under `services/yourservice`.
2. Run `go mod init github.com/obakengphikiso/go-monorepo/services/yourservice`.
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

- Use the shared library via `require github.com/obakengphikiso/go-monorepo/libs/shared vX.Y.Z`.
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

## Continuous Integration (CI)

The monorepo uses GitHub Actions for continuous integration, implementing an efficient build system that only builds services affected by changes.

### Workflow Overview

The CI workflow (`/.github/workflows/ci.yml`) consists of two main jobs:

1. **detect-affected-services**
   - Determines which services were affected by changes
   - Runs on every push to main and pull requests
   - Examines changes in the `services/` directory
   - Outputs a JSON array of affected service names

2. **build**
   - Builds each affected service
   - Only runs if there are affected services
   - Uses matrix strategy to build services in parallel
   - Skips removed services automatically

### How It Works

#### Change Detection

The workflow detects changes by:

- For pull requests: comparing against the base branch
- For pushes: comparing against the previous commit
- Only considers changes in the `services/` directory
- Handles service removal gracefully

Example:

```bash
# If you change files in services/auth/
→ Only builds auth service

# If you change files in services/auth/ and services/orders/
→ Builds both auth and orders services

# If you delete a service
→ Skips building the deleted service
```

#### Service Building

For each affected service:

1. Checks if the service directory exists
2. Sets up Go 1.24
3. Builds the service using `go build`
4. Runs tests if present

### CI Skip

To skip CI for a commit, include `[skip ci]` in your commit message:

```bash
git commit -m "Update docs [skip ci]"
```

### Best Practices

1. **Commit Organization**
   - Group changes by service
   - Use clear commit messages
   - Reference issues/PRs when applicable

2. **Service Dependencies**
   - Keep services independent when possible
   - Document shared dependencies
   - Use shared library for common code

3. **Testing**
   - Add tests for new features
   - Run tests locally before pushing
   - Use meaningful test names

## Development Workflow

### Setting Up Local Environment

1. **Clone the repository:**

```sh
git clone https://github.com/ObakengPhikiso/monorepo.git
cd monorepo
```

1. **Install dependencies:**
   - Docker and Docker Compose
   - Go 1.21 or later
   - MongoDB (for local development without Docker)

1. **Set up Go workspace:**

```sh
# Initialize Go workspace
go work init
go work use ./services/api-gateway ./services/auth ./services/orders ./services/payments ./libs/shared
```

1. **Set up environment variables:**

```sh
# Example .env file
JWT_SECRET=your-256-bit-secret
AUTH_DB_URL=mongodb://localhost:27017/auth
ORDERS_DB_URL=mongodb://localhost:27017/orders
PAYMENTS_DB_URL=mongodb://localhost:27017/payments
```

### Development Process

1. **Start dependencies:**

```sh
# Start MongoDB and other services
docker compose up -d mongo
```

1. **Run services locally:**

```sh
# Run auth service
cd services/auth
go run main.go

# In another terminal, run orders service
cd services/orders
go run main.go

# Run API Gateway
cd services/api-gateway
go run main.go
```

1. **Test changes:**
   - Write unit tests for new features
   - Test integration with other services
   - Verify API endpoints with Swagger UI

1. **Submit changes:**
   - Create a feature branch
   - Make your changes
   - Run tests locally
   - Submit a pull request

### Project Dependencies

1. **Core Dependencies:**
   - Go 1.24+
   - MongoDB 6.0+
   - Docker & Docker Compose

1. **Service Dependencies:**
   - **API Gateway:**
     - `libs/shared` for JWT validation and utils
   - **Auth Service:**
     - `libs/shared` for JWT generation and utils
     - MongoDB for user storage
   - **Orders Service:**
     - `libs/shared` for common utils
     - MongoDB for order storage
   - **Payments Service:**
     - `libs/shared` for common utils
     - MongoDB for payment records

1. **Shared Library (`libs/shared`):**
   - Current version: v0.1.0
   - Features:
     - JWT token generation/validation
     - Environment variable management
     - ID generation
     - Logging utilities

### Directory Structure

```plaintext
.github/
  workflows/
    ci.yml           # CI workflow definition
services/
  auth/              # Auth service
  orders/            # Orders service
  payments/          # Payments service
  api-gateway/       # API Gateway service
libs/
  shared/            # Shared library (v0.1.0)
```

### Workflow Status

The workflow status can be checked:

- On GitHub Actions tab
- In pull request checks
- Via status badges in README

### Common Issues

1. **Build Failures**
   - Check the service's logs in the build job
   - Ensure all dependencies are in go.mod
   - Verify Go version compatibility (1.21+)

2. **Service Detection Issues**
   - Ensure changes are in correct service directory
   - Check service directory exists
   - Verify file paths in error messages

3. **Matrix Build Failures**
   - Individual service failures don't fail other services
   - Check specific service build logs
   - Verify service's Dockerfile and dependencies

### Future Improvements

1. Test Coverage
   - Add service-specific tests
   - Implement integration tests
   - Add code coverage reporting

2. Performance
   - Cache Go dependencies
   - Optimize build times
   - Parallel testing

3. Quality Checks
   - Add linting
   - Static code analysis
   - Security scanning

### Contributing

When contributing:

1. Create a feature branch
2. Make your changes
3. Test locally
4. Submit a pull request
5. Wait for CI to pass

For more information, see our [Contributing Guide](CONTRIBUTING.md).

---

**Best Practices:**
- Keep each service independent.
- Use the shared library for common code.
- Use health checks and environment variables for robust orchestration.
- Use Go workspace for local development and easy dependency management.
