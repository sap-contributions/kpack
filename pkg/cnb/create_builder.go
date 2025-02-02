package cnb

import (
	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"

	buildapi "github.com/pivotal/kpack/pkg/apis/build/v1alpha2"
)

type RegistryClient interface {
	Fetch(keychain authn.Keychain, repoName string) (v1.Image, string, error)
	Save(keychain authn.Keychain, tag string, image v1.Image) (string, error)
}

type BuildpackRepository interface {
	FindByIdAndVersion(id, version string) (RemoteBuildpackInfo, error)
}

type LifecycleProvider interface {
	GetImage() (v1.Image, error)
}

type NewBuildpackRepository func(clusterStore *buildapi.ClusterStore) BuildpackRepository

type RemoteBuilderCreator struct {
	RegistryClient         RegistryClient
	NewBuildpackRepository NewBuildpackRepository
	LifecycleProvider      LifecycleProvider
	KpackVersion           string
}

func (r *RemoteBuilderCreator) CreateBuilder(keychain authn.Keychain, clusterStore *buildapi.ClusterStore, clusterStack *buildapi.ClusterStack, spec buildapi.BuilderSpec) (buildapi.BuilderRecord, error) {
	buildpackRepo := r.NewBuildpackRepository(clusterStore)

	buildImage, _, err := r.RegistryClient.Fetch(keychain, clusterStack.Status.BuildImage.LatestImage)
	if err != nil {
		return buildapi.BuilderRecord{}, err
	}

	lifecycleImage, err := r.LifecycleProvider.GetImage()
	if err != nil {
		return buildapi.BuilderRecord{}, err
	}

	builderBldr, err := newBuilderBldr(lifecycleImage, r.KpackVersion)
	if err != nil {
		return buildapi.BuilderRecord{}, err
	}

	err = builderBldr.AddStack(buildImage, clusterStack)
	if err != nil {
		return buildapi.BuilderRecord{}, err
	}

	for _, group := range spec.Order {
		buildpacks := make([]RemoteBuildpackRef, 0, len(group.Group))

		for _, buildpack := range group.Group {
			remoteBuildpack, err := buildpackRepo.FindByIdAndVersion(buildpack.Id, buildpack.Version)
			if err != nil {
				return buildapi.BuilderRecord{}, err
			}

			buildpacks = append(buildpacks, remoteBuildpack.Optional(buildpack.Optional))
		}
		builderBldr.AddGroup(buildpacks...)
	}

	writeableImage, err := builderBldr.WriteableImage()
	if err != nil {
		return buildapi.BuilderRecord{}, err
	}

	identifier, err := r.RegistryClient.Save(keychain, spec.Tag, writeableImage)
	if err != nil {
		return buildapi.BuilderRecord{}, err
	}

	config, err := writeableImage.ConfigFile()
	if err != nil {
		return buildapi.BuilderRecord{}, err
	}

	return buildapi.BuilderRecord{
		Image: identifier,
		Stack: buildapi.BuildStack{
			RunImage: clusterStack.Status.RunImage.LatestImage,
			ID:       clusterStack.Status.Id,
		},
		Buildpacks:              buildpackMetadata(builderBldr.buildpacks()),
		Order:                   builderBldr.order,
		ObservedStackGeneration: clusterStack.Status.ObservedGeneration,
		ObservedStoreGeneration: clusterStore.Status.ObservedGeneration,
		OS:                      config.OS,
	}, nil
}

func buildpackMetadata(buildpacks []DescriptiveBuildpackInfo) buildapi.BuildpackMetadataList {
	m := make(buildapi.BuildpackMetadataList, 0, len(buildpacks))
	for _, b := range buildpacks {
		m = append(m, buildapi.BuildpackMetadata{
			Id:       b.Id,
			Version:  b.Version,
			Homepage: b.Homepage,
		})
	}
	return m
}
