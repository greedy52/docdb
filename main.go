package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/auth"
)

// // setup
// $ export AWS_ACCESS_KEY_ID=<access_id>
// $ export AWS_SECRET_ACCESS_KEY=<secret_key>
// $ export AWS_DEFAULT_REGION=us-west-2
// $ export DOCDB_URL=<docdb_url>

// // sample run
// $ go run .
// == AWS caller identity
// ARN:  arn:aws:sts::<account-id>:assumed-role/steve-poweruser/docdb-test
// == Connect steve-documentdb-test.cluster-<some-random-stuff>.us-west-2.docdb.amazonaws.com:27017
// Preparing...
// Connected.
// Result:  {steve 555}

func main() {
	ctx := context.TODO()

	printAWSIdentity(ctx)
	connectWithAWSIdentity(ctx)
}

func printAWSIdentity(ctx context.Context) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}

	stsService := sts.NewFromConfig(cfg)
	identity, err := stsService.GetCallerIdentity(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("== AWS caller identity")
	fmt.Println("ARN: ", aws.ToString(identity.Arn))
}

func connectWithAWSIdentity(ctx context.Context) {
	fmt.Println("== Connect", os.Getenv("DOCDB_URL"))
	options := options.Client().
		ApplyURI("mongodb://" + os.Getenv("DOCDB_URL")).
		SetDirect(true).
		SetAuth(makeCred()).
		SetTLSConfig(makeTLSConfig())

	fmt.Println("Preparing...")
	client, err := mongo.Connect(ctx, options)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected.")

	result := client.Database("test").Collection("users").FindOne(ctx, bson.D{})
	if result.Err() != nil {
		log.Fatal(err)
	}

	var data = struct {
		Name string
		Age  int
	}{}

	if err := result.Decode(&data); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Result: ", data)
}

func makeCred() options.Credential {
	cred := options.Credential{
		AuthMechanism: auth.MongoDBAWS,
		AuthSource:    "$external",
		Username:      os.Getenv("AWS_ACCESS_KEY_ID"),
		Password:      os.Getenv("AWS_SECRET_ACCESS_KEY"),
	}
	if token := os.Getenv("AWS_SESSION_TOKEN"); token != "" {
		cred.AuthMechanismProperties = map[string]string{
			"AWS_SESSION_TOKEN": token,
		}
	}
	return cred
}

func makeTLSConfig() *tls.Config {
	cas, err := os.ReadFile("global-bundle.pem")
	if err != nil {
		log.Fatal(err)
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(cas)
	return &tls.Config{
		RootCAs: pool,
	}
}
