package firebase

import (
	"context"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

var App *firebase.App

// InitFirebase initializes the Firebase SDK using credentials from the environment variable.
func InitFirebase() {
	ctx := context.Background()

	serviceAccountJSON := os.Getenv("FIREBASE_SERVICE_ACCOUNT")
	if serviceAccountJSON == "" {
		log.Fatal("❌ FIREBASE_SERVICE_ACCOUNT environment variable is not set")
	}

	opt := option.WithCredentialsJSON([]byte(serviceAccountJSON))

	var err error
	App, err = firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatalf("❌ Error initializing Firebase app: %v", err)
	}

	log.Println("✅ Firebase SDK initialized successfully!")
}
