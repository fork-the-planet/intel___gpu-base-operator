/*
Copyright 2026 Intel Corporation. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	core "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha "github.com/intel/gpu-base-operator/api/v1alpha1"
)

// ContentImageVerifier checks whether the firmware content image is reachable and,
// when checksums are declared, verifies each file's SHA256 against the image contents.
type ContentImageVerifier interface {
	Verify(ctx context.Context, spec *v1alpha.GPUFirmwareUpdateSpec) error
}

// DefaultContentImageVerifier implements ContentImageVerifier using the OCI registry API.
type DefaultContentImageVerifier struct {
	k8sReader client.Reader
	namespace string
}

func newContentImageVerifier(k8sReader client.Reader, namespace string) ContentImageVerifier {
	return &DefaultContentImageVerifier{
		k8sReader: k8sReader,
		namespace: namespace,
	}
}

// Verify checks image reachability. All files are checked for existence. If a file declares a Checksum, the
// checksum is verified against the /fwupdate/<filename> content.
func (v *DefaultContentImageVerifier) Verify(ctx context.Context, spec *v1alpha.GPUFirmwareUpdateSpec) error {
	ref, err := name.ParseReference(spec.Content.ContainerImage)
	if err != nil {
		return fmt.Errorf("invalid content image reference %q: %w", spec.Content.ContainerImage, err)
	}

	keychain, err := v.buildKeychain(ctx, spec.ImagePullSecret)
	if err != nil {
		return fmt.Errorf("failed to build registry auth: %w", err)
	}

	remoteOpts := []remote.Option{
		remote.WithAuthFromKeychain(keychain),
		remote.WithContext(ctx),
	}

	if spec.InsecureSkipTLSVerify {
		defaultTransport, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			return fmt.Errorf("unexpected default transport type: %T", http.DefaultTransport)
		}

		insecureTransport := defaultTransport.Clone()
		if insecureTransport.TLSClientConfig != nil {
			insecureTransport.TLSClientConfig = insecureTransport.TLSClientConfig.Clone()
		} else {
			insecureTransport.TLSClientConfig = &tls.Config{}
		}

		//nolint:gosec // InsecureSkipVerify is intentionally requested by the user via the CR field.
		insecureTransport.TLSClientConfig.InsecureSkipVerify = true

		remoteOpts = append(remoteOpts, remote.WithTransport(insecureTransport))
	}

	// Full checksum verification: pull and stream the merged filesystem.
	img, err := remote.Image(ref, remoteOpts...)
	if err != nil {
		return fmt.Errorf("failed to pull content image %q: %w", spec.Content.ContainerImage, err)
	}

	pr, pw := io.Pipe()

	go func() {
		if exportErr := crane.Export(img, pw); exportErr != nil {
			pw.CloseWithError(exportErr)
		} else {
			if closeErr := pw.Close(); closeErr != nil {
				klog.Warningf("failed to close pipe writer after image export: %v", closeErr)
			}
		}
	}()

	verifyErr := verifyChecksumsFromExport(pr, spec.Content.Files)

	// Closing the read end unblocks the export goroutine if it is still running.
	if closeErr := pr.Close(); closeErr != nil && verifyErr == nil {
		return fmt.Errorf("failed to close pipe reader: %w", closeErr)
	}

	return verifyErr
}

// verifyChecksumsFromExport reads a merged-filesystem tar (as produced by crane.Export)
// and checks that each firmware file is present. If a firmware file declares a Checksum,
// it verifies the SHA256 of that file.
// It returns a distinct error for a missing file versus a checksum mismatch.
func verifyChecksumsFromExport(tarStream io.Reader, files []v1alpha.GPUFirmwareFile) error {
	want := make(map[string]string, len(files))

	for _, f := range files {
		// Empty Checksum is treated as "not available"
		want[f.FileName] = f.Checksum
	}

	found := map[string]bool{}

	tr := tar.NewReader(tarStream)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("error reading image filesystem: %w", err)
		}

		// crane.Export paths look like "fwupdate/file.bin" or "./fwupdate/file.bin".
		base := strings.TrimPrefix(hdr.Name, "./")
		if !strings.HasPrefix(base, "fwupdate/") {
			continue
		}

		base = strings.TrimPrefix(base, "fwupdate/")

		expected, ok := want[base]
		if !ok {
			continue
		}

		if expected != "" {
			if hdr.Typeflag != tar.TypeReg {
				return fmt.Errorf("firmware file %q found in image but is not a regular file", base)
			}

			h := sha256.New()
			if _, err := io.Copy(h, tr); err != nil {
				return fmt.Errorf("failed to hash %q from image: %w", base, err)
			}

			actual := fmt.Sprintf("sha256:%x", h.Sum(nil))
			if actual != expected {
				return fmt.Errorf("checksum mismatch for %q: expected %s, got %s", base, expected, actual)
			}

			klog.V(2).Infof("Verified checksum for %q: %s", base, actual)
		} else {
			klog.V(2).Infof("Verified existence for %q without checksum", base)
		}

		found[base] = true

		// All files found and verified, no need to continue reading the tar.
		if len(found) == len(want) {
			break
		}
	}

	for filename := range want {
		if !found[filename] {
			return fmt.Errorf("firmware file %q not found under /fwupdate/ in the content image", filename)
		}
	}

	return nil
}

// dockerConfigJSON mirrors the .dockerconfigjson secret format.
type dockerConfigJSON struct {
	Auths map[string]authn.AuthConfig `json:"auths"`
}

// dockerConfigKeychain implements authn.Keychain from a parsed .dockerconfigjson secret.
type dockerConfigKeychain struct {
	auths map[string]authn.AuthConfig
}

func (kc *dockerConfigKeychain) Resolve(r authn.Resource) (authn.Authenticator, error) {
	registry := r.RegistryStr()

	for host, auth := range kc.auths {
		// Docker config hosts may include an https:// scheme prefix; strip it for matching.
		h := strings.TrimPrefix(host, "https://")
		h = strings.TrimPrefix(h, "http://")

		if h == registry || host == registry {
			return authn.FromConfig(auth), nil
		}
	}

	return authn.Anonymous, nil
}

func (v *DefaultContentImageVerifier) buildKeychain(ctx context.Context, pullSecretName string) (authn.Keychain, error) {
	if pullSecretName == "" {
		return authn.DefaultKeychain, nil
	}

	secret := &core.Secret{}
	if err := v.k8sReader.Get(ctx, client.ObjectKey{Name: pullSecretName, Namespace: v.namespace}, secret); err != nil {
		return nil, fmt.Errorf("failed to get pull secret %q: %w", pullSecretName, err)
	}

	data, ok := secret.Data[core.DockerConfigJsonKey]
	if !ok {
		return nil, fmt.Errorf("pull secret %q missing key %q", pullSecretName, core.DockerConfigJsonKey)
	}

	cfg := &dockerConfigJSON{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse pull secret %q: %w", pullSecretName, err)
	}

	return &dockerConfigKeychain{auths: cfg.Auths}, nil
}
