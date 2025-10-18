package db

import (
	"context"
	"errors"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStore struct {
	Client *mongo.Client
	DB     *mongo.Database
}

func NewMongoStore(ctx context.Context, uri, dbName string) (*MongoStore, error) {
	if uri == "" {
		uri = os.Getenv("MONGODB_URI")
		if uri == "" {
			uri = "mongodb://localhost:27017"
		}
	}
	if dbName == "" {
		dbName = os.Getenv("MONGODB_DATABASE")
	}

	if dbName == "" {
		return nil, errors.New("database name required (set dbName or MONGODB_DATABASE)")
	}

	clientOpts := options.Client().ApplyURI(uri).
		SetMaxPoolSize(100)

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(connectCtx, clientOpts)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(connectCtx, nil); err != nil {
		_ = client.Disconnect(connectCtx)
		return nil, err
	}

	store := &MongoStore{
		Client: client,
		DB:     client.Database(dbName),
	}
	return store, nil
}

func (m *MongoStore) Close(ctx context.Context) error {
	if m == nil || m.Client == nil {
		return nil
	}
	disconnectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return m.Client.Disconnect(disconnectCtx)
}

func (m *MongoStore) Ping(ctx context.Context) error {
	if m == nil || m.Client == nil {
		return errors.New("mongo client is nil")
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return m.Client.Ping(pingCtx, nil)
}
