package services

import (
	"context"
	"errors"
	"time"

	"srv_recipes/models"

	"github.com/go-redis/redis"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	// "srv_recipes/config"
)

type UserHandler struct {
	collection  *mongo.Collection
	ctx         context.Context
	redisClient *redis.Client
}

func NewUserHandler(ctx context.Context, collection *mongo.Collection, redisClient *redis.Client) *UserHandler {
	return &UserHandler{
		collection:  collection,
		ctx:         ctx,
		redisClient: redisClient,
	}
}

func (handler *UserHandler) CreateUser(email, hashedPassword string) error {
	u := models.User{Email: email, Password: hashedPassword, Verified: false, Created: time.Now()}
	_, err := handler.collection.InsertOne(context.Background(), u)
	if mongo.IsDuplicateKeyError(err) {
		return errors.New("user exists")
	}
	return err
}

func (handler *UserHandler) GetUserByEmail(email string) (*models.User, error) {
	var u models.User
	err := handler.collection.
		FindOne(context.Background(), bson.M{"email": email}).
		Decode(&u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (handler *UserHandler) VerifyUser(email string) error {
	_, err := handler.collection.
		UpdateOne(context.Background(), bson.M{"email": email}, bson.M{"$set": bson.M{"verified": true}})
	return err
}
