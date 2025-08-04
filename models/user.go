package models

import (
	"time"
)

// API user credentials
// It is used to sign in
//
// swagger:model user
type User struct {
	// User ID (может быть string или interface{})
	ID interface{} `bson:"_id,omitempty" json:"id"`
	
	// User's login
	//
	// required: true
	Email    string    `bson:"email" json:"email"`

	// User's password
	//
	// required: true
	Password string    `bson:"password" json:"password"`
	Verified bool      `bson:"verified" json:"verified"`
	Created  time.Time `bson:"created" json:"created"`
}
