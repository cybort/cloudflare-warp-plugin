Cloudflare Warp Plugin for Caddy
================================

Thisis a native Caddy integration of the [Cloudflare Warp](https://www.cloudflare.com/products/cloudflare-warp/) client.

Serve your site on the Internet without exposing your server to the Internet!

More information about Warp:

- Blog post: [Introducing Cloudflare Warp: Hide Behind The Edge](https://blog.cloudflare.com/introducing-cloudflare-warp/)
- [Warp documentation](https://developers.cloudflare.com/warp/)

**THIS IS A WORK-IN-PROGRESS. Warp is currently in beta.** This plugin is functional, but currently requires a modified version of the official Warp client library.


## Installation

Currently, you must have [this proposed revision](https://github.com/cloudflare/cloudflare-warp/pull/3) of the Cloudflare Warp client in your GOPATH.

Then you must [apply this patch](https://github.com/mholt/caddy/pull/2048) to Caddy so that it understands the `warp` directive. (It just adds one line of code to a list.)

Finally, you will have to plug in this plugin by adding

```
	_ "github.com/caddyserver/cloudflare-warp-plugin"
```

to [the imports in run.go](https://github.com/mholt/caddy/blob/5552dcbbc7f630ada7c7d030b37c2efdce750ace/caddy/caddymain/run.go#L37).

Then you can run `go run build.go` and then a Caddy binary will be made with this plugin installed.


## Usage

First, ensure that you're participating in the Cloudflare Warp beta program. We'll defer the other prerequisites (for example, using a domain with an [active zone](https://support.cloudflare.com/hc/en-us/articles/201720164-Step-2-Create-a-Cloudflare-account-and-add-a-website) on Cloudflare) to the [Warp docs](https://developers.cloudflare.com/warp/).

To use it with Caddy, simply add the `warp` directive to a site you want to warp. Here's an example Caddyfile:

```
example.com
warp
```

For simple sites, you can also run Caddy with `warp` like this without a Caddyfile:

```
caddy -host example.com warp
```

The first time you start Caddy with this plugin, it will open a browser tab and ask you to log in to Cloudflare. Then you will have to authorize Warp. This is a one-time thing: a certificate credential (.pem file) will be downloaded and placed in your .caddy folder. Once you have that certificate, Caddy will reuse it. It will also use a certificate obtained from the official client, if one exists in its default location.

Note that your site will be served only locally and without HTTPS. That's OK: Caddy makes an outbound connection with Cloudflare that serves as an encrypted tunnel to their edge nodes. The outside world accesses your site over HTTPS to Cloudflare, and Cloudflare accesses your local server through the TLS-encrypted tunnel.

By default, Warp allows your site to be accessed over both HTTP and HTTPS. We suggest that you [always use HTTPS](https://blog.cloudflare.com/how-to-make-your-site-https-only/). (TODO: Is there a way for Caddy--i.e. the client--to enforce this?)

