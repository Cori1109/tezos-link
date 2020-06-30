package usecases

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/octo-technology/tezos-link/backend/config"
	"github.com/octo-technology/tezos-link/backend/internal/proxy/domain/repository"
	"github.com/octo-technology/tezos-link/backend/pkg/domain/errors"
	pkgmodel "github.com/octo-technology/tezos-link/backend/pkg/domain/model"
	pkgrepository "github.com/octo-technology/tezos-link/backend/pkg/domain/repository"
	"github.com/octo-technology/tezos-link/backend/pkg/infrastructure/database/inputs"
	"github.com/sirupsen/logrus"
)

// ProxyUsecase contains the repositories and regexes to route paths proxying and store metrics
type ProxyUsecase struct {
	cacheRepo        repository.BlockchainRepository
	proxyRepo        repository.BlockchainRepository
	metricsRepo      pkgrepository.MetricsRepository
	projectRepo      pkgrepository.ProjectRepository
	projectCacheRepo pkgrepository.ProjectRepository
	metricsCacheRepo repository.MetricInputRepository
	whitelisted      []*regexp.Regexp
	blacklisted      []*regexp.Regexp
	dontCache        []*regexp.Regexp
	rollingPatterns  []*regexp.Regexp
	baseArchiveURL   string
	baseRollingURL   string
}

// ProxyUsecaseInterface contains all methods implemented by the proxyRepo use-case
type ProxyUsecaseInterface interface {
	Proxy(request *pkgmodel.Request) (string, bool, error)
}

// NoProxyResponse is the error message when there is no response from the proxyRepo
const NoProxyResponse = "no response from proxy"

// NewProxyUsecase returns a new proxy use-case
func NewProxyUsecase(
	cacheRepo repository.BlockchainRepository,
	proxyRepo repository.BlockchainRepository,
	metricsRepo pkgrepository.MetricsRepository,
	projectRepo pkgrepository.ProjectRepository,
	projectCacheRepo pkgrepository.ProjectRepository,
	metricsCacheRepo repository.MetricInputRepository) *ProxyUsecase {

	baseArchiveURL := "http://" + config.ProxyConfig.Tezos.ArchiveHost + ":" + strconv.Itoa(config.ProxyConfig.Tezos.ArchivePort)
	baseRollingURL := "http://" + config.ProxyConfig.Tezos.RollingHost + ":" + strconv.Itoa(config.ProxyConfig.Tezos.RollingPort)
	return &ProxyUsecase{
		cacheRepo:        cacheRepo,
		proxyRepo:        proxyRepo,
		metricsRepo:      metricsRepo,
		projectRepo:      projectRepo,
		projectCacheRepo: projectCacheRepo,
		metricsCacheRepo: metricsCacheRepo,
		whitelisted:      setupRegexpFor(config.ProxyConfig.Proxy.WhitelistedMethods),
		blacklisted:      setupRegexpFor(config.ProxyConfig.Proxy.BlockedMethods),
		dontCache:        setupRegexpFor(config.ProxyConfig.Proxy.DontCache),
		rollingPatterns:  setupRegexpFor(config.ProxyConfig.Proxy.WhitelistedRolling),
		baseArchiveURL:   baseArchiveURL,
		baseRollingURL:   baseRollingURL,
	}
}

func (p *ProxyUsecase) findInDatabaseIfNotFoundInCache(UUID string) error {
	_, err := p.projectCacheRepo.FindByUUID(UUID)

	if err != nil {
		logrus.Debug("project ID not found in cache: ", UUID, err.Error())
		prj, err := p.projectRepo.FindByUUID(UUID)
		if err != nil {
			logrus.Debug("project ID not found: ", UUID, err.Error())
			return err
		}

		_, err = p.projectCacheRepo.Save(prj.Title, prj.UUID, prj.CreationDate)
	}

	return nil
}


func (p *ProxyUsecase) WriteCachedRequestsRoutine() {
	logrus.Info("Starting to write cached requests to database")
	cachedMetrics, err := p.metricsCacheRepo.GetAll()
	if err != nil {
		logrus.Errorf("could not get cached metrics: %s", err)
	}
	logrus.Infof("got %d cached metrics", len(cachedMetrics))
	err = p.metricsRepo.SaveMany(cachedMetrics)
	if err != nil {
		logrus.Errorf("could not save metrics in database: %s", err)
	}
	logrus.Infof("Successfully saved %d cached metrics in database", len(cachedMetrics))
	time.Sleep(time.Duration(config.ProxyConfig.Proxy.RoutineDelaySeconds) * time.Second)
}

func (p *ProxyUsecase) IsRollingRedirection(url string) bool {
	ret := false
	urls := strings.Split(url, "?")
	url = "/" + strings.Trim(urls[0], "/")

	for _, wl := range p.rollingPatterns {
		if wl.Match([]byte(url)) {
			ret = true
			break
		}
	}

	return ret
}

// Proxy proxy an http request to the right repositories
func (p *ProxyUsecase) Proxy(request *pkgmodel.Request) (string, bool, error) {
	logrus.Info("received proxy request for path: ", request.Path)
	response := []byte("call blacklisted")

	err := p.findInDatabaseIfNotFoundInCache(request.UUID)

	if err != nil {
		return err.Error(), false, err
	}

	if !p.isAllowed(request.Path) {
		logrus.Debug("not allowed to proxy on the path: ", request.Path)
		return string(response), false, nil
	}

	if request.Action == pkgmodel.OBTAIN && p.isCacheable(request.Path) {
		url := p.baseArchiveURL + request.Path

		response, err := p.cacheRepo.Get(request, url)
		if err != nil {
			logrus.Info("path not cached, fetching to node: ", request.Path)

			if p.IsRollingRedirection(request.Path) {
				url = p.baseRollingURL + request.Path
			}

			response, err = p.proxyRepo.Get(request, url)
			if err != nil {
				logrus.Errorf("could not request to proxy: %s", err)
				return errors.ErrNoProxyResponse.Error(), false, errors.ErrNoProxyResponse
			}
			logrus.Info("received response from node: ", string(response.([]byte)))

			_ = p.cacheRepo.Add(request, response)
		}

		// TODO save that it is cached from the LRU or not
		p.saveMetrics(request)
		return string(response.([]byte)), false, nil
	}

	p.saveMetrics(request)
	return "", true, nil
}

func (p *ProxyUsecase) saveMetrics(request *pkgmodel.Request) {
	metrics := inputs.NewMetricsInput(request, time.Now().UTC())

	// add to cache
	err := p.metricsCacheRepo.Add(&metrics)
	if err != nil {
		logrus.Errorf("could not cache the metrics: %s", err)
	}

	logrus.Info("metric input has been added to cache metrics")
	// check limit reached if yes save in database
	nb := p.metricsCacheRepo.Len()
	if nb >= config.ProxyConfig.Proxy.CacheMaxMetricItems {
		logrus.Info("cache metrics limit has been reached, saving cached metrics in database ...")
		allRequests, err := p.metricsCacheRepo.GetAll()
		if err != nil {
			logrus.Errorf("could not retrieve cached metrics: %s", err)
		}
		err = p.metricsRepo.SaveMany(allRequests)
		if err != nil {
			logrus.Errorf("could not save metrics in database: %s", err)
		}
	}

}

func (p *ProxyUsecase) isAllowed(url string) bool {
	ret := false
	urls := strings.Split(url, "?")
	url = "/" + strings.Trim(urls[0], "/")

	for _, wl := range p.whitelisted {
		if wl.Match([]byte(url)) {
			ret = true
			for _, bl := range p.blacklisted {
				if bl.Match([]byte(url)) {
					ret = false
					break
				}
			}
			break
		}
	}

	return ret
}

func (p *ProxyUsecase) isCacheable(url string) bool {
	ret := true

	for _, wl := range p.dontCache {
		if wl.Match([]byte(url)) {
			ret = false
		}
	}

	return ret
}

func setupRegexpFor(regexPaths []string) []*regexp.Regexp {
	var list []*regexp.Regexp

	for _, s := range regexPaths {
		regex, err := regexp.Compile(s)
		if err != nil {
			logrus.Error("could not compile Regexp: ", s)
		} else {
			list = append(list, regex)
		}
	}

	return list
}
