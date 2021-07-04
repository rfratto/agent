package metricspb

import "github.com/rfratto/croissant/id"

type RequestParititions = map[id.ID]*ScrapeTargetsRequest

// CountTargets returns the target count in req.
func CountTargets(req *ScrapeTargetsRequest) (count int) {
	for _, tset := range req.Targets {
		for _, group := range tset.Groups {
			count += len(group.Targets)
		}
	}
	return
}

// GetPartition gets or makes a partition for key.
func GetPartition(ps RequestParititions, key id.ID, ref *ScrapeTargetsRequest) *ScrapeTargetsRequest {
	p, ok := ps[key]
	if !ok {
		p = &ScrapeTargetsRequest{
			InstanceName: ref.InstanceName,
			Targets:      make(map[string]*TargetSet),
			Tombstones:   make(map[string]*TargetSet),
			Ttl:          ref.Ttl,
		}
		ps[key] = p
	}
	return p
}

// GetTargetSet gets or creates a new TargetSet by name.
func GetTargetSet(p *ScrapeTargetsRequest, key string) *TargetSet {
	tset, ok := p.Targets[key]
	if !ok {
		tset = &TargetSet{}
		p.Targets[key] = tset
	}
	return tset
}

// GetTombstones gets or creates a new tombstone TargetSet by name.
func GetTombstones(p *ScrapeTargetsRequest, key string) *TargetSet {
	tset, ok := p.Tombstones[key]
	if !ok {
		tset = &TargetSet{}
		p.Tombstones[key] = tset
	}
	return tset
}

// GetTargetGroup gets or creates a target group based on a reference from tset.
// No targets will be copied if creating a group.
func GetTargetGroup(tset *TargetSet, ref *TargetGroup) *TargetGroup {
	var tgroup *TargetGroup
	for _, g := range tset.Groups {
		if g.Source == ref.Source {
			return g
		}
	}
	tgroup = &TargetGroup{
		Source: ref.Source,
		Labels: ref.Labels,
	}
	tset.Groups = append(tset.Groups, tgroup)
	return tgroup
}
