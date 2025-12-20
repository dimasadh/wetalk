# WeTalk - Backend Chat Application

A WebSocket-based chat application built with Go, MongoDB, and Redis.

## Overview

The main goal of this project is to let me implement the system design for a real-time group chat application using Go, MongoDB, and Redis. As we know, we can't horizontally scale a websocket server because it's stateful and requires maintaining connections. To overcome this limitation, this project uses Redis as a message broker to distribute messages across multiple instances of the server.

WeTalk main features:

- **Personal Chat**: 1-on-1 conversation between two users
- **Group Chat**: Multi-user conversation with admin controls and invitation system

## Tech Stack

**Backend:**

- Go 1.22+
- WebSocket (gorilla/websocket)
- MongoDB (mongo-driver)
- Redis

## Prerequisites

- Go 1.22 or higher
- MongoDB 4.4 or higher
- Web browser with WebSocket support
- Redis 6.2 or higher

## Installation

1. **Clone the repository:**

```bash
git https://github.com/dimasadh/wetalk.git
cd wetalk
```

2. **Install dependencies:**

```bash
go mod download
```

3. **Start MongoDB from docker:**

```bash
docker run -d --name mongodb -p 27017:27017 mongo:4.4
```

4. **Start Redis from docker (optional):**

```bash
docker run -d --name redis -p 6379:6379 redis:6.2
```

5. **Run the application:**

```bash
go run main.go
```

6. **Start the frontend server (if needed):**

You can run the frontend by opening the index.html file in your browser or by running [WeTalk Web](https://github.com/dimasadh/wetalk-web)

## Support

For issues, questions, or contributions, please open an issue on GitHub.
