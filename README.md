
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

* Docker
* Docker Compose

### Installation

1.  **Clone the repository:**

    ```sh
    git clone <repository-url>
    cd go-notify
    ```

2.  **Configuration:**

    The application is configured using the `config.yaml` and `environment varaiables` file. You can modify this file to change the database connection string, Redis address, and other settings.

3.  **Build app**

    - Docker
    ```sh
    docker build -t go-notify-app .
    ```

    - Go Tool
    ```sh
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/go-notify ./cmd/server
    ```

4. _(Optional)_ **Start local Infra Docker Compose:** 

    ```sh    
    docker-compose up
    ```

5. **Run app** 
    Ensure required environmental variable or config.yaml are set properly.

    - Docker 
        - Ensure you pass your environment variable via -e arg
        - --network is need if you are running it against infra setup using Step 4 above.
    ```sh
    docker run --rm -p 8080:8080 \
    --network=go-notify_default \
    -e DATABASE_CONNECTION_STRING="postgresql://user:password@postgres:5432/go_notify_db?sslmode=disable" \
    -e REDIS_ADDRESS="redis:6379" \
    go-notify-app
    ``` 

    - Go tool
        - Ensure required environmental variable or config.yaml are set properly on the machine.
    ```sh
       ./bin/go-notify
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


## Key Points / Notes
- [docker-compose.yml](docker-compose.yml) contains required services to setup local environment : Postgres, Redis
- [Makefile](Makefile) helper script
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
-  A multi-stage [Dockerfile](Dockerfile) is used to create a small and secure production image.
- Needs Go version > 1.24 as it uses the new net/http features for http server.
- Middleware - Auth, LoggerContext, Prometheus etc. were skipped to respect time