// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.17.3
// source: metricspb.proto

package metricspb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ScrapeTargetsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Instance name refers to the metrics.configs[].name field from the Agent config.
	InstanceName string `protobuf:"bytes,1,opt,name=instance_name,json=instanceName,proto3" json:"instance_name,omitempty"`
	// Targets is the set of targets to collect from.
	// Corresponds to map[string][]*targetgroup.Group from Prometheus.
	Targets map[string]*TargetSet `protobuf:"bytes,2,rep,name=targets,proto3" json:"targets,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Tombstones is the set of targets that have been removed since the
	// last discovery.
	Tombstones map[string]*TargetSet `protobuf:"bytes,3,rep,name=tombstones,proto3" json:"tombstones,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Time, in nanoseconds, when these targets were discovered. Receiving a request
	// with a newer discovery time should always override requests with older times.
	DiscoveryTime int64 `protobuf:"varint,4,opt,name=discovery_time,json=discoveryTime,proto3" json:"discovery_time,omitempty"`
	// TTL is how long, in nanoseconds, the group for this discovery_id should
	// last. New groups with a new discovery_id will be sent every TTL/2
	// nanoseconds. If the TTL expires without receiving a new discovery_id,
	// the job should be discarded.
	Ttl int64 `protobuf:"varint,6,opt,name=ttl,proto3" json:"ttl,omitempty"`
}

func (x *ScrapeTargetsRequest) Reset() {
	*x = ScrapeTargetsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metricspb_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScrapeTargetsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScrapeTargetsRequest) ProtoMessage() {}

func (x *ScrapeTargetsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_metricspb_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScrapeTargetsRequest.ProtoReflect.Descriptor instead.
func (*ScrapeTargetsRequest) Descriptor() ([]byte, []int) {
	return file_metricspb_proto_rawDescGZIP(), []int{0}
}

func (x *ScrapeTargetsRequest) GetInstanceName() string {
	if x != nil {
		return x.InstanceName
	}
	return ""
}

func (x *ScrapeTargetsRequest) GetTargets() map[string]*TargetSet {
	if x != nil {
		return x.Targets
	}
	return nil
}

func (x *ScrapeTargetsRequest) GetTombstones() map[string]*TargetSet {
	if x != nil {
		return x.Tombstones
	}
	return nil
}

func (x *ScrapeTargetsRequest) GetDiscoveryTime() int64 {
	if x != nil {
		return x.DiscoveryTime
	}
	return 0
}

func (x *ScrapeTargetsRequest) GetTtl() int64 {
	if x != nil {
		return x.Ttl
	}
	return 0
}

// TargetSet is a list of target groups. Correponds to []*targetgroup.Group
// from Prometheus.
type TargetSet struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Groups []*TargetGroup `protobuf:"bytes,1,rep,name=groups,proto3" json:"groups,omitempty"`
}

func (x *TargetSet) Reset() {
	*x = TargetSet{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metricspb_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TargetSet) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TargetSet) ProtoMessage() {}

func (x *TargetSet) ProtoReflect() protoreflect.Message {
	mi := &file_metricspb_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TargetSet.ProtoReflect.Descriptor instead.
func (*TargetSet) Descriptor() ([]byte, []int) {
	return file_metricspb_proto_rawDescGZIP(), []int{1}
}

func (x *TargetSet) GetGroups() []*TargetGroup {
	if x != nil {
		return x.Groups
	}
	return nil
}

// TargetGroup is a group of targets with a related source. Corresponds to
// *targetgroup.Group from Prometheus.
type TargetGroup struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Targets []*LabelSet `protobuf:"bytes,1,rep,name=targets,proto3" json:"targets,omitempty"`
	Labels  *LabelSet   `protobuf:"bytes,2,opt,name=labels,proto3" json:"labels,omitempty"`
	Source  string      `protobuf:"bytes,3,opt,name=source,proto3" json:"source,omitempty"`
}

func (x *TargetGroup) Reset() {
	*x = TargetGroup{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metricspb_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TargetGroup) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TargetGroup) ProtoMessage() {}

func (x *TargetGroup) ProtoReflect() protoreflect.Message {
	mi := &file_metricspb_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TargetGroup.ProtoReflect.Descriptor instead.
func (*TargetGroup) Descriptor() ([]byte, []int) {
	return file_metricspb_proto_rawDescGZIP(), []int{2}
}

func (x *TargetGroup) GetTargets() []*LabelSet {
	if x != nil {
		return x.Targets
	}
	return nil
}

func (x *TargetGroup) GetLabels() *LabelSet {
	if x != nil {
		return x.Labels
	}
	return nil
}

func (x *TargetGroup) GetSource() string {
	if x != nil {
		return x.Source
	}
	return ""
}

// LabelSet is a set of labels for a target. Corresponds to model.LabelSet
// from Prometheus.
type LabelSet struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Labels map[string]string `protobuf:"bytes,1,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *LabelSet) Reset() {
	*x = LabelSet{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metricspb_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LabelSet) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LabelSet) ProtoMessage() {}

func (x *LabelSet) ProtoReflect() protoreflect.Message {
	mi := &file_metricspb_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LabelSet.ProtoReflect.Descriptor instead.
func (*LabelSet) Descriptor() ([]byte, []int) {
	return file_metricspb_proto_rawDescGZIP(), []int{3}
}

func (x *LabelSet) GetLabels() map[string]string {
	if x != nil {
		return x.Labels
	}
	return nil
}

type ScrapeTargetsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ScrapeTargetsResponse) Reset() {
	*x = ScrapeTargetsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_metricspb_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ScrapeTargetsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ScrapeTargetsResponse) ProtoMessage() {}

func (x *ScrapeTargetsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_metricspb_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ScrapeTargetsResponse.ProtoReflect.Descriptor instead.
func (*ScrapeTargetsResponse) Descriptor() ([]byte, []int) {
	return file_metricspb_proto_rawDescGZIP(), []int{4}
}

var File_metricspb_proto protoreflect.FileDescriptor

var file_metricspb_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x70, 0x62, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x18, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74,
	0x2e, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e, 0x61, 0x2e, 0x76, 0x31, 0x22, 0xf0, 0x03, 0x0a, 0x14,
	0x53, 0x63, 0x72, 0x61, 0x70, 0x65, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x12, 0x23, 0x0a, 0x0d, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x69, 0x6e, 0x73,
	0x74, 0x61, 0x6e, 0x63, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x55, 0x0a, 0x07, 0x74, 0x61, 0x72,
	0x67, 0x65, 0x74, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x3b, 0x2e, 0x6d, 0x65, 0x74,
	0x72, 0x69, 0x63, 0x73, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x67, 0x72, 0x61, 0x66, 0x61,
	0x6e, 0x61, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x63, 0x72, 0x61, 0x70, 0x65, 0x54, 0x61, 0x72, 0x67,
	0x65, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x54, 0x61, 0x72, 0x67, 0x65,
	0x74, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x07, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73,
	0x12, 0x5e, 0x0a, 0x0a, 0x74, 0x6f, 0x6d, 0x62, 0x73, 0x74, 0x6f, 0x6e, 0x65, 0x73, 0x18, 0x03,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x3e, 0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x61,
	0x67, 0x65, 0x6e, 0x74, 0x2e, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e, 0x61, 0x2e, 0x76, 0x31, 0x2e,
	0x53, 0x63, 0x72, 0x61, 0x70, 0x65, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x2e, 0x54, 0x6f, 0x6d, 0x62, 0x73, 0x74, 0x6f, 0x6e, 0x65, 0x73, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x52, 0x0a, 0x74, 0x6f, 0x6d, 0x62, 0x73, 0x74, 0x6f, 0x6e, 0x65, 0x73,
	0x12, 0x25, 0x0a, 0x0e, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x79, 0x5f, 0x74, 0x69,
	0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0d, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76,
	0x65, 0x72, 0x79, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x74, 0x74, 0x6c, 0x18, 0x06,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x03, 0x74, 0x74, 0x6c, 0x1a, 0x5f, 0x0a, 0x0c, 0x54, 0x61, 0x72,
	0x67, 0x65, 0x74, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x39, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x6d, 0x65, 0x74,
	0x72, 0x69, 0x63, 0x73, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x67, 0x72, 0x61, 0x66, 0x61,
	0x6e, 0x61, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x53, 0x65, 0x74, 0x52,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x62, 0x0a, 0x0f, 0x54, 0x6f,
	0x6d, 0x62, 0x73, 0x74, 0x6f, 0x6e, 0x65, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12,
	0x39, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23,
	0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x67,
	0x72, 0x61, 0x66, 0x61, 0x6e, 0x61, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74,
	0x53, 0x65, 0x74, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x4a,
	0x0a, 0x09, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x53, 0x65, 0x74, 0x12, 0x3d, 0x0a, 0x06, 0x67,
	0x72, 0x6f, 0x75, 0x70, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x6d, 0x65,
	0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x67, 0x72, 0x61, 0x66,
	0x61, 0x6e, 0x61, 0x2e, 0x76, 0x31, 0x2e, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x47, 0x72, 0x6f,
	0x75, 0x70, 0x52, 0x06, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x73, 0x22, 0x9f, 0x01, 0x0a, 0x0b, 0x54,
	0x61, 0x72, 0x67, 0x65, 0x74, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x12, 0x3c, 0x0a, 0x07, 0x74, 0x61,
	0x72, 0x67, 0x65, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x6d, 0x65,
	0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x67, 0x72, 0x61, 0x66,
	0x61, 0x6e, 0x61, 0x2e, 0x76, 0x31, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x53, 0x65, 0x74, 0x52,
	0x07, 0x74, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x12, 0x3a, 0x0a, 0x06, 0x6c, 0x61, 0x62, 0x65,
	0x6c, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69,
	0x63, 0x73, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e, 0x61,
	0x2e, 0x76, 0x31, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x53, 0x65, 0x74, 0x52, 0x06, 0x6c, 0x61,
	0x62, 0x65, 0x6c, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x22, 0x8d, 0x01, 0x0a,
	0x08, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x53, 0x65, 0x74, 0x12, 0x46, 0x0a, 0x06, 0x6c, 0x61, 0x62,
	0x65, 0x6c, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x6d, 0x65, 0x74, 0x72,
	0x69, 0x63, 0x73, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e,
	0x61, 0x2e, 0x76, 0x31, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x53, 0x65, 0x74, 0x2e, 0x4c, 0x61,
	0x62, 0x65, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x6c, 0x61, 0x62, 0x65, 0x6c,
	0x73, 0x1a, 0x39, 0x0a, 0x0b, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x17, 0x0a, 0x15,
	0x53, 0x63, 0x72, 0x61, 0x70, 0x65, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x32, 0x7b, 0x0a, 0x07, 0x53, 0x63, 0x72, 0x61, 0x70, 0x65, 0x72,
	0x12, 0x70, 0x0a, 0x0d, 0x53, 0x63, 0x72, 0x61, 0x70, 0x65, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74,
	0x73, 0x12, 0x2e, 0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x61, 0x67, 0x65, 0x6e,
	0x74, 0x2e, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e, 0x61, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x63, 0x72,
	0x61, 0x70, 0x65, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x2f, 0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x61, 0x67, 0x65, 0x6e,
	0x74, 0x2e, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e, 0x61, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x63, 0x72,
	0x61, 0x70, 0x65, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x42, 0x39, 0x5a, 0x37, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d,
	0x2f, 0x67, 0x72, 0x61, 0x66, 0x61, 0x6e, 0x61, 0x2f, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2f, 0x70,
	0x6b, 0x67, 0x2f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x6e, 0x61, 0x6c, 0x2f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x70, 0x62, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_metricspb_proto_rawDescOnce sync.Once
	file_metricspb_proto_rawDescData = file_metricspb_proto_rawDesc
)

func file_metricspb_proto_rawDescGZIP() []byte {
	file_metricspb_proto_rawDescOnce.Do(func() {
		file_metricspb_proto_rawDescData = protoimpl.X.CompressGZIP(file_metricspb_proto_rawDescData)
	})
	return file_metricspb_proto_rawDescData
}

var file_metricspb_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_metricspb_proto_goTypes = []interface{}{
	(*ScrapeTargetsRequest)(nil),  // 0: metrics.agent.grafana.v1.ScrapeTargetsRequest
	(*TargetSet)(nil),             // 1: metrics.agent.grafana.v1.TargetSet
	(*TargetGroup)(nil),           // 2: metrics.agent.grafana.v1.TargetGroup
	(*LabelSet)(nil),              // 3: metrics.agent.grafana.v1.LabelSet
	(*ScrapeTargetsResponse)(nil), // 4: metrics.agent.grafana.v1.ScrapeTargetsResponse
	nil,                           // 5: metrics.agent.grafana.v1.ScrapeTargetsRequest.TargetsEntry
	nil,                           // 6: metrics.agent.grafana.v1.ScrapeTargetsRequest.TombstonesEntry
	nil,                           // 7: metrics.agent.grafana.v1.LabelSet.LabelsEntry
}
var file_metricspb_proto_depIdxs = []int32{
	5, // 0: metrics.agent.grafana.v1.ScrapeTargetsRequest.targets:type_name -> metrics.agent.grafana.v1.ScrapeTargetsRequest.TargetsEntry
	6, // 1: metrics.agent.grafana.v1.ScrapeTargetsRequest.tombstones:type_name -> metrics.agent.grafana.v1.ScrapeTargetsRequest.TombstonesEntry
	2, // 2: metrics.agent.grafana.v1.TargetSet.groups:type_name -> metrics.agent.grafana.v1.TargetGroup
	3, // 3: metrics.agent.grafana.v1.TargetGroup.targets:type_name -> metrics.agent.grafana.v1.LabelSet
	3, // 4: metrics.agent.grafana.v1.TargetGroup.labels:type_name -> metrics.agent.grafana.v1.LabelSet
	7, // 5: metrics.agent.grafana.v1.LabelSet.labels:type_name -> metrics.agent.grafana.v1.LabelSet.LabelsEntry
	1, // 6: metrics.agent.grafana.v1.ScrapeTargetsRequest.TargetsEntry.value:type_name -> metrics.agent.grafana.v1.TargetSet
	1, // 7: metrics.agent.grafana.v1.ScrapeTargetsRequest.TombstonesEntry.value:type_name -> metrics.agent.grafana.v1.TargetSet
	0, // 8: metrics.agent.grafana.v1.Scraper.ScrapeTargets:input_type -> metrics.agent.grafana.v1.ScrapeTargetsRequest
	4, // 9: metrics.agent.grafana.v1.Scraper.ScrapeTargets:output_type -> metrics.agent.grafana.v1.ScrapeTargetsResponse
	9, // [9:10] is the sub-list for method output_type
	8, // [8:9] is the sub-list for method input_type
	8, // [8:8] is the sub-list for extension type_name
	8, // [8:8] is the sub-list for extension extendee
	0, // [0:8] is the sub-list for field type_name
}

func init() { file_metricspb_proto_init() }
func file_metricspb_proto_init() {
	if File_metricspb_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_metricspb_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScrapeTargetsRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_metricspb_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TargetSet); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_metricspb_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TargetGroup); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_metricspb_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LabelSet); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_metricspb_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ScrapeTargetsResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_metricspb_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_metricspb_proto_goTypes,
		DependencyIndexes: file_metricspb_proto_depIdxs,
		MessageInfos:      file_metricspb_proto_msgTypes,
	}.Build()
	File_metricspb_proto = out.File
	file_metricspb_proto_rawDesc = nil
	file_metricspb_proto_goTypes = nil
	file_metricspb_proto_depIdxs = nil
}
