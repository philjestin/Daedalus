package api

import (
	"encoding/json"
	"net/http"

	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/service"
)

// TagsHandler handles tag-related HTTP requests.
type TagsHandler struct {
	service *service.TagService
}

// NewTagsHandler creates a new TagsHandler.
func NewTagsHandler(svc *service.TagService) *TagsHandler {
	return &TagsHandler{service: svc}
}

// List returns all tags.
func (h *TagsHandler) List(w http.ResponseWriter, r *http.Request) {
	tags, err := h.service.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tags == nil {
		tags = []model.Tag{}
	}
	respondJSON(w, http.StatusOK, tags)
}

// CreateTagRequest represents a request to create a new tag.
type CreateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Create creates a new tag.
func (h *TagsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tag := &model.Tag{
		Name:  req.Name,
		Color: req.Color,
	}

	if err := h.service.Create(r.Context(), tag); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, tag)
}

// Get retrieves a single tag by ID.
func (h *TagsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid tag ID")
		return
	}

	tag, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag == nil {
		respondError(w, http.StatusNotFound, "tag not found")
		return
	}

	respondJSON(w, http.StatusOK, tag)
}

// Update updates a tag.
func (h *TagsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid tag ID")
		return
	}

	tag, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tag == nil {
		respondError(w, http.StatusNotFound, "tag not found")
		return
	}

	var req CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != "" {
		tag.Name = req.Name
	}
	if req.Color != "" {
		tag.Color = req.Color
	}

	if err := h.service.Update(r.Context(), tag); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, tag)
}

// Delete removes a tag.
func (h *TagsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid tag ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddTagToPart adds a tag to a part.
func (h *TagsHandler) AddTagToPart(w http.ResponseWriter, r *http.Request) {
	partID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid part ID")
		return
	}
	tagID, err := parseUUID(r, "tagId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid tag ID")
		return
	}

	if err := h.service.AddTagToPart(r.Context(), partID, tagID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveTagFromPart removes a tag from a part.
func (h *TagsHandler) RemoveTagFromPart(w http.ResponseWriter, r *http.Request) {
	partID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid part ID")
		return
	}
	tagID, err := parseUUID(r, "tagId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid tag ID")
		return
	}

	if err := h.service.RemoveTagFromPart(r.Context(), partID, tagID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetPartTags returns all tags for a part.
func (h *TagsHandler) GetPartTags(w http.ResponseWriter, r *http.Request) {
	partID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid part ID")
		return
	}

	tags, err := h.service.GetTagsForPart(r.Context(), partID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tags == nil {
		tags = []model.Tag{}
	}

	respondJSON(w, http.StatusOK, tags)
}

// AddTagToDesign adds a tag to a design.
func (h *TagsHandler) AddTagToDesign(w http.ResponseWriter, r *http.Request) {
	designID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid design ID")
		return
	}
	tagID, err := parseUUID(r, "tagId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid tag ID")
		return
	}

	if err := h.service.AddTagToDesign(r.Context(), designID, tagID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveTagFromDesign removes a tag from a design.
func (h *TagsHandler) RemoveTagFromDesign(w http.ResponseWriter, r *http.Request) {
	designID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid design ID")
		return
	}
	tagID, err := parseUUID(r, "tagId")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid tag ID")
		return
	}

	if err := h.service.RemoveTagFromDesign(r.Context(), designID, tagID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetDesignTags returns all tags for a design.
func (h *TagsHandler) GetDesignTags(w http.ResponseWriter, r *http.Request) {
	designID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid design ID")
		return
	}

	tags, err := h.service.GetTagsForDesign(r.Context(), designID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tags == nil {
		tags = []model.Tag{}
	}

	respondJSON(w, http.StatusOK, tags)
}

// ListPartsByTag returns all parts with a given tag.
func (h *TagsHandler) ListPartsByTag(w http.ResponseWriter, r *http.Request) {
	tagID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid tag ID")
		return
	}

	parts, err := h.service.ListPartsByTag(r.Context(), tagID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if parts == nil {
		parts = []model.Part{}
	}

	respondJSON(w, http.StatusOK, parts)
}

// ListDesignsByTag returns all designs with a given tag.
func (h *TagsHandler) ListDesignsByTag(w http.ResponseWriter, r *http.Request) {
	tagID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid tag ID")
		return
	}

	designs, err := h.service.ListDesignsByTag(r.Context(), tagID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if designs == nil {
		designs = []model.Design{}
	}

	respondJSON(w, http.StatusOK, designs)
}
