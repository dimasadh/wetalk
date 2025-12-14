package repository

import (
	"context"
	"time"
	"wetalk/internal/entity"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, refreshToken entity.RefreshToken) error
	GetByToken(ctx context.Context, token string) (entity.RefreshToken, error)
	GetByUserId(ctx context.Context, userId string) ([]entity.RefreshToken, error)
	Revoke(ctx context.Context, token string) error
	RevokeAllByUserId(ctx context.Context, userId string) error
	DeleteExpired(ctx context.Context) error
	IsRevoked(ctx context.Context, token string) (bool, error)
}

type refreshTokenRepository struct {
	db mongo.Database
}

func NewRefreshTokenRepository(db mongo.Database) RefreshTokenRepository {
	return &refreshTokenRepository{
		db: db,
	}
}

func (r *refreshTokenRepository) Create(ctx context.Context, refreshToken entity.RefreshToken) error {
	collection := r.db.Collection("refresh_tokens")
	
	refreshToken.Id = uuid.New().String()
	refreshToken.CreatedAt = time.Now()
	refreshToken.IsRevoked = false
	
	_, err := collection.InsertOne(ctx, refreshToken)
	return err
}

func (r *refreshTokenRepository) GetByToken(ctx context.Context, token string) (entity.RefreshToken, error) {
	collection := r.db.Collection("refresh_tokens")
	filter := bson.M{"token": token}
	
	var refreshToken entity.RefreshToken
	err := collection.FindOne(ctx, filter).Decode(&refreshToken)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return entity.RefreshToken{}, ErrUserNotFound
		}
		return entity.RefreshToken{}, err
	}
	
	return refreshToken, nil
}

func (r *refreshTokenRepository) GetByUserId(ctx context.Context, userId string) ([]entity.RefreshToken, error) {
	collection := r.db.Collection("refresh_tokens")
	filter := bson.M{
		"userId":    userId,
		"isRevoked": false,
		"expiresAt": bson.M{"$gt": time.Now()},
	}
	
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	
	var tokens []entity.RefreshToken
	if err := cursor.All(ctx, &tokens); err != nil {
		return nil, err
	}
	
	return tokens, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, token string) error {
	collection := r.db.Collection("refresh_tokens")
	filter := bson.M{"token": token}
	now := time.Now()
	
	update := bson.M{
		"$set": bson.M{
			"isRevoked": true,
			"revokedAt": now,
		},
	}
	
	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *refreshTokenRepository) RevokeAllByUserId(ctx context.Context, userId string) error {
	collection := r.db.Collection("refresh_tokens")
	filter := bson.M{
		"userId":    userId,
		"isRevoked": false,
	}
	now := time.Now()
	
	update := bson.M{
		"$set": bson.M{
			"isRevoked": true,
			"revokedAt": now,
		},
	}
	
	_, err := collection.UpdateMany(ctx, filter, update)
	return err
}

func (r *refreshTokenRepository) DeleteExpired(ctx context.Context) error {
	collection := r.db.Collection("refresh_tokens")
	filter := bson.M{
		"expiresAt": bson.M{"$lt": time.Now()},
	}
	
	_, err := collection.DeleteMany(ctx, filter)
	return err
}

func (r *refreshTokenRepository) IsRevoked(ctx context.Context, token string) (bool, error) {
	refreshToken, err := r.GetByToken(ctx, token)
	if err != nil {
		return true, err
	}
	
	return refreshToken.IsRevoked, nil
}