package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jonboulle/clockwork"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

const (
	emailAddressID = "email_address"
	dbName         = "mindful-insights"
	user           = "user"
)

type mongoRepo struct {
	client *mongo.Client
	clock  clockwork.Clock
}

func New(ctx context.Context, uri string, clock clockwork.Clock) (*mongoRepo, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return &mongoRepo{}, fmt.Errorf("db_error: %w", err)
	}

	repo := &mongoRepo{
		client: client,
		clock:  clock,
	}

	if err := repo.ensureIndexes(ctx); err != nil {
		return nil, err
	}
	return repo, nil
}

func (m *mongoRepo) ensureIndexes(ctx context.Context) error {
	mod := mongo.IndexModel{
		Keys: bsonx.Doc{
			bsonx.Elem{Key: emailAddressID, Value: bsonx.Int32(1)},
		},
		Options: options.Index().SetName(emailAddressID).SetUnique(false),
	}

	_, err := m.UserCollection().Indexes().CreateOne(ctx, mod)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	return nil
}

func (m *mongoRepo) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return m.client.Ping(ctx, nil)
}

func (m *mongoRepo) UserCollection() *mongo.Collection {
	m.db().Collection(user)
	return nil
}

func (m *mongoRepo) db() *mongo.Database {
	return m.client.Database(dbName)
}

func (m *mongoRepo) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

// CreateUser creates a new user in mongo
func (m *mongoRepo) CreateUser(ctx context.Context, user User) error {
	_, err := m.UserCollection().InsertOne(ctx, User{
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Password:  user.Password,
		CreatedAt: time.Now(),
	})

	if err != nil {
		var e mongo.WriteException
		if errors.As(err, &e) {
			for _, v := range e.WriteErrors {
				if v.Code == 11000 {
					return DuplicatedError{err: err}
				}

			}
		}
	}
	return nil
}

func (m *mongoRepo) GetUser(ctx context.Context, emailAddress string) (User, error) {
	filter := bson.D{{Key: emailAddressID, Value: emailAddress}}

	var u User
	if err := m.UserCollection().FindOne(ctx, filter).Decode(&u); err != nil {
		switch err {
		case mongo.ErrNoDocuments:
			return User{}, nil
		default:
			return User{}, fmt.Errorf("error finding user: %w", err)
		}
	}

	if u.Password != "" {
		u.Password = ""
	}

	return u, nil
}

type DuplicatedError struct {
	err error
}

func (d DuplicatedError) Error() string {
	return fmt.Sprintf("user found in doc: %s", d.err)
}
