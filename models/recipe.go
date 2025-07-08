package models

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

// swagger:parameters recipes newRecipe
type Recipe struct {
	ID           bson.ObjectID `bson:"_id" json:"id"`
	CustomID     string        `bson:"id" json:"customId"`
	Name         string        `bson:"name" json:"name"`
	Tags         []string      `bson:"tags" json:"tags"`
	Ingredients  []string      `bson:"ingredients" json:"ingredients"`
	Instructions []string      `bson:"instructions" json:"instructions"`
	PublishedAt  string        `bson:"publishedAt" json:"publishedAt"`
}
