package repository

import (
	"context"
	"wetalk/internal/entity"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepository interface {
	Index(ctx context.Context, filter entity.UserIndexFilter) ([]entity.User, error)
	Get(ctx context.Context, userId string) (entity.User, error)
	Create(ctx context.Context, user entity.User) (string, error)
	Update(ctx context.Context, user entity.User) error
	GetOnlineUser(ctx context.Context, userIds []string) ([]entity.User, error)
}

type userRepository struct {
	db mongo.Database
}

func NewUserRepository(db mongo.Database) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) Index(ctx context.Context, filter entity.UserIndexFilter) ([]entity.User, error) {
	collection := r.db.Collection("users")

	var bsonFilter bson.M
	if len(filter.Ids) > 0 {
		bsonFilter = bson.M{"_id": bson.M{"$in": filter.Ids}}
	}

	cursor, err := collection.Find(ctx, bsonFilter)
	if err != nil {
		return nil, err
	}

	var users []entity.User
	err = cursor.All(ctx, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *userRepository) Get(ctx context.Context, userId string) (entity.User, error) {
	collection := r.db.Collection("users")
	filter := bson.M{"_id": userId}

	var user entity.User
	err := collection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		return entity.User{}, err
	}

	return user, nil
}

func (r *userRepository) Create(ctx context.Context, user entity.User) (string, error) {
	collection := r.db.Collection("users")
	user.Id = uuid.New().String()

	_, err := collection.InsertOne(ctx, user)
	if err != nil {
		return "", err
	}

	return user.Id, nil
}

func (r *userRepository) Update(ctx context.Context, user entity.User) error {
	collection := r.db.Collection("users")
	filter := bson.M{"_id": user.Id}
	update := bson.M{
		"$set": bson.M{
			"name":     user.Name,
			"isOnline": user.IsOnline,
		},
	}
	_, err := collection.UpdateOne(ctx, filter, update)

	return err
}

func (r *userRepository) GetOnlineUser(ctx context.Context, userIds []string) ([]entity.User, error) {
	collection := r.db.Collection("users")

	filter := bson.M{"isOnline": true}
	if len(userIds) > 0 {
		filter["_id"] = bson.M{"$in": userIds}
	}

	cursor, err := collection.Find(context.Background(), filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var users []entity.User
	for cursor.Next(context.Background()) {
		var user entity.User
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}
