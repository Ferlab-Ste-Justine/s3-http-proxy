# About

A minimalist s3 http proxy implementation with a small set of very well established direct dependencies that is easy to audit and trust.

It supports serving any path in a bucket with the following optional features:
- Tls support
- Basic auth support
- Adding a path prefix to the download urls

# Usage

A configuration yaml file (in the process' running directory and named **config.yml** by default, though that behavior is customizable with the **S3_HTTP_PROXY_CONFIG_FILE** environment variable) in the following format should be provided:

```
s3:
  endpoint: <ip>:<port> formatted endpoint for your s3 api
  region: Region of your s3 bucket
  bucket: Your s3 bucket
  tls: Whether the s3 endpoint should be access with https or not
  credentials: Yaml file containing your s3 credentials. The file should contain the folllowing keys: access_key, secret_key
server:
  port: Post the server should be listening on
  address: Ip the server should bind on
  basic_auth: Path to yaml basic auth file if you want basic auth
  tls:
    certificate: Path to a server certificate if the server should serve over tls
    key: Path to a server private key if the server should serve over tls
  debug_mode: Whether the server should output debug logs
download_prefix: Prefix that should prepend file path to downloads. For example, if you specify "/download" here and try to download at the path "/download/catA/myfile.txt", then your s3 bucket should contain the key "catA/myfile.txt" 
```

If you are using basic auth, you will also have a basic auth file that looks like this:

```
<username1>: <password1>
<username2>: <password2>
...
```