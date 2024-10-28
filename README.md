# Go Cache Server with Sharding and API Key Authentication

This data store, named **Apollo**, is a high-performance, sharded key-value store tailored to handle a high number of requests from a single client (typically a web server), differentiating it from solutions like Redis. While Redis is optimized for handling multiple simultaneous connections across multiple clients, Apollo is focused on maximizing efficiency for a single, sustained connection with fast retrieval and write operations.

## Key Features

- **Sharded Architecture**: Uses 16 shards to distribute keys, improving data handling efficiency by reducing contention.
- **TTL Management**: Each entry can optionally have a time-to-live (TTL) set, automatically cleaning expired keys.
- **Simple Authentication**: API key-based authentication ensures secure access, allowing only clients with the correct key to interact with the server.
- **Limited Connection Handling**: Designed for single-connection use with a custom worker pool, limiting resource usage while ensuring high throughput for that single client.
- **Customizable Limits**: Configurable maximum key and value sizes allow for flexibility and adaptability to various use cases.
- **In-Memory Only**: Optimized for performance with data storage exclusively in memory.

## Differences from Redis

Apollo differs from Redis in several significant ways, making it ideal for scenarios that prioritize single-client, high-frequency access:

- **Single Client Focus**: Apollo is optimized for single-client access, whereas Redis handles multiple concurrent connections.
- **Simplified API**: Supports basic commands (`GET`, `SET`, `DEL`, `RAL` - Remove All), focusing solely on core key-value storage with optional TTL.
- **Lightweight Sharding**: Apollo uses 16 shards to improve performance for a single connection, while Redis is a fully distributed system designed for broader scalability.
- **Reduced Scalability Needs**: This data store is single-instance, making it a streamlined solution for single-client use cases without Redis's distributed setup.

## Getting Started

### Prerequisites

- **Go**: Ensure you have Go installed to build and run the server.
- **Environment File (.env)**: The server requires an API key specified in a `.env` file.

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/SteveStef/Apollo.git
   cd Apollo
   go build -o Apollo

2. Set up the environment file:
   - Create a `.env` file in the project root and add your API key:
     ```
     API_KEY=<your_api_key>
     ```

### Running the Server

To start Apollo, simply run:

```bash
./apollo
```

The server will start listening on `PORT=4000` by default.

### Usage

Apollo supports the following commands via TCP:

- **SET**: Store a key-value pair with an optional TTL.
- **GET**: Retrieve a value by key.
- **DEL**: Delete a key.
- **RAL**: Remove all keys in the data store.

### Example Commands

To interact with Apollo, you can use a TCP client to send commands in the following formats:

- **SET**: `SET <key> <value> <TTL(optional)>`
- **GET**: `GET <key>`
- **DEL**: `DEL <key>`
- **RAL**: `RAL` (clears all keys)

### Configuration

- **API Key**: Set your API key in the `.env` file for secure access.
- **Port**: By default, Apollo listens on port `4000`. You can change this in the code if needed.
- **Custom Limits**: Modify constants such as `MAX_KEY_SIZE` and `MAX_VALUE_SIZE` in the code to adjust key and value length constraints.

## Contributing

Contributions are welcome to enhance Apollo’s functionality! Please fork the repository, create a feature branch, and submit a pull request with your changes.

---

Apollo offers a streamlined, in-memory solution for high-frequency, single-client data storage, bridging simplicity and efficiency for specialized applications. Enjoy!

