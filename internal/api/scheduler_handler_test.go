package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akshaysangma/go-notify/internal/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockScheduler is a mock of the SchedulerController interface.
type MockScheduler struct {
	mock.Mock
}

func (m *MockScheduler) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockScheduler) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockScheduler) IsRunning() bool {
	args := m.Called()
	return args.Bool(0)
}

func TestSchedulerHandler_getSchedulerStatus(t *testing.T) {
	t.Run("Status Running", func(t *testing.T) {
		mockScheduler := new(MockScheduler)
		handler := NewSchedulerHandler(mockScheduler, zap.NewNop())
		mockScheduler.On("IsRunning").Return(true).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler", nil)
		rr := httptest.NewRecorder()

		handler.getSchedulerStatus(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var body SchedulerStatusResponse
		err := json.Unmarshal(rr.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.Equal(t, "running", body.Status)
		mockScheduler.AssertExpectations(t)
	})

	t.Run("Status Stopped", func(t *testing.T) {
		mockScheduler := new(MockScheduler)
		handler := NewSchedulerHandler(mockScheduler, zap.NewNop())
		mockScheduler.On("IsRunning").Return(false).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/scheduler", nil)
		rr := httptest.NewRecorder()

		handler.getSchedulerStatus(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var body SchedulerStatusResponse
		err := json.Unmarshal(rr.Body.Bytes(), &body)
		assert.NoError(t, err)
		assert.Equal(t, "stopped", body.Status)
		mockScheduler.AssertExpectations(t)
	})
}

func TestSchedulerHandler_schedulerControl(t *testing.T) {
	t.Run("Start Success", func(t *testing.T) {
		mockScheduler := new(MockScheduler)
		handler := NewSchedulerHandler(mockScheduler, zap.NewNop())
		mockScheduler.On("Start").Return(nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler?action=start", nil)
		rr := httptest.NewRecorder()

		handler.schedulerControl(rr, req)

		assert.Equal(t, http.StatusAccepted, rr.Code)
		var body SuccessResponse
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		assert.Equal(t, "Scheduler start signal sent.", body.Message)
		mockScheduler.AssertExpectations(t)
	})

	t.Run("Start Conflict - Already Running", func(t *testing.T) {
		mockScheduler := new(MockScheduler)
		handler := NewSchedulerHandler(mockScheduler, zap.NewNop())
		mockScheduler.On("Start").Return(scheduler.ErrAlreadyRunning).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler?action=start", nil)
		rr := httptest.NewRecorder()

		handler.schedulerControl(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)
		mockScheduler.AssertExpectations(t)
	})

	t.Run("Stop Success", func(t *testing.T) {
		mockScheduler := new(MockScheduler)
		handler := NewSchedulerHandler(mockScheduler, zap.NewNop())
		mockScheduler.On("Stop").Return(nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler?action=stop", nil)
		rr := httptest.NewRecorder()

		handler.schedulerControl(rr, req)

		assert.Equal(t, http.StatusAccepted, rr.Code)
		var body SuccessResponse
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		assert.Equal(t, "Scheduler stop signal sent.", body.Message)
		mockScheduler.AssertExpectations(t)
	})

	t.Run("Stop Conflict - Not Running", func(t *testing.T) {
		mockScheduler := new(MockScheduler)
		handler := NewSchedulerHandler(mockScheduler, zap.NewNop())
		mockScheduler.On("Stop").Return(scheduler.ErrNotRunning).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler?action=stop", nil)
		rr := httptest.NewRecorder()

		handler.schedulerControl(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)
		mockScheduler.AssertExpectations(t)
	})

	t.Run("Internal Server Error on Start", func(t *testing.T) {
		mockScheduler := new(MockScheduler)
		handler := NewSchedulerHandler(mockScheduler, zap.NewNop())
		internalErr := errors.New("something broke")
		mockScheduler.On("Start").Return(internalErr).Once()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler?action=start", nil)
		rr := httptest.NewRecorder()

		handler.schedulerControl(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockScheduler.AssertExpectations(t)
	})

	t.Run("Invalid Action", func(t *testing.T) {
		mockScheduler := new(MockScheduler)
		handler := NewSchedulerHandler(mockScheduler, zap.NewNop())
		req := httptest.NewRequest(http.MethodPost, "/api/v1/scheduler?action=invalid", nil)
		rr := httptest.NewRecorder()

		handler.schedulerControl(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockScheduler.AssertNotCalled(t, "Start")
		mockScheduler.AssertNotCalled(t, "Stop")
	})
}
