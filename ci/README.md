# GCS Resource Concourse Pipeline

In order to run the GCS Resource Concourse Pipeline you must have an existing [Concourse](http://concourse.ci) environment.

* Target your Concourse CI environment:

```
fly -t <CONCOURSE TARGET NAME> login -c <YOUR CONCOURSE URL>
```

* Update the [credentials.yml](https://github.com/frodenas/gcs-resource/blob/master/ci/credentials.yml) file.

* Set the GCS Resource Concourse Pipeline:

```
fly -t <CONCOURSE TARGET NAME> set-pipeline -p gcs-resource -c pipeline.yml -l credentials.yml
```

* Unpause the GCS Resource Concourse Pipeline:

```
fly -t <CONCOURSE TARGET NAME> unpause-pipeline -p gcs-resource
```
