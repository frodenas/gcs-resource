# GCS Resource

Versions objects in a [Google Cloud Storage](https://cloud.google.com/storage/) (GCS) bucket, by pattern-matching filenames to identify version numbers.

This resource is based on the official [S3 resource](https://github.com/concourse/s3-resource).

## Source Configuration

* `bucket` (*required*): the name of the bucket.

* `json_key` (*required*): the contents of your GCS Account JSON Key file to use when accessing the bucket. Example:
  ```
  json_key: |
    {
      "private_key_id": "...",
      "private_key": "...",
      "client_email": "...",
      "client_id": "...",
      "type": "service_account"
    }
  ```

### File Names

One of the following two options must be specified:

* `regexp`: the pattern to match filenames against within GCS. The first grouped match is used to extract the version, or if a group is explicitly named `version`, that group is used.

  The version extracted from this pattern is used to version the resource. Semantic versions, or just numbers, are supported. Accordingly, full regular expressions are supported, to specify the capture groups.

* `versioned_file`: if you [enable versioning](https://cloud.google.com/storage/docs/object-versioning#_Enabling) for your GCS bucket then you can keep the file name the same and upload new versions of your file without resorting to version numbers. This property is the path to the file in your GCS bucket.

## Behavior

### `check`: Extract versions from the bucket.

Objects will be found via the pattern configured by `regexp` or `versioned_file`. The versions will be used to order them (using [semver](http://semver.org/)). Each object's filename is the resulting version.

### `in`: Fetch an object from the bucket.

Places the following files in the destination:

* `(filename)`: the file fetched from the bucket.

* `url`: a file containing the URL of the object.

* `version`: the version identified in the file name (only if using `regexp`).

* `generation`: the object's generation (only if using `versioned_file`).

#### Parameters

*None*

### `out`: Upload an object to the bucket.

Given a file specified by `file`, upload it to the GCS bucket. If `regexp` is
specified, the new file will be uploaded to the directory that the regex
searches in. If `versioned_file` is specified, the new file will be uploaded as
a new version of that file.

#### Parameters

* `file` (*required*): path to the file to upload, provided by an output of a
  task. If multiple files are matched by the glob, an error is raised. The file which matches will be placed into the directory structure on GCS as defined in `regexp` in the resource definition. The matching syntax is bash glob expansion, so no capture groups, etc.

* `predefined_acl` (*optional*): the predefined ACL for the object. Acceptable values are:
  - `authenticatedRead`: Object owner gets OWNER access, and allAuthenticatedUsers get READER access.
  - `bucketOwnerFullControl`: Object owner gets OWNER access, and project team owners get OWNER access.
  - `bucketOwnerRead`: Object owner gets OWNER access, and project team owners get READER access.
  - `private`: Object owner gets OWNER access.
  - `projectPrivate`: Object owner gets OWNER access, and project team members get access according to their roles.
  - `publicRead`: Object owner gets OWNER access, and allUsers get READER access.

## Example Configuration

### Resource Type

```yaml
resource_types:
  - name: gcs-resource
    type: docker-image
    source:
      repository: frodenas/gcs-resource
```

### Resource

``` yaml
resources:
  - name: release
    type: gcs-resource
    source:
      bucket: releases
      json_key: <GCS-ACCOUNT-JSON-KEY-CONTENTS>
      regexp: directory_on_gcs/release-(.*).tgz
```

### Plan

``` yaml
- get: release
```

``` yaml
- put: release
  params:
    file: path/to/release-*.tgz
    predefined_acl: publicRead
```

## Developing on this resource

First get the resource via: `go get github.com/frodenas/gcs-resource`

Run the `unit-tests`: `make`

Run the `integration-tests`: `make integration-tests`

## Developing using Concourse
Clone this repository and just run one-off task with concourse

```bash
fly -t ConcourseTarget execute -c build.yml -i gcs-resource=. -o built-resource=.
```


Just build the Docker image to be use inside your pipeline

```bash
 docker build -t frodenas/gcs-resource .
```


