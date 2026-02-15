package firebase

import (
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

var App *firebase.App

// InitFirebase initializes the Firebase SDK using credentials from the environment variable.
func InitFirebase() {
	ctx := context.Background()

	// User requested to load from file: config/firebase-service-account.json
	// Ensure this file exists or is handled.
	opt := option.WithCredentialsFile("config/firebase-service-account.json")

	var err error
	App, err = firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatalf("❌ Error initializing Firebase app: %v", err)
	}

	log.Println("✅ Firebase SDK initialized successfully!")
}
