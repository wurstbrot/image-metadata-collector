#!/bin/bash
#shellcheck disable=SC2034
# sdase-version-collertor container image

# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.

# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.

# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.
set -xe

trap cleanup INT EXIT
cleanup() {
  (test -n "${base_img}" && buildah rm "${base_img}") || true
  (test -n "${golang_img}" && buildah rm "${golang_img}") || true
}

### Scratch Build

case "$(uname -i)" in
  x86_64|amd64)
    arch="x86_64"
    build="amd64";;
  aarch64|arm*)
    arch="aarch64"
    build="arm64";;
  *)
    echo "unsupported: $(uname -i)"
    exit 1;;
esac

dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
build_dir="${dir}/build"

buildah_from_options=""
if [ -n "$1" ]; then
  buildah_from_options="${buildah_from_options} --creds $1"
fi

base_img="$( buildah from --pull --platform=linux/${build} gcr.io/distroless/static-debian11 )"
base_mnt="$( buildah mount "${base_img}" )"

golang_img="$( buildah from --pull --quiet golang:1.18 )"
golang_mnt="$( buildah mount "${golang_img}" )"
mkdir "${golang_mnt}/go/src/app"
cp -r "./go.mod" "./go.sum" "./cmd" "./internal" "${golang_mnt}/go/src/app/"
buildah run "${golang_img}" -- /bin/bash -c "cd /go/src/app/ && go get -d -v ./..."
buildah run "${golang_img}" -- /bin/bash -c "cd /go/src/app/ && GOARCH=${build} GOOS=linux go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest && ls -la && GOARCH=${build} GOOS=linux cyclonedx-gomod mod -json=true -output /bom.json"

buildah run "${golang_img}" -- /bin/bash -c "cd /go/src/app/ && GOOS=linux GOARCH=${build} CGO_ENABLED=0 go build -o /go/bin/app cmd/collector/main.go"
cp -r "./internal/cmd/imagecollector/configs/" "${base_mnt}/configs"
buildah copy --chown 1001:1001 "${base_img}" "${golang_mnt}/go/bin/app" "/app"
buildah copy --chown 1001:1001 "${base_img}" "${golang_mnt}/bom.json" "/bom.json"
buildah umount "${golang_img}"
buildah rm "${golang_img}"

revision="$( git rev-parse HEAD )"
# Get bill of materials hash – the content
# of this script is included in hash, too.
bill_of_materials_hash="$( ( cat "${0}";
  echo "${revision}" \
) | sha256sum | awk '{ print $1; }' )"

oci_prefix="org.opencontainers.image"

descr="sdase-version-collector Image"

buildah config \
  --label "${oci_prefix}.authors=SDA SE Engineers <engineers@sda-se.io>" \
  --label "${oci_prefix}.url=https://quay.io/sdase/sdase-version-collector" \
  --label "${oci_prefix}.source=https://github.com/SDA-SE/sdase-version-collector" \
  --label "${oci_prefix}.revision=${revision}" \
  --label "${oci_prefix}.vendor=SDA SE Open Industry Solutions" \
  --label "${oci_prefix}.licenses=MIT" \
  --label "${oci_prefix}.title=sdase-version-collector" \
  --label "${oci_prefix}.description=${descr}" \
  --label "io.sda-se.image.bill-of-materials-hash=${bill_of_materials_hash}" \
  --env ANNOTATION_NAME_ENGAGEMENT_TAG="clusterscanner.sdase.org/engagement-tags" \
  --env DEFAULT_ENGAGEMENT_TAGS="cluster-image-scanner" \
  --user 1001 \
  --cmd '["/app"]' \
  --author "SDA SE Engineers" \
  --created-by "DevOps 5xx" \
  "${base_img}"

image="sdase-version-collector"
# create a individual image id
image_build="${image}.${RANDOM}"
buildah commit --quiet --rm "${base_img}" "${image_build}" && base_img=

if [ -n "${BUILD_EXPORT_OCI_ARCHIVES}" ]
then
  mkdir --parent "${build_dir}"
  buildah push --quiet "${image_build}" \
    "oci-archive:${build_dir}/${image//:/-}.tar"

  buildah rmi "${image_build}"
fi

cleanup
