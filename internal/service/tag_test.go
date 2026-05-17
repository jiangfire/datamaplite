package service

import (
	"errors"
	"testing"

	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTagService_CreateTag_Success(t *testing.T) {
	mockStore := new(MockStore)
	svc := NewTagService(mockStore)

	req := &model.TagRequest{
		Name:  "PII",
		Color: "#ff0000",
	}

	mockStore.On("GetTagByName", mock.Anything, "PII").Return(nil, errors.New("not found"))
	mockStore.On("CreateTag", mock.Anything, mock.Anything).Return("tag-1", nil)

	resp, err := svc.CreateTag(t.Context(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "tag-1", resp.ID)
	assert.Equal(t, "PII", resp.Name)
	assert.Equal(t, "#ff0000", resp.Color)
	mockStore.AssertExpectations(t)
}

func TestTagService_CreateTag_InvalidColor(t *testing.T) {
	svc := NewTagService(nil)

	req := &model.TagRequest{
		Name:  "PII",
		Color: "invalid",
	}

	resp, err := svc.CreateTag(t.Context(), req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid color format")
}

func TestTagService_CreateTag_DuplicateName(t *testing.T) {
	mockStore := new(MockStore)
	svc := NewTagService(mockStore)

	req := &model.TagRequest{
		Name:  "PII",
		Color: "#ff0000",
	}

	mockStore.On("GetTagByName", mock.Anything, "PII").Return(&store.TagRow{ID: "tag-1", Name: "PII"}, nil)

	resp, err := svc.CreateTag(t.Context(), req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "already exists")
	mockStore.AssertExpectations(t)
}

func TestTagService_GetTag_Success(t *testing.T) {
	mockStore := new(MockStore)
	svc := NewTagService(mockStore)

	mockStore.On("GetTag", mock.Anything, "tag-1").Return(&store.TagRow{
		ID:    "tag-1",
		Name:  "PII",
		Color: "#ff0000",
	}, nil)

	resp, err := svc.GetTag(t.Context(), "tag-1")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "PII", resp.Name)
	mockStore.AssertExpectations(t)
}

func TestTagService_ListTags_Success(t *testing.T) {
	mockStore := new(MockStore)
	svc := NewTagService(mockStore)

	mockStore.On("ListTags", mock.Anything).Return([]*store.TagRow{
		{ID: "tag-1", Name: "PII", Color: "#ff0000"},
		{ID: "tag-2", Name: "Sensitive", Color: "#00ff00"},
	}, nil)

	resp, err := svc.ListTags(t.Context())
	assert.NoError(t, err)
	assert.Len(t, resp, 2)
	mockStore.AssertExpectations(t)
}

func TestTagService_DeleteTag_Success(t *testing.T) {
	mockStore := new(MockStore)
	svc := NewTagService(mockStore)

	mockStore.On("DeleteTag", mock.Anything, "tag-1").Return(nil)

	err := svc.DeleteTag(t.Context(), "tag-1")
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestTagService_AddTagToColumn_Success(t *testing.T) {
	mockStore := new(MockStore)
	svc := NewTagService(mockStore)

	mockStore.On("AddTagToColumn", mock.Anything, "col-1", "tag-1").Return(nil)

	err := svc.AddTagToColumn(t.Context(), "col-1", "tag-1")
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}
