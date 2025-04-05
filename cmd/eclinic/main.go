package main

import (
	"fmt"

	"github.com/abiiranathan/apigen/v2/models"
	"github.com/abiiranathan/apigen/v2/services"
	"gorm.io/gorm/logger"
)

var dsn = "dbname=apigen user=**** password=****** host=127.0.0.1 port=5432 sslmode=disable TimeZone=Africa/Kampala"

func main() {
	db, err := services.PostgresConnection(dsn, "Africa/Kampala", logger.Error)
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate([]any{
		&models.User{},
		&models.Role{},
		&models.Tag{},
		&models.Issue{},
		&models.Question{},
		&models.Comment{},
	}...)

	if err != nil {
		panic(err)
	}

	svc := services.NewService(db)
	// create a new role
	role := models.Role{
		Name: "Admin",
	}

	err = svc.RoleService.Create(&role)
	if err != nil {
		panic(err)
	}

	// Create a new user
	user := models.User{
		Name:     "John Doe",
		Age:      30,
		Discount: 10.0,
		RoleID:   role.ID,
	}
	err = svc.UserService.Create(&user)
	if err != nil {
		panic(err)
	}

	// Fetch all users
	users, err := svc.UserService.GetAll()
	if err != nil {
		panic(err)
	}
	for _, user := range users {
		fmt.Println("user:", user.Name)
	}

	// get all paginated users
	paginatedUsers, err := svc.UserService.GetPaginated(1, 10)
	if err != nil {
		panic(err)
	}

	for _, user := range paginatedUsers.Results {
		fmt.Println("paginated user:", user.Name)
	}
}
