package server_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	. "nidavellir/server"
	"nidavellir/services/store"
)

func NewSourceHandler() *SourceHandler {
	db := &MockSourceStore{db: map[int]*store.Source{
		1: {
			Id:         1,
			Name:       "Source1",
			UniqueName: "source1",
			RepoUrl:    "http://github.com/somewhere/source1",
			Interval:   3600,
			State:      store.ScheduleNoop,
			NextTime:   time.Now(),
			Secrets: []store.Secret{
				{
					Id:       1,
					SourceId: 1,
					Key:      "secret-key",
					Value:    "secret-value",
				},
				{
					Id:       2,
					SourceId: 1,
					Key:      "name",
					Value:    "some-name",
				},
			},
		},
	}}

	return &SourceHandler{DB: db}
}

func TestSourceHandler_GetSources(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewSourceHandler()

	w := httptest.NewRecorder()
	r := NewTestRequest("GET", "/", nil, nil)
	handler.GetSources()(w, r)
	assert.Equal(http.StatusOK, w.Code)

	var sources []*store.Source
	err := readJson(w, &sources)
	assert.NoError(err)
	assert.Len(sources, 1)
}

func TestSourceHandler_GetSource(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewSourceHandler()

	tests := []struct {
		Id         string
		StatusCode int
		Type       interface{}
	}{
		{"1", http.StatusOK, &store.Source{}},
		{"2", http.StatusBadRequest, nil},
	}

	for _, test := range tests {
		w := httptest.NewRecorder()

		r := NewTestRequest("GET", "/", nil, map[string]string{"id": test.Id})
		handler.GetSource()(w, r)
		assert.Equal(test.StatusCode, w.Code)

		if test.StatusCode == http.StatusOK {
			var source *store.Source
			err := readJson(w, &source)
			assert.NoError(err)
			assert.IsType(test.Type, source)
		} else {
			errMsg := w.Body.String()
			assert.NotEmpty(errMsg)
		}
	}
}

func TestSourceHandler_CreateSource(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewSourceHandler()

	w := httptest.NewRecorder()
	r := NewTestRequest("POST", "/", strings.NewReader(`{
"name": "UniqueName",
"repo_url": "https://git/repo/project.git",
"interval": 5000,
"secrets": [
	{
		"key": "SecretKey", 
		"value": "SecretValue"
	}
]
}`), nil)

	handler.CreateSource()(w, r)
	assert.EqualValues(http.StatusOK, w.Code)

	var source *store.Source
	err := readJson(w, &source)
	assert.NoError(err)
	assert.IsType(&store.Source{}, source)
	assert.Len(source.Secrets, 1)
}

func TestSourceHandler_CreateSource_WithFaultyJsonReturnsError(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewSourceHandler()

	w := httptest.NewRecorder()
	r := NewTestRequest("POST", "/", strings.NewReader(`{
"id": 1,
"name": "",
"repo_url": "",
"interval": "",
"state": ""
}`), nil)

	handler.CreateSource()(w, r)
	assert.EqualValues(http.StatusBadRequest, w.Code)
}

// Tests errors out as id should not be specified
func TestSourceHandler_CreateSource_WithInvalidKeyReturnsError(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewSourceHandler()

	w := httptest.NewRecorder()
	r := NewTestRequest("POST", "/", strings.NewReader(`{
"id": 1,
"name": "UniqueName",
"repo_url": "https://git/repo/project.git",
"interval": 5000
}`), nil)

	handler.CreateSource()(w, r)
	assert.EqualValues(http.StatusBadRequest, w.Code)
}

func TestSourceHandler_UpdateSource(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewSourceHandler()

	tests := []struct {
		Id         int
		StatusCode int
		Type       interface{}
	}{
		{1, http.StatusOK, &store.Source{}},
		{2, http.StatusBadRequest, nil},
	}

	for _, test := range tests {

		w := httptest.NewRecorder()
		r := NewTestRequest("PUT", "/", strings.NewReader(fmt.Sprintf(`{
"id": %d,
"name": "UniqueName",
"repo_url": "https://git/repo/project.git",
"interval": 5000,
"secrets": [
	{
		"key": "SecretKey", 
		"value": "SecretValue"
	}
]
}`, test.Id)), nil)

		handler.UpdateSource()(w, r)
		assert.Equal(test.StatusCode, w.Code)

		if test.StatusCode == http.StatusOK {
			var source *store.Source
			err := readJson(w, &source)
			assert.NoError(err)
			assert.IsType(test.Type, source)
		} else {
			errMsg := w.Body.String()
			assert.NotEmpty(errMsg)
		}
	}
}

func TestSourceHandler_DeleteSource(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewSourceHandler()

	tests := []struct {
		Id         string
		StatusCode int
		Type       interface{}
	}{
		{"1", http.StatusOK, &store.Source{}},
		{"2", http.StatusBadRequest, nil},
	}

	for _, test := range tests {
		w := httptest.NewRecorder()
		r := NewTestRequest("PUT", "/", nil, map[string]string{"id": test.Id})

		handler.DeleteSource()(w, r)
		assert.Equal(test.StatusCode, w.Code)
	}

}

func TestSourceHandler_AddSecret(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewSourceHandler()

	w := httptest.NewRecorder()
	r := NewTestRequest("POST", "/", strings.NewReader(`{
"source_id": 1,
"key": "NewKey",
"value": "NewValue"
}`), map[string]string{"sourceId": "1"})

	handler.AddSecret()(w, r)
	assert.Equal(http.StatusOK, w.Code)

	var secret *store.Secret
	err := readJson(w, &secret)
	assert.NoError(err)
	assert.IsType(&store.Secret{}, secret)
}

func TestSourceHandler_GetSecrets(t *testing.T) {
	t.Parallel()

	assert := require.New(t)
	handler := NewSourceHandler()

	w := httptest.NewRecorder()
	r := NewTestRequest("GET", "/", nil, map[string]string{"sourceId": "1"})

	handler.GetSecrets()(w, r)
	assert.Equal(http.StatusOK, w.Code)

	var secrets []*store.Secret
	err := readJson(w, &secrets)
	assert.NoError(err)
	assert.IsType([]*store.Secret{}, secrets)
	assert.Len(secrets, 2)
}

func TestSourceHandler_UpdateSecret(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewSourceHandler()

	w := httptest.NewRecorder()
	r := NewTestRequest("PUT", "/", strings.NewReader(`{
"id": 1,
"source_id": 1,
"key": "NewKey",
"value": "NewValue"
}`), map[string]string{"sourceId": "1"})

	handler.AddSecret()(w, r)
	assert.Equal(http.StatusOK, w.Code)

	var secret *store.Secret
	err := readJson(w, &secret)
	assert.NoError(err)
	assert.IsType(&store.Secret{}, secret)
	assert.Equal(secret.Key, "NewKey")
	assert.Equal(secret.Value, "NewValue")
}

func TestSourceHandler_DeleteSecret(t *testing.T) {
	t.Parallel()
	assert := require.New(t)
	handler := NewSourceHandler()

	w := httptest.NewRecorder()
	r := NewTestRequest("DELETE", "/", nil, map[string]string{"sourceId": "1", "id": "1"})

	handler.DeleteSecret()(w, r)
	assert.Equal(http.StatusOK, w.Code)
}
