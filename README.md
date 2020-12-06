# img2sto

Image To Storage proxy for uploading/downloading images to/from Minio storage with resizing on-the-fly.

---

## Commands

For running, testing, deploying see [Makefile](./Makefile).


## API Docs

`make doc`


## Testing

Currently you need to manually run `make docker-run-dev` before tests,
for launching minio & imaginary containers.


## HTTP Communication

A = your app
P = img2sto
R = resizer
S = storage

Upload: multipart/form-data file (A) -> P -> R -> P -> S -> P -> image url

Download: browser url (A) -> P -> S -> P -> R -> P -> (A)


## Requirements

Resizer: [Imaginary](https://github.com/h2non/imaginary)

Storage: [Minio](https://github.com/minio/minio) required!


## Resizing on-the-fly

Set width and (optionally) height in query params to resize image on-the-fly:

{protocol}://{host}/{id}?resize={W|WxH}

http://localhost:8080/{bucketName}/ec0c5dbc-e4dc-44d6-b12a-55380e8eebbf.png?resize=600
http://localhost:8080/{bucketName}/ec0c5dbc-e4dc-44d6-b12a-55380e8eebbf.png?resize=600x200


## Development

`docker run minio/mc alias set minio http://127.0.0.1:8111 {ACCESS_KEY} {ACCESS_TOKEN} --api S3v4`
