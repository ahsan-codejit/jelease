<!--
SPDX-FileCopyrightText: 2022 Risk.Ident GmbH <contact@riskident.com>

SPDX-License-Identifier: CC-BY-4.0
-->

<p align="center">
  <img src="./docs/jelease-gopher-card-512.jpg" alt="jelease gopher logo"/>
</p>

<h1 align="center">jelease - A newreleases.io ➡️ Jira connector</h1>

[![REUSE status](https://api.reuse.software/badge/github.com/RiskIdent/jelease)](https://api.reuse.software/info/github.com/RiskIdent/jelease)

Automatically create Jira tickets when a newreleases.io release
is detected using webhooks.

## Configuration

The application requires the following environment variables to be set:

<!--lint disable maximum-line-length-->

- Connection and authentication

  - `JELEASE_AUTH_TYPE`: One of \[pat, token]. Determines whether to authenticate using personal access token (on premise) or jira api token (jira cloud)
  - `JELEASE_JIRA_TOKEN`: Jira API token, can also be a password in self-hosted instances
  - `JELEASE_JIRA_URL`: The URL of your Jira instance
  - `JELEASE_JIRA_USER`: Jira username to authenticate API requests
  - `JELEASE_PORT`: The port the application is expecting traffic on
  - `JELEASE_INSECURE_SKIP_CERT_VERIFY`: Skips verification of Jira server certs when performing http requests.

- Jira ticket creation:

  - `JELEASE_ADD_LABELS`: Comma-separated list of labels to add to the created jira ticket
  - `JELEASE_DEFAULT_STATUS`: The status the created tickets are supposed to have
  - `JELEASE_DRY_RUN`: Don't create tickets, log when a ticket would be created
  - `JELEASE_ISSUE_DESCRIPTION`: The description for created issues
  - `JELEASE_ISSUE_TYPE`: The issue type for created issues. E.g `Task`, `Story` (default), or `Bug`
  - `JELEASE_PROJECT`: Jira Project key the tickets will be created in
  - `JELEASE_PROJECT_NAME_CUSTOM_FIELD`: Custom field ID (uint) to store project ID in. If left at 0 (default) then Jelease will use labels instead.
  - `JELEASE_LOG_FORMAT`: Logging format. One of: `pretty` (default), `json`
  - `JELEASE_LOG_LEVEL`: Logging minimum level/severity. One of: `trace`, `debug` (default), `info`, `warn`, `error`, `fatal`, `panic`

<!--lint enable maximum-line-length-->

They can also be specified using a `.env` file in the application directory.

## Local usage

1. Populate a `.env` file with configuration values
2. `go run main.go` / `./jelease`
3. Direct newreleases.io webhooks to the `host:port/webhook` route.

## Building the application and docker image

The application uses [earthly](https://earthly.dev/get-earthly) for building
and pushing a docker image.

After installing earthly, the image can be built by running

```bash
earhtly +docker

# if you want to push a new image version
earhtly --push +docker --VERSION=v0.4.1
```

You can also persist build flags in a `.env` file, e.g:

```properties
# Inside the .env file
REGISTRY=docker.io/my-username
```

## Deployment

**TODO**

## Logo

The gopher logo is designed by an employee at [Risk.Ident](https://riskident.com).

The gopher logo of Jelease was inspired by the original Go gopher,
designed by [Renee French](https://reneefrench.blogspot.com/).

## License

This repository complies with the [REUSE recommendations](https://reuse.software/).

Different licenses are used for different files. In general:

- Go code is licensed under GNU General Public License v3.0 or later ([LICENSES/GPL-3.0-or-later.txt](LICENSES/GPL-3.0-or-later.txt)).
- Documentation licensed under Creative Commons Attribution 4.0 International ([LICENSES/CC-BY-4.0.txt](LICENSES/CC-BY-4.0.txt)).
- The logo is licensed under Creative Commons Attribution 4.0 International ([LICENSES/CC-BY-4.0.txt](LICENSES/CC-BY-4.0.txt)).
- Miscellaneous files, e.g `.gitignore`, are licensed under CC0 1.0 Universal ([LICENSES/CC0-1.0.txt](LICENSES/CC0-1.0.txt)).

Please see each file's header or accompanied `.license` file for specifics.
