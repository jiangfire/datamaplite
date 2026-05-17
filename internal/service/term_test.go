package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jiangfire/datamaplite/internal/model"
	"github.com/jiangfire/datamaplite/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupTermService(_ *testing.T) (*TermService, *MockStore) {
	mockStore := new(MockStore)
	service := NewTermService(mockStore)
	return service, mockStore
}

func TestTermService_CreateTerm(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupTermService(t)

	req := &model.BusinessTermRequest{
		Name:        "CustomerID",
		Description: "Unique identifier for a customer",
		Category:    "identifier",
	}

	mockStore.On("CreateBusinessTerm", ctx, mock.AnythingOfType("*store.BusinessTermCreate")).Return("term-123", nil)
	mockStore.On("GetBusinessTerm", ctx, "term-123").Return(&store.BusinessTermRow{
		ID:          "term-123",
		Name:        "CustomerID",
		Description: strPtr("Unique identifier for a customer"),
		Category:    "identifier",
	}, nil)

	term, err := service.CreateTerm(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "term-123", term.ID)
	assert.Equal(t, "CustomerID", term.Name)
	assert.Equal(t, "identifier", term.Category)
	mockStore.AssertExpectations(t)
}

func TestTermService_ListTerms(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupTermService(t)

	mockStore.On("ListBusinessTerms", ctx, "").Return([]*store.BusinessTermRow{
		{
			ID:       "term-1",
			Name:     "CustomerID",
			Category: "identifier",
		},
		{
			ID:       "term-2",
			Name:     "OrderDate",
			Category: "date",
		},
	}, nil)

	terms, err := service.ListTerms(ctx, "")

	require.NoError(t, err)
	assert.Len(t, terms, 2)
	assert.Equal(t, "CustomerID", terms[0].Name)
	assert.Equal(t, "OrderDate", terms[1].Name)
	mockStore.AssertExpectations(t)
}

func TestTermService_ListTerms_WithCategory(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupTermService(t)

	mockStore.On("ListBusinessTerms", ctx, "identifier").Return([]*store.BusinessTermRow{
		{
			ID:       "term-1",
			Name:     "CustomerID",
			Category: "identifier",
		},
	}, nil)

	terms, err := service.ListTerms(ctx, "identifier")

	require.NoError(t, err)
	assert.Len(t, terms, 1)
	assert.Equal(t, "CustomerID", terms[0].Name)
	mockStore.AssertExpectations(t)
}

func TestTermService_GetTerm(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupTermService(t)

	termID := "term-123"
	mockStore.On("GetBusinessTerm", ctx, termID).Return(&store.BusinessTermRow{
		ID:          termID,
		Name:        "CustomerID",
		Description: strPtr("Unique identifier for a customer"),
		Category:    "identifier",
	}, nil)

	term, err := service.GetTerm(ctx, termID)

	require.NoError(t, err)
	assert.Equal(t, termID, term.ID)
	assert.Equal(t, "CustomerID", term.Name)
	mockStore.AssertExpectations(t)
}

func TestTermService_GetTerm_NotFound(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupTermService(t)

	termID := "non-existent"
	mockStore.On("GetBusinessTerm", ctx, termID).Return(nil, errors.New("term not found"))

	_, err := service.GetTerm(ctx, termID)

	assert.Error(t, err)
	mockStore.AssertExpectations(t)
}

func TestTermService_UpdateTerm(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupTermService(t)

	termID := "term-123"
	req := &model.BusinessTermRequest{
		Name:        "CustomerIdentifier",
		Description: "Updated description",
		Category:    "id",
	}

	mockStore.On("UpdateBusinessTerm", ctx, termID, mock.AnythingOfType("*store.BusinessTermUpdate")).Return(nil)

	err := service.UpdateTerm(ctx, termID, req)

	require.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestTermService_DeleteTerm(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupTermService(t)

	termID := "term-123"
	mockStore.On("DeleteBusinessTerm", ctx, termID).Return(nil)

	err := service.DeleteTerm(ctx, termID)

	require.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestTermService_AssignTermToColumn(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupTermService(t)

	columnID := "col-123"
	termID := "term-456"
	req := &model.AssignTermRequest{
		TermID: &termID,
	}

	mockStore.On("AssignTermToColumn", ctx, columnID, &termID).Return(nil)

	err := service.AssignTermToColumn(ctx, columnID, req)

	require.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestTermService_AssignTermToColumn_Unassign(t *testing.T) {
	ctx := context.Background()
	service, mockStore := setupTermService(t)

	columnID := "col-123"
	req := &model.AssignTermRequest{
		TermID: nil, // 取消分配
	}

	mockStore.On("AssignTermToColumn", ctx, columnID, (*string)(nil)).Return(nil)

	err := service.AssignTermToColumn(ctx, columnID, req)

	require.NoError(t, err)
	mockStore.AssertExpectations(t)
}
