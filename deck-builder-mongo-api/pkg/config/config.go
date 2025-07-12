package config

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func GetConnection() (*mongo.Client, error) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error cargando las variables de entorno")
	}

	mongodbURI := os.Getenv("MONGODB_URI2")
	if mongodbURI == "" {
		log.Fatal("Las variables de entorno no estan configuradas correctamente")
	}

	log.Println("Conectando a MongoDB en:", mongodbURI)

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().ApplyURI(mongodbURI).SetServerAPIOptions(serverAPI)
	if clientOptions == nil {
		log.Fatal("Error al crear las opciones del cliente de MongoDB")
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		log.Fatal("Error al conectar a MongoDB:", err)
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil{
		log.Fatal("Error al hacer ping a MongoDB:", err)
		return nil, err
	}

	log.Println("Conexión exitosa a MongoDB")
	return client, nil

}

func GetCollection(collectionName string) *mongo.Collection {
	client, err := GetConnection()
    if err != nil {
        log.Fatalf("No se pudo conectar a MongoDB: %v", err)
    }
    return client.Database(os.Getenv("MONGODB_DBNAME")).Collection(collectionName)
}

func GetContext() (context.Context, context.CancelFunc){
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if ctx == nil {
		log.Fatal("Error al crear el contexto de MongoDB")
		return nil, cancel
	}
	return ctx, cancel
}