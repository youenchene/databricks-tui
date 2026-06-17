---
name: start-project
description: Initialize Go project with hexagonal architecture, DDD bounded contexts, and ports-and-adapters structure. Use when user wants to scaffold, bootstrap, or start a new Go project with hexagonal or clean architecture.
---

Create a new Go project following hexagonal architecture (ports and adapters) with DDD structure.

**Discovery Process:**

First, understand what the user wants to build, then suggest appropriate bounded contexts (domains) based on DDD principles.

**Example Domain Suggestions by Project Type:**

- **E-commerce Platform**: user, product, cart, order, payment, inventory, shipping, review
- **Food Delivery App**: user, restaurant, menu, order, delivery, payment, notification
- **Social Media Platform**: user, post, comment, like, follow, feed, notification, message
- **Banking System**: customer, account, transaction, card, loan, payment, audit
- **Healthcare System**: patient, doctor, appointment, prescription, billing, medical-record
- **Inventory Management**: product, warehouse, stock, supplier, purchase-order, shipment
- **Hotel Booking**: user, hotel, room, booking, payment, review, availability
- **Learning Platform**: user, course, lesson, enrollment, assignment, grade, discussion

**Domain Selection Guidelines:**

- Suggest 3-6 core domains based on the project
- Each domain should represent a distinct bounded context with its own ubiquitous language
- Domains should have clear boundaries and responsibilities
- Allow user to select which domains to create initially (they can add more later)

**Project Structure to Create:**

```
project-name/
├── cmd/
│   ├── main.go                     # Entry point with Cobra root command
│   ├── root.go                     # Root command configuration
│   ├── serve/                      # Serve commands for microservices
│   │   ├── user.go                 # Serve user microservice
│   │   ├── order.go                # Serve order microservice
│   │   └── all.go                  # Serve all microservices
│   └── migrate/                    # Database migration commands
│       └── migrate.go
├── api/                            # API definitions (if using gRPC)
│   └── proto/
│       └── [domain]/
│           ├── [domain].proto      # Proto definitions
│           └── [domain].pb.go      # Generated code
├── internal/
│   ├── [domain-name]/              # e.g., "user", "order", "product" (bounded context)
│   │   ├── domain/                 # Domain layer
│   │   │   ├── entity.go           # Domain entities
│   │   │   ├── repository.go       # Repository interfaces (ports)
│   │   │   ├── service.go          # Domain services (if needed)
│   │   │   └── errors.go           # Domain-specific errors
│   │   ├── application/            # Application layer
│   │   │   ├── usecase/            # Use cases
│   │   │   └── dto/                # Data transfer objects
│   │   └── infrastructure/         # Infrastructure layer
│   │       ├── persistence/        # Repository implementations
│   │       │   └── postgres/       # Database-specific implementation
│   │       ├── http/               # HTTP REST handlers (if using REST)
│   │       │   └── handler.go
│   │       └── grpc/               # gRPC handlers (if using gRPC)
│   │           └── server.go
│   └── shared/                     # Shared kernel (cross-domain)
│       ├── valueobject/            # Shared value objects
│       ├── middleware/             # HTTP middleware
│       ├── config/                 # Configuration
│       └── database/               # Database connection
├── test/
│   ├── architecture/               # Architecture tests with goarchtest
│   │   └── architecture_test.go    # Hexagonal architecture constraints
│   ├── features/                   # Godog BDD feature files
│   │   └── [domain]/               # Feature files per domain
│   ├── integration/                # Integration tests
│   │   └── [domain]/
│   └── unit/                       # Unit tests
│       └── [domain]/
├── pkg/                            # Public packages (if needed)
├── scripts/                        # Build/deploy scripts
│   ├── generate-proto.sh           # Generate gRPC code (if using gRPC)
│   └── run-tests.sh
├── go.mod
├── go.sum
├── Makefile
├── .env.example
├── buf.yaml                        # Buf config (if using gRPC with buf)
└── README.md
```

**Setup Steps:**

1. **Discover the project** - Ask the user:
   - What type of project are you building? (e.g., "E-commerce platform", "User management system", "Food delivery app", "Inventory management")
   - What is the main business goal?

2. **Analyze and suggest domains** based on the project type:
   - For **E-commerce**: suggest domains like "user", "product", "order", "payment", "inventory", "shipping"
   - For **Food Delivery**: suggest "user", "restaurant", "order", "delivery", "payment"
   - For **Social Media**: suggest "user", "post", "comment", "notification", "feed"
   - For **Banking**: suggest "account", "transaction", "customer", "card", "loan"
   - For **Healthcare**: suggest "patient", "appointment", "doctor", "prescription", "billing"
   - Use your knowledge to suggest relevant bounded contexts

3. **Let user select domains** using AskUserQuestion with multiSelect:
   - Present suggested domains as options
   - Allow user to select which domains to create initially
   - They can always add more domains later with `/new-feature`

4. **Ask for technical preferences using AskUserQuestion:**
   - Project name (text input)
   - Go module path (text input - e.g., github.com/username/project-name)
   - **Database**: PostgreSQL, MongoDB, MySQL, SQLite, Multiple, None (for later)
   - **Communication Protocol**: REST API, gRPC, GraphQL, Both REST + gRPC, None (internal services only)
   - **Web Framework** (if using REST API): Gin, Echo, Fiber, Chi, Standard Library
   - **gRPC Framework** (if using gRPC): Standard gRPC-Go, Connect (buf.build), Both

5. Initialize Go module with `go mod init`

6. Create the complete directory structure for **each selected domain**:
   - `cmd/` with Cobra CLI structure
   - `internal/[domain]/domain/` - Domain layer for each selected bounded context
   - `internal/[domain]/application/` - Application layer
   - `internal/[domain]/infrastructure/` - Infrastructure layer
   - `internal/shared/` - Shared kernel

7. Generate Cobra CLI structure:
   - `cmd/main.go` - Entry point
   - `cmd/root.go` - Root command with common flags
   - `cmd/serve/[domain].go` - Command for **each selected domain** microservice
   - `cmd/serve/all.go` - Command to run all selected microservices together
   - Each microservice command starts its own server (HTTP/gRPC based on protocol choice) on a different port
   - If using gRPC, add `api/proto/[domain]/` directory with .proto files

8. Generate initial files:
   - `Makefile` with commands for **each selected domain**:
     - `make serve-[domain]` - Run specific microservice (for each domain)
     - `make serve-all` - Run all microservices
     - `make build` - Build single binary
     - `make proto-gen` - Generate gRPC code from .proto files (if using gRPC)
     - `make test` - Run all tests (unit + integration + architecture + BDD)
     - `make test-unit` - Run unit tests only
     - `make test-integration` - Run integration tests
     - `make test-arch` - Run architecture tests (goarchtest)
     - `make test-bdd` - Run BDD tests (godog)
     - `make lint`, `make docker-up`, `make docker-down`
   - `.env.example` with environment variables (ports per microservice, database configs)
   - `README.md` with:
     - Project description and business domain
     - Microservices architecture explanation
     - List of domains/bounded contexts and their responsibilities
     - How to run each microservice
   - Basic example implementation in **one of the selected domains** (usually the most important one):
     - Simple domain entity with business logic
     - Repository interface (port)
     - Basic use case in application layer
     - Repository implementation in infrastructure/persistence
     - API handler based on protocol choice:
       - REST API: HTTP handler with one endpoint (e.g., GET /health, POST /create)
       - gRPC: gRPC service implementation with one RPC method
       - Both: Include both handlers

9. Install required dependencies based on selections:
   - `github.com/spf13/cobra` for CLI
   - `github.com/spf13/viper` for configuration
   - **Web Framework** (if REST API): gin-gonic/gin, labstack/echo, gofiber/fiber, or go-chi/chi
   - **gRPC** (if using gRPC): google.golang.org/grpc, google.golang.org/protobuf
   - **Database drivers**:
     - PostgreSQL: lib/pq or pgx
     - MongoDB: go.mongodb.org/mongo-driver
     - MySQL: go-sql-driver/mysql
     - SQLite: modernc.org/sqlite
   - **Testing**:
     - `github.com/cucumber/godog` for BDD
     - `github.com/stretchr/testify` for assertions
     - `github.com/solrac97gr/goarchtest` for architecture testing
   - `github.com/joho/godotenv` for environment variables
   - `github.com/google/uuid` for ID generation

10. Create example feature file in `features/[example-domain]/example.feature` showing Godog syntax for the domain with example implementation

11. Create initial test setup files for unit, integration, and BDD tests organized by domain

12. **Create architecture tests** using goarchtest in `test/architecture/architecture_test.go`:
    - Install `github.com/solrac97gr/goarchtest`
    - Define architectural constraints based on hexagonal architecture:
      - Domain layer should NOT depend on application or infrastructure
      - Application layer should NOT depend on infrastructure
      - Infrastructure CAN depend on domain and application
      - Each domain should NOT depend on other domains (except through shared)
      - Naming conventions (e.g., repositories end with "Repository")
    - Use predefined pattern or custom rules based on selected domains
    - Example test structure:
      ```go
      func TestHexagonalArchitecture(t *testing.T) {
          projectPath, _ := filepath.Abs("../../")

          // Domain layer purity
          result := goarchtest.InPath(projectPath).
              That().
              ResideInNamespace("internal/*/domain").
              ShouldNot().
              HaveDependencyOn("internal/*/infrastructure").
              And().
              ShouldNot().
              HaveDependencyOn("internal/*/application").
              GetResult()

          assert.True(t, result.IsSuccessful, "Domain layer violated")
      }
      ```

13. Create `docker-compose.yml` for local development:
    - Database services (based on user preference)
    - Service definitions for **each selected domain microservice**
    - Port mappings for both HTTP and gRPC (if applicable)
    - Network configuration
    - Volume mounts for hot reload (development)

14. Create Dockerfile for building the binary:
    - Multi-stage build (builder + runtime)
    - Supports running any microservice via CMD
    - Example: `CMD ["serve", "user", "--port", "8080"]`

**Additional gRPC Setup (if applicable):**

15. If using gRPC:
    - Create `api/proto/[domain]/[domain].proto` with example service definition
    - Add `buf.yaml` or `buf.gen.yaml` for code generation (if using buf)
    - Add proto generation script in `scripts/generate-proto.sh`
    - Generate initial Go code from proto files
    - Add proto-gen target to Makefile

**Important**: Create complete structure for all selected domains, but only add a working example implementation in ONE domain (ask user which one should have the example, or choose the most central domain).

**Architecture Principles to Follow:**

- **Microservices Monorepo**: All microservices in one repo, each domain can run independently
- **Single Binary, Multiple Commands**: One binary built with Cobra, different commands for each microservice
- **Domain-First Organization**: Code is organized by bounded context/domain (e.g., `internal/user/`, `internal/order/`)
- **Hexagonal Layers within Domain**: Each domain has its own domain/application/infrastructure layers
- **Domain Layer Purity**: Domain layer has NO external dependencies (only stdlib)
- **Dependency Direction**: Infrastructure → Application → Domain (dependencies point inward)
- **Port & Adapter Pattern**: Repository interfaces (ports) in domain, implementations (adapters) in infrastructure
- **Dependency Injection**: Wire dependencies in each microservice command, pass them down
- **Independent Testing**: Each layer is testable in isolation
- **Shared Kernel**: Common code (value objects, middleware, config) lives in `internal/shared/`
- **Independent Deployment**: Each microservice can be deployed separately (same binary, different command)

**Microservice Execution:**

```bash
# Build single binary
go build -o bin/app cmd/main.go

# Run specific microservice (REST)
./bin/app serve user --port 8080
./bin/app serve order --port 8081

# Run specific microservice (gRPC)
./bin/app serve user --grpc-port 9090
./bin/app serve order --grpc-port 9091

# Run all microservices (for local development)
./bin/app serve all

# Generate gRPC code (if using gRPC)
make proto-gen
# or
buf generate
```

**When to Create New Domains:**

- Each bounded context gets its own folder under `internal/`
- Each domain can be its own microservice
- If two concepts have different lifecycles or different teams own them → separate domains
- If they share the same ubiquitous language → same domain

Be concise and create a production-ready structure.
