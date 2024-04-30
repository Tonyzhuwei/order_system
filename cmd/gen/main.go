package main

import (
	"gorm.io/driver/postgres"

	"gorm.io/gen"
	"gorm.io/gorm"
	"order_system/model"
)

func main() {
	g := gen.NewGenerator(gen.Config{
		OutPath: "./dal",
		Mode:    gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface, // generate mode
	})

	dsn := "host=localhost user=postgres password=password dbname=order_system sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	g.UseDB(db) // reuse your gorm db

	// Generate basic type-safe DAO API for struct `model.User` following conventions
	g.ApplyBasic(model.ALL_ORDER_TABLES...)

	// Generate the code
	g.Execute()
}
