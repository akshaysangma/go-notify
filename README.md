
# Go Notify

Go Notify is a robust and scalable message-sending service designed to automatically dispatch notifications. It features a scheduler for sending pending messages in batches, API endpoints for controlling the service and retrieving message history. Redis is used for caching.

## Features

* **Automatic Message Dispatch**: A scheduler periodically sends pending messages from the database.
* **API Control**: Endpoints to start and stop the message sending scheduler and to retrieve sent messages.
* **Database Integration**: Utilizes a PostgreSQL database for message storage and retrieval.
* **Cache Integration**: Cache mechanism enabled with Redis.
* **Configuration Management**: Easily configurable through a `config.yaml` file.
* **Swagger Documentation**: API documentation using Swagger.


## Project Structure
* `api`: Handles HTTP requests, routing, and response handling.
* `cmd`: Main application entry point.
* `config`: Manages application configuration.
* `database`: Manages the database connection and repository implementations.
* `docs`: Contains Swagger documentation files.
* `external`: Houses clients for external services like Redis and webhooks.
* `internal`: Contains the core business logic of the application.
* `scheduler`: Implements the message dispatch scheduler.

## Getting Started

### Prerequisites

* For Containize experience : Docker and Docker Compose
* Go (version > 1.24) as it uses the new net/http features for http server.
* Make (optional, for using the Makefile)

### Installation & Running

1.  **Clone the repository:**

    ```sh
    git clone <repository-url>
    cd go-notify
    ```

2.  **Configuration:**

    The application is configured using the `config.yaml` and environment variables. You can modify this file to change the database connection string, Redis address, and other settings.

3.  **Run Locally:**
    * **Using Docker (Recommended):**

        * **Start the infrastructure (Postgres & Redis):**
            ```sh
            docker-compose up -d
            ```
        * **Build:**
            ```sh
            docker build -t go-notify-app .
            ```
        * **Run the application:**
            ```sh
            docker run --rm -p 8080:8080 \
            --network=go-notify_default \
            -e DATABASE_CONNECTION_STRING="postgresql://user:password@postgres:5432/go_notify_db?sslmode=disable" \
            -e REDIS_ADDRESS="redis:6379" \
            go-notify-app
            ```

    * **Using Go & Makefile:**

        * **Start the infrastructure (Postgres & Redis):**
            ```sh
            make up
            ```
        * **Build the application:**
            ```sh
            make build
            ```
        * **Run the application:**
            ```sh
            make run
            ```

## API Documentation

The API is documented using Swagger. Once the application is running, you can access the Swagger UI at:
[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

## Database Schema & Queries

This directory contains the PostgreSQL database schema definitions and SQL queries used by the application.
[Database Schemas and Queries](sql)

### Endpoints

#### Scheduler

* `POST /api/v1/scheduler?action={start|stop}`: Start or stop the message sending scheduler.
* `GET /api/v1/scheduler`: Get the current status of the scheduler.

#### Messages

* `GET /api/v1/messages/sent`: Retrieve a list of sent messages.
* `POST /api/v1/messages`: Create a new message for multiple recipients.

## Key Points / Notes
- Assumption:
    - `retrieve a list of sent messages` means all sent messages in the database (with basic offset, limit pagination) and not via [get the sent message list](https://docs.webhook.site/api/examples.html#get-all-data-sent-to-url) api of `webhook.site`. Data was not retrieved from cache as it has only 24 hours data (ephemeral).
    - `Upon project deployment, automatic message sending should -start` means the scheduler will start by default on app startup.
- `Scheduler Startup Behavior:` By default, the scheduler processes its first message batch after an initial delay defined by runs_every. To execute the first batch immediately upon startup, [an initial s.execute()](https://github.com/akshaysangma/go-notify/blob/main/internal/scheduler/message_dispatcher.go#L92) call can be enabled, and subsequent ticker intervals will then align from the completion of this initial run.  We can add a `delayed_start` boolean flag to the scheduler's configuration for conditionally wrapping the initial message batch run, providing a flexible way to control whether processing begins immediately on startup or after the configured delay.
- [Work Pool Implementation](internal/messages/service.go) : Killing worker pool after every run (Implemented Approach) vs reusing worker pool 
    - Pros: 
        - Simple State Management: When the scheduler is paused, no background processes are running, making the "off" state very clean.
        - Resource Efficiency When Stopped: All goroutines and associated memory are freed completely when the work is done
    - Cons:
        - Performance Overhead: Constantly creating and destroying goroutines for every run introduces unnecessary performance overhead.
- [Message Table Schema](sql/schema/20250708142121_create_message_table.sql) - content lenght on the database can be enforced using VARCHAR(size). Opted to try conditional CONSTRAINT.
- In [Message Domain Model](internal/messages/model.go), The `Status` field in the `Message` domain model could be implemented as an iota constant with a `map[int]string` for better type safety, but this was skipped to avoid the need for custom marshalling methods.
- Failure to intialize/write via [Cache client](external/redis/client.go) will not stop application from running.
- The Stop API command is a blocking call until scheduler has shutdown `(status code : 200)` where as Start API is non-blocking `(status code : 202)`.
- The Scheduler config `grace_period` defines the timeout for each processing cycle (`runs_every` - `grace_period`) to prevent job overlaps, ensuring scheduler stability. The `timeout jobs` will rerun next `tick`.
- Add `job_timeout` to avoid hang up due to I/O block during graceful shutdown.
- [docker-compose.yml](docker-compose.yml) contains required services to setup local environment : Postgres, Redis
- [Makefile](Makefile) contains helper scripts. Run `make help` for more info.
- A multi-stage [Dockerfile](Dockerfile) is used to create a small and secure production image.
- For failed messages to send, we can maintain a seperate table for dead_letter_message which tracks
the msg ID and reason etc. Current implementation uses one table only.
- To prioritize core functionality, middleware for features like authentication and monitoring was deferred
- Test for only core components added.
- CICD not added.

## Third-Party Tools & Libraries

This project utilizes several third-party Go libraries and tools to facilitate development:

### Libraries

* **`github.com/spf13/viper`**: configuration manager.
* **`go.uber.org/zap`**: structured logger.
* **`github.com/jackc/pgx/v5`**: PostgreSQL driver and toolkit.
* **`github.com/go-redis/redis/v8`**: Redis client for Go.
* **`github.com/google/uuid`**: Provides an implementation for UUIDs.
* **`github.com/swaggo/http-swagger`**: Provides handler to automatically serve Swagger UI.
* **`github.com/stretchr/testify`**: Provides helpers for assertion and mocking.

### Tools

* **`github.com/swaggo/swag`**: A tool to automatically generate RESTful API documentation with Swagger 2.0.
* **`github.com/pressly/goose`**: A database migration tool. It's used to manage and apply database schema changes.
* **`github.com/sqlc-dev/sqlc`**: A command-line tool that generates type-safe, idiomatic Go code from SQL.
