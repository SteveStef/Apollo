
# Go Cache Server with Sharding and API Key Authentication

This project implements a simple, multithreaded TCP cache server in Go, featuring sharding, TTL (Time-To-Live), API key-based authentication, and a connection queue system. The server can handle up to 100 concurrent connections and utilizes mutex locks for safe data access in a multithreaded environment.

## Features
- **Sharding**: Data is distributed across multiple shards to improve performance.
- **TTL (Time-To-Live)**: Keys have optional expiration times.
- **Multithreading**: Handles concurrent connections with locks.
- **Queue System**: Incoming connections are queued when the server reaches its maximum capacity.
- **API Key Authentication**: Only requests with the correct API key are processed.
- **Simple Protocol**: Supports basic commands like `GET`, `SET`, `DEL`, and `RAL` (Remove All).

## Getting Started

### Prerequisites
- [Go](https://golang.org/dl/) 1.20 or later
- Docker (optional, if you want to run it in a container)

### Installation

1. Clone the repository:
    ```bash
    git clone <repository-url>
    cd <repository-directory>
    ```

2. Install dependencies:
    ```bash
    go mod download
    ```

3. Create a `.env` file in the project root and add your API key:
    ```env
    API_KEY=your_api_key_here
    ```

4. Run the server:
    ```bash
    go run main.go
    ```

The server will start listening on port `4000`.

### Running with Docker

To run the server using Docker, follow these steps:

1. Build the Docker image:
    ```bash
    docker build -t go-cache-server .
    ```

2. Run the container:
    ```bash
    docker run -p 4000:4000 -e API_KEY=your_api_key_here go-cache-server
    ```

The server will be running and accessible on `localhost:4000`.

## Usage

The server listens for TCP connections and processes commands in a simple protocol. 

### Commands

- **SET**: Set a key with an optional TTL (Time-To-Live).
- **GET**: Retrieve a value by key.
- **DEL**: Delete a key-value pair.
- **RAL**: Remove all keys.

Example commands:
- `SET`: `SET <key-length> <key> <value-length> <value> <ttl>`
- `GET`: `GET <key-length> <key>`
- `DEL`: `DEL <key-length> <key>`
- `RAL`: `RAL`

### Example Code for Client Connection

Here's an example of how you could interact with the server from a client:

```go
conn, _ := net.Dial("tcp", "localhost:4000")
conn.Write([]byte("your_api_key"))
conn.Write([]byte("SET"))
conn.Write([]byte("<key and value details>"))
