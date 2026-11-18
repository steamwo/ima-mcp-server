# Security

## Credential boundary

`IMA_OPENAPI_CLIENTID` / `IMA_OPENAPI_APIKEY` are sent only to the configured IMA OpenAPI base URL. The default and recommended base URL is `https://ima.qq.com`.

A custom base URL is rejected unless `IMA_ALLOW_CUSTOM_BASE_URL=1` is explicitly set. This switch is intended only for trusted test endpoints because enabling it allows credentials to be sent to that endpoint.

COS uploads use only the short-lived credentials returned by IMA `create_media`; the IMA Client ID and API Key are never sent to COS.

## Network fetching

Remote image and URL ingestion blocks loopback, private, link-local, multicast, unspecified, carrier-grade NAT, and benchmark IP ranges. Redirects are limited and revalidated. IMA credentials are never attached to these requests.

## HTTP transport

The Streamable HTTP mode has no built-in end-user authentication. It defaults to `127.0.0.1`. If it is deployed remotely, place it behind TLS and an authenticated reverse proxy and treat the configured IMA account as shared service credentials.

## Reporting vulnerabilities

Please report vulnerabilities privately to the repository owner instead of opening a public issue when disclosure could expose credentials or user data.
