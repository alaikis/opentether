package handler

import (
	"github.com/alaikis/opentether/internal/models"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) CloudPublicProducts(c *fiber.Ctx) error {
	rows, err := h.services.Cloud.ListProducts(true)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}

func (h *Handler) CloudPublicReleases(c *fiber.Ctx) error {
	rows, err := h.services.Cloud.ListReleases(true)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}

func (h *Handler) CloudPublicRelease(c *fiber.Ctx) error {
	row, err := h.services.Cloud.GetRelease(c.Params("version"), true)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "release not found"})
	}
	return c.JSON(row)
}

func (h *Handler) CloudPublicSite(c *fiber.Ctx) error {
	row, err := h.services.Cloud.GetSiteContent(c.Params("key"), true)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "content not found"})
	}
	return c.JSON(row)
}

func (h *Handler) CloudPublicDownload(c *fiber.Ctx) error {
	artifact, err := h.services.Cloud.DownloadArtifact(c.UserContext(), c.Params("id"), c.IP(), c.Get("User-Agent"))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "artifact not found"})
	}
	return c.Redirect(artifact.FileURL, fiber.StatusFound)
}

func (h *Handler) CloudAdminProducts(c *fiber.Ctx) error {
	rows, err := h.services.Cloud.ListProducts(false)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}

func (h *Handler) CloudAdminSaveProduct(c *fiber.Ctx) error {
	var row models.CloudProduct
	if err := c.BodyParser(&row); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if id := c.Params("id"); id != "" {
		row.ID = id
	}
	if err := h.services.Cloud.SaveProduct(&row); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(row)
}

func (h *Handler) CloudAdminDeleteProduct(c *fiber.Ctx) error {
	if err := h.services.Cloud.DeleteProduct(c.Params("id")); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) CloudAdminReleases(c *fiber.Ctx) error {
	rows, err := h.services.Cloud.ListReleases(false)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}

func (h *Handler) CloudAdminSaveRelease(c *fiber.Ctx) error {
	var row models.CloudRelease
	if err := c.BodyParser(&row); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if id := c.Params("id"); id != "" {
		row.ID = id
	}
	if err := h.services.Cloud.SaveRelease(&row); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(row)
}

func (h *Handler) CloudAdminPublishRelease(c *fiber.Ctx) error {
	published := c.Query("published", "true") != "false"
	row, err := h.services.Cloud.PublishRelease(c.Params("id"), published)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(row)
}

func (h *Handler) CloudAdminDeleteRelease(c *fiber.Ctx) error {
	if err := h.services.Cloud.DeleteRelease(c.Params("id")); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) CloudAdminArtifacts(c *fiber.Ctx) error {
	rows, err := h.services.Cloud.ListArtifacts(c.Query("release_id"))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}

func (h *Handler) CloudAdminSaveArtifact(c *fiber.Ctx) error {
	var row models.CloudArtifact
	if err := c.BodyParser(&row); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if id := c.Params("id"); id != "" {
		row.ID = id
	}
	if err := h.services.Cloud.SaveArtifact(&row); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(row)
}

func (h *Handler) CloudAdminDeleteArtifact(c *fiber.Ctx) error {
	if err := h.services.Cloud.DeleteArtifact(c.Params("id")); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) CloudAdminSiteContents(c *fiber.Ctx) error {
	rows, err := h.services.Cloud.ListSiteContents(false)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rows)
}

func (h *Handler) CloudAdminSaveSiteContent(c *fiber.Ctx) error {
	var row models.CloudSiteContent
	if err := c.BodyParser(&row); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if id := c.Params("id"); id != "" {
		row.ID = id
	}
	if err := h.services.Cloud.SaveSiteContent(&row); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(row)
}

func (h *Handler) CloudAdminDeleteSiteContent(c *fiber.Ctx) error {
	if err := h.services.Cloud.DeleteSiteContent(c.Params("id")); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(204)
}

func (h *Handler) CloudAdminStats(c *fiber.Ctx) error {
	stats, err := h.services.Cloud.DownloadStats()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(stats)
}
