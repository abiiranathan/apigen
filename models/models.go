package models

type Sex string

const (
	Male   Sex = "Male"
	Female Sex = "Female"
)

type (
	User struct {
		ID       int     `json:"id" gorm:"autoIncrement"`
		Name     string  `json:"name" gorm:"default:''"`
		Age      int     `json:"age"`
		Discount float64 `json:"discount" gorm:"constraint:positive_discount CHECK (discount > 0)"`
		RoleID   int64   `json:"role_id" gorm:"not null"`
		Role     Role    `json:"role" gorm:"foreignKey:RoleID"`
		Tags     []Tag   `json:"tags" gorm:"many2many:user_tags"`
	}

	Role struct {
		ID     int64  `json:"id"`
		Name   string `json:"name"`
		Gender Sex    `json:"gender"`
	}

	Tag struct {
		ID     int64   `json:"id"`
		Name   string  `json:"name"`
		Issues []Issue `gorm:"many2many:tag_issues" json:"issues"`
	}

	Issue struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
)
