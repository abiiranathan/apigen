package main

import (
	"log"

	"github.com/abiiranathan/apigen/models"
	"github.com/abiiranathan/apigen/services"
	"gorm.io/gorm/logger"
)

func main() {
	db, err := services.Sqlite3Connection(":memory:", logger.Silent)
	if err != nil {
		panic(err)
	}

	log.Println("Connected to sqlite3 database")

	err = db.AutoMigrate(&models.User{}, &models.Role{}, &models.Tag{}, &models.Issue{}, &models.Question{}, &models.Comment{})
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Auto migrated models")

	svc := services.NewService(db)

	userSvcTx, err := svc.UserService.Begin()
	if err != nil {
		log.Fatalln(err)
	}
	defer userSvcTx.Rollback()

	user := models.User{
		Name:     "John Doe",
		Age:      30,
		Discount: 0.1,
	}

	err = userSvcTx.Create(&user)
	if err != nil {
		log.Fatalln(err)
	}

	err = userSvcTx.Commit()
	if err != nil {
		log.Fatalln(err)
	}

	// fetch user
	user, err = svc.UserService.Get(user.ID)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("User:", user)

	// Service-level transactions
	tx, err := svc.Begin()
	if err != nil {
		log.Fatalln(err)
	}
	defer tx.Rollback()

	// create a new role
	role := models.Role{
		Name:   "Admin",
		Gender: "Male",
	}

	err = tx.RoleService.Create(&role)
	if err != nil {
		log.Fatalln(err)
	}

	// create a new tag
	tag := models.Tag{
		Name:   "Go",
		RoleID: role.ID,
	}

	err = tx.TagService.Create(&tag)
	if err != nil {
		log.Fatalln(err)
	}

	options := services.NewOptions(4).
		Where("name = ?", "Go").
		WhereIf(role.ID != 0, "role_id = ?", role.ID).
		Append(services.Order("id desc"))

	// fetch tag
	tag, err = tx.TagService.Get(tag.ID, options)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Tag:", tag)

	// create a new issue
	issue := models.Issue{
		Name: "Issue 1",
	}

	err = tx.IssueService.Create(&issue)
	if err != nil {
		log.Fatalln(err)
	}

	// commit the transaction
	err = tx.Commit()
	if err != nil {
		log.Fatalln(err)
	}

	// fetch role
	role, err = svc.RoleService.Get(role.ID)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Role:", role)

	// fetch tag
	tag, err = svc.TagService.Get(tag.ID)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Tag:", tag)

	// fetch issue
	issue, err = svc.IssueService.Get(issue.ID)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Issue:", issue)

}
