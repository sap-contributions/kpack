package cnb

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
	buildapi "github.com/pivotal/kpack/pkg/apis/build/v1alpha2"
)

type RemoteBuildpackInfo struct {
	BuildpackInfo DescriptiveBuildpackInfo
	Layers        []buildpackLayer
}

func (i RemoteBuildpackInfo) Optional(optional bool) RemoteBuildpackRef {
	return RemoteBuildpackRef{
		DescriptiveBuildpackInfo: i.BuildpackInfo,
		Optional:                 optional,
		Layers:                   i.Layers,
	}
}

type RemoteBuildpackRef struct {
	DescriptiveBuildpackInfo DescriptiveBuildpackInfo
	Optional                 bool
	Layers                   []buildpackLayer
}

func (r RemoteBuildpackRef) buildpackRef() buildapi.BuildpackRef {
	return buildapi.BuildpackRef{
		BuildpackInfo: r.DescriptiveBuildpackInfo.BuildpackInfo,
		Optional:      r.Optional,
	}
}

type buildpackLayer struct {
	v1Layer            v1.Layer
	BuildpackInfo      DescriptiveBuildpackInfo
	BuildpackLayerInfo BuildpackLayerInfo
}
