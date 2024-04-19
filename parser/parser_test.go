package parser_test

import (
	"errors"
	"log"
	"testing"

	"github.com/abiiranathan/apigen/models"
	"github.com/abiiranathan/apigen/services"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTransactionCommit(t *testing.T) {
	db, err := services.Sqlite3Connection(":memory:", logger.Silent)
	if err != nil {
		panic(err)
	}

	log.Println("Connected to sqlite3 database")
	err = db.AutoMigrate(&models.User{}, &models.Role{}, &models.Tag{}, &models.Issue{}, &models.Question{}, &models.Comment{})
	if err != nil {
		t.Fatal(err)
	}

	svc := services.NewService(db)

	// create a use service transaction
	userSvcTx, err := svc.UserService.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer userSvcTx.Rollback()

	user := models.User{
		Name:     "John Doe",
		Age:      30,
		Discount: 0.1,
	}

	err = userSvcTx.Create(&user)
	if err != nil {
		t.Fatal(err)
	}

	err = userSvcTx.Commit()
	if err != nil {
		t.Fatal(err)
	}

	// fetch user
	if fetchedUser, err := svc.UserService.Get(user.ID); err == nil {
		if fetchedUser.Name != user.Name {
			t.Fatalf("Expected user name to be %s, got %s", user.Name, fetchedUser.Name)
		}

		log.Println("User:", fetchedUser)
	} else {
		t.Fatal(err)
	}
}

func TestTransactionRollback(t *testing.T) {
	db, err := services.Sqlite3Connection(":memory:", logger.Silent)
	if err != nil {
		panic(err)
	}

	log.Println("Connected to sqlite3 database")

	err = db.AutoMigrate(&models.User{}, &models.Role{}, &models.Tag{}, &models.Issue{}, &models.Question{}, &models.Comment{})
	if err != nil {
		t.Fatal(err)
	}

	svc := services.NewService(db)

	// create a use service transaction
	userSvcTx, err := svc.UserService.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer userSvcTx.Rollback()

	user := models.User{
		Name:     "John Doe",
		Age:      30,
		Discount: 0.1,
	}

	err = userSvcTx.Create(&user)
	if err != nil {
		t.Fatal(err)
	}

	// rollback transaction
	err = userSvcTx.Rollback()
	if err != nil {
		t.Fatal(err)
	}

	// fetch user
	_, err = svc.UserService.Get(user.ID)
	if err == nil {
		t.Fatal("Expected to get an error, got nil")
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Expected error to be %v, got %v", gorm.ErrRecordNotFound, err)
	}
}

func TestServiceTransactionCommit(t *testing.T) {
	db, err := services.Sqlite3Connection(":memory:", logger.Silent)
	if err != nil {
		panic(err)
	}

	log.Println("Connected to sqlite3 database")

	err = db.AutoMigrate(&models.User{}, &models.Role{}, &models.Tag{}, &models.Issue{}, &models.Question{}, &models.Comment{})
	if err != nil {
		t.Fatal(err)
	}

	svc := services.NewService(db)

	// Service-level transactions
	tx, err := svc.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	// create a new role
	role := models.Role{
		Name:   "Admin",
		Gender: "Male",
	}

	err = tx.RoleService.Create(&role)
	if err != nil {
		t.Fatal(err)
	}

	// create a new tag
	tag := models.Tag{
		Name:   "Go",
		RoleID: role.ID,
	}

	err = tx.TagService.Create(&tag)
	if err != nil {
		t.Fatal(err)
	}

	// create a new issue
	issue := models.Issue{
		Name: "Issue 1",
	}

	err = tx.IssueService.Create(&issue)
	if err != nil {
		t.Fatal(err)
	}

	// commit the transaction
	err = tx.Commit()
	if err != nil {
		t.Fatal(err)
	}

	// fetch role
	// fetch role
	role, err = svc.RoleService.Get(role.ID)
	if err != nil {
		t.Fatal(err)
	}

	log.Println("Role:", role)

	// fetch tag
	tag, err = svc.TagService.Get(tag.ID)
	if err != nil {
		t.Fatal(err)
	}

	log.Println("Tag:", tag)

	// fetch issue
	issue, err = svc.IssueService.Get(issue.ID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestServiceTransactionRollback(t *testing.T) {
	db, err := services.Sqlite3Connection(":memory:", logger.Silent)
	if err != nil {
		panic(err)
	}

	log.Println("Connected to sqlite3 database")

	err = db.AutoMigrate(&models.User{}, &models.Role{}, &models.Tag{}, &models.Issue{}, &models.Question{}, &models.Comment{})
	if err != nil {
		t.Fatal(err)
	}

	svc := services.NewService(db)

	// Service-level transactions
	tx, err := svc.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()

	// create a new role
	role := models.Role{
		Name:   "Admin",
		Gender: "Male",
	}

	err = tx.RoleService.Create(&role)
	if err != nil {
		t.Fatal(err)
	}

	// create a new tag
	tag := models.Tag{
		Name:   "Go",
		RoleID: role.ID,
	}

	err = tx.TagService.Create(&tag)
	if err != nil {
		t.Fatal(err)
	}

	// create a new issue
	issue := models.Issue{
		Name: "Issue 1",
	}

	err = tx.IssueService.Create(&issue)
	if err != nil {
		t.Fatal(err)
	}

	// commit the transaction
	err = tx.Rollback()
	if err != nil {
		t.Fatal(err)
	}

	// fetch role
	// fetch role
	role, err = svc.RoleService.Get(role.ID)
	if err == nil {
		t.Fatal("Expected to get an error, got nil")
	}

	// fetch tag
	tag, err = svc.TagService.Get(tag.ID)
	if err == nil {
		t.Fatal("Expected to get an error, got nil")
	}

	// fetch issue
	issue, err = svc.IssueService.Get(issue.ID)
	if err == nil {
		t.Fatal("Expected to get an error, got nil")
	}

}
