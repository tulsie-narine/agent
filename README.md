## Running the Project with Docker

This project provides Dockerfiles for each major component and a `docker-compose.yml` for orchestrating the services. Below are the key details for running the project using Docker:

### Project-Specific Requirements
- **Go Services (`agent`, `api`)**: Require Go version `1.22.1` (as set in Dockerfiles).
- **Web Service (`web`)**: Requires Node.js version `22.13.1` (as set in Dockerfile).
- **Alpine Linux** is used for final images of Go services for minimal footprint.

### Environment Variables
- Each service supports environment variables via `.env` files (see `.env.example` in each directory for required variables).
  - Uncomment the `env_file` lines in `docker-compose.yml` and provide the appropriate `.env` files if needed.
- No secrets are included in the example configs; ensure you provide production values as needed.

### Build and Run Instructions
1. **Clone the repository** and ensure Docker and Docker Compose are installed.
2. **(Optional)**: Review and update `.env.example` files in `agent`, `api`, and `web` directories. Copy to `.env` and fill in required values.
3. **Build and start all services:**
   ```sh
   docker compose up --build
   ```
   This will build and start:
   - `go-agent` (from `./agent`)
   - `go-api` (from `./api`)
   - `ts-web` (from `./web`)

### Ports Exposed
- **go-api**: `8080` (mapped to host `8080`)
- **ts-web**: `3000` (mapped to host `3000`)
- **go-agent**: No ports exposed by default (uncomment in compose/Dockerfile if needed)

### Special Configuration
- All services are connected via the `appnet` bridge network for inter-service communication.
- Database service is not included by default; add and configure as needed (see `depends_on` in compose file).
- Migrations for the API service are copied into the container; ensure your database is accessible if using migrations at runtime.
- Healthchecks are not enabled by default; you may uncomment and configure them in the Dockerfiles as needed.

### Summary Table
| Service   | Directory | Dockerfile Version | Exposed Port |
|-----------|-----------|-------------------|--------------|
| go-agent  | ./agent   | Go 1.22.1         | (none)       |
| go-api    | ./api     | Go 1.22.1         | 8080         |
| ts-web    | ./web     | Node 22.13.1      | 3000         |

For more details on deployment, see `DOCKER_DEPLOYMENT.md` and `QUICK_START_DOCKER.md` in the project root.
