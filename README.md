# GCS Resource [![Build Status](https://travis-ci.org/frodenas/gcs-resource.png)](https://travis-ci.org/frodenas/gcs-resource)

Versions objects in a [Google Cloud Storage][gcs] (GCS) bucket, by pattern-matching filenames to identify version numbers.

This resource is based on the official [S3 resource][s3-resource].

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

* `versioned_file`: if you [enable versioning][gsc-versioning] for your GCS bucket then you can keep the file name the same and upload new versions of your file without resorting to version numbers. This property is the path to the file in your GCS bucket.

### Initial state

If no resource versions exist you can set up this resource to emit an initial version with a specified content. This won't create a real resource in GCS but only create an initial version for Concourse. The resource file will be created as usual when you `get` a resource with an initial version.

You can define one of the following two options:

* `initial_path`: *Optional.* Must be used with the `regexp` option. You should set this to the file path containing the initial version which would match the given regexp. E.g. if `regexp` is `file/build-(.*).zip`, then `initial_path` might be `file/build-0.0.0.zip`. The resource version will be `0.0.0` in this case.

* `initial_version`: *Optional.* Must be used with the `versioned_file` option. This will be the resource version.

By default the resource file will be created with no content when `get` runs. You can set the content by using one of the following options:

* `initial_content_text`: *Optional.* Initial content as a string.

* `initial_content_binary`: *Optional.* You can pass binary content as a base64-encoded string.


## Behavior

### `check`: Extract versions from the bucket.

Objects will be found via the pattern configured by `regexp` or `versioned_file`. The versions will be used to order them (using [semver][semver]). Each object's filename is the resulting version.

### `in`: Fetch an object from the bucket.

Places the following files in the destination:

* `(filename)`: the file fetched from the bucket.

* `url`: a file containing the URL of the object.

* `version`: the version identified in the file name (only if using `regexp`).

* `generation`: the object's generation (only if using `versioned_file`).

#### Parameters

* `skip_download`: *Optional.* If true, skip downloading object from GCS.

  This is useful to trigger a job that does not utilize the file, or to skip the implicit `get` after uploading a file to GCS using `put` (using `get_params`).

* `unpack`: *Optional.* If true and the file is an archive (tar, gzipped tar, other gzipped file, or zip), unpack the file. Gzipped tarballs will be both ungzipped and untarred.

### `out`: Upload an object to the bucket.

Given a file specified by `file`, upload it to the GCS bucket. If `regexp` is
specified, the new file will be uploaded to the directory that the regex
searches in. If `versioned_file` is specified, the new file will be uploaded as
a new version of that file.

#### Parameters

* `file` (*required*): path to the file to upload, provided by an output of a
  task. If multiple files are matched by the glob, an error is raised. The file which matches will be placed into the directory structure on GCS as defined in `regexp` in the resource definition. The matching syntax is bash glob expansion, so no capture groups, etc.

* `predefined_acl` (*optional*): the [predefined ACL][gcs-acls] for the object. Acceptable values are:
  - `authenticatedRead`: Object owner gets OWNER access, and allAuthenticatedUsers get READER access.
  - `bucketOwnerFullControl`: Object owner gets OWNER access, and project team owners get OWNER access.
  - `bucketOwnerRead`: Object owner gets OWNER access, and project team owners get READER access.
  - `private`: Object owner gets OWNER access.
  - `projectPrivate`: Object owner gets OWNER access, and project team members get access according to their roles.
  - `publicRead`: Object owner gets OWNER access, and allUsers get READER access.
  - `publicReadWrite`: Object owner gets OWNER access, and allUsers get READER and WRITER access.

* `content_type` (*optional*): sets the MIME type for the object to be uploaded, eg. `application/octet-stream`.

* `cache_control` (*optional*): sets the Cache-Control directive for the object to be uploaded.

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
    content_type: application/octet-stream
    cache_control: max-age=3600
```

## Development

* Get the resource: `go get github.com/frodenas/gcs-resource`
* Run the `unit-tests`: `make`
* Run the `integration-tests`: `make integration-tests`
* Build the source code using concourse: `fly -t ConcourseTarget execute -c ci/tasks/build.yml -i gcs-resource-src=. -o built-resource=.`
* Build  the Docker image to be use inside your pipeline: `docker build -t frodenas/gcs-resource .`

## Contributing

Refer to the [contributing guidelines][contributing].

## License

Apache License 2.0, see [LICENSE][license].

[contributing]: https://github.com/frodenas/gcs-resource/blob/master/CONTRIBUTING.md
[gcs]: https://cloud.google.com/storage/
[gcs-acls]: https://cloud.google.com/storage/docs/access-control/lists#predefined-acl
[gsc-versioning]: https://cloud.google.com/storage/docs/object-versioning#_Enabling
[license]: https://github.com/frodenas/gcs-resource/blob/master/LICENSE
[s3-resource]: https://github.com/concourse/s3-resource
[semver]: http://semver.org/
