package multihttp

import (
	"fmt"
	"hash/fnv"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/prometheus/common/model"
	prom_config "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/relabel"
	"github.com/prometheus/prometheus/scrape"
)

// Service refers to a single HTTP or HTTPS endpoint that exposes targets.
type Service struct {
	labelsMut        sync.RWMutex
	discoveredLabels labels.Labels // Pre-relabeling labels

	labels labels.Labels // Post-relabeling labels
	params url.Values    // Query parameters

	mut                 sync.RWMutex
	lastError           error
	lastRefresh         time.Time
	lastRefreshDuration time.Duration
	health              scrape.TargetHealth
}

// NewService initializes a a service for querying.
func NewService(labels, discoveredLabels labels.Labels, params url.Values) *Service {
	return &Service{
		labels:           labels,
		discoveredLabels: discoveredLabels,
		params:           params,
		health:           scrape.HealthUnknown,
	}
}

// String returns the URL of the service.
func (s *Service) String() string { return s.URL().String() }

// Labels returns a copy of the public labels of the service.
func (s *Service) Labels() labels.Labels {
	lset := make(labels.Labels, 0, len(s.labels))
	for _, l := range s.labels {
		if !strings.HasPrefix(l.Name, model.ReservedLabelPrefix) {
			lset = append(lset, l)
		}
	}
	return lset
}

// DiscoveredLabels returns a copy of the service's labels before any
// processing.
func (s *Service) DiscoveredLabels() labels.Labels {
	s.labelsMut.RLock()
	defer s.labelsMut.RUnlock()

	lset := make(labels.Labels, len(s.discoveredLabels))
	copy(lset, s.discoveredLabels)
	return lset
}

func (s *Service) setDiscoveredLabels(ls labels.Labels) {
	s.labelsMut.Lock()
	defer s.labelsMut.Unlock()
	s.discoveredLabels = ls
}

// URL returns a copy of the service's URL.
func (s *Service) URL() *url.URL {
	params := url.Values{}

	for k, v := range s.params {
		params[k] = make([]string, len(v))
		copy(params[k], v)
	}
	for _, l := range s.labels {
		if !strings.HasPrefix(l.Name, model.ParamLabelPrefix) {
			continue
		}
		ks := l.Name[len(model.ParamLabelPrefix):]

		if len(params[ks]) > 0 {
			params[ks][0] = l.Value
		} else {
			params[ks] = []string{l.Value}
		}
	}

	return &url.URL{
		Scheme:   s.labels.Get(model.SchemeLabel),
		Host:     s.labels.Get(model.AddressLabel),
		Path:     s.labels.Get(APIPathLabel),
		RawQuery: params.Encode(),
	}
}

// Report sets information about the most recent query.
func (s *Service) Report(start time.Time, dur time.Duration, err error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	if err == nil {
		s.health = scrape.HealthGood
	} else {
		s.health = scrape.HealthBad
	}

	s.lastError = err
	s.lastRefresh = start
	s.lastRefreshDuration = dur
}

// LastError returns the error encountered during the most recent query.
func (s *Service) LastError() error {
	s.mut.RLock()
	defer s.mut.RUnlock()
	return s.lastError
}

// LastRefresh returns the last time a refresh ran for a service.
func (s *Service) LastRefresh() time.Time {
	s.mut.RLock()
	defer s.mut.RUnlock()
	return s.lastRefresh
}

// LastRefreshDuration returns the error encountered during the most recent query.
func (s *Service) LastRefreshDuration() time.Duration {
	s.mut.RLock()
	defer s.mut.RUnlock()
	return s.lastRefreshDuration
}

// Health returns the current health state of the service.
func (s *Service) Health() scrape.TargetHealth {
	s.mut.RLock()
	defer s.mut.RUnlock()
	return s.health
}

// hash hashes the service.
func (s *Service) hash() uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(fmt.Sprintf("%016d", s.labels.Hash())))
	_, _ = h.Write([]byte(s.URL().String()))
	return h.Sum64()
}

// Services is a sortable list of services.
type Services []*Service

// Len implements sort.Interface.
func (ss Services) Len() int { return len(ss) }

// Less implements sort.Interface.
func (ss Services) Less(i, j int) bool { return ss[i].URL().String() < ss[j].URL().String() }

// Swap implements sort.Interface.
func (ss Services) Swap(i, j int) { ss[i], ss[j] = ss[j], ss[i] }

// servicesFromGroup builds a set of services from a target group. Dropped
// services will have an empty label set.
func servicesFromGroup(group *targetgroup.Group, cfg *Config) ([]*Service, error) {
	var (
		res  = make([]*Service, 0, len(group.Targets))
		errs *multierror.Error
	)

	for i, targetLabels := range group.Targets {
		// Combine the labels from the target and group, where labels from the
		// group are only used if the target doesn't also have them.
		allLabels := make([]labels.Label, 0, len(targetLabels)+len(group.Labels))

		for k, v := range targetLabels {
			allLabels = append(allLabels, labels.Label{Name: string(k), Value: string(v)})
		}
		for k, v := range group.Labels {
			if _, exist := targetLabels[k]; !exist {
				allLabels = append(allLabels, labels.Label{Name: string(k), Value: string(v)})
			}
		}

		lset := labels.New(allLabels...)
		finalLabels, origLabels, err := populateLabels(lset, cfg)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("instance %d in group %q", i, group))
		}
		if finalLabels != nil || origLabels != nil {
			res = append(res, NewService(finalLabels, origLabels, cfg.Params))
		}
	}

	return res, errs.ErrorOrNil()
}

// populateLabels builds a label set based on the multihttp config. The final
// set of labels will be nil if the target was dropped during relabeling.
func populateLabels(lset labels.Labels, cfg *Config) (final, orig labels.Labels, err error) {
	// NOTE(rfratto): Prometheus does things very differently here, since the
	// intent is to generate targets that will be scraped. However, we're
	// generating services to collect targets from, so we want to generate different labels:
	//
	// __api_path__    API path where targets are exposed
	// __scheme__      URL scheme to use for collecting targets
	// __address__     host:port where targets are exposed
	// __param_<name>  URL query parameter
	//
	// This function does _not_ explicitly generate any label not prefixed with __.
	// Labels not prefixed with __ from relabeling will be copied to the final
	// discovered targets.

	defaultLabels := []labels.Label{
		{Name: APIPathLabel, Value: cfg.APIPath},
		{Name: model.SchemeLabel, Value: cfg.Scheme},
	}
	lb := labels.NewBuilder(lset)

	for _, dl := range defaultLabels {
		if !lset.Has(dl.Name) {
			lb.Set(dl.Name, dl.Value)
		}
	}
	// Encode the query parameters as labels.
	for k, v := range cfg.Params {
		if len(v) > 0 {
			lb.Set(model.ParamLabelPrefix+k, v[0])
		}
	}

	orig = lb.Labels()
	lset = relabel.Process(orig, cfg.RelabelConfigs...)

	// lset will be nil if the target got dropped.
	if lset == nil {
		return nil, orig, nil
	} else if !lset.Has(model.AddressLabel) {
		return nil, nil, fmt.Errorf("missing address label")
	}

	lb = labels.NewBuilder(lset)

	addr := lset.Get(model.AddressLabel)
	if !hasPort(addr) {
		// Append a port if one is missing.
		switch lset.Get(model.SchemeLabel) {
		case "http", "":
			addr = addr + ":80"
		case "https":
			addr = addr + ":443"
		default:
			return nil, nil, fmt.Errorf("invalid scheme: %q", cfg.Scheme)
		}
		lb.Set(model.AddressLabel, addr)
	}

	if err := prom_config.CheckTargetAddress(model.LabelValue(addr)); err != nil {
		return nil, nil, err
	}

	// Meta labels are deleted after relabelling. Other internal labels propagate
	// to the service which decides whether they will be part of their label set.
	for _, l := range lset {
		if strings.HasPrefix(l.Name, model.MetaLabelPrefix) {
			lb.Del(l.Name)
		}
	}

	final = lb.Labels()
	for _, l := range final {
		// Check label values are valid, drop the target if not.
		if !model.LabelValue(l.Value).IsValid() {
			return nil, nil, fmt.Errorf("invalid label value for %q: %q", l.Name, l.Value)
		}
	}
	return final, orig, nil
}

// hasPort returns true if the addr has a port in it.
func hasPort(addr string) bool {
	// If we can split, a port exists and we don't have to add one.
	if _, _, err := net.SplitHostPort(addr); err == nil {
		return true
	}
	// If adding a port makes it valid, then the prevous error was because the
	// port wasn't present.
	_, _, err := net.SplitHostPort(addr + ":1234")
	return err != nil
}
