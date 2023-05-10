package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// Set up all routes for registered models
func SetupRoutes(api fiber.Router, h *Handlers) {

	// API prefix for users
	usersPrefix := api.Group("/users")

	usersPrefix.Get("/", h.GetAllUsers())
	usersPrefix.Get("/:id<int>", h.GetUser())
	usersPrefix.Get("/paginated", h.GetPaginatedUsers())
	usersPrefix.Post("", h.CreateUser())
	usersPrefix.Put("/:id<int>", h.UpdateUser())
	usersPrefix.Patch("/:id<int>", h.PartialUserUpdate())
	usersPrefix.Delete("/:id<int>", h.DeleteUser())

	// API prefix for roles
	rolesPrefix := api.Group("/roles")

	rolesPrefix.Get("/", h.GetAllRoles())
	rolesPrefix.Get("/:id<int>", h.GetRole())
	rolesPrefix.Get("/paginated", h.GetPaginatedRoles())
	rolesPrefix.Post("", h.CreateRole())
	rolesPrefix.Put("/:id<int>", h.UpdateRole())
	rolesPrefix.Patch("/:id<int>", h.PartialRoleUpdate())
	rolesPrefix.Delete("/:id<int>", h.DeleteRole())

	// API prefix for tags
	tagsPrefix := api.Group("/tags")

	tagsPrefix.Get("/", h.GetAllTags())
	tagsPrefix.Get("/:id<int>", h.GetTag())
	tagsPrefix.Get("/paginated", h.GetPaginatedTags())
	tagsPrefix.Post("", h.CreateTag())
	tagsPrefix.Put("/:id<int>", h.UpdateTag())
	tagsPrefix.Patch("/:id<int>", h.PartialTagUpdate())
	tagsPrefix.Delete("/:id<int>", h.DeleteTag())

	// API prefix for issues
	issuesPrefix := api.Group("/issues")

	issuesPrefix.Get("/", h.GetAllIssues())
	issuesPrefix.Get("/:id<int>", h.GetIssue())
	issuesPrefix.Get("/paginated", h.GetPaginatedIssues())
	issuesPrefix.Post("", h.CreateIssue())
	issuesPrefix.Put("/:id<int>", h.UpdateIssue())
	issuesPrefix.Patch("/:id<int>", h.PartialIssueUpdate())
	issuesPrefix.Delete("/:id<int>", h.DeleteIssue())

}
