package main

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const dbName = "fiber-hrms"
const mongoURI = "mongodb://localhost:27017/" + dbName

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

func Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	db := client.Database(dbName)
	if err != nil {
		return err
	}
	mg = MongoInstance{
		Client: client,
		Db:     db,
	}
	return nil
}

func main() {
	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()
	app.Get("/employee", func(ctx *fiber.Ctx) error {
		query := bson.D{{}}
		cursor, err := mg.Db.Collection("employees").Find(ctx.Context(), query)
		if err != nil {
			return ctx.Status(500).SendString(err.Error())
		}
		var employees []Employee = make([]Employee, 0)
		if err := cursor.All(ctx.Context(), &employees); err != nil {
			return ctx.Status(500).SendString(err.Error())
		}
		return ctx.JSON(employees)
	})
	app.Post("/employee", func(ctx *fiber.Ctx) error {
		collection := mg.Db.Collection("employees")
		employee := new(Employee)
		if err := ctx.BodyParser(employee); err != nil {
			return ctx.Status(400).SendString(err.Error())
		}
		employee.ID = ""
		insertionResult, err := collection.InsertOne(ctx.Context(), employee)
		if err != nil {
			return ctx.Status(500).SendString(err.Error())
		}
		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
		createdRecord := collection.FindOne(ctx.Context(), filter)
		createdEmployee := Employee{}
		createdRecord.Decode(createdEmployee)
		return ctx.Status(201).JSON(createdEmployee)
	})
	app.Put("/employee/:id", func(ctx *fiber.Ctx) error {
		idParam := ctx.Params("id")

		employeeId, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return ctx.SendStatus(400)
		}

		employee := new(Employee)

		if err := ctx.BodyParser(employee); err != nil {
			return ctx.Status(400).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: employeeId}}
		update := bson.D{
			{
				Key: "$set", Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "age", Value: employee.Age},
					{Key: "salary", Value: employee.Salary},
				},
			},
		}
		err = mg.Db.Collection("employees").FindOneAndUpdate(ctx.Context(), query, update).Err()
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return ctx.SendStatus(400)
			}
			return ctx.SendStatus(500)
		}
		employee.ID = idParam
		return ctx.Status(200).JSON(employee)
	})
	app.Delete("/employee/:id", func(ctx *fiber.Ctx) error {
		idParam := ctx.Params("id")

		employeeId, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return ctx.SendStatus(400)
		}
		query := bson.D{{Key: "_id", Value: employeeId}}
		result, err := mg.Db.Collection("employees").DeleteOne(ctx.Context(), &query)
		if err != nil {
			return ctx.SendStatus(500)
		}
		if result.DeletedCount < 1 {
			return ctx.SendStatus(404)
		}
		return ctx.Status(200).JSON("deleted")
	})
	log.Fatal(app.Listen(":3000"))
}
