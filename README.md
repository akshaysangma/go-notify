
# Go Notify

Go Notify is a robust and scalable message-sending service designed to automatically dispatch notifications. It features a scheduler for sending pending messages in batches, API endpoints for controlling the service and retrieving message history, and Redis caching for enhanced performance.

## Features

* **Automatic Message Dispatch**: A scheduler periodically sends pending messages from the database.
* **API Control**: Endpoints to start and stop the message sending scheduler and to retrieve sent messages.
* **Database Integration**: Utilizes a PostgreSQL database for message storage and retrieval.
* **Configuration Management**: Easily configurable through a `config.yaml` file.
* **Swagger Documentation**: Comprehensive API documentation using Swagger.

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

3.  **Using Docker (Recommended):**

    * **Start the infrastructure (Postgres & Redis):**
        ```sh
        docker-compose up -d
        ```
    * **Build and run the application:**
        ```sh
        docker build -t go-notify-app .
        docker run --rm -p 8080:8080 \
        --network=go-notify_default \
        -e DATABASE_CONNECTION_STRING="postgresql://user:password@postgres:5432/go_notify_db?sslmode=disable" \
        -e REDIS_ADDRESS="redis:6379" \
        go-notify-app
        ```

4.  **Using Go & Makefile:**

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

### Endpoints

#### Scheduler

* `POST /api/v1/scheduler?action={start|stop}`: Start or stop the message sending scheduler.
* `GET /api/v1/scheduler`: Get the current status of the scheduler.

#### Messages

* `GET /api/v1/messages/sent`: Retrieve a list of sent messages.
* `POST /api/v1/messages`: Create a new message for multiple recipients.

## Third-Party Tools & Libraries

This project utilizes several third-party Go libraries and tools to facilitate development:

### Libraries

* **`github.com/spf13/viper`**: Used for handling application configuration from files, environment variables, and more.
* **`go.uber.org/zap`**: A blazing fast, structured, leveled logger.
* **`github.com/jackc/pgx/v5`**: A modern, feature-rich PostgreSQL driver and toolkit for Go.
* **`github.com/go-redis/redis/v8`**: A high-performance Redis client for Go.
* **`github.com/google/uuid`**: Provides an implementation for UUIDs (Universally Unique Identifiers).
* **`github.com/swaggo/http-swagger`**: A middleware to automatically serve Swagger UI for your API.

### Tools

* **`github.com/swaggo/swag`**: A tool to automatically generate RESTful API documentation with Swagger 2.0.
* **`github.com/pressly/goose`**: A database migration tool. It's used to manage and apply database schema changes.
* **`github.com/sqlc-dev/sqlc`**: A command-line tool that generates type-safe, idiomatic Go code from SQL.


## Key Points / Notes
- [docker-compose.yml](docker-compose.yml) contains required services to setup local environment : Postgres, Redis
- [Makefile](Makefile) contains helper scripts. Run `make help` for more info.
- [Cache client](external/redis/client.go) failure to intialize cache client will not stop application from running.
- For failed messages to send, we can maintain a seperate table for dead_letter_message which tracks
the msg ID and reason etc. Current implementation uses one table only.
- In [Message Domain Model](internal/messages/model.go), The `Status` field in the `Message` domain model could be implemented as an iota constant with a `map[int]string` for better type safety, but this was skipped to avoid the need for custom marshalling methods.
- [Work Pool Implementation](internal/messages/service.go) : Killing worker pool after every run (Implemented Approach) vs reusing worker pool 
    - Pros: 
        - Simple State Management: When the scheduler is paused, no background processes are running, making the "off" state very clean.
        - Resource Efficiency When Stopped: All goroutines and associated memory are freed completely when the work is done
    - Cons:
        - Performance Overhead: Constantly creating and destroying goroutines for every run introduces unnecessary performance overhead.
- [Message Table Schema](sql/schema/20250708142121_create_message_table.sql) - content lenght on the database can be enforced using VARCHAR(size). Opted to try conditional CONSTRAINT.
- The Scheduler config `grace_period` defines the timeout for each processing cycle (`runs_every` - `grace_period`) to prevent job overlaps, ensuring scheduler stability. The `timeout jobs` will rerun next `tick`.
- Add `job_timeout` to avoid hang up due to I/O block during graceful shutdown.
- A multi-stage [Dockerfile](Dockerfile) is used to create a small and secure production image.
- To prioritize core functionality, middleware for features like authentication and monitoring was deferred
- Assumption: `retrieve a list of sent messages` means sent all messages in the database (with basic offset, limit pagination) and not via [get the sent message list](https://docs.webhook.site/api/examples.html#get-all-data-sent-to-url) api of `webhook.site`
- The Stop API command is blocking call until scheduler has shutdown in current Implementation. 