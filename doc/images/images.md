# Images

The Oracle Cloud Native Operating System is distributed in a variety of formats.
The most common formats are a bootable image and a container image that contains
an ostree commit.  The bootable image is used to create boot media for
virtualized platforms.  The ostree image is used to serve updates to existing
installations as well as server as the basis for an ostree archive for
customized installations.

Creating images requires access to a Kubernetes cluster.  Any running cluster
can be used.  If there is no cluster available, an ephemeral cluster is created
automatically using the libvirt provider with the default configuration.  Image
conversion requires a significant amount of space.  It is recommended to allocate
at least 20Gi of storage to any cluster nodes.  See `ocne-config.yaml(5)` and
`ocne-defaults.yaml(5)` for details.

## Bootable Images

The bootable image is nothing but a container image that contains a single
virtual machine image in the qcow2 format.  This image is used as the boot media
for clusters created with the libvirt and oci providers.  By default, the image
is configured to work with the libvirt provider.  A conversion process must be
performed to allow the image to boot in Oracle Cloud Infrastructure (OCI).  In
most cases, the conversion is performed automatically.  It is possible to
manually convert and import the image to OCI if required.

The image downloaded directly from a container registry is in the qcow2 format
and is suitable for launching virtual machines.  In some cases, the image must
be edited to allow it to be used on other platforms.

### OCI Compute Images

Creating clusters with the oci provider requires a custom compute image in the
target compartment.  The bootable container image must be customized to work
properly in OCI, and must be converted into an appropriate format.  Once an
appropriate image has been created, it must be imported to the target
compartment.

#### Creating an OCI VM Image

Images can be created using the `ocne image create` command with the `oci` type.
The result can be imported to OCI and used as the boot volume for an OCI compute
instance.

By default, the image is created for the architecture of the system where the
command is executed.
```
$ ocne image create -t oci
INFO[2024-07-05T15:57:23Z] Creating Image                               
INFO[2024-07-05T15:58:22Z] Preparing pod used to create image           y: ok 
INFO[2024-07-05T15:58:38Z] Waiting for pod ocne-system/ocne-image-builder to be ready: ok 
INFO[2024-07-05T15:58:38Z] Getting local boot image for architecture: amd64 
Getting image source signatures
Copying blob 7872e1e151ed done   | 
Copying config de749e691d done   | 
Writing manifest to image destination
INFO[2024-07-05T15:59:33Z] Uploading boot image to pod ocne-system/ocne-image-builder: ok 
INFO[2024-07-05T16:00:38Z] Downloading boot image from pod ocne-system/ocne-image-builder: ok 
INFO[2024-07-05T16:00:38Z] New boot image was created successfully at /home/opc/.ocne/images/boot.qcow2-1.28.3-amd64.oci 
```

Images can be created for other architectures by providing the appropriate argument
```
$ ocne image create -t oci -a arm64
INFO[2024-07-05T16:01:56Z] Creating Image                               
INFO[2024-07-05T16:02:45Z] Preparing pod used to create image           y: ok 
INFO[2024-07-05T16:03:32Z] Waiting for pod ocne-system/ocne-image-builder to be ready: ok 
INFO[2024-07-05T16:03:32Z] Getting local boot image for architecture: arm64 
Getting image source signatures
Copying blob 3beb0eb62dea done   | 
Copying config f75a36c0ca done   | 
Writing manifest to image destination
INFO[2024-07-05T16:04:14Z] Uploading boot image to pod ocne-system/ocne-image-builder: ok 
INFO[2024-07-05T16:05:17Z] Downloading boot image from pod ocne-system/ocne-image-builder: ok 
INFO[2024-07-05T16:05:17Z] New boot image was created successfully at /home/opc/.ocne/images/boot.qcow2-1.28.3-arm64.oci
```

#### Uploading a VM Image to OCI

Once an image is created, it can be uploaded to OCI and imported as a custom
compute image.  First, a converted image is uploaded to an object in an OCI
object storage bucket.  Once the upload is complete, the object is imported
as a custom compute image.  The object in the bucket is left behind and must
be cleaned up manually.

```
$ ocne image upload --arch amd64 --type oci --compartment mycompartment --file /home/opc/.ocne/images/boot.qcow2-1.28.3-amd64.oci
INFO[2024-07-05T16:49:16Z] Uploading image to object storage: ok 
INFO[2024-07-05T17:04:35Z] Importing compute image: [##########]: ok 
INFO[2024-07-05T17:04:35Z] Image OCID is ocid1.image.oc1.iad.aaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
```

### Ostree Archive Images

Custom installation of Oracle Cloud Native Operating System are done using the
Anaconda and Kickstart automatic installtion feature of Oracle Linux.  Anaconda
requires that ostree content be available in a particular format to install onto
the root filesystem of the target host: an ostree archive served over http.  The
ostree container image used for updates is not in this format.  To use the
content with Anaconda, it must be converted to an archive with an http server.

#### Creating an Ostree Archive Container Image

Images can be created using the `ocne image create` command with the `ostree`
type.  The resulting container image is stored in the Open Container Initiative
archive format and can be imported to the local cache or pushed to a container
registry.  Building the container image pulls a base image from within the
Kubernetes cluster that is used by the image creation command.  It is necessary
to set any proxy parameters that are required to access the base image.

By default, the image is created for the architecture of the system where the
command is executed.

```
$ ocne image create --type ostree
INFO[2024-07-05T17:07:55Z] Creating Image                               
INFO[2024-07-05T17:08:46Z] Preparing pod used to create image           y: ok 
INFO[2024-07-05T17:09:07Z] Waiting for pod ocne-system/ocne-image-builder to be ready: ok 
INFO[2024-07-05T17:14:27Z] Generating container image: ok   
INFO[2024-07-05T17:15:45Z] Saving container image: ok       
INFO[2024-07-05T17:15:47Z] Saved image to /home/opc/.ocne/images/ock-1.28.3-amd64-ostree.tar
```

#### Uploading a Container Image

Images can be uploaded to container registry.  A login prompt is provided if
credentials are not already available for the target registry.

```
$ ocne image upload --type ostree --file /home/opc/.ocne/images/ock-1.28.3-amd64-ostree.tar --destination docker://myregistry.com/ock-ostree:latest --arch amd64
Getting image source signatures
Copying blob 3f3139aed2bd [--------------------------------------] 8.0b / 51.2MiB | 524.4 KiB/s
Copying blob 76a7d9b9b348 [--------------------------------------] 8.0b / 1.6GiB | 581.1 KiB/s
INFO[2024-07-05T17:29:58Z] Log in to myregistry.com
Username: myuser
Password: 
Login Succeeded!
Getting image source signatures
Copying blob 3f3139aed2bd done   | 
Copying blob 76a7d9b9b348 done   | 
Copying config e7bd66a3d6 done   | 
Writing manifest to image destination
```

The typical use case for uploading ostree archive images to is move them to
a container image registry.  It is also possible to move the image to any
target that is supported by the Open Container Initiative transports and
formats.  See `containers-transports(5)` for available options.

```
$ ocne image upload --type ostree --file /home/opc/.ocne/images/ock-1.28.3-amd64-ostree.tar --destination dir:ock-ostree --arch amd64
Getting image source signatures
Copying blob 3f3139aed2bd done   | 
Copying blob 76a7d9b9b348 done   | 
Copying config e7bd66a3d6 done   | 
Writing manifest to image destination

ls ock-ostree/
3f3139aed2bde60870b297d50b4e5f7982dd551993ab434260c9af3b2ab57fef  76a7d9b9b3487aedd6a64b926c8c5bdc2aa3a3de9ede1102a849c11e52a56e6f  e7bd66a3d654a7f4e4f05899eeb1e6db7bf5a0f71e54bce73915c06373512ad5  manifest.json  version
```

#### Running the OStree Archive Container Image Locally

The archive can be loaded into the local cache and executed with a container
runtime.  The container runs an instance of nginx and serves the static ostree
archive.  It is useful to expose the nginx server port.

```
# Load the archive into the local cache
$podman load < /home/opc/.ocne/images/ock-1.28.3-amd64-ostree.tar
Getting image source signatures
Copying blob 3f3139aed2bd done  
Copying blob 76a7d9b9b348 done  
Copying config e7bd66a3d6 done  
Writing manifest to image destination
Storing signatures
Loaded image: localhost/ock-ostree:latest

# Start a container
$ podman run -d -p 8080:80 localhost/ock-ostree:latest
144505def639759ea757173bdcd13718180a50757a6481d150ef8a8724009110

# Fetch the ostree commit of the osnos ref
$ curl http://localhost:8080/ostree/refs/heads/ock
789672394f9c0242ec602191cc6f2f808bf8476686256aa71556a11bdf6695db
```
