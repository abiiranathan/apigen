package handlers

import (
	"github.com/abiiranathan/apigen/cmd/apigen/services"
	"github.com/abiiranathan/apigen/models"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handlers struct {
	db  *gorm.DB
	svc *services.Service
}

func New(db *gorm.DB, svc *services.Service) *Handlers {
	return &Handlers{db: db, svc: svc}
}

func (h *Handlers) CreateUser(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var user models.User
		if err := BodyParser(c, &user); err != nil {
			return err
		}
		if err := h.svc.UserService.Create(&user, options...); err != nil {
			return err
		}
		return c.Status(fiber.StatusCreated).JSON(user)
	}
}

func (h *Handlers) GetUser(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userId := int64(GetParam(c, "id"))
		user, err := h.svc.UserService.Get(int64(userId), options...)
		if err != nil {
			return err
		}
		return c.JSON(user)
	}
}

func (h *Handlers) GetAllUsers(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		users, err := h.svc.UserService.GetAll(options...)
		if err != nil {
			return err
		}
		return c.JSON(users)
	}
}

func (h *Handlers) GetPaginatedUsers(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		page := GetQuery(c, "page")
		pageSize := GetQuery(c, "limit")

		if pageSize <= 0 {
			pageSize = 25
		}

		users, err := h.svc.UserService.GetPaginated(page, pageSize, options...)
		if err != nil {
			return err
		}
		return c.JSON(users)
	}
}

func (h *Handlers) FindManyUsers(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		users, err := h.svc.UserService.FindMany(options...)
		if err != nil {
			return err
		}
		return c.JSON(users)
	}
}

func (h *Handlers) UpdateUser(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var user models.User
		if err := BodyParser(c, &user); err != nil {
			return err
		}

		// Since we expect a full update, user must have an ID
		if user.ID == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Record missing id field.")
		}

		updatedUser, err := h.svc.UserService.Update(user.ID, &user, options...)
		if err != nil {
			return err
		}
		return c.JSON(updatedUser)
	}
}

func (h *Handlers) PartialUserUpdate(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userId := int64(GetParam(c, "id"))

		var user models.User
		if err := BodyParser(c, &user); err != nil {
			return err
		}

		// Specify options for Where, Order, Omit or Select, etc.

		updatedUser, err := h.svc.UserService.PartialUpdate(userId, user, options...)
		if err != nil {
			return err
		}
		return c.JSON(updatedUser)
	}
}

func (h *Handlers) DeleteUser(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		userId := int64(GetParam(c, "id"))
		err := h.svc.UserService.Delete(userId)
		if err != nil {
			return err
		}
		return c.JSON("record deleted successfully")
	}
}

func (h *Handlers) DeleteUsersWhere(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Specify conditions for delete with arguments
		condition := ""
		args := []any{}

		err := h.svc.UserService.DeleteWhere(condition, args...)
		if err != nil {
			return err
		}
		return c.JSON("records deleted successfully")
	}
}

func (h *Handlers) CreateRole(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var role models.Role
		if err := BodyParser(c, &role); err != nil {
			return err
		}
		if err := h.svc.RoleService.Create(&role, options...); err != nil {
			return err
		}
		return c.Status(fiber.StatusCreated).JSON(role)
	}
}

func (h *Handlers) GetRole(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		roleId := int64(GetParam(c, "id"))
		role, err := h.svc.RoleService.Get(int64(roleId), options...)
		if err != nil {
			return err
		}
		return c.JSON(role)
	}
}

func (h *Handlers) GetAllRoles(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		roles, err := h.svc.RoleService.GetAll(options...)
		if err != nil {
			return err
		}
		return c.JSON(roles)
	}
}

func (h *Handlers) GetPaginatedRoles(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		page := GetQuery(c, "page")
		pageSize := GetQuery(c, "limit")

		if pageSize <= 0 {
			pageSize = 25
		}

		roles, err := h.svc.RoleService.GetPaginated(page, pageSize, options...)
		if err != nil {
			return err
		}
		return c.JSON(roles)
	}
}

func (h *Handlers) FindManyRoles(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		roles, err := h.svc.RoleService.FindMany(options...)
		if err != nil {
			return err
		}
		return c.JSON(roles)
	}
}

func (h *Handlers) UpdateRole(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var role models.Role
		if err := BodyParser(c, &role); err != nil {
			return err
		}

		// Since we expect a full update, role must have an ID
		if role.ID == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Record missing id field.")
		}

		updatedRole, err := h.svc.RoleService.Update(role.ID, &role, options...)
		if err != nil {
			return err
		}
		return c.JSON(updatedRole)
	}
}

func (h *Handlers) PartialRoleUpdate(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		roleId := int64(GetParam(c, "id"))

		var role models.Role
		if err := BodyParser(c, &role); err != nil {
			return err
		}

		// Specify options for Where, Order, Omit or Select, etc.

		updatedRole, err := h.svc.RoleService.PartialUpdate(roleId, role, options...)
		if err != nil {
			return err
		}
		return c.JSON(updatedRole)
	}
}

func (h *Handlers) DeleteRole(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		roleId := int64(GetParam(c, "id"))
		err := h.svc.RoleService.Delete(roleId)
		if err != nil {
			return err
		}
		return c.JSON("record deleted successfully")
	}
}

func (h *Handlers) DeleteRolesWhere(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Specify conditions for delete with arguments
		condition := ""
		args := []any{}

		err := h.svc.RoleService.DeleteWhere(condition, args...)
		if err != nil {
			return err
		}
		return c.JSON("records deleted successfully")
	}
}

func (h *Handlers) CreateTag(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var tag models.Tag
		if err := BodyParser(c, &tag); err != nil {
			return err
		}
		if err := h.svc.TagService.Create(&tag, options...); err != nil {
			return err
		}
		return c.Status(fiber.StatusCreated).JSON(tag)
	}
}

func (h *Handlers) GetTag(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		tagId := int64(GetParam(c, "id"))
		tag, err := h.svc.TagService.Get(int64(tagId), options...)
		if err != nil {
			return err
		}
		return c.JSON(tag)
	}
}

func (h *Handlers) GetAllTags(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		tags, err := h.svc.TagService.GetAll(options...)
		if err != nil {
			return err
		}
		return c.JSON(tags)
	}
}

func (h *Handlers) GetPaginatedTags(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		page := GetQuery(c, "page")
		pageSize := GetQuery(c, "limit")

		if pageSize <= 0 {
			pageSize = 25
		}

		tags, err := h.svc.TagService.GetPaginated(page, pageSize, options...)
		if err != nil {
			return err
		}
		return c.JSON(tags)
	}
}

func (h *Handlers) FindManyTags(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		tags, err := h.svc.TagService.FindMany(options...)
		if err != nil {
			return err
		}
		return c.JSON(tags)
	}
}

func (h *Handlers) UpdateTag(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var tag models.Tag
		if err := BodyParser(c, &tag); err != nil {
			return err
		}

		// Since we expect a full update, tag must have an ID
		if tag.ID == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Record missing id field.")
		}

		updatedTag, err := h.svc.TagService.Update(tag.ID, &tag, options...)
		if err != nil {
			return err
		}
		return c.JSON(updatedTag)
	}
}

func (h *Handlers) PartialTagUpdate(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		tagId := int64(GetParam(c, "id"))

		var tag models.Tag
		if err := BodyParser(c, &tag); err != nil {
			return err
		}

		// Specify options for Where, Order, Omit or Select, etc.

		updatedTag, err := h.svc.TagService.PartialUpdate(tagId, tag, options...)
		if err != nil {
			return err
		}
		return c.JSON(updatedTag)
	}
}

func (h *Handlers) DeleteTag(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		tagId := int64(GetParam(c, "id"))
		err := h.svc.TagService.Delete(tagId)
		if err != nil {
			return err
		}
		return c.JSON("record deleted successfully")
	}
}

func (h *Handlers) DeleteTagsWhere(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Specify conditions for delete with arguments
		condition := ""
		args := []any{}

		err := h.svc.TagService.DeleteWhere(condition, args...)
		if err != nil {
			return err
		}
		return c.JSON("records deleted successfully")
	}
}

func (h *Handlers) CreateIssue(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var issue models.Issue
		if err := BodyParser(c, &issue); err != nil {
			return err
		}
		if err := h.svc.IssueService.Create(&issue, options...); err != nil {
			return err
		}
		return c.Status(fiber.StatusCreated).JSON(issue)
	}
}

func (h *Handlers) GetIssue(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		issueId := int64(GetParam(c, "id"))
		issue, err := h.svc.IssueService.Get(int64(issueId), options...)
		if err != nil {
			return err
		}
		return c.JSON(issue)
	}
}

func (h *Handlers) GetAllIssues(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		issues, err := h.svc.IssueService.GetAll(options...)
		if err != nil {
			return err
		}
		return c.JSON(issues)
	}
}

func (h *Handlers) GetPaginatedIssues(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		page := GetQuery(c, "page")
		pageSize := GetQuery(c, "limit")

		if pageSize <= 0 {
			pageSize = 25
		}

		issues, err := h.svc.IssueService.GetPaginated(page, pageSize, options...)
		if err != nil {
			return err
		}
		return c.JSON(issues)
	}
}

func (h *Handlers) FindManyIssues(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		issues, err := h.svc.IssueService.FindMany(options...)
		if err != nil {
			return err
		}
		return c.JSON(issues)
	}
}

func (h *Handlers) UpdateIssue(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var issue models.Issue
		if err := BodyParser(c, &issue); err != nil {
			return err
		}

		// Since we expect a full update, issue must have an ID
		if issue.ID == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "Record missing id field.")
		}

		updatedIssue, err := h.svc.IssueService.Update(issue.ID, &issue, options...)
		if err != nil {
			return err
		}
		return c.JSON(updatedIssue)
	}
}

func (h *Handlers) PartialIssueUpdate(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		issueId := int64(GetParam(c, "id"))

		var issue models.Issue
		if err := BodyParser(c, &issue); err != nil {
			return err
		}

		// Specify options for Where, Order, Omit or Select, etc.

		updatedIssue, err := h.svc.IssueService.PartialUpdate(issueId, issue, options...)
		if err != nil {
			return err
		}
		return c.JSON(updatedIssue)
	}
}

func (h *Handlers) DeleteIssue(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		issueId := int64(GetParam(c, "id"))
		err := h.svc.IssueService.Delete(issueId)
		if err != nil {
			return err
		}
		return c.JSON("record deleted successfully")
	}
}

func (h *Handlers) DeleteIssuesWhere(options ...services.Option) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		// Specify conditions for delete with arguments
		condition := ""
		args := []any{}

		err := h.svc.IssueService.DeleteWhere(condition, args...)
		if err != nil {
			return err
		}
		return c.JSON("records deleted successfully")
	}
}
