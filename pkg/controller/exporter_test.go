package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v8"
	"github.com/mvisonneau/gitlab-ci-pipelines-exporter/pkg/config"
	"github.com/mvisonneau/gitlab-ci-pipelines-exporter/pkg/gitlab"
	"github.com/mvisonneau/gitlab-ci-pipelines-exporter/pkg/ratelimit"
	"github.com/mvisonneau/gitlab-ci-pipelines-exporter/pkg/storage"
	"github.com/stretchr/testify/assert"
	goGitlab "github.com/xanzy/go-gitlab"
)

func resetGlobalValues() {
	cfgUpdateLock.Lock()
	defer cfgUpdateLock.Unlock()

	cfg = config.New()
	gitlabClient = nil
	redisClient = nil
	taskFactory = nil
	pullingQueue = nil
	store = storage.NewLocalStorage()
}

func configureMockedGitlabClient() (*http.ServeMux, *httptest.Server) {
	cfgUpdateLock.Lock()
	defer cfgUpdateLock.Unlock()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	opts := []goGitlab.ClientOptionFunc{
		goGitlab.WithBaseURL(server.URL),
		goGitlab.WithoutRetries(),
	}

	gc, _ := goGitlab.NewClient("", opts...)

	gitlabClient = &gitlab.Client{
		Client:      gc,
		RateLimiter: ratelimit.NewLocalLimiter(100),
	}

	return mux, server
}

// func TestConfigure(t *testing.T) {
// 	resetGlobalValues()

// 	_cfg := config.New()
// 	_cfg.Gitlab.URL = "http://foo.bar"
// 	_cfg.Pull.MaximumGitLabAPIRequestsPerSecond = 1

// 	assert.NoError(t, Configure(_cfg, ""))
// 	assert.Equal(t, _cfg, cfg)
// }

func TestConfigureGitlabClient(t *testing.T) {
	resetGlobalValues()

	cfg.Pull.MaximumGitLabAPIRequestsPerSecond = 1
	configureGitlabClient("yolo")
	assert.NotNil(t, gitlabClient)
}

func TestConfigureRedisClient(t *testing.T) {
	resetGlobalValues()

	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	c := redis.NewClient(&redis.Options{Addr: s.Addr()})
	assert.NoError(t, ConfigureRedisClient(c))
	assert.Equal(t, redisClient, c)

	s.Close()
	assert.Error(t, ConfigureRedisClient(c))
}

// TODO: Sort out why this creates loads of race issues across
func TestConfigurePullingQueue(t *testing.T) {
	resetGlobalValues()

	// TODO: Test with redis client, miniredis does not seem to support it yet
	configurePullingQueue()
	assert.Equal(t, "pull", pullingQueue.Options().Name)
}

func TestConfigureStore(t *testing.T) {
	resetGlobalValues()

	cfg = config.Config{
		Projects: []config.Project{
			{
				Name: "foo/bar",
			},
		},
	}

	// Test with local storage
	configureStore()
	assert.NotNil(t, store)

	projects, err := store.Projects()
	assert.NoError(t, err)

	expectedProjects := config.Projects{
		"3861188962": config.Project{
			Name: "foo/bar",
		},
	}
	assert.Equal(t, expectedProjects, projects)

	// Test with redis storage
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	c := redis.NewClient(&redis.Options{Addr: s.Addr()})
	assert.NoError(t, ConfigureRedisClient(c))

	configureStore()
	projects, err = store.Projects()
	assert.NoError(t, err)
	assert.Equal(t, expectedProjects, projects)
}

func TestProcessPullingQueue(_ *testing.T) {
	resetGlobalValues()

	// TODO: Test with redis client, miniredis does not seem to support it yet
	processPullingQueue(context.TODO())
}